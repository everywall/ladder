package ruleset

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// ParseURL is the common transport boundary. Credentials in URLs are rejected
// rather than copied into metadata or redirect requests.
func ParseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return nil, fmt.Errorf("invalid URL")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Opaque != "" || u.User != nil {
		return nil, fmt.Errorf("URL must be absolute HTTP(S), with a host and no embedded credentials")
	}
	if !ValidDomain(u.Hostname()) {
		return nil, fmt.Errorf("URL has an invalid hostname")
	}
	if strings.ContainsAny(u.Host, "\r\n\t ") {
		return nil, fmt.Errorf("URL has an invalid host")
	}
	if p := u.Port(); p != "" {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("invalid URL port")
			}
			n = n*10 + int(c-'0')
			if n > 65535 {
				return nil, fmt.Errorf("invalid URL port")
			}
		}
		if n == 0 {
			return nil, fmt.Errorf("invalid URL port")
		}
	}
	u.Fragment = ""
	return u, nil
}
func normalizeDomain(s string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".")
}
func ValidDomain(s string) bool {
	s = normalizeDomain(s)
	if s == "" {
		return false
	}
	if net.ParseIP(s) != nil {
		return true
	}
	if len(s) > 253 || strings.ContainsAny(s, "/:@?#*[] ") {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
func MatchDomain(host, domain string) bool {
	host = normalizeDomain(host)
	domain = normalizeDomain(domain)
	if host == "" || domain == "" {
		return false
	}
	if net.ParseIP(host) != nil || net.ParseIP(domain) != nil {
		return host == domain
	}
	return host == domain || strings.HasSuffix(host, "."+domain)
}
func MatchPath(actual, prefix string) bool {
	if prefix == "/" {
		return strings.HasPrefix(actual, "/") || actual == ""
	}
	prefix = strings.TrimSuffix(prefix, "/")
	return actual == prefix || strings.HasPrefix(actual, prefix+"/")
}
func (rs RuleSet) Match(u *url.URL) Rule {
	for _, r := range rs {
		domains := append([]string{}, r.Domains...)
		if r.Domain != "" {
			domains = append(domains, r.Domain)
		}
		for _, d := range domains {
			domain := normalizeDomain(d)
			subdomainsOnly := strings.HasPrefix(domain, ".")
			domain = strings.TrimPrefix(domain, ".")
			if !MatchDomain(u.Hostname(), domain) || (subdomainsOnly && normalizeDomain(u.Hostname()) == domain) {
				continue
			}
			if len(r.Paths) == 0 {
				return r
			}
			for _, p := range r.Paths {
				if u.Path == p {
					return r
				}
			}
		}
	}
	return Rule{}
}
func (rs RuleSet) Validate() error {
	for i, r := range rs {
		for _, d := range append(append([]string{}, r.Domains...), r.Domain) {
			if d == "" {
				continue
			}
			normalized := normalizeDomain(d)
			valid := ValidDomain(normalized)
			if strings.HasPrefix(normalized, ".") {
				suffix := strings.TrimPrefix(normalized, ".")
				valid = ValidDomain(suffix) && net.ParseIP(suffix) == nil
			}
			if !valid {
				return fmt.Errorf("legacy rule %d: invalid domain", i)
			}
		}
		for _, rr := range append(append(append([]Regex{}, r.URLMods.Domain...), r.URLMods.Path...), r.RegexRules...) {
			if _, err := regexp.Compile(rr.Match); err != nil {
				return fmt.Errorf("legacy rule %d: %w", i, err)
			}
		}
		for _, v := range []string{r.Headers.UserAgent, r.Headers.XForwardedFor, r.Headers.Referer, r.Headers.Cookie, r.Headers.CSP} {
			if strings.ContainsAny(v, "\r\n") {
				return fmt.Errorf("legacy rule %d: invalid header value", i)
			}
		}
	}
	return nil
}
func ModifyURL(u *url.URL, domains, paths []Regex, query []KV, googleCache bool) (*url.URL, error) {
	v := *u
	for _, rr := range domains {
		re, err := regexp.Compile(rr.Match)
		if err != nil {
			return nil, err
		}
		v.Host = re.ReplaceAllString(v.Host, rr.Replace)
	}
	for _, rr := range paths {
		re, err := regexp.Compile(rr.Match)
		if err != nil {
			return nil, err
		}
		v.Path = re.ReplaceAllString(v.Path, rr.Replace)
		v.RawPath = ""
	}
	if len(query) > 0 {
		q := v.Query()
		for _, kv := range query {
			if kv.Value == "" {
				q.Del(kv.Key)
			} else {
				q.Set(kv.Key, kv.Value)
			}
		}
		v.RawQuery = q.Encode()
	}
	if googleCache {
		return ParseURL("https://webcache.googleusercontent.com/search?q=" + url.QueryEscape("cache:"+v.String()))
	}
	return ParseURL(v.String())
}
