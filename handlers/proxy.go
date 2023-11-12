package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
)

var (
	UserAgent      = getenv("USER_AGENT", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	ForwardedFor   = getenv("X_FORWARDED_FOR", "66.249.66.1")
	rulesSet       = loadRules()
	allowedDomains = strings.Split(os.Getenv("ALLOWED_DOMAINS"), ",")
)

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

func ProxySite(c *fiber.Ctx) error {
	// Get the url from the URL
	url, err := extractUrl(c)
	if err != nil {
		log.Println("ERROR In URL extraction:", err)
	}

	queries := c.Queries()
	body, _, resp, err := fetchSite(url, queries)
	if err != nil {
		log.Println("ERROR:", err)
		c.SendStatus(fiber.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Set("Content-Security-Policy", resp.Header.Get("Content-Security-Policy"))

	return c.SendString(body)
}

func modifyURL(uri string, rule Rule) (string, error) {
	newUrl, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	for _, urlMod := range rule.UrlMods.Domain {
		re := regexp.MustCompile(urlMod.Match)
		newUrl.Host = re.ReplaceAllString(newUrl.Host, urlMod.Replace)
	}

	for _, urlMod := range rule.UrlMods.Path {
		re := regexp.MustCompile(urlMod.Match)
		newUrl.Path = re.ReplaceAllString(newUrl.Path, urlMod.Replace)
	}

	v := newUrl.Query()
	for _, query := range rule.UrlMods.Query {
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
	client := &http.Client{}
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

	if rule.Headers.Cookie != "" {
		req.Header.Set("Cookie", rule.Headers.Cookie)
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
		log.Println(rule.Headers.CSP)
		resp.Header.Set("Content-Security-Policy", rule.Headers.CSP)
	}

	//log.Print("rule", rule) TODO: Add a debug mode to print the rule
	body := rewriteHtml(bodyB, u, rule)
	return body, req, resp, nil
}

func rewriteHtml(bodyB []byte, u *url.URL, rule Rule) string {
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

func loadRules() RuleSet {
	rulesUrl := os.Getenv("RULESET")
	if rulesUrl == "" {
		RulesList := RuleSet{}
		return RulesList
	}
	log.Println("Loading rules")

	var ruleSet RuleSet
	if strings.HasPrefix(rulesUrl, "http") {

		resp, err := http.Get(rulesUrl)
		if err != nil {
			log.Println("ERROR:", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			log.Println("ERROR:", resp.StatusCode, rulesUrl)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("ERROR:", err)
		}
		yaml.Unmarshal(body, &ruleSet)

		if err != nil {
			log.Println("ERROR:", err)
		}
	} else {
		yamlFile, err := os.ReadFile(rulesUrl)
		if err != nil {
			log.Println("ERROR:", err)
		}
		yaml.Unmarshal(yamlFile, &ruleSet)
	}

	domains := []string{}
	for _, rule := range ruleSet {

		domains = append(domains, rule.Domain)
		domains = append(domains, rule.Domains...)
		if os.Getenv("ALLOWED_DOMAINS_RULESET") == "true" {
			allowedDomains = append(allowedDomains, domains...)
		}
	}

	log.Println("Loaded ", len(ruleSet), " rules for", len(domains), "Domains")
	return ruleSet
}

func fetchRule(domain string, path string) Rule {
	if len(rulesSet) == 0 {
		return Rule{}
	}
	rule := Rule{}
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

func applyRules(body string, rule Rule) string {
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
