package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"ladder/internal/service"
	"ladder/internal/startup"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureService struct {
	request service.Request
	err     error
	passed  bool
}

func (f *fixtureService) Read(_ context.Context, r service.Request) (service.Document, error) {
	f.request = r
	return service.Document{Title: "Fixture", Metadata: service.Metadata{SourceURL: r.URL, FinalURL: r.URL, UpstreamStatus: 200}, Content: service.Content{Markdown: "# Heading", Text: "Plain body", HTML: "<p>Safe body</p>"}}, f.err
}
func (f *fixtureService) TestURL(_ context.Context, r service.Request) (service.TestResult, error) {
	f.request = r
	return service.TestResult{Passed: f.passed, Checks: []service.CheckResult{{Name: "status", Passed: f.passed, Detail: "fixture"}}}, f.err
}
func (*fixtureService) Packs() []service.PackInfo {
	return []service.PackInfo{{ID: "fixture", Version: "1.0.0", Rules: 1, SHA256: strings.Repeat("a", 64)}}
}

func testDeps(f *fixtureService) dependencies {
	return dependencies{newService: func(options) (backend, error) { return f, nil }, checkPacks: func([]string) ([]service.PackInfo, error) { return f.Packs(), nil }}
}

func TestFetchFormatsJSONAndFlagsAfterURL(t *testing.T) {
	for _, format := range []string{"markdown", "text", "html"} {
		t.Run(format, func(t *testing.T) {
			f := &fixtureService{}
			var stdout, stderr bytes.Buffer
			handled, code := run([]string{"fetch", "https://example.test/one?x=a%2Fb", "--format", format, "--refresh", "--json"}, strings.NewReader(""), &stdout, &stderr, testDeps(f))
			if !handled || code != 0 || stderr.Len() != 0 {
				t.Fatalf("handled=%v code=%d stderr=%s", handled, code, &stderr)
			}
			var value map[string]any
			dec := json.NewDecoder(&stdout)
			if err := dec.Decode(&value); err != nil {
				t.Fatal(err)
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				t.Fatal("stdout contains more than one JSON document")
			}
			want := map[string]string{"markdown": "# Heading", "text": "Plain body", "html": "<p>Safe body</p>"}[format]
			if value["content"] != want || value["format"] != format || value["metadata"].(map[string]any)["sourceUrl"] != f.request.URL {
				t.Fatalf("result %+v", value)
			}
			if f.request.Cache != "reload" || f.request.URL != "https://example.test/one?x=a%2Fb" {
				t.Fatalf("URL or cache changed: %+v", f.request)
			}
		})
	}
}

func TestFetchErrorsExitNonzeroAndRemainMachineReadable(t *testing.T) {
	for _, status := range []int{400, 502} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			f := &fixtureService{err: &service.Error{Code: "fixture_failure", Status: status, Message: "fixture failure"}}
			var stdout, stderr bytes.Buffer
			_, code := run([]string{"fetch", "bad", "--json"}, nil, &stdout, &stderr, testDeps(f))
			if code == 0 || !json.Valid(stdout.Bytes()) || stderr.Len() != 0 {
				t.Fatalf("code %d stdout %s stderr %s", code, &stdout, &stderr)
			}
		})
	}
	for _, args := range [][]string{{"fetch"}, {"fetch", "https://example.test", "--format", "bad"}, {"fetch", "https://example.test", "--refresh", "--cache", "bypass"}, {"fetch", "https://example.test", "--cache", "bad"}} {
		var stdout, stderr bytes.Buffer
		_, code := run(args, nil, &stdout, &stderr, testDeps(&fixtureService{}))
		if code != 2 {
			t.Fatalf("%v code %d", args, code)
		}
	}
}

func TestRulesCheckListAndFailedTest(t *testing.T) {
	f := &fixtureService{}
	d := testDeps(f)
	for _, args := range [][]string{{"rules", "list", "--json"}, {"rules", "check", "pack.yaml", "--json"}} {
		var stdout, stderr bytes.Buffer
		_, code := run(args, nil, &stdout, &stderr, d)
		if code != 0 || !json.Valid(stdout.Bytes()) {
			t.Fatalf("%v code %d out %s err %s", args, code, &stdout, &stderr)
		}
	}
	var stdout, stderr bytes.Buffer
	_, code := run([]string{"rules", "test", "https://example.test", "--json"}, nil, &stdout, &stderr, d)
	if code != 1 || !json.Valid(stdout.Bytes()) {
		t.Fatalf("failed rule test code %d output %s", code, &stdout)
	}
	d.checkPacks = func([]string) ([]service.PackInfo, error) { return nil, errors.New("invalid pack") }
	_, code = run([]string{"rules", "check", "bad.yaml"}, nil, io.Discard, io.Discard, d)
	if code != 1 {
		t.Fatal("invalid pack succeeded")
	}
}

