package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"ladder/pkg/ruleset"

	"github.com/stretchr/testify/assert"
)

// TestApplyRulesUnit exercises applyRules directly: a regex strip and a head
// injection should both land on the input body.
func TestApplyRulesUnit(t *testing.T) {
	// Seed rulesSet so applyRules doesn't early-return when len(rulesSet)==0.
	prev := rulesSet
	rulesSet = ruleset.RuleSet{{Domain: "example.test"}}
	t.Cleanup(func() { rulesSet = prev })

	input := `<html><head></head><body><script src="/paywall.js"></script><p>hello</p></body></html>`

	rule := ruleset.Rule{
		Domain: "example.test",
		RegexRules: []ruleset.Regex{
			{Match: `<script[^>]*></script>`, Replace: ""},
		},
	}
	rule.Injections = append(rule.Injections, struct {
		Position string `yaml:"position,omitempty"`
		Append   string `yaml:"append,omitempty"`
		Prepend  string `yaml:"prepend,omitempty"`
		Replace  string `yaml:"replace,omitempty"`
	}{Position: "head", Append: `<style>.paywall{display:none}</style>`})

	out := applyRules(input, rule)

	assert.NotContains(t, out, `<script src="/paywall.js">`, "regexRules should strip the script tag")
	assert.Contains(t, out, `<style>.paywall{display:none}</style>`, "injections should append to <head>")
	assert.Contains(t, out, `<p>hello</p>`, "non-matching content should pass through unchanged")
}

// TestFetchSiteInvokesApplyRules is the regression test for the dead-code bug
// where applyRules was defined but never called from fetchSite. The test
// configures a rule with both regexRules and injections, then verifies they
// take effect on a response served by an httptest.NewServer.
//
// Without the fix this test fails: the script tag passes through unmangled
// (or only mangled by rewriteHtml's src→script rename, which won't match the
// strict regex below) and the injection is absent from <head>.
func TestFetchSiteInvokesApplyRules(t *testing.T) {
	const upstreamHTML = `<html><head><title>t</title></head><body><script src="https://example.test/paywall.js"></script><p>body content</p></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, upstreamHTML)
	}))
	t.Cleanup(server.Close)

	u, err := url.Parse(server.URL)
	assert.NoError(t, err)

	rule := ruleset.Rule{
		Domain: u.Host,
		RegexRules: []ruleset.Regex{
			// Strict pattern requiring src= — only matches if applyRules runs
			// BEFORE rewriteHtml's src→script rename. Confirms ordering too.
			{Match: `<script[^>]*src="https://example\.test/paywall\.js"[^>]*></script>`, Replace: ""},
		},
	}
	rule.Injections = append(rule.Injections, struct {
		Position string `yaml:"position,omitempty"`
		Append   string `yaml:"append,omitempty"`
		Prepend  string `yaml:"prepend,omitempty"`
		Replace  string `yaml:"replace,omitempty"`
	}{Position: "head", Append: `<style id="bpc-marker">body{filter:none}</style>`})

	prev := rulesSet
	rulesSet = ruleset.RuleSet{rule}
	t.Cleanup(func() { rulesSet = prev })

	body, _, _, err := fetchSite(server.URL, map[string]string{})
	assert.NoError(t, err)

	assert.NotContains(t, body, `paywall.js`, "regexRules in fetchSite should strip the upstream script")
	assert.Contains(t, body, `<style id="bpc-marker">`, "injections in fetchSite should append to <head>")
}

// TestApplyRulesOrderingBeforeRewriteHtml documents the ordering requirement:
// applyRules must run BEFORE rewriteHtml because rewriteHtml renames `src="/..."`
// to `script="..."` on relative-URL script tags, which would otherwise prevent
// `<script[^>]*src="..."` regex patterns from matching.
func TestApplyRulesOrderingBeforeRewriteHtml(t *testing.T) {
	const upstreamHTML = `<html><head></head><body><script src="/relative-paywall.js"></script></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, upstreamHTML)
	}))
	t.Cleanup(server.Close)

	u, _ := url.Parse(server.URL)

	rule := ruleset.Rule{
		Domain: u.Host,
		RegexRules: []ruleset.Regex{
			{Match: `<script[^>]*src="/relative-paywall\.js"[^>]*></script>`, Replace: ""},
		},
	}

	prev := rulesSet
	rulesSet = ruleset.RuleSet{rule}
	t.Cleanup(func() { rulesSet = prev })

	body, _, _, err := fetchSite(server.URL, map[string]string{})
	assert.NoError(t, err)

	assert.NotContains(t, body, `relative-paywall.js`, "applyRules must see the original src= attribute before rewriteHtml renames it")
}

// Tiny sanity util so callers can spot test failures faster when reading output.
var _ = io.EOF
var _ = strings.TrimSpace
