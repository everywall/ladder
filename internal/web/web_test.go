package web

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ladder/internal/localstate"
	"ladder/internal/service"
	"ladder/internal/startup"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	hits := new(atomic.Int64)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/article?literal=%252F", 302)
			return
		case "/missing":
			w.WriteHeader(404)
			fmt.Fprint(w, "<html><title>Page not found</title><main><h1>Page not found</h1><p>The requested story no longer exists. Try the archive.</p></main></html>")
			return
		case "/empty":
			fmt.Fprint(w, "<html><title>Empty page</title><body><script>parent.evil=true</script></body></html>")
			return
		}
		fmt.Fprint(w, `<html><head><title>A useful &amp; quiet story</title></head><body><nav>Navigation clutter</nav><main><h1>A useful &amp; quiet story</h1><p>This is the story, with enough real words to make a readable paragraph. A link leads to <a href="https://example.org/context">more context</a>.</p><p>A second paragraph gives this page a little structure and more useful text. It is a deliberate fixture for the reading view.</p><script>parent.evil=true</script><img src="https://example.org/tracker" onerror="alert(1)"></main></body></html>`)
	}))
	t.Cleanup(upstream.Close)
	return upstream, hits
}

func testApp(t *testing.T, cfg service.Config, options Options) (*fiber.App, *service.Service) {
	t.Helper()
	svc, err := service.New(cfg)
	require.NoError(t, err)
	if options.StartupStatus == nil {
		options.StartupStatus = func() (startup.State, error) { return startup.State{Supported: true, Manager: "test manager"}, nil }
	}
	app := fiber.New(fiber.Config{StrictRouting: true})
	Register(app.Group(normalizeBase(options.BasePath)), svc, options)
	return app, svc
}

func store(t *testing.T) *localstate.Store {
	t.Helper()
	state, err := localstate.Open(t.TempDir())
	require.NoError(t, err)
	return state
}

func request(t *testing.T, app *fiber.App, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("X-Ladder-Client", "reader")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	response, err := app.Test(req, 10000)
	require.NoError(t, err)
	t.Cleanup(func() { response.Body.Close() })
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response, data
}

