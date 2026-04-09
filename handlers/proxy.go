package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ladder/pkg/ruleset"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
)

// FlareSolverrRequest represents the request structure for FlareSolverr API
type FlareSolverrRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
}

// FlareSolverrResponse represents the response structure from FlareSolverr API
type FlareSolverrResponse struct {
	Solution struct {
		URL     string `json:"url"`
		Status  int    `json:"status"`
		Cookies []struct {
			Name     string  `json:"name"`
			Value    string  `json:"value"`
			Domain   string  `json:"domain"`
			Path     string  `json:"path"`
			Expires  float64 `json:"expires"`
			Size     int     `json:"size"`
			HTTPOnly bool    `json:"httpOnly"`
			Secure   bool    `json:"secure"`
			Session  bool    `json:"session"`
			SameSite string  `json:"sameSite"`
		} `json:"cookies"`
		Response string            `json:"response"`
		Headers  map[string]string `json:"headers"`
	} `json:"solution"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

var (
	UserAgent        = getenv("USER_AGENT", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	ForwardedFor     = getenv("X_FORWARDED_FOR", "66.249.66.1")
	flareSolverrHost = os.Getenv("FLARESOLVERR_HOST")
	rulesSet         = ruleset.NewRulesetFromEnv()
	allowedDomains   = []string{}
	defaultTimeout   = 15 // in seconds
)

const ladderParamPrefix = "__ladder_"

type debugUIOptions struct {
	Enabled     bool
	ShowRequest bool
	HasUI       bool
	ShowUI      bool
}

func init() {
	allowedDomains = strings.Split(os.Getenv("ALLOWED_DOMAINS"), ",")
	if os.Getenv("ALLOWED_DOMAINS_RULESET") == "true" {
		allowedDomains = append(allowedDomains, rulesSet.Domains()...)
	}
	if timeoutStr := os.Getenv("HTTP_TIMEOUT"); timeoutStr != "" {
		defaultTimeout, _ = strconv.Atoi(timeoutStr)
	}
}

// extracts a URL from the request ctx. If the URL in the request
// is a relative path, it reconstructs the full URL using the referer header.
func extractUrl(c *fiber.Ctx) (string, error) {
	// try to extract url-encoded
	reqUrl, err := url.QueryUnescape(c.Params("*"))
	if err != nil {
		// fallback
		reqUrl = c.Params("*")
	}

	// Extract the actual path from req ctx
	urlQuery, err := url.Parse(reqUrl)
	if err != nil {
		return "", fmt.Errorf("error parsing request URL '%s': %v", reqUrl, err)
	}

	isRelativePath := urlQuery.Scheme == ""

	// eg: https://localhost:8080/images/foobar.jpg -> https://realsite.com/images/foobar.jpg
	if isRelativePath {
		// Parse the referer URL from the request header.
		refererUrl, err := url.Parse(c.Get("referer"))
		if err != nil {
			return "", fmt.Errorf("error parsing referer URL from req: '%s': %v", reqUrl, err)
		}

		// Extract the real url from referer path
		realUrl, err := url.Parse(strings.TrimPrefix(refererUrl.Path, "/"))
		if err != nil {
			return "", fmt.Errorf("error parsing real URL from referer '%s': %v", refererUrl.Path, err)
		}

		// reconstruct the full URL using the referer's scheme, host, and the relative path / queries
		fullUrl := &url.URL{
			Scheme:   realUrl.Scheme,
			Host:     realUrl.Host,
			Path:     urlQuery.Path,
			RawQuery: urlQuery.RawQuery,
		}

		if os.Getenv("LOG_URLS") == "true" {
			log.Printf("modified relative URL: '%s' -> '%s'", reqUrl, fullUrl.String())
		}
		return fullUrl.String(), nil

	}

	// default behavior:
	// eg: https://localhost:8080/https://realsite.com/images/foobar.jpg -> https://realsite.com/images/foobar.jpg
	return urlQuery.String(), nil
}

// getFlareSolverrCookies retrieves cookies from FlareSolverr for the given URL
func getFlareSolverrCookies(targetURL string) (string, error) {
	if flareSolverrHost == "" {
		return "", fmt.Errorf("FLARESOLVERR_HOST environment variable not set")
	}

	reqBody := FlareSolverrRequest{
		Cmd:        "request.get",
		URL:        targetURL,
		MaxTimeout: 60000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(flareSolverrHost+"/v1", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var fsResp FlareSolverrResponse
	if err := json.NewDecoder(resp.Body).Decode(&fsResp); err != nil {
		return "", err
	}

	if fsResp.Status != "ok" {
		return "", fmt.Errorf("FlareSolverr error: %s", fsResp.Message)
	}

	// Build cookie string from the response
	var cookies []string
	for _, cookie := range fsResp.Solution.Cookies {
		cookies = append(cookies, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}

	return strings.Join(cookies, "; "), nil
}

func ProxySite(rulesetPath string) fiber.Handler {
	if rulesetPath != "" {
		rs, err := ruleset.NewRuleset(rulesetPath)
		if err != nil {
			panic(err)
		}
		rulesSet = rs
	}

	return func(c *fiber.Ctx) error {
		// Get the url from the URL
		url, err := extractUrl(c)
		if err != nil {
			log.Println("ERROR In URL extraction:", err)
		}

		queries := c.Queries()
		proxyQueries, debugOptions := splitProxyQueries(queries)
		body, _, resp, err := fetchSite(url, proxyQueries)
		if err != nil {
			log.Println("ERROR:", err)
			c.SendStatus(fiber.StatusInternalServerError)
			return c.SendString(err.Error())
		}

		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(contentType, "text/html") && shouldInjectDebugUI(debugOptions) {
			body = injectDebugUI(body, debugOptions)
		}

		c.Cookie(&fiber.Cookie{})
		c.Set("Content-Type", resp.Header.Get("Content-Type"))
		c.Set("Content-Security-Policy", resp.Header.Get("Content-Security-Policy"))

		return c.SendString(body)
	}
}

func shouldInjectDebugUI(options debugUIOptions) bool {
	if os.Getenv("DEBUG_UI") == "false" {
		return false
	}

	if options.HasUI {
		return options.ShowUI
	}

	if os.Getenv("DEBUG_UI") == "true" {
		return true
	}

	return true
}

func splitProxyQueries(queries map[string]string) (map[string]string, debugUIOptions) {
	proxyQueries := map[string]string{}
	options := debugUIOptions{}

	for key, value := range queries {
		switch key {
		case "__ladder_debug":
			options.Enabled = value == "1"
			continue
		case "__ladder_show_request":
			options.ShowRequest = value == "1"
			continue
		case "__ladder_ui":
			options.HasUI = true
			options.ShowUI = value == "1"
			continue
		}

		if strings.HasPrefix(key, ladderParamPrefix) {
			continue
		}

		proxyQueries[key] = value
	}

	return proxyQueries, options
}

func injectDebugUI(body string, options debugUIOptions) string {
	if !strings.Contains(strings.ToLower(body), "<html") {
		return body
	}

	debugChecked := ""
	if options.Enabled {
		debugChecked = "checked"
	}

	showRequestChecked := ""
	if options.ShowRequest {
		showRequestChecked = "checked"
	}

	requestInfo := ""
	if options.Enabled {
		requestInfo = `<div id="__ladderDebugInfo" style="margin-top:12px;padding:8px;border:1px solid #cbd5e1;border-radius:8px;font-size:12px;color:#334155;background:#f8fafc;word-break:break-all;">Request: <span id="__ladderRequestValue"></span></div>`
	}

	injection := fmt.Sprintf(`
<button id="__ladderGear" type="button" aria-label="Open Properties" title="Open Properties" style="position:fixed;top:12px;right:12px;z-index:2147483646;width:32px;height:32px;border:1px solid #cbd5e1;border-radius:9999px;background:#fff;color:#475569;display:flex;align-items:center;justify-content:center;box-shadow:0 1px 2px rgba(15,23,42,.12);cursor:pointer;">
	<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
		<path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z"></path>
		<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h.01a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z"></path>
	</svg>
</button>
<aside id="__ladderPanel" aria-hidden="true" style="position:fixed;top:0;right:0;height:100vh;width:min(300px,85vw);transform:translateX(100%%);transition:transform .2s ease-in-out;z-index:2147483647;background:#fff;border-left:1px solid #e2e8f0;padding:12px 14px;font-family:ui-sans-serif,system-ui,-apple-system,'Segoe UI',sans-serif;color:#334155;box-shadow:-1px 0 2px rgba(15,23,42,.08);">
	<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px;">
		<strong style="font-size:14px;">Properties</strong>
		<button id="__ladderClose" type="button" aria-label="Close Properties" style="font-size:18px;line-height:1;color:#64748b;cursor:pointer;">×</button>
	</div>
	<label style="display:flex;align-items:center;justify-content:space-between;gap:10px;font-size:13px;margin-bottom:10px;">
		<span>Debug mode</span>
		<input id="__ladderOptDebug" type="checkbox" data-param="__ladder_debug" %s>
	</label>
	<label style="display:flex;align-items:center;justify-content:space-between;gap:10px;font-size:13px;margin-bottom:10px;">
		<span>Show request</span>
		<input id="__ladderOptRequest" type="checkbox" data-param="__ladder_show_request" %s>
	</label>
	%s
</aside>
<script>
(function () {
	var gear = document.getElementById('__ladderGear');
	var panel = document.getElementById('__ladderPanel');
	var closeBtn = document.getElementById('__ladderClose');
	var controls = panel ? panel.querySelectorAll('input[data-param]') : [];

	function setOpen(isOpen) {
		if (!panel) { return; }
		panel.style.transform = isOpen ? 'translateX(0)' : 'translateX(100%%)';
		panel.setAttribute('aria-hidden', String(!isOpen));
	}

	function applyAndReload(input) {
		var param = input.getAttribute('data-param');
		if (!param) { return; }
		var params = new URLSearchParams(window.location.search);
		if (input.checked) {
			params.set(param, '1');
		} else {
			params.delete(param);
		}
		var next = window.location.pathname;
		var query = params.toString();
		if (query.length > 0) {
			next += '?' + query;
		}
		window.location.href = next;
	}

	if (gear) {
		gear.addEventListener('click', function () {
			var hidden = panel && panel.getAttribute('aria-hidden') === 'true';
			setOpen(hidden);
		});
	}

	if (closeBtn) {
		closeBtn.addEventListener('click', function () { setOpen(false); });
	}

	document.addEventListener('keydown', function (event) {
		if (event.key === 'Escape') {
			setOpen(false);
		}
	});

	document.addEventListener('click', function (event) {
		if (!panel || panel.getAttribute('aria-hidden') === 'true') { return; }
		if (!panel.contains(event.target) && (!gear || !gear.contains(event.target))) {
			setOpen(false);
		}
	});

	for (var i = 0; i < controls.length; i++) {
		controls[i].addEventListener('change', function () { applyAndReload(this); });
	}

	var reqValue = document.getElementById('__ladderRequestValue');
	if (reqValue) {
		reqValue.textContent = decodeURIComponent(window.location.pathname.replace(/^\//, ''));
	}
})();
</script>
`, debugChecked, showRequestChecked, requestInfo)

	bodyClose := regexp.MustCompile(`(?i)</body>`)
	if bodyClose.MatchString(body) {
		return bodyClose.ReplaceAllString(body, injection+"</body>")
	}

	return body + injection
}

func modifyURL(uri string, rule ruleset.Rule) (string, error) {
	newUrl, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	for _, urlMod := range rule.URLMods.Domain {
		re := regexp.MustCompile(urlMod.Match)
		newUrl.Host = re.ReplaceAllString(newUrl.Host, urlMod.Replace)
	}

	for _, urlMod := range rule.URLMods.Path {
		re := regexp.MustCompile(urlMod.Match)
		newUrl.Path = re.ReplaceAllString(newUrl.Path, urlMod.Replace)
	}

	v := newUrl.Query()
	for _, query := range rule.URLMods.Query {
		if query.Value == "" {
			v.Del(query.Key)
			continue
		}
		v.Set(query.Key, query.Value)
	}
	newUrl.RawQuery = v.Encode()

	if rule.GoogleCache {
		newUrl, err = url.Parse("https://webcache.googleusercontent.com/search?q=cache:" + newUrl.String())
		if err != nil {
			return "", err
		}
	}

	return newUrl.String(), nil
}

func fetchSite(urlpath string, queries map[string]string) (string, *http.Request, *http.Response, error) {
	urlQuery := "?"
	if len(queries) > 0 {
		for k, v := range queries {
			urlQuery += k + "=" + v + "&"
		}
	}
	urlQuery = strings.TrimSuffix(urlQuery, "&")
	urlQuery = strings.TrimSuffix(urlQuery, "?")

	u, err := url.Parse(urlpath)
	if err != nil {
		return "", nil, nil, err
	}

	if len(allowedDomains) > 0 && !StringInSlice(u.Host, allowedDomains) {
		return "", nil, nil, fmt.Errorf("domain not allowed. %s not in %s", u.Host, allowedDomains)
	}

	if os.Getenv("LOG_URLS") == "true" {
		log.Println(u.String() + urlQuery)
	}

	// Modify the URI according to ruleset
	rule := fetchRule(u.Host, u.Path)
	url, err := modifyURL(u.String()+urlQuery, rule)
	if err != nil {
		return "", nil, nil, err
	}

	// Fetch the site
	client := &http.Client{
		Timeout: time.Second * time.Duration(defaultTimeout),
	}
	req, _ := http.NewRequest("GET", url, nil)

	if rule.Headers.UserAgent != "" {
		req.Header.Set("User-Agent", rule.Headers.UserAgent)
	} else {
		req.Header.Set("User-Agent", UserAgent)
	}

	if rule.Headers.XForwardedFor != "" {
		if rule.Headers.XForwardedFor != "none" {
			req.Header.Set("X-Forwarded-For", rule.Headers.XForwardedFor)
		}
	} else {
		req.Header.Set("X-Forwarded-For", ForwardedFor)
	}

	if rule.Headers.Referer != "" {
		if rule.Headers.Referer != "none" {
			req.Header.Set("Referer", rule.Headers.Referer)
		}
	} else {
		req.Header.Set("Referer", u.String())
	}

	// Handle FlareSolverr integration
	cookieValue := rule.Headers.Cookie
	debug := os.Getenv("LOG_URLS") == "true"

	if rule.UseFlareSolverr && flareSolverrHost != "" {
		if fsCookies, err := getFlareSolverrCookies(url); err == nil {
			if cookieValue != "" {
				cookieValue = cookieValue + "; " + fsCookies
			} else {
				cookieValue = fsCookies
			}
			if debug {
				log.Printf("Using FlareSolverr cookies for %s", url)
			}
		} else if debug {
			log.Printf("FlareSolverr error for %s: %v", url, err)
		}
	}

	if cookieValue != "" {
		req.Header.Set("Cookie", cookieValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, nil, err
	}
	defer resp.Body.Close()

	bodyB, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, err
	}

	if rule.Headers.CSP != "" {
		// log.Println(rule.Headers.CSP)
		resp.Header.Set("Content-Security-Policy", rule.Headers.CSP)
	}

	// log.Print("rule", rule) TODO: Add a debug mode to print the rule
	body := rewriteHtml(bodyB, u, rule)
	return body, req, resp, nil
}

func rewriteHtml(bodyB []byte, u *url.URL, rule ruleset.Rule) string {
	// Rewrite the HTML
	body := string(bodyB)

	// images
	imagePattern := `<img\s+([^>]*\s+)?src="(/)([^"]*)"`
	re := regexp.MustCompile(imagePattern)
	body = re.ReplaceAllString(body, fmt.Sprintf(`<img $1 src="%s$3"`, "/https://"+u.Host+"/"))

	// scripts
	scriptPattern := `<script\s+([^>]*\s+)?src="(/)([^"]*)"`
	reScript := regexp.MustCompile(scriptPattern)
	body = reScript.ReplaceAllString(body, fmt.Sprintf(`<script $1 script="%s$3"`, "/https://"+u.Host+"/"))

	// body = strings.ReplaceAll(body, "srcset=\"/", "srcset=\"/https://"+u.Host+"/") // TODO: Needs a regex to rewrite the URL's
	body = strings.ReplaceAll(body, "href=\"/", "href=\"/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url('/", "url('/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url(/", "url(/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "href=\"https://"+u.Host, "href=\"/https://"+u.Host+"/")

	if os.Getenv("RULESET") != "" {
		body = applyRules(body, rule)
	}
	return body
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func fetchRule(domain string, path string) ruleset.Rule {
	if len(rulesSet) == 0 {
		return ruleset.Rule{}
	}
	rule := ruleset.Rule{}
	for _, rule := range rulesSet {
		domains := rule.Domains
		if rule.Domain != "" {
			domains = append(domains, rule.Domain)
		}
		for _, ruleDomain := range domains {
			if ruleDomain == domain || strings.HasSuffix(domain, ruleDomain) {
				if len(rule.Paths) > 0 && !StringInSlice(path, rule.Paths) {
					continue
				}
				// return first match
				return rule
			}
		}
	}
	return rule
}

func applyRules(body string, rule ruleset.Rule) string {
	if len(rulesSet) == 0 {
		return body
	}

	for _, regexRule := range rule.RegexRules {
		re := regexp.MustCompile(regexRule.Match)
		body = re.ReplaceAllString(body, regexRule.Replace)
	}
	for _, injection := range rule.Injections {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
		if err != nil {
			log.Fatal(err)
		}
		if injection.Replace != "" {
			doc.Find(injection.Position).ReplaceWithHtml(injection.Replace)
		}
		if injection.Append != "" {
			doc.Find(injection.Position).AppendHtml(injection.Append)
		}
		if injection.Prepend != "" {
			doc.Find(injection.Position).PrependHtml(injection.Prepend)
		}
		body, err = doc.Html()
		if err != nil {
			log.Fatal(err)
		}
	}

	return body
}

func StringInSlice(s string, list []string) bool {
	for _, x := range list {
		if strings.HasPrefix(s, x) {
			return true
		}
	}
	return false
}
