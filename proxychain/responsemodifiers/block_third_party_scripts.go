package responsemodifiers

import (
	_ "embed"
	"fmt"
	"strings"

	"ladder/proxychain"
	"ladder/proxychain/responsemodifiers/rewriters"
)

// BlockThirdPartyScripts rewrites HTML and injects JS to block all third party JS from loading.
func BlockThirdPartyScripts() proxychain.ResponseModification {
	// TODO: monkey patch fetch and XMLHttpRequest to firewall 3P JS as well.
	return func(chain *proxychain.ProxyChain) error {
		// don't add rewriter if it's not even html
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		// proxyURL is the URL of the ladder: http://localhost:8080 (ladder)
		originalURI := chain.Context.Request().URI()
		proxyURL := fmt.Sprintf("%s://%s", originalURI.Scheme(), originalURI.Host())

		// replace http.Response.Body with a readcloser that wraps the original, modifying the html attributes
		rr := rewriters.NewBlockThirdPartyScriptsRewriter(chain.Request.URL, proxyURL)
		blockJSRewriter := rewriters.NewHTMLRewriter(chain.Response.Body, rr)
		chain.Response.Body = blockJSRewriter

		return nil
	}
}
