package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var staticAsset = regexp.MustCompile(`(?i)\.(css|js|mjs|map|json|xml|png|jpe?g|gif|webp|avif|svg|ico|bmp|woff2?|ttf|otf|eot|mp4|webm|mp3|m4a|pdf|zip)$`)

func (s *Service) flare(ctx context.Context, target *url.URL, headers http.Header) (FetchResult, error) {
	payload, _ := json.Marshal(map[string]any{"cmd": "request.get", "url": target.String(), "maxTimeout": s.cfg.Timeout.Milliseconds()})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.FlareSolverrURL, "/")+"/v1", bytes.NewReader(payload))
	if err != nil {
		return FetchResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	// FlareSolverr uses its own browser headers. Never send configured cookies or
	// application credentials to the solver endpoint.
	resp, err := s.client.Do(req)
	if err != nil {
		return FetchResult{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, s.cfg.MaxBodyBytes+1))
	if err != nil || int64(len(data)) > s.cfg.MaxBodyBytes {
		return FetchResult{}, failure("solver_response", 502, "FlareSolverr response exceeded limit")
	}
	var solved struct {
		Status   string `json:"status"`
		Solution struct {
			URL      string            `json:"url"`
			Status   int               `json:"status"`
			Response string            `json:"response"`
			Headers  map[string]string `json:"headers"`
		} `json:"solution"`
	}
	if err = json.Unmarshal(data, &solved); err != nil || solved.Status != "ok" || solved.Solution.Status < 100 || solved.Solution.Status > 599 {
		return FetchResult{}, failure("solver_response", 502, "FlareSolverr returned an invalid response")
	}
	final, err := s.validate(solved.Solution.URL)
	if err != nil {
		return FetchResult{}, err
	}
	h := http.Header{}
	for k, v := range solved.Solution.Headers {
		h.Set(k, v)
	}
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	return FetchResult{Metadata: Metadata{FinalURL: final.String(), UpstreamStatus: solved.Solution.Status, MediaType: "text/html"}, Header: h, RequestHeader: headers.Clone(), Body: []byte(solved.Solution.Response)}, nil
}
