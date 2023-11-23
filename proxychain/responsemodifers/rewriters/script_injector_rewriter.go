package rewriters

import (
	_ "embed"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ScriptInjectorRewriter implements HTMLTokenRewriter
// ScriptInjectorRewriter is a struct that injects JS into the page
// It uses an HTML tokenizer to process HTML content and injects JS at a specified location
type ScriptInjectorRewriter struct {
	execTime ScriptExecTime
	script   string
}

type ScriptExecTime int

const (
	BeforeDOMContentLoaded ScriptExecTime = iota
	AfterDOMContentLoaded
	AfterDOMIdle
)

func (r *ScriptInjectorRewriter) ShouldModify(token *html.Token) bool {
	// modify if token == <head>
	return token.DataAtom == atom.Head && token.Type == html.StartTagToken
}

//go:embed after_dom_idle_script_injector.js
var afterDomIdleScriptInjector string

func (r *ScriptInjectorRewriter) ModifyToken(token *html.Token) (string, string) {
	switch {
	case r.execTime == BeforeDOMContentLoaded:
		return "", fmt.Sprintf("\n<script>\n%s\n</script>\n", r.script)

	case r.execTime == AfterDOMContentLoaded:
		return "", fmt.Sprintf("\n<script>\ndocument.addEventListener('DOMContentLoaded', () => { %s });\n</script>", r.script)

	case r.execTime == AfterDOMIdle:
		s := strings.Replace(afterDomIdleScriptInjector, `'SCRIPT_CONTENT_PARAM'`, r.script, 1)
		return "", fmt.Sprintf("\n<script>\n%s\n</script>\n", s)

	default:
		return "", ""
	}
}

// NewScriptInjectorRewriter implements a HtmlTokenRewriter
// and injects JS into the page for execution at a particular time
func NewScriptInjectorRewriter(script string, execTime ScriptExecTime) *ScriptInjectorRewriter {
	return &ScriptInjectorRewriter{
		execTime: execTime,
		script:   script,
	}
}
