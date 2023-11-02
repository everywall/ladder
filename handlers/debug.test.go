// BEGIN: 7f8d9e6d4b5c
package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestDebug(t *testing.T) {
	app := fiber.New()
	app.Get("/debug/*", Debug)

	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "valid url",
			url:      "https://www.google.com",
			expected: "<!doctype html>",
		},
		{
			name:     "invalid url",
			url:      "invalid-url",
			expected: "parse invalid-url: invalid URI for request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/debug/"+tc.url, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status OK; got %v", resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(string(body), tc.expected) {
				t.Errorf("expected body to contain %q; got %q", tc.expected, string(body))
			}
		})
	}
}

// END: 7f8d9e6d4b5c