func read(t *testing.T, app *fiber.App, prefix, target, cache string) service.Document {
	t.Helper()
	payload, err := json.Marshal(service.Request{URL: target, Cache: cache})
	require.NoError(t, err)
	response, body := request(t, app, "POST", prefix+"/api/v1/read", string(payload), nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	var doc service.Document
	require.NoError(t, json.Unmarshal(body, &doc))
	return doc
}

func TestReadPreservesMetadataAndIsolatesOutboundLinks(t *testing.T) {
	upstream, hits := fixture(t)
	state := store(t)
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	app, svc := testApp(t, service.Config{CacheTTL: time.Minute, Now: func() time.Time { return now }}, Options{Store: state})
	target := upstream.URL + "/redirect"
	doc := read(t, app, "", target, "default")
	require.Equal(t, "A useful & quiet story", doc.Title)
	require.Equal(t, target, doc.Metadata.SourceURL)
	require.Equal(t, upstream.URL+"/article?literal=%252F", doc.Metadata.FinalURL)
	require.Equal(t, 200, doc.Metadata.UpstreamStatus)
	require.Equal(t, "text/html", doc.Metadata.MediaType)
	require.Equal(t, now, doc.Metadata.FetchedAt)
	require.Equal(t, int64(60), doc.Metadata.Cache.TTLSeconds)
	require.Equal(t, now.Add(time.Minute), doc.Metadata.Cache.ExpiresAt)
	require.Contains(t, doc.Content.Markdown, "more context")
	require.Contains(t, doc.Content.Text, "This is the story")
	require.Contains(t, doc.Content.HTML, `target="_blank"`)
	require.Contains(t, doc.Content.HTML, `rel="noopener noreferrer"`)
	require.NotContains(t, doc.Content.HTML, "<script")
	require.NotContains(t, doc.Content.HTML, "<img")
	require.NotContains(t, doc.Content.HTML, "onerror")
	require.Equal(t, int64(2), hits.Load())
	now = now.Add(10 * time.Second)
	doc = read(t, app, "", target, "default")
	require.True(t, doc.Metadata.Cache.Hit)
	require.Equal(t, int64(10), doc.Metadata.Cache.AgeSeconds)
	require.Equal(t, int64(2), hits.Load())
	doc = read(t, app, "", target, "reload")
	require.False(t, doc.Metadata.Cache.Hit)
	require.Equal(t, int64(4), hits.Load())
	response, body := request(t, app, "DELETE", "/api/v1/cache", "", nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	require.False(t, read(t, app, "", target, "default").Metadata.Cache.Hit)
	require.Equal(t, int64(6), hits.Load())
	snapshot, err := state.SnapshotChecked()
	require.NoError(t, err)
	require.Len(t, snapshot.Entries, 1)
	require.Equal(t, target, snapshot.Entries[0].URL)
	// Non-web service reads do not become browser history.
	_, err = svc.Read(context.Background(), service.Request{URL: upstream.URL + "/article"})
	require.NoError(t, err)
	snapshot, err = state.SnapshotChecked()
	require.NoError(t, err)
	require.Len(t, snapshot.Entries, 1)
}

func TestUpstreamErrorsAndEmptyPagesRemainExplicit(t *testing.T) {
	upstream, _ := fixture(t)
	app, _ := testApp(t, service.Config{}, Options{Store: store(t)})
	doc := read(t, app, "", upstream.URL+"/missing", "")
	require.Equal(t, 404, doc.Metadata.UpstreamStatus)
	require.Contains(t, doc.Content.Text, "no longer exists")
	require.Contains(t, doc.Problems, service.Problem{Code: "upstream_status", Message: "upstream returned an HTTP error"})
	doc = read(t, app, "", upstream.URL+"/empty", "")
	require.Empty(t, doc.Content.Text)
	require.NotEmpty(t, doc.Problems)
	upstream.Close()
	payload, err := json.Marshal(service.Request{URL: upstream.URL + "/article", Cache: "bypass"})
	require.NoError(t, err)
	response, body := request(t, app, "POST", "/api/v1/read", string(payload), nil)
	require.Equal(t, 502, response.StatusCode, string(body))
	require.Contains(t, string(body), "upstream_error")
	require.Equal(t, "application/json", strings.Split(response.Header.Get("Content-Type"), ";")[0])
}

func TestAPIOriginAndHostBoundary(t *testing.T) {
	app, _ := testApp(t, service.Config{}, Options{Store: store(t), TrustedHosts: []string{"ladder.local"}})
	for _, tc := range []struct {
		name, host, origin, marker, site string
		status                           int
	}{
		{"same origin", "localhost", "http://localhost", "reader", "same-origin", 200},
		{"header only for same-origin GET", "localhost", "", "reader", "", 200},
		{"missing marker", "localhost", "http://localhost", "", "", 403},
		{"opaque legacy frame", "localhost", "null", "reader", "", 403},
		{"foreign origin", "localhost", "https://evil.example", "reader", "", 403},
		{"lookalike origin", "localhost", "http://localhost.evil.example", "reader", "", 403},
		{"origin path is invalid", "localhost", "http://localhost/", "reader", "", 403},
		{"rebinding host", "evil.example", "http://evil.example", "reader", "", 403},
		{"explicit custom host", "ladder.local:8080", "http://ladder.local:8080", "reader", "same-origin", 200},
		{"cross-site fetch metadata", "localhost", "http://localhost", "reader", "cross-site", 403},
		{"IPv6 loopback", "[::1]:8080", "http://[::1]:8080", "reader", "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://"+tc.host+"/api/v1/history", nil)
			require.NoError(t, err)
			req.Header.Set("X-Ladder-Client", tc.marker)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Sec-Fetch-Site", tc.site)
			req.Header.Set("X-Forwarded-Host", "localhost")
			response, err := app.Test(req)
			require.NoError(t, err)
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, tc.status, response.StatusCode, string(body))
			require.Empty(t, response.Header.Get("Access-Control-Allow-Origin"))
			if tc.name == "rebinding host" {
				require.Contains(t, string(body), "TRUSTED_HOSTS")
			}
		})
	}
	response, body := request(t, app, "OPTIONS", "/api/v1/history", "", map[string]string{"X-Ladder-Client": "", "Origin": "https://evil.example", "Access-Control-Request-Headers": "X-Ladder-Client"})
	require.Equal(t, 403, response.StatusCode, string(body))
	require.Empty(t, response.Header.Get("Access-Control-Allow-Origin"))
}

