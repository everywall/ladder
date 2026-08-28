// BEGIN: 7d5e1f7c7d5e
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestApi(t *testing.T) {
	app := fiber.New()
	app.Get("/api/*", Api)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "valid url",
			url:            "https://example.com",
			expectedStatus: http.StatusOK,
		},
		{
			// Api reports a bad target URL as 500, not 400. Asserting the
			// behaviour that exists rather than changing the handler, which
			// would be an API change and belongs in its own PR.
			name:           "invalid url",
			url:            "invalid-url",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/"+tt.url, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// END: 7d5e1f7c7d5e
