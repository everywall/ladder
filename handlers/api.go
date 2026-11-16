package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"ladder/internal/service"
	"ladder/pkg/ruleset"
)

type JsonRequest struct {
	URL string `json:"url"`
}

var version = "dev"

func Api(c *fiber.Ctx) error {
	var target string
	var err error
	if c.Method() == http.MethodPost {
		media, _, parseErr := mime.ParseMediaType(c.Get("Content-Type"))
		if parseErr != nil || media != "application/json" {
			return c.Status(415).JSON(fiber.Map{"error": "Content-Type must be application/json"})
		}
		var input JsonRequest
		dec := json.NewDecoder(bytes.NewReader(c.Body()))
		if dec.Decode(&input) != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON request"})
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON request"})
		}
		target = input.URL
	} else {
		target, err = routeTarget(c, false)
	}
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid target URL"})
	}
	s, err := configured()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invalid transport configuration"})
	}
	f, err := s.Fetch(c.UserContext(), service.Request{URL: target})
	if err != nil {
		status := 502
		code := "upstream_error"
		message := "Upstream request failed"
		if e, ok := err.(*service.Error); ok {
			status = e.Status
			code = e.Code
			message = e.Message
		}
		return c.Status(status).JSON(fiber.Map{"ok": false, "error": message, "code": code})
	}
	body := f.Body
	if service.IsHTML(f.Metadata.MediaType) {
		body, err = s.TransformHTML(f)
		if err != nil {
			return legacyError(c, err)
		}
		u, _ := url.Parse(f.Metadata.FinalURL)
		body = []byte(rewriteHtml(body, u, ruleset.Rule{}))
	}
	response := Response{Version: version, OK: f.Metadata.UpstreamStatus >= 200 && f.Metadata.UpstreamStatus < 300, UpstreamStatus: f.Metadata.UpstreamStatus, FinalURL: f.Metadata.FinalURL, Body: string(body), Metadata: f.Metadata}
	if !utf8.Valid(body) || (!strings.HasPrefix(f.Metadata.MediaType, "text/") && !service.IsHTML(f.Metadata.MediaType) && !strings.Contains(f.Metadata.MediaType, "json") && !strings.Contains(f.Metadata.MediaType, "javascript") && !strings.Contains(f.Metadata.MediaType, "xml")) {
		response.Body = base64.StdEncoding.EncodeToString(body)
		response.BodyEncoding = "base64"
	}
	response.Request.Headers = headerList(f.RequestHeader)
	response.Response.Headers = headerList(f.Header)
	response.Response.Status = f.Metadata.UpstreamStatus
	response.Response.FinalURL = f.Metadata.FinalURL
	return c.JSON(response)
}
func headerList(h http.Header) []any {
	safe := service.RedactHeaders(h)
	keys := make([]string, 0, len(safe))
	for k := range safe {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		for _, v := range safe[k] {
			out = append(out, map[string]string{"key": k, "value": v})
		}
	}
	return out
}

type Response struct {
	Version        string           `json:"version"`
	OK             bool             `json:"ok"`
	UpstreamStatus int              `json:"upstreamStatus"`
	FinalURL       string           `json:"finalURL"`
	Body           string           `json:"body"`
	BodyEncoding   string           `json:"bodyEncoding,omitempty"`
	Metadata       service.Metadata `json:"metadata"`
	Request        struct {
		Headers []any `json:"headers"`
	} `json:"request"`
	Response struct {
		Headers  []any  `json:"headers"`
		Status   int    `json:"status"`
		FinalURL string `json:"finalURL"`
	} `json:"response"`
}
