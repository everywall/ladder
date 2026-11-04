package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"ladder/pkg/ruleset"
)

func newService(t *testing.T, cfg Config) *Service {
	t.Helper()
	s, err := New(cfg)
	require.NoError(t, err)
	return s
}
func fetch(t *testing.T, s *Service, target, cache string) FetchResult {
	t.Helper()
	f, err := s.Fetch(context.Background(), Request{URL: target, Cache: cache})
	require.NoError(t, err)
	return f
}
func TestFetchTTLBoundaryBypassReloadAndCopies(t *testing.T) {
	var hits atomic.Int64
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "payload")
	}))
	defer origin.Close()
	now := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	s := newService(t, Config{CacheTTL: time.Minute, Now: func() time.Time { return now }})
	f := fetch(t, s, origin.URL, "")
	require.False(t, f.Metadata.Cache.Hit)
	require.Equal(t, now.Add(time.Minute), f.Metadata.Cache.ExpiresAt)
	f.Body[0] = 'X'
	f.Header.Set("Content-Type", "bad")
	now = now.Add(20 * time.Second)
	f = fetch(t, s, origin.URL, "")
	require.True(t, f.Metadata.Cache.Hit)
	require.Equal(t, int64(20), f.Metadata.Cache.AgeSeconds)
	require.Equal(t, "payload", string(f.Body))
	require.Equal(t, "text/plain", f.Header.Get("Content-Type"))
	require.Equal(t, int64(1), hits.Load())
	now = now.Add(40 * time.Second)
	require.False(t, fetch(t, s, origin.URL, "").Metadata.Cache.Hit)
	require.Equal(t, int64(2), hits.Load())
	require.False(t, fetch(t, s, origin.URL, "bypass").Metadata.Cache.Hit)
	require.Equal(t, int64(3), hits.Load())
	require.True(t, fetch(t, s, origin.URL, "").Metadata.Cache.Hit)
	require.False(t, fetch(t, s, origin.URL, "reload").Metadata.Cache.Hit)
	require.Equal(t, int64(4), hits.Load())
	s.ClearCache()
	require.False(t, fetch(t, s, origin.URL, "").Metadata.Cache.Hit)
	require.Equal(t, int64(5), hits.Load())
}
func TestCacheExclusionsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name, cacheControl, cookie, age string
		status                          int
	}{{"no-store", "no-store", "", "", 200}, {"private", "private, max-age=60", "", "", 200}, {"cookie", "", "session=private", "", 200}, {"aged", "max-age=60", "", "60", 200}, {"error", "", "", "", 500}} {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int64
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.Header().Set("Cache-Control", tc.cacheControl)
				w.Header().Set("Set-Cookie", tc.cookie)
				w.Header().Set("Age", tc.age)
				w.WriteHeader(tc.status)
				io.WriteString(w, "ok")
			}))
			defer origin.Close()
			s := newService(t, Config{CacheTTL: time.Minute})
			fetch(t, s, origin.URL, "")
			require.False(t, fetch(t, s, origin.URL, "").Metadata.Cache.Hit)
			require.Equal(t, int64(2), hits.Load())
		})
	}
	var hits atomic.Int64
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); io.WriteString(w, r.URL.Path) }))
	defer origin.Close()
	s := newService(t, Config{CacheTTL: time.Minute, CacheMaxEntries: 1, CacheMaxBytes: 2048})
	fetch(t, s, origin.URL+"/one", "")
	fetch(t, s, origin.URL+"/two", "")
	require.False(t, fetch(t, s, origin.URL+"/one", "").Metadata.Cache.Hit)
	require.Equal(t, int64(3), hits.Load())
}
func TestRedirectStripsConfiguredCredentialsAndNeverCaches(t *testing.T) {
	var destinationHeaders http.Header
	var destinationHits atomic.Int64
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destinationHeaders = r.Header.Clone()
		destinationHits.Add(1)
		w.Header().Set("Set-Cookie", "private=result")
		io.WriteString(w, "destination")
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "private=initial", r.Header.Get("Cookie"))
		http.Redirect(w, r, destination.URL, 302)
	}))
	defer source.Close()
	u, _ := url.Parse(source.URL)
	r := ruleset.Rule{Domain: u.Hostname()}
	r.Headers.Cookie = "private=initial"
	s := newService(t, Config{Rules: ruleset.RuleSet{r}, CacheTTL: time.Minute})
	for range 2 {
		f := fetch(t, s, source.URL, "")
		require.False(t, f.Metadata.Cache.Hit)
		require.Equal(t, destination.URL, f.Metadata.FinalURL)
		require.Equal(t, "[REDACTED]", f.Header.Get("Set-Cookie"))
		require.Empty(t, f.RequestHeader.Get("Cookie"))
		require.Empty(t, destinationHeaders.Get("Cookie"))
	}
	require.Equal(t, int64(2), destinationHits.Load())
	require.Equal(t, "[REDACTED]", s.LegacyRules()[0].Headers.Cookie)
	// Same-origin redirects must retain configured cookies but metadata redacts.
	same := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/end", 302)
			return
		}
		io.WriteString(w, "ok")
	}))
	defer same.Close()
	f := fetch(t, s, same.URL+"/start", "")
	require.Equal(t, "[REDACTED]", f.RequestHeader.Get("Cookie"))
	require.False(t, f.Metadata.Cache.Hit)
}
func TestRedirectCookieAndClearInflightCannotPopulateCache(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var hits atomic.Int64
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/redirect" {
			w.Header().Set("Set-Cookie", "s=private")
			http.Redirect(w, r, "/end", 302)
			return
		}
		if r.URL.Path == "/slow" {
			started <- struct{}{}
			<-release
		}
		io.WriteString(w, "ok")
	}))
	defer origin.Close()
	s := newService(t, Config{CacheTTL: time.Minute})
	fetch(t, s, origin.URL+"/redirect", "")
	require.False(t, fetch(t, s, origin.URL+"/redirect", "").Metadata.Cache.Hit)
	require.Equal(t, int64(4), hits.Load())
	done := make(chan error)
	go func() { _, err := s.Fetch(context.Background(), Request{URL: origin.URL + "/slow"}); done <- err }()
	<-started
	s.ClearCache()
	close(release)
	require.NoError(t, <-done)
	s.mu.Lock()
	_, cached := s.cache[origin.URL+"/slow"]
	s.mu.Unlock()
	require.False(t, cached)
}
func TestFetchInputPolicyLimitsAndCancellation(t *testing.T) {
	var hits atomic.Int64
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/slow" {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
		if r.URL.Path == "/loop" {
			http.Redirect(w, r, "/loop", 302)
			return
		}
		io.WriteString(w, "123456789")
	}))
	defer origin.Close()
	s := newService(t, Config{MaxBodyBytes: 4, Timeout: 30 * time.Millisecond})
	for _, raw := range []string{"", "invalid-url", "ftp://example.org/a", "file:///etc/passwd", "http://user:password@example.org", "http://example.org/%xx"} {
		_, err := s.Fetch(context.Background(), Request{URL: raw})
		var e *Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, 400, e.Status)
	}
	require.Zero(t, hits.Load())
	_, err := s.Fetch(context.Background(), Request{URL: origin.URL})
	var e *Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, "body_too_large", e.Code)
	_, err = s.Fetch(context.Background(), Request{URL: origin.URL + "/slow"})
	require.ErrorAs(t, err, &e)
	require.Equal(t, 504, e.Status)
	_, err = s.Fetch(context.Background(), Request{URL: origin.URL + "/loop"})
	require.ErrorAs(t, err, &e)
	require.Equal(t, "redirect_limit", e.Code)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Fetch(ctx, Request{URL: origin.URL})
	require.ErrorAs(t, err, &e)
	require.Equal(t, "canceled", e.Code)
	_, err = s.Fetch(context.Background(), Request{URL: origin.URL, Cache: "invalid"})
	require.ErrorAs(t, err, &e)
	require.Equal(t, 400, e.Status)
}
func TestAllowedDomainRecheckedAfterRedirectAndRuleModification(t *testing.T) {
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("disallowed destination reached") }))
	defer destination.Close()
	dest, _ := url.Parse(destination.URL)
	dest.Host = "localhost:" + dest.Port()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, dest.String(), 302) }))
	defer source.Close()
	s := newService(t, Config{AllowedDomains: []string{"127.0.0.1"}})
	_, err := s.Fetch(context.Background(), Request{URL: source.URL})
	var e *Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, 403, e.Status)
	r := ruleset.Rule{Domain: "127.0.0.1"}
	r.URLMods.Domain = []ruleset.Regex{{Match: "127.0.0.1", Replace: "localhost"}}
	s = newService(t, Config{Rules: ruleset.RuleSet{r}, AllowedDomains: []string{"127.0.0.1"}})
	_, err = s.Fetch(context.Background(), Request{URL: source.URL})
	require.ErrorAs(t, err, &e)
	require.Equal(t, 403, e.Status)
}
func TestReadAndFlareUseActualFinalURL(t *testing.T) {
	final := "https://example.org/redirected/article"
	solver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1", r.URL.Path)
		require.Empty(t, r.Header.Get("Cookie"))
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "solution": map[string]any{"url": final, "status": 404, "response": "<article><h1>Solver article</h1><p>The final page is missing but this text is preserved.</p></article>"}})
	}))
	defer solver.Close()
	rule := ruleset.Rule{Domain: "example.org", UseFlareSolverr: true}
	s := newService(t, Config{Rules: ruleset.RuleSet{rule}, FlareSolverrURL: solver.URL})
	f := fetch(t, s, "https://example.org/start", "")
	require.Equal(t, final, f.Metadata.FinalURL)
	require.Equal(t, 404, f.Metadata.UpstreamStatus)
	require.False(t, f.Metadata.Cache.Hit)
	doc, err := s.Read(context.Background(), Request{URL: "https://example.org/start"})
	require.NoError(t, err)
	require.Equal(t, final, doc.Metadata.FinalURL)
	require.Contains(t, doc.Content.Text, "final page")
	require.Equal(t, "upstream_status", doc.Problems[0].Code)
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "<script>unsafe()</script>\n```\n[link](javascript:alert(1))")
	}))
	defer plain.Close()
	s = newService(t, Config{})
	doc, err = s.Read(context.Background(), Request{URL: plain.URL})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(doc.Content.Markdown, "````\n"))
	require.NotContains(t, doc.Content.HTML, "<script>")
}
func TestVersionedPackTransformsAndDeclaredAssertions(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "honest-client", r.Header.Get("User-Agent"))
		require.Empty(t, r.Header.Get("Referer"))
		require.Empty(t, r.Header.Get("X-Forwarded-For"))
		require.Equal(t, []string{"one", "two"}, r.URL.Query()["dup"])
		require.Empty(t, r.URL.Query().Get("tracking"))
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<article><h1>Old title</h1><p>Old article text remains readable.</p><aside class="ad">advertisement</aside></article>`)
	}))
	defer origin.Close()
	u, _ := url.Parse(origin.URL)
	pack := `schemaVersion: 1
