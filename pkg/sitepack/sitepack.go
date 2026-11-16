// Package sitepack loads explicit, versioned local rule packs. Packs are never
// discovered recursively alongside legacy RULESET files.
package sitepack

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"gopkg.in/yaml.v3"
	"ladder/pkg/ruleset"
)

type Pack struct {
	SchemaVersion int    `yaml:"schemaVersion"`
	ID            string `yaml:"id"`
	Version       string `yaml:"version"`
	Rules         []Rule `yaml:"rules"`
	SHA256        string `yaml:"-"`
}
type Rule struct {
	ID      string   `yaml:"id"`
	Domains []string `yaml:"domains"`
	Paths   []string `yaml:"paths,omitempty"`
	Headers struct {
		Set    map[string]string `yaml:"set,omitempty"`
		Remove []string          `yaml:"remove,omitempty"`
	} `yaml:"headers,omitempty"`
	URL struct {
		Domain []ruleset.Regex `yaml:"domain,omitempty"`
		Path   []ruleset.Regex `yaml:"path,omitempty"`
		Query  []ruleset.KV    `yaml:"query,omitempty"`
	} `yaml:"url,omitempty"`
	Regex  []ruleset.Regex `yaml:"regex,omitempty"`
	Remove []string        `yaml:"remove,omitempty"`
	Tests  []Test          `yaml:"tests"`
}
type Test struct {
	Name        string   `yaml:"name"`
	URL         string   `yaml:"url"`
	Status      int      `yaml:"status,omitempty"`
	Contains    []string `yaml:"contains,omitempty"`
	NotContains []string `yaml:"notContains,omitempty"`
}