func TestJSONPayloadValidationDoesNotRunOrMutate(t *testing.T) {
	upstream, hits := fixture(t)
	state := store(t)
	app, _ := testApp(t, service.Config{}, Options{Store: state})
	for _, tc := range []struct {
		name, path, method, body, contentType string
		status                                int
	}{
		{"wrong content type", "/read", "POST", `{"url":"` + upstream.URL + `"}`, "text/plain", 415},
		{"broken JSON", "/read", "POST", `{"url":`, "application/json", 400},
		{"unknown field", "/read", "POST", `{"url":"` + upstream.URL + `","surprise":true}`, "application/json", 400},
		{"array", "/read", "POST", `[]`, "application/json", 400},
		{"null", "/read", "POST", `null`, "application/json", 400},
		{"trailing object", "/read", "POST", `{} {}`, "application/json", 400},
		{"oversized", "/read", "POST", `{"url":"` + strings.Repeat("a", maxJSONBytes) + `"}`, "application/json", 413},
		{"invalid URL", "/read", "POST", `{"url":"javascript:alert(1)"}`, "application/json", 400},
		{"invalid cache", "/read", "POST", `{"url":"` + upstream.URL + `","cache":"forever"}`, "application/json", 400},
		{"settings missing flag", "/settings", "PATCH", `{}`, "application/json", 400},
		{"settings null", "/settings", "PATCH", `{"historyEnabled":null}`, "application/json", 400},
		{"settings wrong type", "/settings", "PATCH", `{"historyEnabled":"false"}`, "application/json", 400},
		{"GET cache never clears", "/cache", "GET", "", "", 404},
		{"GET history clear never clears", "/history/clear", "GET", "", "", 404},
		{"invalid entry ID", "/history/not-an-id", "DELETE", "", "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, body := request(t, app, tc.method, "/api/v1"+tc.path, tc.body, map[string]string{"Content-Type": tc.contentType})
			require.Equal(t, tc.status, response.StatusCode, string(body))
			require.Contains(t, string(body), `"error"`)
		})
	}
	require.Zero(t, hits.Load())
	require.True(t, state.Snapshot().HistoryEnabled)
	require.Empty(t, state.Snapshot().Entries)
}

