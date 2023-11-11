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

func ProxySite(c *fiber.Ctx) error {
	// Get the url from the URL
	url := c.Params("*")

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

	rule := fetchRule(u.Host, u.Path)

	if rule.GoogleCache {
		u, err = url.Parse("https://webcache.googleusercontent.com/search?q=cache:" + u.String())
		if err != nil {
			return "", nil, nil, err
		}
	}

	// Fetch the site
	client := &http.Client{}
	req, _ := http.NewRequest("GET", u.String()+urlQuery, nil)

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
