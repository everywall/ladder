package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"ladder/internal/service"
	"ladder/pkg/ruleset"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
)

var (
	sharedMu      sync.RWMutex
	sharedService *service.Service
	basePath      string
	rulesSet      ruleset.RuleSet
)

// Configure connects all legacy routes to the same bounded transport used by
// the reader, CLI and MCP. main calls this before accepting requests.
func Configure(s *service.Service) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	sharedService = s
	basePath = normalizeBasePath(os.Getenv("BASE_PATH"))
	if s != nil {
		rulesSet = s.LegacyRules()
	} else {
		rulesSet = nil
	}
}
func configured() (*service.Service, error) {
	sharedMu.RLock()
	s := sharedService
	sharedMu.RUnlock()
	if s != nil {
		return s, nil
	}
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedService == nil {
		var err error
		sharedService, err = service.NewFromEnv()
		if err != nil {
			return nil, err
		}
		rulesSet = sharedService.LegacyRules()
		basePath = normalizeBasePath(os.Getenv("BASE_PATH"))
	}
	return sharedService, nil
}
func normalizeBasePath(p string) string {
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimRight(p, "/")
}

// extractUrl decodes only a fully route-encoded target. Escapes in a normal
// absolute target (including %2F in paths and %26 in queries) remain intact.
func extractUrl(c *fiber.Ctx) (string, error) {
	target := c.Params("*")
	if !strings.Contains(target, "://") {
		if decoded, err := url.PathUnescape(target); err == nil {
			target = decoded
		}
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("invalid target URL")
	}
	if u.Scheme == "" {
		referer, err := url.Parse(c.Get("Referer"))
		if err != nil || referer == nil {
			return "", fmt.Errorf("relative target needs a proxied referer")
		}
		proxied := strings.TrimPrefix(strings.TrimPrefix(referer.EscapedPath(), basePath), "/")
		if !strings.Contains(proxied, "://") {
			proxied, _ = url.PathUnescape(proxied)
		}
		real, err := ruleset.ParseURL(proxied)
		if err != nil {
			return "", fmt.Errorf("relative target needs a proxied referer")
		}
		// The wildcard has no leading slash. These requests are browser root paths.
		u.Path = "/" + strings.TrimPrefix(u.Path, "/")
		u = real.ResolveReference(u)
	}
	validated, err := ruleset.ParseURL(u.String())
	if err != nil {
		return "", err
	}
	return validated.String(), nil
}
func routeTarget(c *fiber.Ctx, relative bool) (string, error) {
	var target string
	var err error
	if relative {
		target, err = extractUrl(c)
	} else {
		target = c.Params("*")
		if !strings.Contains(target, "://") {
			if decoded, e := url.PathUnescape(target); e == nil {
				target = decoded
			}
		}
		var u *url.URL
		u, err = ruleset.ParseURL(target)
		if err == nil {
			target = u.String()
		}
	}
	if err != nil {
		return "", err
	}
	// fasthttp's original query bytes retain encoding and repeated parameters.
	if query := string(c.Request().URI().QueryString()); query != "" {
		separator := "?"
		if strings.Contains(target, "?") {
			separator = "&"
		}
		target += separator + query
	}
	return target, nil
}
func ProxySite(rulesetPath string) fiber.Handler {
	var setupErr error
	if rulesetPath != "" {
		cfg, err := service.ConfigFromLookup(func(k string) (string, bool) {
			if k == "RULESET" {
				return rulesetPath, true
			}
			return os.LookupEnv(k)
		})
		if err == nil {
			var s *service.Service
			s, err = service.New(cfg)
			if err == nil {
				Configure(s)
			}
		}
		setupErr = err
	}
	return func(c *fiber.Ctx) error {
		if setupErr != nil {
			return legacyError(c, &service.Error{Code: "configuration", Status: 500, Message: "invalid transport configuration"})
		}
		target, err := routeTarget(c, true)
		if err != nil {
			return legacyError(c, &service.Error{Code: "invalid_url", Status: 400, Message: err.Error()})
		}
		s, err := configured()
		if err != nil {
			return legacyError(c, &service.Error{Code: "configuration", Status: 500, Message: "invalid transport configuration"})
		}
		result, err := s.Fetch(c.UserContext(), service.Request{URL: target})
		if err != nil {
			return legacyError(c, err)
		}
		body := result.Body
		if service.IsHTML(result.Metadata.MediaType) {
			body, err = s.TransformHTML(result)
			if err != nil {
				return legacyError(c, err)
			}
			u, _ := url.Parse(result.Metadata.FinalURL)
			body = []byte(rewriteHtml(body, u, ruleset.Rule{}))
		}
		sandbox(c)
		sendHeaders(c, result.Header)
		return c.Status(result.Metadata.UpstreamStatus).Send(body)
	}
}

