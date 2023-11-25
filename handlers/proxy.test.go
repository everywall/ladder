// BEGIN: 6f8b3f5d5d5d
package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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

// END: 6f8b3f5d5d5d
