package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestSetDefaultScheme(t *testing.T) {
	original := DefaultScheme()
	t.Cleanup(func() { _ = SetDefaultScheme(original) })

	assert.NoError(t, SetDefaultScheme("http"))
	assert.Equal(t, "http", DefaultScheme())

	assert.NoError(t, SetDefaultScheme("HTTPS"))
	assert.Equal(t, "https", DefaultScheme())

	assert.NoError(t, SetDefaultScheme("  https  "))
	assert.Equal(t, "https", DefaultScheme())

	for _, bad := range []string{"ftp", "ws", "", "javascript", "file"} {
		err := SetDefaultScheme(bad)
		assert.Error(t, err, "expected error for %q", bad)
	}
	// Last successful value should still be in place after rejected attempts.
	assert.Equal(t, "https", DefaultScheme())
}

func TestEnsureScheme(t *testing.T) {
	original := DefaultScheme()
	t.Cleanup(func() { _ = SetDefaultScheme(original) })

	require := func(scheme string) {
		t.Helper()
		if err := SetDefaultScheme(scheme); err != nil {
			t.Fatalf("SetDefaultScheme(%q) failed: %v", scheme, err)
		}
	}

	require("https")
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"already https", "https://example.com/page", "https://example.com/page"},
		{"already http", "http://example.com", "http://example.com"},
		{"schemeless host", "example.com/page", "https://example.com/page"},
		{"schemeless host with query", "example.com/page?q=1", "https://example.com/page?q=1"},
		{"schemeless host with port", "example.com:8443/page", "https://example.com:8443/page"},
		{"localhost with port", "localhost:8080/api", "https://localhost:8080/api"},
		{"truly relative path", "/images/foo.jpg", "/images/foo.jpg"},
		{"single segment no dot", "about", "about"},
		{"empty", "", ""},
		{"protocol-relative looks like host", "example.com", "https://example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ensureScheme(tc.in))
		})
	}

	require("http")
	assert.Equal(t, "http://example.com/page", ensureScheme("example.com/page"),
		"schemeless URL should pick up the configured non-default scheme")
	assert.Equal(t, "https://example.com/page", ensureScheme("https://example.com/page"),
		"explicit scheme must never be overridden")
}

func TestExtractUrlPrependsDefaultScheme(t *testing.T) {
	original := DefaultScheme()
	t.Cleanup(func() { _ = SetDefaultScheme(original) })
	require := func(scheme string) {
		t.Helper()
		if err := SetDefaultScheme(scheme); err != nil {
			t.Fatalf("SetDefaultScheme(%q) failed: %v", scheme, err)
		}
	}

	run := func(t *testing.T, target string, headers map[string]string) string {
		t.Helper()
		app := fiber.New()
		var captured string
		app.Get("/*", func(c *fiber.Ctx) error {
			u, err := extractUrl(c)
			if err != nil {
				return c.Status(500).SendString(err.Error())
			}
			captured = u
			return c.SendString("ok")
		})
		req := httptest.NewRequest("GET", target, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		return captured
	}

	require("https")
	t.Run("schemeless URL with no referer gets https", func(t *testing.T) {
		got := run(t, "/example.com/page", nil)
		assert.Equal(t, "https://example.com/page", got)
	})

	t.Run("https URL is unchanged", func(t *testing.T) {
		got := run(t, "/https://example.com/page", nil)
		assert.Equal(t, "https://example.com/page", got)
	})

	t.Run("http URL is unchanged", func(t *testing.T) {
		got := run(t, "/http://example.com/page", nil)
		assert.Equal(t, "http://example.com/page", got)
	})

	t.Run("relative path with proxied referer still reconstructs", func(t *testing.T) {
		got := run(t, "/images/foo.jpg", map[string]string{
			"Referer": "http://localhost:8080/https://realsite.com/article",
		})
		assert.Equal(t, "https://realsite.com/images/foo.jpg", got)
	})

	require("http")
	t.Run("configured http scheme is used", func(t *testing.T) {
		got := run(t, "/example.com/page", nil)
		assert.Equal(t, "http://example.com/page", got)
	})
}
