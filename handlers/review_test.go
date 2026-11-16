package handlers

import (
	"bytes"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"io"
	"ladder/internal/service"
	"ladder/pkg/ruleset"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLegacyRuleCSPIsEnforcedAlongsideOpaqueSandbox(t *testing.T) {
	app, origin, _ := fixtureApp(t)
	u, _ := url.Parse(origin.URL)
	rule := ruleset.Rule{Domain: u.Hostname()}
	rule.Headers.CSP = "script-src 'self';"
	svc, err := service.New(service.Config{Rules: ruleset.RuleSet{rule}})
	require.NoError(t, err)
	Configure(svc)
	for _, prefix := range []string{"/", "/raw/"} {
		response, _ := appRequest(t, app, "GET", prefix+origin.URL+"/redirected/page.html", "", nil)
		require.Equal(t, 200, response.StatusCode)
		policies := response.Header.Values("Content-Security-Policy")
		require.Len(t, policies, 2, "both independent policies must reach the browser")
		require.Contains(t, strings.Join(policies, "\n"), "sandbox allow-scripts")
		require.Contains(t, policies, rule.Headers.CSP)
		require.NotContains(t, strings.Join(policies, "\n"), "allow-same-origin")
	}
}

func TestLegacySameOriginPolicyAndExplicitRefererResolution(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page":
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, `<script>const dynamic="/dynamic.png";</script>`)
		case "/style.css":
			w.Header().Set("Content-Type", "text/css")
			io.WriteString(w, `body{background:url(/dynamic.png)}`)
		case "/dynamic.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte{0, 255, 42})
		default:
			w.WriteHeader(404)
		}
	}))
	defer origin.Close()
	svc, err := service.New(service.Config{})
	require.NoError(t, err)
	Configure(svc)
	t.Cleanup(func() { Configure(nil) })
	app := fiber.New()
	app.Get("/*", ProxySite(""))
	for _, path := range []string{"/page", "/style.css"} {
		response, _ := appRequest(t, app, "GET", "/"+origin.URL+path, "", nil)
		require.Equal(t, "same-origin", response.Header.Get("Referrer-Policy"), "the browser must be allowed to return the proxy URL only to Ladder")
		request := httptest.NewRequest("GET", "/dynamic.png", nil)
		request.Header.Set("Referer", "http://example.com/"+origin.URL+path)
		asset, err := app.Test(request, 5000)
		require.NoError(t, err)
		got, err := io.ReadAll(asset.Body)
		asset.Body.Close()
		require.NoError(t, err)
		require.Equal(t, 200, asset.StatusCode)
		require.True(t, bytes.Equal([]byte{0, 255, 42}, got))
	}
}

func TestLegacyRawBodyLimitCanBeRaisedThroughEnvironment(t *testing.T) {
	payload := bytes.Repeat([]byte{0x5a}, 9<<20)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write(payload)
	}))
	defer origin.Close()
	app := fiber.New()
	app.Get("/raw/*", Raw)
	t.Cleanup(func() { Configure(nil) })
	cfg, err := service.ConfigFromLookup(func(string) (string, bool) { return "", false })
	require.NoError(t, err)
	svc, err := service.New(cfg)
	require.NoError(t, err)
	Configure(svc)
	response, _ := appRequest(t, app, "GET", "/raw/"+origin.URL, "", nil)
	require.Equal(t, 502, response.StatusCode)
	cfg, err = service.ConfigFromLookup(func(key string) (string, bool) {
		if key == "MAX_BODY_BYTES" {
			return "16777216", true
		}
		return "", false
	})
	require.NoError(t, err)
	require.Equal(t, int64(16<<20), cfg.MaxBodyBytes)
	svc, err = service.New(cfg)
	require.NoError(t, err)
	Configure(svc)
	response, body := appRequest(t, app, "GET", "/raw/"+origin.URL, "", nil)
	require.Equal(t, 200, response.StatusCode)
	require.Equal(t, "application/pdf", response.Header.Get("Content-Type"))
	require.True(t, bytes.Equal(payload, body), "raised limit must serve exact original bytes")
}