func TestHistorySettingsDeleteClearAndPersistence(t *testing.T) {
	upstream, _ := fixture(t)
	dir := t.TempDir()
	state, err := localstate.Open(dir)
	require.NoError(t, err)
	app, _ := testApp(t, service.Config{}, Options{Store: state})
	read(t, app, "", upstream.URL+"/article", "")
	read(t, app, "", upstream.URL+"/missing", "")
	response, body := request(t, app, "GET", "/api/v1/history", "", nil)
	require.Equal(t, 200, response.StatusCode)
	var snapshot localstate.Snapshot
	require.NoError(t, json.Unmarshal(body, &snapshot))
	require.Len(t, snapshot.Entries, 2)
	require.NotContains(t, string(body), "This is the story")
	response, body = request(t, app, "PATCH", "/api/v1/settings", `{"historyEnabled":false}`, nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	read(t, app, "", upstream.URL+"/other", "")
	require.Len(t, state.Snapshot().Entries, 2)
	reopened, err := localstate.Open(dir)
	require.NoError(t, err)
	require.False(t, reopened.Snapshot().HistoryEnabled)
	response, body = request(t, app, "DELETE", "/api/v1/history/"+snapshot.Entries[0].ID, "", nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	require.Len(t, state.Snapshot().Entries, 1)
	response, body = request(t, app, "DELETE", "/api/v1/history", "", nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	require.Empty(t, state.Snapshot().Entries)
	response, body = request(t, app, "PATCH", "/api/v1/settings", `{"historyEnabled":true}`, nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	read(t, app, "", upstream.URL+"/other", "")
	require.Len(t, state.Snapshot().Entries, 1)
}

func TestHistoryFailuresAreVisibleAndDoNotHideThePage(t *testing.T) {
	upstream, _ := fixture(t)
	dir := t.TempDir()
	state, err := localstate.Open(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "history.json"), []byte("broken JSON"), 0600))
	app, _ := testApp(t, service.Config{}, Options{Store: state})
	response, body := request(t, app, "GET", "/api/v1/history", "", nil)
	require.Equal(t, 500, response.StatusCode, string(body))
	require.Contains(t, string(body), "history_read_failed")
	response, body = request(t, app, "PATCH", "/api/v1/settings", `{"historyEnabled":false}`, nil)
	require.Equal(t, 500, response.StatusCode, string(body))
	require.Contains(t, string(body), "history_write_failed")
	payload, err := json.Marshal(service.Request{URL: upstream.URL + "/article"})
	require.NoError(t, err)
	response, body = request(t, app, "POST", "/api/v1/read", string(payload), nil)
	require.Equal(t, 200, response.StatusCode)
	require.Contains(t, string(body), "historyError")
	require.Contains(t, string(body), "A useful")
	response, body = request(t, app, "GET", "/api/v1/settings", "", nil)
	require.Equal(t, 200, response.StatusCode)
	require.Contains(t, string(body), `"historyAvailable":false`)
	require.Contains(t, string(body), "historyError")
}

func TestRulePacksAndTestURLResultsDoNotAddHistory(t *testing.T) {
	upstream, _ := fixture(t)
	packFile := filepath.Join(t.TempDir(), "pack.yaml")
	pack := fmt.Sprintf("schemaVersion: 1\nid: local-fixture\nversion: 1.2.3\nrules:\n  - id: article\n    domains: [127.0.0.1]\n    tests:\n      - name: readable\n        url: %s/article\n        status: 200\n        contains: [story]\n", upstream.URL)
	require.NoError(t, os.WriteFile(packFile, []byte(pack), 0600))
	state := store(t)
	app, _ := testApp(t, service.Config{PackPaths: []string{packFile}}, Options{Store: state})
	response, body := request(t, app, "GET", "/api/v1/rules", "", nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	var packs struct {
		Packs []service.PackInfo `json:"packs"`
	}
	require.NoError(t, json.Unmarshal(body, &packs))
	require.Len(t, packs.Packs, 1)
	require.Equal(t, "1.2.3", packs.Packs[0].Version)
	require.Len(t, packs.Packs[0].SHA256, 64)
	payload, err := json.Marshal(service.Request{URL: upstream.URL + "/article", Cache: "bypass"})
	require.NoError(t, err)
	response, body = request(t, app, "POST", "/api/v1/rules/test", string(payload), nil)
	require.Equal(t, 200, response.StatusCode, string(body))
	var result service.TestResult
	require.NoError(t, json.Unmarshal(body, &result))
	require.True(t, result.Passed)
	require.Equal(t, 200, result.Metadata.UpstreamStatus)
	require.Len(t, result.Matched, 1)
	require.NotEmpty(t, result.Checks)
	require.Empty(t, state.Snapshot().Entries)
}

func TestShellAssetsCSPAndBasePath(t *testing.T) {
	upstream, _ := fixture(t)
	app, _ := testApp(t, service.Config{}, Options{BasePath: " /shelf/ ", Store: store(t)})
	for _, path := range []string{"/shelf/", "/shelf/reader"} {
		response, body := request(t, app, "GET", path, "", nil)
		require.Equal(t, 200, response.StatusCode, string(body))
		require.Contains(t, string(body), `data-base-path="/shelf"`)
		require.Contains(t, string(body), `src="/shelf/reader-assets/app.js"`)
		require.Contains(t, string(body), `sandbox="allow-popups allow-popups-to-escape-sandbox"`)
		require.NotContains(t, string(body), "allow-scripts")
		require.NotContains(t, string(body), "allow-same-origin")
		require.NotContains(t, string(body), "<script>")
		require.NotContains(t, string(body), "onclick=")
		require.NotContains(t, string(body), "https://fonts")
		require.Contains(t, response.Header.Get("Content-Security-Policy"), "script-src 'self'")
		require.Contains(t, response.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'")
		require.NotContains(t, response.Header.Get("Content-Security-Policy"), "unsafe-inline")
		require.Equal(t, "no-store", response.Header.Get("Cache-Control"))
		require.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
		require.Equal(t, "no-referrer", response.Header.Get("Referrer-Policy"))
	}
	response, body := request(t, app, "GET", "/shelf/reader-assets/app.js", "", nil)
	require.Equal(t, 200, response.StatusCode)
	sum := sha256.Sum256([]byte(ReaderStyle))
	require.Contains(t, string(body), base64.StdEncoding.EncodeToString(sum[:]))
	require.Contains(t, string(body), ".srcdoc =")
	require.NotContains(t, string(body), ".innerHTML")
	require.Contains(t, string(body), "encodeURIComponent(location.href)")
	require.Contains(t, string(body), "window.location.origin")
	response, body = request(t, app, "GET", "/shelf/reader-assets/style.css", "", nil)
	require.Equal(t, 200, response.StatusCode)
	require.Contains(t, string(body), "prefers-reduced-motion")
	require.NotContains(t, string(body), "@import")
	doc := read(t, app, "/shelf", upstream.URL+"/article", "")
	require.Equal(t, 200, doc.Metadata.UpstreamStatus)
}

func TestSettingsReportsActualInjectedStartupStatusWithoutEnabling(t *testing.T) {
	calls := 0
	app, _ := testApp(t, service.Config{}, Options{Store: store(t), StartupStatus: func() (startup.State, error) {
		calls++
		return startup.State{Supported: true, Installed: true, Enabled: true, Running: false, Manager: "fixture-manager"}, nil
	}})
	response, body := request(t, app, "GET", "/api/v1/settings", "", nil)
	require.Equal(t, 200, response.StatusCode)
	require.Contains(t, string(body), `"enabled":true`)
	require.Contains(t, string(body), `"running":false`)
	require.Contains(t, string(body), `"historyLimit":100`)
	require.Equal(t, 1, calls)
	app, _ = testApp(t, service.Config{}, Options{StartupStatus: func() (startup.State, error) { return startup.State{}, errors.New("fixture manager unavailable") }})
	response, body = request(t, app, "GET", "/api/v1/settings", "", nil)
	require.Equal(t, 200, response.StatusCode)
	require.Contains(t, string(body), "fixture manager unavailable")
	require.Contains(t, string(body), `"historyAvailable":false`)
	response, body = request(t, app, "GET", "/api/v1/history", "", nil)
	require.Equal(t, 503, response.StatusCode)
	require.Contains(t, string(body), "history_unavailable")
}
