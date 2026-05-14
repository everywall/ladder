// BEGIN: 6f8b3f5d5d5d
package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"ladder/pkg/ruleset"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestProxySite(t *testing.T) {
	app := fiber.New()
	app.Get("/:url", ProxySite(""))

	req := httptest.NewRequest("GET", "/https://example.com", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProxySiteRelativeUrlWithoutReferer(t *testing.T) {
	app := fiber.New()
	app.Get("/*", ProxySite(""))

	// Request a relative path without a Referer header.
	// The handler should return a helpful error instead of a
	// confusing "http: no Host in request URL" from the Go HTTP client.
	req := httptest.NewRequest("GET", "/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "Referer")
}

func TestProxySiteRelativeUrlWithRefererMissingHost(t *testing.T) {
	app := fiber.New()
	app.Get("/*", ProxySite(""))

	// Request a relative path with a referer that has no scheme/host.
	// The handler should return a helpful error about the malformed referer.
	req := httptest.NewRequest("GET", "/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1", nil)
	req.Header.Set("Referer", "/relative/path")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(body), "no host") || strings.Contains(string(body), "Referer"))
}

func TestRewriteHtml(t *testing.T) {
	bodyB := []byte(`
		<html>
			<head>
				<title>Test Page</title>
			</head>
			<body>
				<img src="/image.jpg">
				<script src="/script.js"></script>
				<a href="/about">About Us</a>
				<div style="background-image: url('/background.jpg')"></div>
			</body>
		</html>
	`)
	u := &url.URL{Host: "example.com"}

	expected := `
		<html>
			<head>
				<title>Test Page</title>
			</head>
			<body>
				<img src="/https://example.com/image.jpg">
				<script script="/https://example.com/script.js"></script>
				<a href="/https://example.com/about">About Us</a>
				<div style="background-image: url('/https://example.com/background.jpg')"></div>
			</body>
		</html>
	`

	actual := rewriteHtml(bodyB, u, ruleset.Rule{})
	assert.Equal(t, expected, actual)
}

// END: 6f8b3f5d5d5d
