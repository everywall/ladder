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

func TestRaw(t *testing.T) {
	app := fiber.New()
	app.Get("/raw/*", Raw)

	testCases := []struct {
		name           string
		url            string
		expectedStatus int
		expected       string
	}{
		{
			name:           "valid url",
			url:            "https://example.com",
			expectedStatus: http.StatusOK,
			expected:       "<!doctype html>",
		},
		{
			// Raw reports a bad target URL as 500 and returns the error from
			// the HTTP client, not a url.Parse error. Both assertions here
			// were guesses that had never been run.
			name:           "invalid url",
			url:            "invalid-url",
			expectedStatus: http.StatusInternalServerError,
			expected:       `unsupported protocol scheme`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/raw/"+tc.url, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d; got %v", tc.expectedStatus, resp.Status)
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
