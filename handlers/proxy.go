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

var UserAgent = getenv("USER_AGENT", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
var ForwardedFor = getenv("X_FORWARDED_FOR", "66.249.66.1")
var rulesSet = loadRules()
var allowedDomains = strings.Split(os.Getenv("ALLOWED_DOMAINS"), ",")

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

	if os.Getenv("LOG_URLS	") == "true" {
		log.Println(u.String() + urlQuery)
	}

	// Fetch the site
	client := &http.Client{}
	req, _ := http.NewRequest("GET", u.String()+urlQuery, nil)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("X-Forwarded-For", ForwardedFor)
	req.Header.Set("Referer", u.String())
	req.Header.Set("Host", u.Host)
	resp, err := client.Do(req)

	if err != nil {
		return "", nil, nil, err
	}
	defer resp.Body.Close()

	bodyB, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, err
	}

	body := rewriteHtml(bodyB, u)
	return body, req, resp, nil
}

func rewriteHtml(bodyB []byte, u *url.URL) string {
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

	//body = strings.ReplaceAll(body, "srcset=\"/", "srcset=\"/https://"+u.Host+"/") // TODO: Needs a regex to rewrite the URL's
	body = strings.ReplaceAll(body, "href=\"/", "href=\"/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url('/", "url('/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url(/", "url(/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "href=\"https://"+u.Host, "href=\"/https://"+u.Host+"/")

	if os.Getenv("RULESET") != "" {
		body = applyRules(u.Host, u.Path, body)
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

	for _, rule := range ruleSet {
		//log.Println("Loaded rules for", rule.Domain)
		if os.Getenv("ALLOWED_DOMAINS_RULESET") == "true" {
			allowedDomains = append(allowedDomains, rule.Domain)
			for _, domainRule := range rule.Domains {
				allowedDomains = append(allowedDomains, domainRule)
			}
		}
	}

	log.Println("Loaded rules for", len(ruleSet), "Domains")
	return ruleSet
}

func applyRules(domain string, path string, body string) string {
	if len(rulesSet) == 0 {
		return body
	}

	for _, rule := range rulesSet {
		if rule.Domain != "" && rule.Domain != domain {
			continue
		}
		if len(rule.Domains) > 0 {
			if !StringInSlice(domain, rule.Domains) {
				continue
			}
			if len(rule.Paths) > 0 && !StringInSlice(path, rule.Paths) {
				continue
			}
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
	}
	return body
}

type Rule struct {
	Match   string `yaml:"match"`
	Replace string `yaml:"replace"`
}

type RuleSet []struct {
	Domain      string   `yaml:"domain,omitempty"`
	Domains     []string `yaml:"domains,omitempty"`
	Paths       []string `yaml:"paths,omitempty"`
	GoogleCache bool     `yaml:"googleCache,omitempty"`
	RegexRules  []Rule   `yaml:"regexRules"`
	Injections  []struct {
		Position string `yaml:"position"`
		Append   string `yaml:"append"`
		Prepend  string `yaml:"prepend"`
		Replace  string `yaml:"replace"`
	} `yaml:"injections"`
}

func StringInSlice(s string, list []string) bool {
	for _, x := range list {
		if strings.HasPrefix(s, x) {
			return true
		}
	}
	return false
}
