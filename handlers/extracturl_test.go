package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// callExtractUrl runs extractUrl against a real request so that the fiber
// wildcard param and the referer header behave exactly as they do in the proxy.
func callExtractUrl(t *testing.T, target string, referer string) (string, error) {
	t.Helper()

	var (
		got    string
		gotErr error
	)

	app := fiber.New()
	app.Get("/*", func(c *fiber.Ctx) error {
		got, gotErr = extractUrl(c)
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, target, nil)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	_, err := app.Test(req)
	require.NoError(t, err)

	return got, gotErr
}

func TestExtractUrlAbsolute(t *testing.T) {
	got, err := callExtractUrl(t, "/https://example.com/images/foobar.jpg", "")

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/images/foobar.jpg", got)
}

// A root-relative asset request carries the target site only in the referer.
// The query string is not part of the wildcard param, so it is not expected
// here; fetchSite re-attaches it from c.Queries().
func TestExtractUrlRelativeWithProxiedReferer(t *testing.T) {
	got, err := callExtractUrl(t,
		"/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1?ray=92fef3e48ae3f7e5",
		"http://localhost:8080/https://www.philanthropy.com/article/crypto-charity-leader-sentenced-for-tax-evasion")

	assert.NoError(t, err)
	assert.Equal(t,
		"https://www.philanthropy.com/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1",
		got)
}

// Regression test for #90. A relative path with no referer used to be returned
// verbatim with a nil error, so it reached the HTTP client as a URL with no
// scheme and failed there with `unsupported protocol scheme ""`.
func TestExtractUrlRelativeWithoutRefererIsAnError(t *testing.T) {
	cases := []struct {
		name    string
		target  string
		referer string
	}{
		{
			name:   "cloudflare challenge fetch, no referer",
			target: "/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1?ray=92fef3e48ae3f7e5",
		},
		{
			name:   "cloudflare redirect target, no referer",
			target: "/techestination/claim-your-free-google-cloud-credits?__cf_chl_rt_tk=lXRX198gSrWZJ8aDan",
		},
		{
			name:    "referer is the ladder form, not a proxied page",
			target:  "/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1?ray=92fef3e48ae3f7e5",
			referer: "http://localhost:8080/",
		},
		{
			name:    "referer path is not a URL",
			target:  "/assets/app.js",
			referer: "http://localhost:8080/not-a-url",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := callExtractUrl(t, tc.target, tc.referer)

			assert.Error(t, err, "an unresolvable relative path must be reported, not passed on")
			assert.Empty(t, got, "no URL should be returned when it cannot be resolved")
		})
	}
}

// The proxied URL sits after BASE_PATH in the referer, not at the root.
func TestExtractUrlRelativeWithBasePathReferer(t *testing.T) {
	original := basePath
	basePath = "/proxy"
	defer func() { basePath = original }()

	got, err := callExtractUrl(t, "/assets/app.js",
		"http://localhost:8080/proxy/https://example.com/article")

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/assets/app.js", got)
}

func TestExtractUrlRejectsNonHTTPScheme(t *testing.T) {
	got, err := callExtractUrl(t, "/ftp://example.com/pub/file.txt", "")

	assert.Error(t, err)
	assert.Empty(t, got)
}

// The handler must stop on an unresolvable URL instead of fetching it: the old
// code only logged the extraction error and carried on into fetchSite, which is
// where `unsupported protocol scheme ""` surfaced as a 500.
func TestProxySiteRejectsUnresolvableRelativePath(t *testing.T) {
	app := fiber.New()
	app.Get("/*", ProxySite(""))

	req := httptest.NewRequest(http.MethodGet,
		"/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1?ray=92fef3e48ae3f7e5", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.NotContains(t, string(body), "unsupported protocol scheme")
}
