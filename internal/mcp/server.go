// Package mcp exposes the reader directly over newline-delimited JSON-RPC stdio.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"ladder/internal/service"
	"strings"
	"sync"
)

const (
	Version        = "2026-07-28"
	LegacyVersion  = "2025-11-25"
	maxInputBytes  = 1 << 20
	maxOutputBytes = 8 << 20
	maxConcurrent  = 16
)

type Backend interface {
	Read(context.Context, service.Request) (service.Document, error)
	TestURL(context.Context, service.Request) (service.TestResult, error)
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type pending struct {
	cancel    context.CancelFunc
	cancelled bool
}

type server struct {
	backend  Backend
	out      io.Writer
	mu       sync.Mutex
	writeMu  sync.Mutex
	active   map[string]*pending
	closed   bool
	legacy   bool
	writeErr error
}

// Serve requires no HTTP listener. EOF cancels active work and returns promptly.
// Responses are bounded and serialized; only protocol messages reach output.
func Serve(ctx context.Context, input io.Reader, output io.Writer, backend Backend) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s := &server{backend: backend, out: output, active: make(map[string]*pending)}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maxInputBytes)
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req request
		if !json.Valid(line) {
			s.send(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: -32700, Message: "Parse error"}})
			continue
		}
		if json.Unmarshal(line, &req) != nil || req.JSONRPC != "2.0" || req.Method == "" || !validID(req.ID) {
			s.send(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: -32600, Message: "Invalid Request"}})
			continue
		}
		if len(req.ID) == 0 {
			s.notify(req)
			continue
		}
		params := map[string]json.RawMessage{}
		if len(req.Params) > 0 && (json.Unmarshal(req.Params, &params) != nil || params == nil) {
			s.fail(req.ID, -32602, "params must be an object", nil)
			continue
		}
		modern, versionErr := s.protocol(req.Method, params)
		if versionErr != nil {
			s.send(response{JSONRPC: "2.0", ID: req.ID, Error: versionErr})
			continue
		}
		switch req.Method {
		case "server/discover":
			s.result(req.ID, true, map[string]any{"supportedVersions": []string{Version, LegacyVersion}, "capabilities": map[string]any{"tools": map[string]any{}}, "_meta": map[string]any{"io.modelcontextprotocol/serverInfo": map[string]any{"name": "Ladder", "version": "1.0.0"}}})
		case "initialize":
			s.legacy = true
			s.result(req.ID, false, map[string]any{"protocolVersion": LegacyVersion, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "Ladder", "version": "1.0.0"}})
		case "ping":
			s.result(req.ID, modern, map[string]any{})
		case "tools/list":
			s.result(req.ID, modern, map[string]any{"tools": tools()})
		case "tools/call":
			call, err := parseCall(params)
			if err != nil {
				s.fail(req.ID, -32602, err.Error(), nil)
				continue
			}
			s.mu.Lock()
			if _, exists := s.active[string(req.ID)]; exists {
				s.mu.Unlock()
				s.fail(req.ID, -32600, "Request ID is already active", nil)
				continue
			}
			if len(s.active) >= maxConcurrent {
				s.mu.Unlock()
				s.fail(req.ID, -32000, "Too many active requests", nil)
				continue
			}
			callCtx, stop := context.WithCancel(ctx)
			p := &pending{cancel: stop}
			s.active[string(req.ID)] = p
			s.mu.Unlock()
			go func(req request, modern bool, call toolCall, p *pending) {
				defer stop()
				result := s.call(callCtx, call)
				s.mu.Lock()
				delete(s.active, string(req.ID))
				cancelled := p.cancelled || s.closed
				s.mu.Unlock()
				if !cancelled {
					s.result(req.ID, modern, result)
				}
			}(req, modern, call, p)
		default:
			s.fail(req.ID, -32601, "Method not found", nil)
		}
		s.writeMu.Lock()
		writeErr := s.writeErr
		s.writeMu.Unlock()
		if writeErr != nil {
			break
		}
	}
	// Do not wait for a backend that ignores cancellation after the client closes.
	s.mu.Lock()
	s.closed = true
	for _, p := range s.active {
		p.cancelled = true
		p.cancel()
	}
	s.mu.Unlock()
	cancel()
	s.writeMu.Lock()
	writeErr := s.writeErr
	s.writeMu.Unlock()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP input (maximum %d bytes per message): %w", maxInputBytes, err)
	}
	return writeErr
}

func validID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true
	}
	var value any
	if json.Unmarshal(id, &value) != nil {
		return false
	}
	switch value.(type) {
	case string, float64:
		return true
	}
	return false
}

func (s *server) protocol(method string, params map[string]json.RawMessage) (bool, *rpcError) {
	var meta struct {
		Version      string          `json:"io.modelcontextprotocol/protocolVersion"`
		Capabilities json.RawMessage `json:"io.modelcontextprotocol/clientCapabilities"`
	}
	if raw, ok := params["_meta"]; ok {
		if json.Unmarshal(raw, &meta) != nil || string(raw) == "null" {
			return true, &rpcError{Code: -32602, Message: "_meta must be an object"}
		}
	}
	if method == "initialize" {
		var version string
		if json.Unmarshal(params["protocolVersion"], &version) != nil || strings.TrimSpace(version) == "" {
			return false, &rpcError{Code: -32602, Message: "protocolVersion is required"}
		}
		// Legacy initialization negotiates our supported version in the result.
		return false, nil
	}
	if meta.Version != "" {
		if meta.Version != Version {
			return true, unsupported(meta.Version)
		}
		var capabilities map[string]any
		if json.Unmarshal(meta.Capabilities, &capabilities) != nil || capabilities == nil {
			return true, &rpcError{Code: -32602, Message: "params._meta.io.modelcontextprotocol/clientCapabilities must be an object"}
		}
		return true, nil
	}
	if s.legacy && method != "server/discover" {
		return false, nil
	}
	return true, &rpcError{Code: -32602, Message: "params._meta.io.modelcontextprotocol/protocolVersion is required"}
}