func TestCompatibilityAndNoImplicitStartup(t *testing.T) {
	calls := []string{}
	d := testDeps(&fixtureService{})
	d.startupStatus = func(startup.Options) (startup.State, error) {
		calls = append(calls, "status")
		return startup.State{}, nil
	}
	d.startupEnable = func(o startup.Options) (startup.State, error) {
		calls = append(calls, "enable")
		if o.Port != 4567 || o.Executable != "/fixture/ladder" {
			t.Fatalf("startup args %+v", o)
		}
		return startup.State{Enabled: true}, nil
	}
	d.startupDisable = func(startup.Options) (startup.State, error) {
		calls = append(calls, "disable")
		return startup.State{}, nil
	}
	for _, args := range [][]string{nil, {"--port", "8080"}, {"serve"}} {
		handled, code := run(args, nil, io.Discard, io.Discard, d)
		if handled || code != 0 {
			t.Fatalf("legacy args intercepted %v", args)
		}
	}
	if len(calls) != 0 {
		t.Fatal("startup enabled implicitly")
	}
	for _, args := range [][]string{{"startup", "status", "--json"}, {"startup", "enable", "--port", "4567", "--executable", "/fixture/ladder", "--json"}, {"startup", "disable", "--json"}} {
		var stdout, stderr bytes.Buffer
		_, code := run(args, nil, &stdout, &stderr, d)
		if code != 0 || !json.Valid(stdout.Bytes()) {
			t.Fatalf("startup %v code %d stderr %s", args, code, &stderr)
		}
	}
	if strings.Join(calls, ",") != "status,enable,disable" {
		t.Fatal(calls)
	}
	handled, code := run([]string{"fech"}, nil, io.Discard, io.Discard, d)
	if !handled || code != 2 {
		t.Fatal("unknown command was allowed to start a server")
	}
}

func TestConnectorRequiresExplicitStdioAndHasProtocolOnlyStdout(t *testing.T) {
	d := testDeps(&fixtureService{})
	var stdout, stderr bytes.Buffer
	_, code := run([]string{"mcp"}, nil, &stdout, &stderr, d)
	if code != 2 || stdout.Len() != 0 {
		t.Fatal("MCP started without --stdio")
	}
	stdout.Reset()
	stderr.Reset()
	input := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}` + "\n"
	_, code = run([]string{"mcp", "--stdio"}, strings.NewReader(input), &stdout, &stderr, d)
	if code != 0 || !json.Valid(stdout.Bytes()) || stderr.Len() != 0 {
		t.Fatalf("stdio code %d stdout %s stderr %s", code, &stdout, &stderr)
	}
}

func clearServiceEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"RULESET", "RULE_PACKS", "ALLOWED_DOMAINS", "ALLOWED_DOMAINS_RULESET", "FLARESOLVERR_HOST", "CACHE_TTL", "CACHE_MAX_ENTRIES", "HTTP_TIMEOUT"} {
		t.Setenv(key, "")
	}
}

func TestRealServiceFetchAndUpstreamFailureWithoutWebServer(t *testing.T) {
	clearServiceEnv(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/failed" {
			w.WriteHeader(503)
		}
		io.WriteString(w, "<html><title>Local fixture</title><article><h1>Local fixture</h1><p>Readable fixture body.</p></article></html>")
	}))
	defer upstream.Close()
	for _, suffix := range []string{"/ok", "/failed"} {
		var stdout, stderr bytes.Buffer
		_, code := Run([]string{"fetch", upstream.URL + suffix, "--format", "text", "--json"}, nil, &stdout, &stderr)
		if !json.Valid(stdout.Bytes()) {
			t.Fatalf("stdout %s", &stdout)
		}
		if suffix == "/ok" && (code != 0 || !strings.Contains(stdout.String(), "Readable fixture body")) {
			t.Fatalf("read code %d out %s err %s", code, &stdout, &stderr)
		}
		if suffix == "/failed" && code == 0 {
			t.Fatalf("upstream 503 succeeded: %s", &stdout)
		}
	}
}

func TestRulesetFlagOverridesInvalidEnvWithoutMutation(t *testing.T) {
	clearServiceEnv(t)
	t.Setenv("RULESET", filepath.Join(t.TempDir(), "missing.yaml"))
	file := filepath.Join(t.TempDir(), "empty.yaml")
	if err := os.WriteFile(file, []byte("[]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	_, code := Run([]string{"rules", "list", "--ruleset", file, "--json"}, nil, &stdout, &stderr)
	if code != 0 || !json.Valid(stdout.Bytes()) {
		t.Fatalf("override code %d stdout %s stderr %s", code, &stdout, &stderr)
	}
	if os.Getenv("RULESET") == file {
		t.Fatal("command mutated process environment")
	}
}
