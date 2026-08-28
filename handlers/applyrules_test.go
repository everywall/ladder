package handlers

import (
	"net/url"
	"strings"
	"testing"

	"ladder/pkg/ruleset"

	"github.com/stretchr/testify/assert"
)

// withRuleset installs a ruleset for the duration of a test. applyRules returns
// early unless one is loaded.
func withRuleset(t *testing.T, rules ruleset.RuleSet) {
	t.Helper()
	original := rulesSet
	rulesSet = rules
	t.Cleanup(func() { rulesSet = original })
}

func TestApplyRulesRegexRules(t *testing.T) {
	rule := ruleset.Rule{
		Domain: "example.com",
		RegexRules: []ruleset.Regex{
			{Match: `<div class="paywall">.*?</div>`, Replace: ""},
			{Match: `http://`, Replace: "https://"},
		},
	}
	withRuleset(t, ruleset.RuleSet{rule})

	body := `<p>keep</p><div class="paywall">pay up</div><a href="http://x.test">x</a>`
	got := applyRules(body, rule)

	assert.NotContains(t, got, "pay up")
	assert.Contains(t, got, "<p>keep</p>")
	assert.Contains(t, got, `href="https://x.test"`)
}

func TestApplyRulesInjections(t *testing.T) {
	rule := ruleset.Rule{
		Domain: "example.com",
		Injections: []struct {
			Position string `yaml:"position,omitempty"`
			Append   string `yaml:"append,omitempty"`
			Prepend  string `yaml:"prepend,omitempty"`
			Replace  string `yaml:"replace,omitempty"`
		}{
			{Position: "head", Append: "<script>injected()</script>"},
			{Position: "h1", Replace: "<h1>replaced</h1>"},
		},
	}
	withRuleset(t, ruleset.RuleSet{rule})

	body := `<html><head><title>t</title></head><body><h1>original</h1></body></html>`
	got := applyRules(body, rule)

	assert.Contains(t, got, "injected()")
	assert.Contains(t, got, "<h1>replaced</h1>")
	assert.NotContains(t, got, "original")
}

// A ruleset can be loaded from a remote URL, so a malformed pattern is
// untrusted input. It must not take the process down.
func TestApplyRulesInvalidRegexIsSkipped(t *testing.T) {
	rule := ruleset.Rule{
		Domain: "example.com",
		RegexRules: []ruleset.Regex{
			{Match: `[unclosed`, Replace: "boom"},
			{Match: `keep`, Replace: "kept"},
		},
	}
	withRuleset(t, ruleset.RuleSet{rule})

	assert.NotPanics(t, func() {
		got := applyRules("<p>keep</p>", rule)
		assert.Contains(t, got, "kept", "a later valid rule should still be applied")
	})
}

func TestApplyRulesNoRulesetIsANoOp(t *testing.T) {
	withRuleset(t, ruleset.RuleSet{})
	body := `<p>untouched</p>`
	assert.Equal(t, body, applyRules(body, ruleset.Rule{
		RegexRules: []ruleset.Regex{{Match: "untouched", Replace: "changed"}},
	}))
}

// The regression this fixes: rewriteHtml dropped its applyRules call in
// 8fa0531, so ruleset regexRules and injections silently did nothing.
func TestRewriteHtmlAppliesRuleset(t *testing.T) {
	rule := ruleset.Rule{
		Domain:     "example.com",
		RegexRules: []ruleset.Regex{{Match: `<div id="paywall"></div>`, Replace: ""}},
	}
	withRuleset(t, ruleset.RuleSet{rule})

	body := []byte(`<html><body><div id="paywall"></div><img src="/a.jpg"></body></html>`)
	got := rewriteHtml(body, &url.URL{Host: "example.com"}, rule)

	assert.NotContains(t, got, `id="paywall"`, "ruleset regexRules must reach rewriteHtml output")
	assert.True(t, strings.Contains(got, "/https://example.com/a.jpg"),
		"existing URL rewriting must still happen")
}