id: fixture
version: "1.0.0"
rules:
 - id: article
   domains: [` + u.Hostname() + `]
   paths: [/article]
   headers:
     set: {User-Agent: honest-client}
     remove: [Referer, X-Forwarded-For]
   url:
     query: [{key: tracking, value: ""}]
   regex: [{match: Old, replace: New}]
   remove: [.ad]
   tests:
     - name: fixture
       url: ` + origin.URL + `/article?dup=one&dup=two&tracking=x
       status: 200
       contains: [New title]
       notContains: [advertisement]
`
	filename := filepath.Join(t.TempDir(), "pack.yaml")
	require.NoError(t, os.WriteFile(filename, []byte(pack), 0600))
	s := newService(t, Config{PackPaths: []string{filename}, CacheTTL: time.Minute})
	target := origin.URL + "/article?dup=one&dup=two&tracking=x"
	f := fetch(t, s, target, "")
	require.Contains(t, string(f.Body), "Old title")
	require.Contains(t, string(f.Body), "advertisement")
	require.Equal(t, []RuleMatch{{PackID: "fixture", PackVersion: "1.0.0", RuleID: "article"}}, f.Metadata.Rules)
	doc, err := s.Read(context.Background(), Request{URL: target})
	require.NoError(t, err)
	require.Contains(t, doc.Content.Text, "New title")
	require.NotContains(t, doc.Content.Text, "advertisement")
	result, err := s.TestURL(context.Background(), Request{URL: target})
	require.NoError(t, err)
	require.True(t, result.Passed)
	require.GreaterOrEqual(t, len(result.Checks), 6)
	require.Len(t, s.Packs()[0].SHA256, 64)
	// A wrong declared assertion is an observable failing test, not validation.
	badFile := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(badFile, []byte(strings.Replace(pack, "contains: [New title]", "contains: [Missing title]", 1)), 0600))
	bad := newService(t, Config{PackPaths: []string{badFile}})
	badResult, err := bad.TestURL(context.Background(), Request{URL: target})
	require.NoError(t, err)
	require.False(t, badResult.Passed)
	require.NotEqual(t, s.Packs()[0].SHA256, bad.Packs()[0].SHA256)
}
func TestConcurrentFetchClearAndImmutableMetadata(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "concurrent payload") }))
	defer origin.Close()
	s := newService(t, Config{CacheTTL: time.Minute})
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				f, err := s.Fetch(context.Background(), Request{URL: origin.URL})
				if err != nil {
					t.Error(err)
					return
				}
				if len(f.Body) > 0 {
					f.Body[0] = 'X'
				}
				s.ClearCache()
			}
		}()
	}
	wg.Wait()
	require.Equal(t, "concurrent payload", string(fetch(t, s, origin.URL, "").Body))
}
func TestConfigOverlayNoEnvironmentMutation(t *testing.T) {
	t.Setenv("RULESET", "/missing/environment/rules.yaml")
	cfg, err := ConfigFromLookup(func(key string) (string, bool) {
		switch key {
		case "RULESET":
			return "", true
		case "CACHE_TTL":
			return "0s", true
		case "HTTP_TIMEOUT":
			return "2", true
		}
		return os.LookupEnv(key)
	})
	require.NoError(t, err)
	require.Equal(t, time.Duration(0), cfg.CacheTTL)
	require.Equal(t, 2*time.Second, cfg.Timeout)
	require.Equal(t, "/missing/environment/rules.yaml", os.Getenv("RULESET"))
}

func TestConfigLoadsAllBundledLegacyRules(t *testing.T) {
	absolute, err := filepath.Abs("../../rulesets")
	require.NoError(t, err)
	cfg, err := ConfigFromLookup(func(key string) (string, bool) {
		if key == "RULESET" {
			return absolute, true
		}
		if key == "RULE_PACKS" {
			return "", true
		}
		return "", false
	})
	require.NoError(t, err)
	require.Equal(t, 10, len(cfg.Rules))
	s, err := New(cfg)
	require.NoError(t, err)
	require.Equal(t, 10, len(s.LegacyRules()))
	require.Empty(t, s.Packs())
}

func TestLegacyUnusedDocumentOperationsRemainInactive(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<article><h1>Original title</h1><p>Original body text remains visible to readers.</p></article>")
	}))
	defer origin.Close()
	rule := ruleset.Rule{Domain: "127.0.0.1", RegexRules: []ruleset.Regex{{Match: "Original", Replace: "Injected"}}}
	rule.Injections = append(rule.Injections, struct {
		Position string `yaml:"position,omitempty"`
		Append   string `yaml:"append,omitempty"`
		Prepend  string `yaml:"prepend,omitempty"`
		Replace  string `yaml:"replace,omitempty"`
	}{Position: "article", Replace: "<p>Unexpected injection</p>"})
	s := newService(t, Config{Rules: ruleset.RuleSet{rule}})
	raw := fetch(t, s, origin.URL, "")
	transformed, err := s.TransformHTML(raw)
	require.NoError(t, err)
	require.Equal(t, raw.Body, transformed)
	doc, err := s.Read(context.Background(), Request{URL: origin.URL})
	require.NoError(t, err)
	require.Contains(t, doc.Content.Text, "Original body")
	require.NotContains(t, doc.Content.Text, "Unexpected injection")
}

func TestInitialCredentialsStrippedByURLRewriteStillDisableCache(t *testing.T) {
	var hits atomic.Int64
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		require.Empty(t, r.Header.Get("Cookie"))
		io.WriteString(w, "ok")
	}))
	defer destination.Close()
	target, err := url.Parse(destination.URL)
	require.NoError(t, err)
	rule := ruleset.Rule{Domain: "source.example"}
	rule.Headers.Cookie = "private=source"
	rule.URLMods.Domain = []ruleset.Regex{{Match: "source.example", Replace: target.Host}}
	s := newService(t, Config{Rules: ruleset.RuleSet{rule}, CacheTTL: time.Minute})
	fetch(t, s, "http://source.example/article", "")
	require.False(t, fetch(t, s, "http://source.example/article", "").Metadata.Cache.Hit)
	require.Equal(t, int64(2), hits.Load())
}

func TestOrderedFirstRulePerPackAndPackOrder(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<article><h1>Original</h1><p>Readable ordered transformation fixture.</p></article>")
	}))
	defer origin.Close()
	makePack := func(id, from, to string, second bool) string {
		text := "schemaVersion: 1\nid: " + id + "\nversion: '1.0.0'\nrules:\n - id: first\n   domains: [127.0.0.1]\n   regex: [{match: " + from + ", replace: " + to + "}]\n   tests: [{name: fixture, url: '" + origin.URL + "', status: 200}]\n"
		if second {
			text += " - id: second\n   domains: [127.0.0.1]\n   regex: [{match: First, replace: Wrong}]\n   tests: [{name: fixture, url: '" + origin.URL + "', status: 200}]\n"
		}
		filename := filepath.Join(t.TempDir(), id+".yaml")
		require.NoError(t, os.WriteFile(filename, []byte(text), 0600))
		return filename
	}
	first := makePack("first-pack", "Original", "First", true)
	second := makePack("second-pack", "First", "Final", false)
	s := newService(t, Config{PackPaths: []string{first, second}})
	f := fetch(t, s, origin.URL, "")
	transformed, err := s.TransformHTML(f)
	require.NoError(t, err)
	require.Contains(t, string(transformed), "<h1>Final</h1>")
	require.NotContains(t, string(transformed), "Wrong")
	require.Equal(t, []RuleMatch{{PackID: "first-pack", PackVersion: "1.0.0", RuleID: "first"}, {PackID: "second-pack", PackVersion: "1.0.0", RuleID: "first"}}, f.Metadata.Rules)
}

func TestCacheMetadataReflectsStoredEntriesAndReloadEvictsOldSuccess(t *testing.T) {
	var fail atomic.Bool
	var hits atomic.Int64
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if fail.Load() {
			w.WriteHeader(500)
		}
		io.WriteString(w, "payload")
	}))
	defer origin.Close()
	tiny := newService(t, Config{CacheTTL: time.Minute, CacheMaxBytes: 1})
	f := fetch(t, tiny, origin.URL, "")
	require.True(t, f.Metadata.Cache.ExpiresAt.IsZero())
	require.False(t, fetch(t, tiny, origin.URL, "").Metadata.Cache.Hit)
	s := newService(t, Config{CacheTTL: time.Minute})
	fetch(t, s, origin.URL, "")
	fail.Store(true)
	f = fetch(t, s, origin.URL, "reload")
	require.Equal(t, 500, f.Metadata.UpstreamStatus)
	require.True(t, f.Metadata.Cache.ExpiresAt.IsZero())
	fail.Store(false)
	require.False(t, fetch(t, s, origin.URL, "").Metadata.Cache.Hit)
	require.Equal(t, int64(5), hits.Load())
}

func TestResponsePrivacyHeadersAcrossFieldLines(t *testing.T) {
	cases := []struct {
		name    string
		headers http.Header
	}{
		{"no-store", http.Header{"Cache-Control": {"max-age=60", "no-store"}}},
		{"private", http.Header{"Cache-Control": {"max-age=60", "private"}}},
		{"no-cache", http.Header{"Cache-Control": {"max-age=60", "no-cache"}}},
		{"vary-star", http.Header{"Cache-Control": {"max-age=60"}, "Vary": {"Accept-Language", "Accept-Encoding, *"}}},
		{"later-cookie", http.Header{"Cache-Control": {"max-age=60"}, "Set-Cookie": {"", "session=private"}}},
		{"ambiguous-age", http.Header{"Cache-Control": {"max-age=60"}, "Age": {"0", "50"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int64
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				for key, values := range tc.headers {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.Header().Set("Content-Type", "text/plain")
				io.WriteString(w, "private payload")
			}))
			defer origin.Close()
			s := newService(t, Config{CacheTTL: time.Minute})
			for range 2 {
				f := fetch(t, s, origin.URL, "")
				require.False(t, f.Metadata.Cache.Hit, "privacy response must not be reused")
				require.True(t, f.Metadata.Cache.ExpiresAt.IsZero(), "uncached response must not claim freshness")
			}
			require.Equal(t, int64(2), hits.Load())
		})
	}
}
