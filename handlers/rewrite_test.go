package handlers

import (
	"net/url"
	"testing"

	"ladder/pkg/ruleset"

	"github.com/stretchr/testify/assert"
)

// Root-relative <script src="/..."> tags were rewritten to script="/..." , which
// is not an attribute any browser loads, so those scripts silently never ran.
// This lives in a correctly named file so it is actually executed; the existing
// handlers/proxy.test.go is named with a dot and is never picked up by go test.
func TestRewriteHtmlKeepsScriptSrcAttribute(t *testing.T) {
	u := &url.URL{Host: "example.com"}

	got := rewriteHtml([]byte(`<script src="/app.js"></script>`), u, ruleset.Rule{})

	assert.Contains(t, got, `src="/https://example.com/app.js"`,
		"the src attribute must survive the rewrite")
	assert.NotContains(t, got, `script="`,
		"src must not be renamed to a script attribute")
}

func TestRewriteHtmlScriptWithOtherAttributes(t *testing.T) {
	u := &url.URL{Host: "example.com"}

	got := rewriteHtml([]byte(`<script defer type="module" src="/a.js"></script>`), u, ruleset.Rule{})

	assert.Contains(t, got, `src="/https://example.com/a.js"`)
	assert.Contains(t, got, `defer`, "existing attributes must be preserved")
	assert.Contains(t, got, `type="module"`)
}

// Absolute script URLs were never matched by the pattern and must stay untouched.
func TestRewriteHtmlLeavesAbsoluteScriptSrcAlone(t *testing.T) {
	u := &url.URL{Host: "example.com"}

	body := `<script src="https://cdn.example.net/x.js"></script>`
	assert.Equal(t, body, rewriteHtml([]byte(body), u, ruleset.Rule{}))
}
