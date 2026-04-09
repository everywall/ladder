// BEGIN: 6f8b3f5d5d5d
package handlers

import (
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

func TestSplitProxyQueries(t *testing.T) {
	queries := map[string]string{
		"q":                     "news",
		"page":                  "2",
		"__ladder_debug":        "1",
		"__ladder_show_request": "1",
		"__ladder_other":        "x",
	}

	proxyQueries, options := splitProxyQueries(queries)

	assert.Equal(t, map[string]string{
		"q":    "news",
		"page": "2",
	}, proxyQueries)
	assert.True(t, options.Enabled)
	assert.True(t, options.ShowRequest)
}

func TestInjectDebugUI(t *testing.T) {
	body := `<html><body><p>hello</p></body></html>`
	updated := injectDebugUI(body, debugUIOptions{Enabled: true, ShowRequest: true})

	assert.True(t, strings.Contains(updated, `id="__ladderGear"`))
	assert.True(t, strings.Contains(updated, `id="__ladderPanel"`))
	assert.True(t, strings.Contains(updated, `data-param="__ladder_debug" checked`))
	assert.True(t, strings.Contains(updated, `data-param="__ladder_show_request" checked`))
	assert.True(t, strings.Contains(updated, `id="__ladderDebugInfo"`))
}

// END: 6f8b3f5d5d5d