var identifier = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var version = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`)

func Load(filename string) (Pack, error) {
	var p Pack
	f, err := os.Open(filename)
	if err != nil {
		return p, fmt.Errorf("pack %s: %w", filename, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 2*1024*1024+1))
	if err != nil {
		return p, err
	}
	if len(data) > 2*1024*1024 {
		return p, fmt.Errorf("pack %s exceeds 2 MiB", filename)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err = dec.Decode(&p); err != nil {
		return p, fmt.Errorf("pack %s: %w", filename, err)
	}
	var extra any
	if err = dec.Decode(&extra); err != io.EOF {
		return p, fmt.Errorf("pack %s: exactly one YAML document is required", filename)
	}
	if err = p.Validate(); err != nil {
		return p, fmt.Errorf("pack %s: %w", filename, err)
	}
	sum := sha256.Sum256(data)
	p.SHA256 = hex.EncodeToString(sum[:])
	return p, nil
}
func (p Pack) Validate() error {
	if p.SchemaVersion != 1 {
		return fmt.Errorf("schemaVersion: supported version is 1")
	}
	if !identifier.MatchString(p.ID) {
		return fmt.Errorf("id: use a stable lowercase identifier")
	}
	if !version.MatchString(p.Version) {
		return fmt.Errorf("version: expected semantic version such as 1.0.0")
	}
	if len(p.Rules) == 0 || len(p.Rules) > 1000 {
		return fmt.Errorf("rules: require 1 to 1000 rules")
	}
	ids := map[string]bool{}
	for i, r := range p.Rules {
		loc := fmt.Sprintf("rules[%d] (%s)", i, r.ID)
		if !identifier.MatchString(r.ID) || ids[r.ID] {
			return fmt.Errorf("%s.id: invalid or duplicate id", loc)
		}
		ids[r.ID] = true
		if len(r.Domains) == 0 {
			return fmt.Errorf("%s.domains: at least one domain is required", loc)
		}
		for _, d := range r.Domains {
			if !ruleset.ValidDomain(d) {
				return fmt.Errorf("%s.domains: invalid hostname %q", loc, d)
			}
		}
		for _, p := range r.Paths {
			if !strings.HasPrefix(p, "/") || strings.ContainsAny(p, "?#") {
				return fmt.Errorf("%s.paths: require absolute URL paths", loc)
			}
		}
		for _, rr := range append(append(append([]ruleset.Regex{}, r.URL.Domain...), r.URL.Path...), r.Regex...) {
			if _, err := regexp.Compile(rr.Match); err != nil {
				return fmt.Errorf("%s regex: %w", loc, err)
			}
		}
		for _, s := range r.Remove {
			if _, err := cascadia.Compile(s); err != nil {
				return fmt.Errorf("%s.remove: invalid selector %q: %w", loc, s, err)
			}
		}
		for k, v := range r.Headers.Set {
			if !validHeaderName(k) || strings.ContainsAny(v, "\r\n") {
				return fmt.Errorf("%s.headers.set: invalid header", loc)
			}
			if forbiddenHeader(k) {
				return fmt.Errorf("%s.headers.set: transport header %s is unsupported", loc, k)
			}
		}
		for _, k := range r.Headers.Remove {
			if !validHeaderName(k) {
				return fmt.Errorf("%s.headers.remove: invalid header", loc)
			}
		}
		if len(r.Tests) == 0 {
			return fmt.Errorf("%s.tests: declare at least one status or content assertion", loc)
		}
		names := map[string]bool{}
		for j, t := range r.Tests {
			tl := fmt.Sprintf("%s.tests[%d]", loc, j)
			u, err := ruleset.ParseURL(t.URL)
			if err != nil {
				return fmt.Errorf("%s.url: %w", tl, err)
			}
			if !r.Matches(u) {
				return fmt.Errorf("%s.url: does not match its rule", tl)
			}
			if strings.TrimSpace(t.Name) == "" || names[t.Name] {
				return fmt.Errorf("%s.name: require unique nonempty name", tl)
			}
			names[t.Name] = true
			if t.Status != 0 && (t.Status < 100 || t.Status > 599) {
				return fmt.Errorf("%s.status: invalid HTTP status", tl)
			}
			if t.Status == 0 && len(t.Contains) == 0 && len(t.NotContains) == 0 {
				return fmt.Errorf("%s: no assertions", tl)
			}
			for _, s := range append(append([]string{}, t.Contains...), t.NotContains...) {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("%s: empty text assertion", tl)
				}
			}
		}
	}
	return nil
}
func validHeaderName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
			return false
		}
	}
	return true
}
func forbiddenHeader(s string) bool {
	switch http.CanonicalHeaderKey(s) {
	case "Host", "Connection", "Content-Length", "Transfer-Encoding", "Upgrade", "Proxy-Authorization":
		return true
	}
	return false
}
func (r Rule) Matches(u *url.URL) bool {
	matched := false
	for _, d := range r.Domains {
		if ruleset.MatchDomain(u.Hostname(), d) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	if len(r.Paths) == 0 {
		return true
	}
	for _, p := range r.Paths {
		if ruleset.MatchPath(u.Path, p) {
			return true
		}
	}
	return false
}
func (r Rule) ApplyURL(u *url.URL) (*url.URL, error) {
	return ruleset.ModifyURL(u, r.URL.Domain, r.URL.Path, r.URL.Query, false)
}
func (r Rule) ApplyHeaders(h http.Header) {
	for k, v := range r.Headers.Set {
		h.Set(k, v)
	}
	for _, k := range r.Headers.Remove {
		h.Del(k)
	}
}
func (r Rule) ApplyHTML(body []byte) ([]byte, error) {
	out := string(body)
	for _, rr := range r.Regex {
		re, err := regexp.Compile(rr.Match)
		if err != nil {
			return nil, err
		}
		out = re.ReplaceAllString(out, rr.Replace)
	}
	if len(r.Remove) == 0 {
		return []byte(out), nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(out))
	if err != nil {
		return nil, err
	}
	for _, selector := range r.Remove {
		doc.Find(selector).Remove()
	}
	out, err = doc.Html()
	return []byte(out), err
}
