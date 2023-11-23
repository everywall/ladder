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

	"ladder/pkg/ruleset"
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
)

var (
	UserAgent      = getenv("USER_AGENT", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	ForwardedFor   = getenv("X_FORWARDED_FOR", "66.249.66.1")
	rulesSet       = ruleset.NewRulesetFromEnv()
	allowedDomains = []string{}
)

func init() {
	allowedDomains = strings.Split(os.Getenv("ALLOWED_DOMAINS"), ",")
	if os.Getenv("ALLOWED_DOMAINS_RULESET") == "true" {
		allowedDomains = append(allowedDomains, rulesSet.Domains()...)
	}
}

type ProxyOptions struct {
	RulesetPath string
	Verbose     bool
}

func NewProxySiteHandler(opts *ProxyOptions) fiber.Handler {
	/*
		var rs ruleset.RuleSet
		if opts.RulesetPath != "" {
			r, err := ruleset.NewRuleset(opts.RulesetPath)
			if err != nil {
				panic(err)
			}
			rs = r
		}
	*/

	return func(c *fiber.Ctx) error {
		proxychain := proxychain.
			NewProxyChain().
			SetFiberCtx(c).
			SetDebugLogging(opts.Verbose).
			SetRequestModifications(
				rx.DeleteOutgoingCookies(),
				//rx.RequestArchiveIs(),
				rx.MasqueradeAsGoogleBot(),
			).
			AddResponseModifications(
				tx.BypassCORS(),
				tx.BypassContentSecurityPolicy(),
				tx.DeleteIncomingCookies(),
				tx.RewriteHTMLResourceURLs(),
				tx.PatchDynamicResourceURLs(),
			).
			Execute()

		return proxychain
	}

}

func modifyURL(uri string, rule ruleset.Rule) (string, error) {
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
		//log.Println(rule.Headers.CSP)
		resp.Header.Set("Content-Security-Policy", rule.Headers.CSP)
	}

	//log.Print("rule", rule) TODO: Add a debug mode to print the rule
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
