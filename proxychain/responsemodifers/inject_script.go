package responsemodifers

import (
	_ "embed"
	"ladder/proxychain"
	"ladder/proxychain/responsemodifers/rewriters"
	"strings"
)

// InjectScript modifies HTTP responses
// to execute javascript at a particular time.
func InjectScript(js string, execTime rewriters.ScriptExecTime) proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// don't add rewriter if it's not even html
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		// the rewriting actually happens in chain.Execute() as the client is streaming the response body back
		rr := rewriters.NewScriptInjectorRewriter(js, execTime)
		// we just queue it up here
		chain.AddHTMLTokenRewriter(rr)

		return nil
	}
}
