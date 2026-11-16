package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"ladder/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fixtureBackend struct {
	read func(context.Context, service.Request) (service.Document, error)
}

func (f fixtureBackend) Read(ctx context.Context, r service.Request) (service.Document, error) {
	if f.read != nil {
		return f.read(ctx, r)
	}
	return service.Document{Title: "Fixture", Metadata: service.Metadata{SourceURL: r.URL, FinalURL: r.URL, UpstreamStatus: 200}, Content: service.Content{Markdown: "# Fixture", Text: "Fixture text", HTML: "<p>Fixture</p>"}}, nil
}
func (fixtureBackend) TestURL(_ context.Context, r service.Request) (service.TestResult, error) {
	return service.TestResult{Passed: true, Metadata: service.Metadata{SourceURL: r.URL}}, nil
}

type client struct {
	t         *testing.T
	input     *io.PipeWriter
	responses chan map[string]any
	done      chan error
}

func openClient(t *testing.T, b Backend) *client {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	c := &client{t: t, input: inW, responses: make(chan map[string]any, 32), done: make(chan error, 1)}
	go func() { err := Serve(context.Background(), inR, outW, b); outW.Close(); c.done <- err }()
	go func() {
		scan := bufio.NewScanner(outR)
		scan.Buffer(make([]byte, 4096), maxOutputBytes+1)
		for scan.Scan() {
			var v map[string]any
			if json.Unmarshal(scan.Bytes(), &v) == nil {
				c.responses <- v
			}
		}
		close(c.responses)
	}()
	t.Cleanup(func() {
		inW.Close()
		select {
		case <-c.done:
		case <-time.After(time.Second):
			t.Error("MCP did not exit on EOF")
		}
	})
	return c
}
func (c *client) send(value any) {
	c.t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		c.t.Fatal(err)
	}
	data = append(data, '\n')
	if _, err = c.input.Write(data); err != nil {
		c.t.Fatal(err)
	}
}
func (c *client) receive() map[string]any {
	c.t.Helper()
	select {
	case value, ok := <-c.responses:
		if !ok {
			c.t.Fatal("stdio closed unexpectedly")
		}
		return value
	case <-time.After(2 * time.Second):
		c.t.Fatal("MCP response timed out")
		return nil
	}
}
func modern(id int, method string, params map[string]any) map[string]any {
	if params == nil {
		params = map[string]any{}
	}
	params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": Version, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}
	return map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
}
func result(t *testing.T, v map[string]any) map[string]any {
	t.Helper()
	if v["error"] != nil {
		t.Fatalf("protocol error: %+v", v)
	}
	r, ok := v["result"].(map[string]any)
	if !ok {
		t.Fatalf("result missing: %+v", v)
	}
	return r
}

func TestModernDiscoverToolsAndStructuredFetch(t *testing.T) {
	c := openClient(t, fixtureBackend{})
	c.send(modern(1, "server/discover", nil))
	r := result(t, c.receive())
	if r["resultType"] != "complete" || len(r["supportedVersions"].([]any)) != 2 {
		t.Fatalf("discovery %+v", r)
	}
	c.send(modern(2, "tools/list", nil))
	r = result(t, c.receive())
	if r["resultType"] != "complete" || len(r["tools"].([]any)) != 2 {
		t.Fatalf("tools %+v", r)
	}
	c.send(modern(3, "tools/call", map[string]any{"name": "ladder_fetch", "arguments": map[string]any{"url": "https://example.test/one", "format": "text", "cache": "reload"}}))
	r = result(t, c.receive())
	content := r["structuredContent"].(map[string]any)
	if r["isError"] != false || r["resultType"] != "complete" || content["content"] != "Fixture text" || content["metadata"].(map[string]any)["sourceUrl"] != "https://example.test/one" {
		t.Fatalf("fetch %+v", r)
	}
	text := r["content"].([]any)[0].(map[string]any)["text"].(string)
	if !json.Valid([]byte(text)) {
		t.Fatal("compatibility text is not JSON")
	}
}

func TestLegacyInitializeAndProtocolErrors(t *testing.T) {
	c := openClient(t, fixtureBackend{})
	c.send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": LegacyVersion, "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "fixture", "version": "1"}}})
	r := result(t, c.receive())
	if r["protocolVersion"] != LegacyVersion || r["resultType"] != nil {
		t.Fatalf("legacy initialize %+v", r)
	}
	c.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	c.send(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	result(t, c.receive())
	c.send(map[string]any{"jsonrpc": "2.0", "id": 3, "method": "missing"})
	if c.receive()["error"].(map[string]any)["code"] != float64(-32601) {
		t.Fatal("unknown method code")
	}
	c.send(modern(4, "tools/call", map[string]any{"name": "ladder_fetch", "arguments": map[string]any{"url": true}}))
	if c.receive()["error"].(map[string]any)["code"] != float64(-32602) {
		t.Fatal("invalid arguments code")
	}
	bad := modern(5, "server/discover", nil)
	bad["params"].(map[string]any)["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": "2099-01-01"}
	c.send(bad)
	err := c.receive()["error"].(map[string]any)
	if err["code"] != float64(-32022) || err["data"].(map[string]any)["requestedVersion"] != "2099-01-01" {
		t.Fatalf("version error %+v", err)
	}
}

func TestToolFailureUsesResultNotProtocolError(t *testing.T) {
	c := openClient(t, fixtureBackend{read: func(context.Context, service.Request) (service.Document, error) {
		return service.Document{}, &service.Error{Code: "upstream_failed", Status: 502, Message: "fixture upstream unavailable"}
	}})
	c.send(modern(1, "tools/call", map[string]any{"name": "ladder_fetch", "arguments": map[string]any{"url": "https://example.test"}}))
	r := result(t, c.receive())
	if r["isError"] != true || r["structuredContent"].(map[string]any)["error"].(map[string]any)["code"] != "upstream_failed" {
		t.Fatalf("tool failure %+v", r)
	}
}

