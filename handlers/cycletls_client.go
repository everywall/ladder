package handlers

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
)

// fetchWithCycleTLS performs an HTTP GET using CycleTLS to spoof JA3, JA4r and/or
// HTTP/2 fingerprints. It returns a synthetic *http.Response that is fully compatible
// with the rest of the proxy pipeline (rewriteHtml, applyRules, etc.).
func fetchWithCycleTLS(targetURL string, opts cycletls.Options) (*http.Response, []byte, error) {
	client := cycletls.Init()
	defer client.Close()

	clResp, err := client.Do(targetURL, opts, "GET")
	if err != nil {
		return nil, nil, err
	}

	// Build a synthetic *http.Response so callers don't need to know about CycleTLS.
	bodyBytes := []byte(clResp.Body)

	syntheticResp := &http.Response{
		StatusCode: clResp.Status,
		Status:     strconv.Itoa(clResp.Status) + " " + http.StatusText(clResp.Status),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
	}

	for k, v := range clResp.Headers {
		syntheticResp.Header.Set(k, v)
	}

	return syntheticResp, bodyBytes, nil
}
