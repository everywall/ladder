package responsemodifers

import (
	_ "embed"
	"strings"

	"ladder/proxychain"
	"ladder/proxychain/responsemodifers/rewriters"
)

// injectScript modifies HTTP responses
// to execute javascript at a particular time.
func injectScript(js string, execTime rewriters.ScriptExecTime) proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// don't add rewriter if it's not even html
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		rr := rewriters.NewScriptInjectorRewriter(js, execTime)
		htmlRewriter := rewriters.NewHTMLRewriter(chain.Response.Body, rr)
		chain.Response.Body = htmlRewriter
		return nil
	}
}

// InjectScriptBeforeDOMContentLoaded modifies HTTP responses to inject a JS before DOM Content is loaded (script tag in head)
func InjectScriptBeforeDOMContentLoaded(js string) proxychain.ResponseModification {
	return injectScript(js, rewriters.BeforeDOMContentLoaded)
}

// InjectScriptAfterDOMContentLoaded modifies HTTP responses to inject a JS after DOM Content is loaded (script tag in head)
func InjectScriptAfterDOMContentLoaded(js string) proxychain.ResponseModification {
	return injectScript(js, rewriters.AfterDOMContentLoaded)
}

// InjectScriptAfterDOMIdle modifies HTTP responses to inject a JS after the DOM is idle (ie: js framework loaded)
func InjectScriptAfterDOMIdle(js string) proxychain.ResponseModification {
	return injectScript(js, rewriters.AfterDOMIdle)
}
