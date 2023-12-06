package rewriters

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"crypto/md5"
	"encoding/hex"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ScriptInjectorRewriter implements HTMLTokenRewriter
// ScriptInjectorRewriter is a struct that injects JS into the page
// It uses an HTML tokenizer to process HTML content and injects JS at a specified location
type ScriptInjectorRewriter struct {
	execTime  ScriptExecTime
	script    string
	scriptMD5 string
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

func (r *ScriptInjectorRewriter) ModifyToken(_ *html.Token) (string, string) {
	switch {
	case r.execTime == BeforeDOMContentLoaded:
		return "", fmt.Sprintf("\n<script id='%s'>\n%s\n</script>\n", r.scriptMD5, r.script)

	case r.execTime == AfterDOMContentLoaded:
		return "", fmt.Sprintf("\n<script id='%s'>\ndocument.addEventListener('DOMContentLoaded', () => { %s });\n</script>", r.scriptMD5, r.script)

	case r.execTime == AfterDOMIdle:
		s := strings.Replace(afterDomIdleScriptInjector, `'{{AFTER_DOM_IDLE_SCRIPT}}'`, r.script, 1)
		return "", fmt.Sprintf("\n<script id='%s'>\n%s\n</script>\n", r.scriptMD5, s)

	default:
		return "", ""
	}
}

// GenerateMD5Hash takes a string and returns its MD5 hash as a hexadecimal string
func generateMD5Hash(input string) string {
	hasher := md5.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}

// applies parameters by string replacement of the template script
func (r *ScriptInjectorRewriter) applyParams(params map[string]string) {
	// Sort the keys by length in descending order
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, key := range keys {
		r.script = strings.ReplaceAll(r.script, key, params[key])
	}
}

// NewScriptInjectorRewriter implements a HtmlTokenRewriter
// and injects JS into the page for execution at a particular time
func NewScriptInjectorRewriter(script string, execTime ScriptExecTime) *ScriptInjectorRewriter {
	scriptMD5 := generateMD5Hash(script)
	executeOnceScript := fmt.Sprintf(`if (!document.getElementById("x-%s")) { %s; document.getElementById("%s").id = "x-%s" };`, scriptMD5, script, scriptMD5, scriptMD5)

	return &ScriptInjectorRewriter{
		execTime:  execTime,
		script:    executeOnceScript,
		scriptMD5: scriptMD5,
	}
}

// NewScriptInjectorRewriterWith implements a HtmlTokenRewriter
// and injects JS into the page for execution at a particular time
// accepting arguments into the script, which will be added via a string replace
// the params map represents the key-value pair of the params.
// the key will be string replaced with the value
func NewScriptInjectorRewriterWithParams(script string, execTime ScriptExecTime, params map[string]string) *ScriptInjectorRewriter {
	rr := NewScriptInjectorRewriter(script, execTime)
	rr.applyParams(params)
	return rr
}
