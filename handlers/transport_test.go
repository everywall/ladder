package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"ladder/internal/service"
	"ladder/pkg/ruleset"
)

func fixtureApp(t *testing.T) (*fiber.App, *httptest.Server, []byte) {
	t.Helper()
	binary := []byte{0, 255, 254, 60, 105, 109, 103, 32, 115, 114, 99, 61, 34, 47, 120, 34, 62, 0}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status/404":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(404)
			io.WriteString(w, "<p>missing marker</p>")
		case "/status/500":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(500)
			io.WriteString(w, "<p>failed marker</p>")
		case "/redirect-error":
			http.Redirect(w, r, "/status/404", 302)
		case "/redirect":
			http.Redirect(w, r, "/redirected/page.html", 302)
		case "/redirected/page.html":
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, `<h1>Fixture article</h1><img src="nested.png"><script src="/assets/app.js"></script><link href="//`+r.Host+`/assets/style.css"><img src='/assets/pixel.png'><div style="background:url('/assets/pixel.png')"></div>`)
		case "/assets/app.js":
			w.Header().Set("Content-Type", "application/javascript")
			io.WriteString(w, "window.fixture=1;")
		case "/assets/style.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			io.WriteString(w, "body { color: green }")
		case "/assets/pixel.png", "/redirected/nested.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(binary)
		case "/binary":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(binary)
		case "/query":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(r.URL.Query())
		default:
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, "<h1>Fixture OK</h1>")
		}
	}))
	t.Cleanup(origin.Close)
	s, err := service.New(service.Config{})
	require.NoError(t, err)
	Configure(s)
	t.Cleanup(func() { Configure(nil) })
	app := fiber.New()
	app.Post("/api", Api)
	app.Get("/api/*", Api)
	app.Get("/raw/*", Raw)
	app.Get("/*", ProxySite(""))
	return app, origin, binary
}
func appRequest(t *testing.T, app *fiber.App, method, target, contentType string, body []byte) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := app.Test(req, 5000)
	require.NoError(t, err)
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, got
}
func TestLegacyStatusAndAPIExplicitMetadata(t *testing.T) {
	app, origin, _ := fixtureApp(t)
	for _, tc := range []struct {
		path   string
		status int
		marker string
	}{{"/", 200, "Fixture OK"}, {"/status/404", 404, "missing marker"}, {"/status/500", 500, "failed marker"}, {"/redirect-error", 404, "missing marker"}} {
		for _, prefix := range []string{"/", "/raw/", "/api/"} {
			t.Run(prefix+tc.path, func(t *testing.T) {
				resp, body := appRequest(t, app, "GET", prefix+origin.URL+tc.path, "", nil)
				if prefix == "/api/" {
					require.Equal(t, 200, resp.StatusCode)
					var got Response
					require.NoError(t, json.Unmarshal(body, &got))
					require.Equal(t, tc.status, got.UpstreamStatus)
					require.Equal(t, tc.status, got.Response.Status)
					require.Equal(t, tc.status < 400, got.OK)
					require.Contains(t, got.Body, tc.marker)
					if tc.path == "/redirect-error" {
						require.Equal(t, origin.URL+"/status/404", got.FinalURL)
					}
				} else {
					require.Equal(t, tc.status, resp.StatusCode)
					require.Contains(t, string(body), tc.marker)
				}
			})
		}
	}
}
func TestLegacyQueryEncodingAndJSONCharset(t *testing.T) {
	app, origin, _ := fixtureApp(t)
	query := "a=hello%20world&literal=a%26b%3Dc&plus=a%2Bb&dup=one&dup=two&empty=&unicode=%C3%A9"
	expected := url.Values{"a": {"hello world"}, "literal": {"a&b=c"}, "plus": {"a+b"}, "dup": {"one", "two"}, "empty": {""}, "unicode": {"é"}}
	for _, prefix := range []string{"/", "/raw/", "/api/"} {
		resp, body := appRequest(t, app, "GET", prefix+origin.URL+"/query?"+query, "", nil)
		require.Equal(t, 200, resp.StatusCode)
		if prefix == "/api/" {
			var got Response
			require.NoError(t, json.Unmarshal(body, &got))
			body = []byte(got.Body)
		}
		var values url.Values
		require.NoError(t, json.Unmarshal(body, &values))
		require.Equal(t, expected, values)
	}
	input, _ := json.Marshal(JsonRequest{URL: origin.URL + "/query?" + query})
	resp, body := appRequest(t, app, "POST", "/api", "application/json; charset=utf-8", input)
	require.Equal(t, 200, resp.StatusCode)
	var got Response
	require.NoError(t, json.Unmarshal(body, &got))
	var values url.Values
	require.NoError(t, json.Unmarshal([]byte(got.Body), &values))
	require.Equal(t, expected, values)
	for _, tc := range []struct {
		content, body string
		status        int
	}{{"application/json", "{", 400}, {"application/json", `{"url":"invalid-url"}`, 400}, {"text/plain", `{"url":"` + origin.URL + `"}`, 415}, {"application/json", `{} {}`, 400}} {
		resp, _ := appRequest(t, app, "POST", "/api", tc.content, []byte(tc.body))
		require.Equal(t, tc.status, resp.StatusCode)
	}
	for _, prefix := range []string{"/", "/raw/", "/api/"} {
		resp, _ := appRequest(t, app, "GET", prefix+"invalid-url", "", nil)
		require.Equal(t, 400, resp.StatusCode)
	}
}
func TestLegacyRawAndBinaryPreserveBytesTypesAndSandbox(t *testing.T) {
	app, origin, binary := fixtureApp(t)
	for _, tc := range []struct{ path, kind string }{{"/binary", "application/octet-stream"}, {"/assets/pixel.png", "image/png"}, {"/assets/app.js", "application/javascript"}, {"/assets/style.css", "text/css; charset=utf-8"}, {"/query", "application/json; charset=utf-8"}, {"/redirected/page.html", "text/html"}} {
		direct, err := http.Get(origin.URL + tc.path)
		require.NoError(t, err)
		expected, err := io.ReadAll(direct.Body)
		direct.Body.Close()
		require.NoError(t, err)
		resp, got := appRequest(t, app, "GET", "/raw/"+origin.URL+tc.path, "", nil)
		require.Equal(t, expected, got)
		require.Equal(t, tc.kind, resp.Header.Get("Content-Type"))
		require.Contains(t, resp.Header.Get("Content-Security-Policy"), "sandbox")
		require.NotContains(t, resp.Header.Get("Content-Security-Policy"), "allow-same-origin")
	}
	for _, prefix := range []string{"/", "/raw/"} {
		resp, body := appRequest(t, app, "GET", prefix+origin.URL+"/binary", "", nil)
		require.Equal(t, binary, body)
		require.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
	}
	_, body := appRequest(t, app, "GET", "/api/"+origin.URL+"/binary", "", nil)
	var got Response
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, "base64", got.BodyEncoding)
	decoded, err := base64.StdEncoding.DecodeString(got.Body)
	require.NoError(t, err)
	require.Equal(t, binary, decoded)
}
func TestLegacyRewriteFollowsActualFinalAssetURLs(t *testing.T) {
	app, origin, binary := fixtureApp(t)
	resp, body := appRequest(t, app, "GET", "/"+origin.URL+"/redirect", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Security-Policy"), "sandbox")
	require.NotContains(t, resp.Header.Get("Content-Security-Policy"), "allow-same-origin")
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	require.NoError(t, err)
	for _, tc := range []struct {
		selector, attr, expected, kind string
		payload                        []byte
	}{{"img", "src", origin.URL + "/redirected/nested.png", "image/png", binary}, {"script", "src", origin.URL + "/assets/app.js", "application/javascript", []byte("window.fixture=1;")}, {"link", "href", origin.URL + "/assets/style.css", "text/css; charset=utf-8", []byte("body { color: green }")}} {
		actual, ok := doc.Find(tc.selector).First().Attr(tc.attr)
		require.True(t, ok)
		require.Equal(t, "/"+tc.expected, actual)
		asset, got := appRequest(t, app, "GET", actual, "", nil)
		require.Equal(t, 200, asset.StatusCode)
		require.Equal(t, tc.payload, got)
		require.Equal(t, tc.kind, asset.Header.Get("Content-Type"))
	}
	// A different authority is respected by a network-path reference.
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "second authority") }))
	defer second.Close()
	u, _ := url.Parse(origin.URL + "/page")
	host := strings.TrimPrefix(second.URL, "http://")
	html := rewriteHtml([]byte(`<img src="//`+host+`/image">`), u, ruleset.Rule{})
	require.Contains(t, html, `src="/`+second.URL+`/image"`)
	httpsBase, _ := url.Parse("https://example.org/page")
	html = rewriteHtml([]byte(`<img src="//cdn.example.net/x"><img src="/y">`), httpsBase, ruleset.Rule{})
	require.Contains(t, html, `/https://cdn.example.net/x`)
	require.Contains(t, html, `/https://example.org/y`)
}