// A response-level sandbox gives every upstream active document an opaque
// origin. In particular, scripts cannot act as the app that owns local state.
func sandbox(c *fiber.Ctx) {
	c.Set("Content-Security-Policy", "sandbox allow-scripts; object-src 'none'; frame-ancestors 'self'")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Referrer-Policy", "same-origin")
}
func sendHeaders(c *fiber.Ctx, h http.Header) {
	// Multiple CSP fields are intersected by the browser; a rule cannot relax the sandbox.
	for _, policy := range h.Values("Content-Security-Policy") {
		if strings.TrimSpace(policy) != "" {
			c.Response().Header.Add("Content-Security-Policy", policy)
		}
	}
	for _, k := range []string{"Content-Type", "Content-Language", "Cache-Control", "ETag", "Last-Modified"} {
		if v := h.Get(k); v != "" {
			c.Set(k, v)
		}
	}
}
func legacyError(c *fiber.Ctx, err error) error {
	status := 502
	message := "upstream request failed"
	if e, ok := err.(*service.Error); ok {
		status = e.Status
		message = e.Message
	}
	return c.Status(status).SendString(message)
}

// Compatibility wrapper used only by old package tests. Retrieval itself is
// exclusively delegated to Service; adapters never construct HTTP clients.
func fetchSite(target string, queries map[string]string) (string, *http.Request, *http.Response, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", nil, nil, err
	}
	if len(queries) > 0 {
		q := u.Query()
		for k, v := range queries {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	s, err := configured()
	if err != nil {
		return "", nil, nil, err
	}
	f, err := s.Fetch(context.Background(), service.Request{URL: u.String()})
	if err != nil {
		return "", nil, nil, err
	}
	req, _ := http.NewRequest(http.MethodGet, f.Metadata.FinalURL, nil)
	req.Header = f.RequestHeader
	resp := &http.Response{StatusCode: f.Metadata.UpstreamStatus, Header: f.Header, Request: req}
	return string(f.Body), req, resp, nil
}
func modifyURL(raw string, r ruleset.Rule) (string, error) {
	u, err := ruleset.ParseURL(raw)
	if err != nil {
		return "", err
	}
	u, err = ruleset.ModifyURL(u, r.URLMods.Domain, r.URLMods.Path, r.URLMods.Query, r.GoogleCache)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
func fetchRule(domain, path string) ruleset.Rule {
	return rulesSet.Match(&url.URL{Host: domain, Path: path})
}
func StringInSlice(s string, list []string) bool {
	for _, x := range list {
		if s == x {
			return true
		}
	}
	return false
}

var cssURL = regexp.MustCompile(`url\(\s*(['"]?)([^)'"\s]+)['"]?\s*\)`)

func rewriteHtml(body []byte, base *url.URL, _ ruleset.Rule) string {
	if base == nil {
		return string(body)
	}
	copyBase := *base
	if copyBase.Scheme == "" {
		copyBase.Scheme = "https"
	}
	base = &copyBase
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return string(body)
	}
	effective := base
	if raw, ok := doc.Find("base[href]").First().Attr("href"); ok {
		if ref, err := url.Parse(raw); err == nil {
			candidate := base.ResolveReference(ref)
			if candidate.Scheme == "http" || candidate.Scheme == "https" {
				effective = candidate
			}
		}
	}
	changed := false
	rewrite := func(raw string) string {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			return raw
		}
		ref, err := url.Parse(trimmed)
		if err != nil {
			return raw
		}
		if ref.Scheme != "" {
			return raw
		} // preserve absolute external resource behavior
		resolved := effective.ResolveReference(ref)
		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			return raw
		}
		changed = true
		return basePath + "/" + resolved.String()
	}
	doc.Find("[src], [href], [poster], [action], [data], [srcset], [style]").Each(func(_ int, s *goquery.Selection) {
		if goquery.NodeName(s) == "base" {
			s.RemoveAttr("href")
			changed = true
			return
		}
		for _, attr := range []string{"src", "href", "poster", "action", "data"} {
			if value, ok := s.Attr(attr); ok {
				s.SetAttr(attr, rewrite(value))
			}
		}
		if value, ok := s.Attr("srcset"); ok && !strings.Contains(value, "data:") {
			parts := strings.Split(value, ",")
			for i, part := range parts {
				fields := strings.Fields(part)
				if len(fields) > 0 {
					fields[0] = rewrite(fields[0])
					parts[i] = strings.Join(fields, " ")
				}
			}
			s.SetAttr("srcset", strings.Join(parts, ", "))
		}
		if value, ok := s.Attr("style"); ok {
			s.SetAttr("style", rewriteCSS(value, rewrite))
		}
	})
	doc.Find("style").Each(func(_ int, s *goquery.Selection) { s.SetText(rewriteCSS(s.Text(), rewrite)) })
	if !changed {
		return string(body)
	}
	out, err := doc.Html()
	if err != nil {
		return string(body)
	}
	return out
}
func rewriteCSS(css string, rewrite func(string) string) string {
	return cssURL.ReplaceAllStringFunc(css, func(value string) string {
		parts := cssURL.FindStringSubmatch(value)
		return "url(\"" + rewrite(parts[2]) + "\")"
	})
}