func unsupported(version string) *rpcError {
	return &rpcError{Code: -32022, Message: "Unsupported protocol version", Data: map[string]any{"supportedVersions": []string{Version, LegacyVersion}, "requestedVersion": version}}
}

func (s *server) notify(req request) {
	if req.Method != "notifications/cancelled" {
		return
	}
	var params struct {
		RequestID json.RawMessage `json:"requestId"`
	}
	if json.Unmarshal(req.Params, &params) != nil {
		return
	}
	s.mu.Lock()
	if p := s.active[string(params.RequestID)]; p != nil {
		p.cancelled = true
		p.cancel()
	}
	s.mu.Unlock()
}

func (s *server) result(id json.RawMessage, modern bool, result map[string]any) {
	if modern {
		result["resultType"] = "complete"
	}
	s.send(response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *server) fail(id json.RawMessage, code int, message string, data any) {
	s.send(response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message, Data: data}})
}

func (s *server) send(resp response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	if len(data) > maxOutputBytes {
		data, _ = json.Marshal(response{JSONRPC: "2.0", ID: resp.ID, Error: &rpcError{Code: -32000, Message: "Response exceeds the stdio output limit"}})
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.writeErr != nil {
		return
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return
	}
	data = append(data, '\n')
	n, err := s.out.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	s.writeErr = err
}

type toolCall struct{ Name, URL, Format, Cache string }

func parseCall(params map[string]json.RawMessage) (toolCall, error) {
	var c toolCall
	if err := json.Unmarshal(params["name"], &c.Name); err != nil {
		return c, errors.New("tool name is required")
	}
	if c.Name != "ladder_fetch" && c.Name != "ladder_test_url" {
		return c, errors.New("unknown tool name")
	}
	var args map[string]json.RawMessage
	if err := json.Unmarshal(params["arguments"], &args); err != nil || args == nil {
		return c, errors.New("arguments must be an object")
	}
	for key := range args {
		if key != "url" && key != "cache" && (key != "format" || c.Name != "ladder_fetch") {
			return c, fmt.Errorf("unknown argument %q", key)
		}
	}
	if json.Unmarshal(args["url"], &c.URL) != nil || strings.TrimSpace(c.URL) == "" {
		return c, errors.New("url must be a nonempty string")
	}
	c.Format = "markdown"
	if raw, ok := args["format"]; ok {
		if json.Unmarshal(raw, &c.Format) != nil {
			return c, errors.New("format must be a string")
		}
	}
	if c.Format != "markdown" && c.Format != "text" && c.Format != "html" {
		return c, errors.New("format must be markdown, text, or html")
	}
	if raw, ok := args["cache"]; ok {
		if json.Unmarshal(raw, &c.Cache) != nil {
			return c, errors.New("cache must be a string")
		}
	}
	if c.Cache != "" && c.Cache != "default" && c.Cache != "reload" && c.Cache != "bypass" {
		return c, errors.New("cache must be default, reload, or bypass")
	}
	return c, nil
}

func (s *server) call(ctx context.Context, c toolCall) map[string]any {
	req := service.Request{URL: c.URL, Cache: c.Cache}
	var value any
	var err error
	isError := false
	if c.Name == "ladder_test_url" {
		var result service.TestResult
		result, err = s.backend.TestURL(ctx, req)
		value = result
		isError = !result.Passed
	} else {
		var doc service.Document
		doc, err = s.backend.Read(ctx, req)
		content := doc.Content.Markdown
		if c.Format == "text" {
			content = doc.Content.Text
		}
		if c.Format == "html" {
			content = doc.Content.HTML
		}
		value = map[string]any{"metadata": doc.Metadata, "title": doc.Title, "byline": doc.Byline, "format": c.Format, "content": content, "problems": doc.Problems}
		isError = doc.Metadata.UpstreamStatus >= 400
	}
	isError = isError || err != nil
	if err != nil {
		problem := service.Problem{Code: "read_failed", Message: err.Error()}
		var serviceErr *service.Error
		if errors.As(err, &serviceErr) {
			problem.Code, problem.Message = serviceErr.Code, serviceErr.Message
		}
		value = map[string]any{"error": problem}
	}
	encoded, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		value = map[string]any{"error": map[string]string{"code": "encoding_failed", "message": "Unable to encode the result"}}
		encoded, _ = json.Marshal(value)
		isError = true
	}
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(encoded)}}, "structuredContent": value, "isError": isError}
}

func tools() []map[string]any {
	properties := func(format bool) map[string]any {
		p := map[string]any{"url": map[string]any{"type": "string", "description": "Absolute HTTP or HTTPS URL"}, "cache": map[string]any{"type": "string", "enum": []string{"default", "reload", "bypass"}, "default": "default"}}
		if format {
			p["format"] = map[string]any{"type": "string", "enum": []string{"markdown", "text", "html"}, "default": "markdown"}
		}
		return p
	}
	return []map[string]any{
		{"name": "ladder_fetch", "description": "Read a web page as sanitized Markdown, plain text, or HTML with source, rule, and cache metadata.", "inputSchema": map[string]any{"type": "object", "properties": properties(true), "required": []string{"url"}, "additionalProperties": false}},
		{"name": "ladder_test_url", "description": "Fetch a URL and report matching rule packs and their content and status checks.", "inputSchema": map[string]any{"type": "object", "properties": properties(false), "required": []string{"url"}, "additionalProperties": false}},
	}
}