func TestCancellationDoesNotBlockLaterRequestsOrSendCancelledResponse(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	c := openClient(t, fixtureBackend{read: func(ctx context.Context, _ service.Request) (service.Document, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return service.Document{}, ctx.Err()
	}})
	c.send(modern(1, "tools/call", map[string]any{"name": "ladder_fetch", "arguments": map[string]any{"url": "https://example.test"}}))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("fetch did not start")
	}
	c.send(modern(2, "ping", nil))
	if c.receive()["id"] != float64(2) {
		t.Fatal("slow fetch blocked ping")
	}
	c.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/cancelled", "params": map[string]any{"requestId": 1}})
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("fetch was not cancelled")
	}
	c.send(modern(3, "ping", nil))
	if c.receive()["id"] != float64(3) {
		t.Fatal("cancelled request produced a response")
	}
}

func TestEOFReturnsWithoutWaitingForBackend(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	c := openClient(t, fixtureBackend{read: func(context.Context, service.Request) (service.Document, error) {
		close(started)
		<-release
		return service.Document{}, nil
	}})
	c.send(modern(1, "tools/call", map[string]any{"name": "ladder_fetch", "arguments": map[string]any{"url": "https://example.test"}}))
	<-started
	c.input.Close()
	select {
	case err := <-c.done:
		if err != nil {
			t.Fatal(err)
		}
		c.done <- nil
	case <-time.After(time.Second):
		t.Fatal("EOF waited for active backend")
	}
	close(release)
}

func TestInputAndOutputAreBounded(t *testing.T) {
	if err := Serve(context.Background(), strings.NewReader(strings.Repeat("x", maxInputBytes+1)), io.Discard, fixtureBackend{}); err == nil {
		t.Fatal("oversized input accepted")
	}
	c := openClient(t, fixtureBackend{read: func(context.Context, service.Request) (service.Document, error) {
		return service.Document{Content: service.Content{Markdown: strings.Repeat("x", maxOutputBytes)}}, nil
	}})
	c.send(modern(1, "tools/call", map[string]any{"name": "ladder_fetch", "arguments": map[string]any{"url": "https://example.test"}}))
	if c.receive()["error"].(map[string]any)["code"] != float64(-32000) {
		t.Fatal("oversized output was not bounded")
	}
}

func TestModernRequiresCapabilitiesInMetaOnEveryRequest(t *testing.T) {
	c := openClient(t, fixtureBackend{})
	for _, method := range []string{"server/discover", "tools/list", "ping"} {
		req := modern(1, method, nil)
		delete(req["params"].(map[string]any)["_meta"].(map[string]any), "io.modelcontextprotocol/clientCapabilities")
		req["params"].(map[string]any)["clientCapabilities"] = map[string]any{}
		c.send(req)
		if c.receive()["error"].(map[string]any)["code"] != float64(-32602) {
			t.Fatalf("%s accepted misplaced capabilities", method)
		}
	}
}

func TestRealServiceCacheAndTestURL(t *testing.T) {
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<article><h1>Fixture</h1><p>A local fixture for the reader.</p></article>")
	}))
	defer upstream.Close()
	svc, err := service.New(service.Config{Client: upstream.Client(), CacheTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	c := openClient(t, svc)
	for i, mode := range []string{"default", "default", "reload", "bypass"} {
		c.send(modern(i+1, "tools/call", map[string]any{"name": "ladder_fetch", "arguments": map[string]any{"url": upstream.URL, "cache": mode}}))
		r := result(t, c.receive())
		meta := r["structuredContent"].(map[string]any)["metadata"].(map[string]any)
		if r["isError"] != false || meta["cache"].(map[string]any)["hit"] != (i == 1) {
			t.Fatalf("cache %s: %+v", mode, r)
		}
	}
	c.send(modern(5, "tools/call", map[string]any{"name": "ladder_test_url", "arguments": map[string]any{"url": upstream.URL}}))
	r := result(t, c.receive())
	if r["structuredContent"].(map[string]any)["checks"] == nil {
		t.Fatal("testURL checks absent")
	}
	if requests != 3 {
		t.Fatalf("cache produced %d upstream requests; want 3", requests)
	}
}

func TestValidNonRequestJSONIsInvalidRequestRatherThanParseError(t *testing.T) {
	var output strings.Builder
	err := Serve(context.Background(), strings.NewReader("[]\n{\n"), &output, fixtureBackend{})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("responses %s", output.String())
	}
	for i, code := range []int{-32600, -32700} {
		var response map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &response); err != nil {
			t.Fatal(err)
		}
		if response["error"].(map[string]any)["code"] != float64(code) {
			t.Fatalf("response %+v", response)
		}
	}
}

func TestLegacyInitializeNegotiatesSupportedVersion(t *testing.T) {
	for _, version := range []string{"2024-11-05", "2025-06-18", "2026-07-28", "2099-01-01"} {
		t.Run(version, func(t *testing.T) {
			c := openClient(t, fixtureBackend{})
			c.send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": version, "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "negotiation fixture", "version": "1"}}})
			initialized := result(t, c.receive())
			if initialized["protocolVersion"] != LegacyVersion {
				t.Fatalf("server must offer its supported legacy version: %+v", initialized)
			}
			c.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
			c.send(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
			listed := result(t, c.receive())
			if len(listed["tools"].([]any)) != 2 {
				t.Fatal("negotiated client cannot use tools")
			}
		})
	}
}