func TestLegacyRouteBoundaryPreservesEscapedPathAndFullyEncodedTarget(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"path": r.URL.EscapedPath(), "query": r.URL.Query()})
	}))
	defer origin.Close()
	Configure(newServiceForHandlers(t))
	defer Configure(nil)
	app := fiber.New()
	app.Get("/raw/*", Raw)
	target := origin.URL + "/encoded/a%2Fb?literal=a%26b%3Dc&plus=a%2Bb&dup=one&dup=two"
	for _, route := range []string{"/raw/" + target, "/raw/" + url.PathEscape(target)} {
		resp, body := appRequest(t, app, "GET", route, "", nil)
		require.Equal(t, 200, resp.StatusCode)
		var got struct {
			Path  string     `json:"path"`
			Query url.Values `json:"query"`
		}
		require.NoError(t, json.Unmarshal(body, &got))
		require.Equal(t, "/encoded/a%2Fb", got.Path)
		require.Equal(t, []string{"one", "two"}, got.Query["dup"])
		require.Equal(t, "a&b=c", got.Query.Get("literal"))
		require.Equal(t, "a+b", got.Query.Get("plus"))
	}
}
func newServiceForHandlers(t *testing.T) *service.Service {
	t.Helper()
	s, err := service.New(service.Config{})
	require.NoError(t, err)
	return s
}

func TestLegacyStructuredMetadataAndRulesNeverExposeCookies(t *testing.T) {
	const sentinel = "fixture-credential-not-for-metadata"
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "session="+sentinel, r.Header.Get("Cookie"))
		w.Header().Set("Set-Cookie", "session="+sentinel)
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<article>Public fixture body</article>")
	}))
	defer origin.Close()
	rule := ruleset.Rule{Domain: "127.0.0.1"}
	rule.Headers.Cookie = "session=" + sentinel
	s, err := service.New(service.Config{Rules: ruleset.RuleSet{rule}})
	require.NoError(t, err)
	Configure(s)
	defer Configure(nil)
	t.Setenv("EXPOSE_RULESET", "true")
	app := fiber.New()
	app.Get("/api/*", Api)
	app.Get("/ruleset", Ruleset)
	for _, target := range []string{"/api/" + origin.URL, "/ruleset"} {
		resp, body := appRequest(t, app, "GET", target, "", nil)
		require.Equal(t, 200, resp.StatusCode)
		require.NotContains(t, string(body), sentinel)
		require.Contains(t, string(body), "[REDACTED]")
	}
}
