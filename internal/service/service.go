package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ladder/pkg/ruleset"
	"ladder/pkg/sitepack"
)

type Error struct {
	Code    string `json:"code"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }
func failure(code string, status int, message string) error {
	return &Error{Code: code, Status: status, Message: message}
}

type fetchPolicyKey struct{}
type fetchPolicy struct{ uncacheable bool }

type cacheEntry struct {
	result  FetchResult
	expires time.Time
	size    int64
}
type Service struct {
	cfg        Config
	client     *http.Client
	packs      []sitepack.Pack
	mu         sync.Mutex
	cache      map[string]cacheEntry
	cacheBytes int64
	cacheEpoch uint64
}

func ConfigFromEnv() (Config, error) { return ConfigFromLookup(os.LookupEnv) }

// ConfigFromLookup permits command flags to overlay environment without process mutation.
func ConfigFromLookup(lookup func(string) (string, bool)) (Config, error) {
	getenv := func(k string) string { v, _ := lookup(k); return v }
	cfg := Config{Timeout: 15 * time.Second, CacheTTL: 5 * time.Minute, CacheMaxEntries: 128, CacheMaxBytes: 32 * 1024 * 1024, MaxBodyBytes: 8 * 1024 * 1024, UserAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", ForwardedFor: "66.249.66.1"}
	var err error
	if s := getenv("RULESET"); s != "" {
		cfg.Rules, err = ruleset.NewRuleset(s)
		if err != nil {
			return cfg, err
		}
	}
	for _, p := range strings.Split(getenv("RULE_PACKS"), ";") {
		if p = strings.TrimSpace(p); p != "" {
			cfg.PackPaths = append(cfg.PackPaths, p)
		}
	}
	for _, d := range strings.Split(getenv("ALLOWED_DOMAINS"), ",") {
		if d = strings.TrimSpace(d); d != "" {
			cfg.AllowedDomains = append(cfg.AllowedDomains, d)
		}
	}
	if getenv("ALLOWED_DOMAINS_RULESET") == "true" {
		for _, domain := range cfg.Rules.Domains() {
			if strings.TrimSpace(domain) != "" {
				cfg.AllowedDomains = append(cfg.AllowedDomains, domain)
			}
		}
	}
	if s, ok := lookup("USER_AGENT"); ok {
		cfg.UserAgent = s
	}
	if s, ok := lookup("X_FORWARDED_FOR"); ok {
		cfg.ForwardedFor = s
	}
	cfg.FlareSolverrURL = getenv("FLARESOLVERR_HOST")
	if s := getenv("HTTP_TIMEOUT"); s != "" {
		if n, e := strconv.Atoi(s); e == nil {
			cfg.Timeout = time.Duration(n) * time.Second
		} else {
			cfg.Timeout, err = time.ParseDuration(s)
			if err != nil {
				return cfg, fmt.Errorf("HTTP_TIMEOUT: expected seconds or Go duration")
			}
		}
		if cfg.Timeout <= 0 {
			return cfg, fmt.Errorf("HTTP_TIMEOUT must be positive")
		}
	}
	if s := getenv("CACHE_TTL"); s != "" {
		cfg.CacheTTL, err = time.ParseDuration(s)
		if err != nil || cfg.CacheTTL < 0 {
			return cfg, fmt.Errorf("CACHE_TTL: expected nonnegative Go duration")
		}
	}
	if s := getenv("CACHE_MAX_ENTRIES"); s != "" {
		cfg.CacheMaxEntries, err = strconv.Atoi(s)
		if err != nil || cfg.CacheMaxEntries < 1 || cfg.CacheMaxEntries > 10000 {
			return cfg, fmt.Errorf("CACHE_MAX_ENTRIES: expected 1 to 10000")
		}
	}
	return cfg, nil
}
func NewFromEnv() (*Service, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return New(cfg)
}
func New(cfg Config) (*Service, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	if cfg.Timeout < 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = 8 * 1024 * 1024
	}
	if cfg.MaxBodyBytes < 1 {
		return nil, fmt.Errorf("body limit must be positive")
	}
	if cfg.CacheMaxEntries == 0 {
		cfg.CacheMaxEntries = 128
	}
	if cfg.CacheMaxEntries < 0 {
		return nil, fmt.Errorf("cache entry limit must be positive")
	}
	if cfg.CacheMaxBytes == 0 {
		cfg.CacheMaxBytes = 32 * 1024 * 1024
	}
	if cfg.CacheMaxBytes < 0 {
		return nil, fmt.Errorf("cache byte limit must be positive")
	}
	if cfg.CacheTTL < 0 {
		return nil, fmt.Errorf("cache TTL cannot be negative")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if err := cfg.Rules.Validate(); err != nil {
		return nil, err
	}
	// Clone caller-owned configuration so later edits cannot change shared policy.
	cfg.Rules = cloneRules(cfg.Rules)
	cfg.PackPaths = append([]string{}, cfg.PackPaths...)
	cfg.AllowedDomains = append([]string{}, cfg.AllowedDomains...)
	for _, d := range cfg.AllowedDomains {
		if !ruleset.ValidDomain(d) {
			return nil, fmt.Errorf("allowed domains must contain hostnames or exact IP addresses")
		}
	}
	if strings.ContainsAny(cfg.UserAgent+cfg.ForwardedFor, "\r\n") {
		return nil, fmt.Errorf("invalid configured header")
	}
	if cfg.FlareSolverrURL != "" {
		if _, err := ruleset.ParseURL(cfg.FlareSolverrURL); err != nil {
			return nil, fmt.Errorf("invalid FlareSolverr URL")
		}
	}
	s := &Service{cfg: cfg, cache: map[string]cacheEntry{}}
	seen := map[string]bool{}
	for _, filename := range cfg.PackPaths {
		p, err := sitepack.Load(filename)
		if err != nil {
			return nil, err
		}
		if seen[p.ID] {
			return nil, fmt.Errorf("duplicate pack id %q", p.ID)
		}
		seen[p.ID] = true
		s.packs = append(s.packs, p)
	}
	client := http.Client{}
	if cfg.Client != nil {
		client = *cfg.Client
	}
	client.Timeout = cfg.Timeout
	previous := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return failure("redirect_limit", 502, "upstream exceeded ten redirects")
		}
		if _, err := s.validate(req.URL.String()); err != nil {
			return err
		}
		crossed := false
		for _, earlier := range via {
			if !sameOrigin(req.URL, earlier.URL) {
				crossed = true
			}
		}
		if crossed {
			stripCredentials(req.Header)
		}
		if policy, ok := req.Context().Value(fetchPolicyKey{}).(*fetchPolicy); ok {
			if req.Response != nil && req.Response.Header.Get("Set-Cookie") != "" {
				policy.uncacheable = true
			}
		}
		if previous != nil {
			if err := previous(req, via); err != nil {
				return err
			}
			if _, err := s.validate(req.URL.String()); err != nil {
				return err
			}
			for _, earlier := range via {
				if !sameOrigin(req.URL, earlier.URL) {
					crossed = true
				}
			}
			if crossed {
				stripCredentials(req.Header)
			}
		}
		return nil
	}

	s.client = &client
	return s, nil
}
func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}
func (s *Service) validate(raw string) (*url.URL, error) {
	if len(raw) > 65536 {
		return nil, failure("invalid_url", 400, "URL exceeds 64 KiB limit")
	}
	u, err := ruleset.ParseURL(raw)
	if err != nil {
		return nil, failure("invalid_url", 400, err.Error())
	}
	if len(s.cfg.AllowedDomains) > 0 {
		ok := false
		for _, d := range s.cfg.AllowedDomains {
			if ruleset.MatchDomain(u.Hostname(), d) {
				ok = true
				break
			}
		}
		if !ok {
			return nil, failure("domain_not_allowed", 403, "target domain is not allowed")
		}
	}
	return u, nil
}
func (s *Service) Packs() []PackInfo {
	out := make([]PackInfo, 0, len(s.packs))
	for _, p := range s.packs {
		out = append(out, PackInfo{ID: p.ID, Version: p.Version, SHA256: p.SHA256, Rules: len(p.Rules)})
	}
	return out
}
func CheckPacks(paths []string) ([]PackInfo, error) {
	s, err := New(Config{PackPaths: paths})
	if err != nil {
		return nil, err
	}
	return s.Packs(), nil
}
func (s *Service) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = map[string]cacheEntry{}
	s.cacheBytes = 0
	s.cacheEpoch++
}
func (s *Service) LegacyRules() ruleset.RuleSet {
	r := cloneRules(s.cfg.Rules)
	for i := range r {
		if r[i].Headers.Cookie != "" {
			r[i].Headers.Cookie = "[REDACTED]"
		}
	}
	return r
}
func cloneRules(r ruleset.RuleSet) ruleset.RuleSet {
	out := append(ruleset.RuleSet{}, r...)
	for i := range out {
		out[i].Domains = append([]string{}, r[i].Domains...)
		out[i].Paths = append([]string{}, r[i].Paths...)
		out[i].RegexRules = append([]ruleset.Regex{}, r[i].RegexRules...)
		out[i].URLMods.Domain = append([]ruleset.Regex{}, r[i].URLMods.Domain...)
		out[i].URLMods.Path = append([]ruleset.Regex{}, r[i].URLMods.Path...)
		out[i].URLMods.Query = append([]ruleset.KV{}, r[i].URLMods.Query...)
		out[i].Injections = append(out[i].Injections[:0:0], r[i].Injections...)
	}
	return out
}

type selectedRule struct {
	pack sitepack.Pack
	rule sitepack.Rule
}

func (s *Service) selectRules(u *url.URL) []selectedRule {
	var out []selectedRule
	for _, p := range s.packs {
		for _, r := range p.Rules {
			if r.Matches(u) {
				out = append(out, selectedRule{p, r})
				break
			}
		}
	}
	return out
}
func matches(selected []selectedRule) []RuleMatch {
	out := make([]RuleMatch, 0, len(selected))
	for _, r := range selected {
		out = append(out, RuleMatch{r.pack.ID, r.pack.Version, r.rule.ID})
	}
	return out
}
func (s *Service) Fetch(ctx context.Context, input Request) (FetchResult, error) {
	var result FetchResult
	u, err := s.validate(input.URL)
	if err != nil {
		return result, err
	}
	if input.Cache != "" && input.Cache != "default" && input.Cache != "reload" && input.Cache != "bypass" {
		return result, failure("invalid_cache", 400, "cache must be default, reload or bypass")
	}
	if err := ctx.Err(); err != nil {
		return result, transportError(err)
	}
	s.mu.Lock()
	epoch := s.cacheEpoch
	s.mu.Unlock()
	key := u.String()
	if input.Cache == "reload" {
		s.mu.Lock()
		if old, ok := s.cache[key]; ok {
			delete(s.cache, key)
			s.cacheBytes -= old.size
		}
		s.mu.Unlock()
	}
	now := s.cfg.Now()
	if input.Cache != "reload" && input.Cache != "bypass" && s.cfg.CacheTTL > 0 && s.client.Jar == nil {
		if found, ok := s.cached(key, now); ok {
			found.Metadata.SourceURL = input.URL
			return found, nil
		}
	}
	selected := s.selectRules(u)
	legacy := s.cfg.Rules.Match(u)
	target, err := ruleset.ModifyURL(u, legacy.URLMods.Domain, legacy.URLMods.Path, legacy.URLMods.Query, legacy.GoogleCache)
	if err != nil {
		return result, failure("invalid_rule_url", 400, "legacy rule produced an invalid URL")
	}
	for _, r := range selected {
		target, err = r.rule.ApplyURL(target)
		if err != nil {
			return result, failure("invalid_rule_url", 400, "pack rule produced an invalid URL")
		}
	}
	if target, err = s.validate(target.String()); err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	policy := &fetchPolicy{}
	ctx = context.WithValue(ctx, fetchPolicyKey{}, policy)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return result, failure("invalid_url", 400, "invalid request URL")
	}
	req.Header.Set("User-Agent", s.cfg.UserAgent)
	if s.cfg.ForwardedFor != "" {
		req.Header.Set("X-Forwarded-For", s.cfg.ForwardedFor)
	}
	req.Header.Set("Referer", u.String())
	applyLegacyHeaders(req.Header, legacy)
	for _, r := range selected {
		r.rule.ApplyHeaders(req.Header)
	}
	// Rules that move a target across origins cannot move its credentials too.
	credentialed := hasCredentials(req.Header)
	if !sameOrigin(u, target) {
		stripCredentials(req.Header)
	}
	if legacy.UseFlareSolverr && s.cfg.FlareSolverrURL != "" && !staticAsset.MatchString(target.Path) {
		// The browser service does not expose every redirect. An allowlist cannot be
		// enforced at each hop there, so use the bounded HTTP transport in that mode.
		if len(s.cfg.AllowedDomains) == 0 {
			result, err = s.flare(ctx, target, req.Header)
			if err == nil {
				return s.finish(input, key, u, selected, result, epoch), nil
			}
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return result, transportError(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, s.cfg.MaxBodyBytes+1))
	if err != nil {
		return result, transportError(err)
	}
	if int64(len(body)) > s.cfg.MaxBodyBytes {
		return result, failure("body_too_large", 502, "upstream body exceeds configured limit")
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	media, _, _ := mime.ParseMediaType(contentType)
	result = FetchResult{Metadata: Metadata{FinalURL: resp.Request.URL.String(), UpstreamStatus: resp.StatusCode, MediaType: media}, Header: resp.Header.Clone(), RequestHeader: resp.Request.Header.Clone(), Body: body}
	if credentialed || policy.uncacheable {
		result.Header.Set("Cache-Control", "no-store")
	}
	result.Header.Set("Content-Type", contentType)
	if legacy.Headers.CSP != "" {
		result.Header.Set("Content-Security-Policy", legacy.Headers.CSP)
	}
	return s.finish(input, key, u, selected, result, epoch), nil
}
func applyLegacyHeaders(h http.Header, r ruleset.Rule) {
	if r.Headers.UserAgent != "" {
		h.Set("User-Agent", r.Headers.UserAgent)
	}
	for k, v := range map[string]string{"X-Forwarded-For": r.Headers.XForwardedFor, "Referer": r.Headers.Referer} {
		if v == "none" {
			h.Del(k)
		} else if v != "" {
			h.Set(k, v)
		}
	}
	if r.Headers.Cookie != "" {
		h.Set("Cookie", r.Headers.Cookie)
	}
}
func transportError(err error) error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return failure("upstream_timeout", 504, "upstream request timed out")
	}
	if errors.Is(err, context.Canceled) {
		return failure("canceled", 499, "request canceled")
	}
	return failure("upstream_error", 502, "upstream request failed")
}
func (s *Service) finish(input Request, key string, u *url.URL, selected []selectedRule, result FetchResult, epoch uint64) FetchResult {
	now := s.cfg.Now()
	result.Metadata.SourceURL = input.URL
	result.Metadata.SourceHost = u.Hostname()
	result.Metadata.FetchedAt = now
	result.Metadata.ServedAt = now
	result.Metadata.Rules = matches(selected)
	ttl, ok := cacheLifetime(result, s.cfg.CacheTTL)
	if s.client.Jar != nil {
		ok = false
	}
	result.Header = RedactHeaders(result.Header)
	result.RequestHeader = RedactHeaders(result.RequestHeader)
	if input.Cache != "bypass" && ok && ttl > 0 && result.Metadata.UpstreamStatus >= 200 && result.Metadata.UpstreamStatus < 300 {
		result.Metadata.Cache = CacheInfo{ExpiresAt: now.Add(ttl), TTLSeconds: int64(ttl.Seconds())}
		if !s.put(key, result, now.Add(ttl), epoch) {
			result.Metadata.Cache = CacheInfo{}
		}
	}
	return result
}

// RedactHeaders returns a copy suitable for structured metadata, never the
// credential-bearing request/response headers owned by net/http.
func RedactHeaders(h http.Header) http.Header {
	out := h.Clone()
	for k := range out {
		if secretHeader(k) {
			out[k] = []string{"[REDACTED]"}
		}
	}
	return out
}
func secretHeader(k string) bool {
	k = strings.ToLower(k)
	return k == "cookie" || k == "set-cookie" || k == "authorization" || k == "proxy-authorization" || strings.Contains(k, "api-key") || strings.Contains(k, "token") || strings.Contains(k, "secret")
}
func hasCredentials(h http.Header) bool {
	for k := range h {
		if secretHeader(k) {
			return true
		}
	}
	return false
}
func stripCredentials(h http.Header) {
	for k := range h {
		if secretHeader(k) {
			h.Del(k)
		}
	}
}
func cacheLifetime(r FetchResult, ttl time.Duration) (time.Duration, bool) {
	if ttl <= 0 || r.RequestHeader.Get("Cookie") != "" || r.RequestHeader.Get("Authorization") != "" || r.Header.Get("Set-Cookie") != "" {
		return 0, false
	}
	for k := range r.RequestHeader {
		if secretHeader(k) {
			return 0, false
		}
	}
	for _, directive := range strings.Split(strings.ToLower(r.Header.Get("Cache-Control")), ",") {
		parts := strings.SplitN(strings.TrimSpace(directive), "=", 2)
		switch parts[0] {
		case "no-store", "private", "no-cache":
			return 0, false
		case "max-age":
			if len(parts) == 2 {
				seconds, err := strconv.ParseInt(strings.Trim(parts[1], "\""), 10, 64)
				if err != nil || seconds <= 0 {
					return 0, false
				}
				if seconds < int64(ttl/time.Second) {
					ttl = time.Duration(seconds) * time.Second
				}
			}
		}
	}
	if age := r.Header.Get("Age"); age != "" {
		seconds, err := strconv.ParseInt(age, 10, 64)
		if err != nil || seconds < 0 {
			return 0, false
		}
		if seconds >= int64(ttl/time.Second) {
			return 0, false
		}
		ttl -= time.Duration(seconds) * time.Second
	}
	if strings.TrimSpace(r.Header.Get("Vary")) == "*" {
		return 0, false
	}
	return ttl, true
}
func cloneResult(r FetchResult) FetchResult {
	r.Body = append([]byte{}, r.Body...)
	r.Header = r.Header.Clone()
	r.RequestHeader = r.RequestHeader.Clone()
	r.Metadata.Rules = append([]RuleMatch{}, r.Metadata.Rules...)
	return r
}
func (s *Service) cached(key string, now time.Time) (FetchResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cache[key]
	if !ok {
		return FetchResult{}, false
	}
	if !now.Before(entry.expires) {
		delete(s.cache, key)
		s.cacheBytes -= entry.size
		return FetchResult{}, false
	}
	r := cloneResult(entry.result)
	r.Metadata.ServedAt = now
	r.Metadata.Cache.Hit = true
	r.Metadata.Cache.AgeSeconds = int64(now.Sub(r.Metadata.FetchedAt).Seconds())
	return r, true
}
func (s *Service) put(key string, r FetchResult, expires time.Time, epoch uint64) bool {
	size := int64(len(r.Body) + len(key) + len(r.Metadata.SourceURL) + len(r.Metadata.FinalURL) + 512)
	for _, h := range []http.Header{r.Header, r.RequestHeader} {
		for k, values := range h {
			size += int64(len(k))
			for _, value := range values {
				size += int64(len(value))
			}
		}
	}
	if size > s.cfg.CacheMaxBytes {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if epoch != s.cacheEpoch {
		return false
	}
	if old, ok := s.cache[key]; ok {
		s.cacheBytes -= old.size
		delete(s.cache, key)
	}
	for len(s.cache) >= s.cfg.CacheMaxEntries || s.cacheBytes+size > s.cfg.CacheMaxBytes {
		var oldestKey string
		var oldest time.Time
		for k, v := range s.cache {
			if oldestKey == "" || v.result.Metadata.FetchedAt.Before(oldest) {
				oldestKey = k
				oldest = v.result.Metadata.FetchedAt
			}
		}
		if oldestKey == "" {
			break
		}
		s.cacheBytes -= s.cache[oldestKey].size
		delete(s.cache, oldestKey)
	}
	s.cache[key] = cacheEntry{cloneResult(r), expires, size}
	s.cacheBytes += size
	return true
}
