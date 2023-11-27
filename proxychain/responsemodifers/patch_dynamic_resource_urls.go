package responsemodifers

import (
	_ "embed"
	"fmt"
	"strings"

	"ladder/proxychain"
	"ladder/proxychain/responsemodifers/rewriters"
)

//go:embed patch_dynamic_resource_urls.js
var patchDynamicResourceURLsScript string

// PatchDynamicResourceURLs patches the javascript runtime to rewrite URLs client-side.
//   - This function is designed to allow the proxified page
//     to still be browsible by routing all resource URLs through the proxy.
//   - Native APIs capable of network requests will be hooked
//     and the URLs arguments modified to point to the proxy instead.
//   - fetch('/relative_path') -> fetch('/https://proxiedsite.com/relative_path')
//   - Element.setAttribute('src', "/assets/img.jpg") -> Element.setAttribute('src', "/https://proxiedsite.com/assets/img.jpg") -> fetch('/https://proxiedsite.com/relative_path')
func PatchDynamicResourceURLs() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// don't add rewriter if it's not even html
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		// this is the original URL sent by client:
		// http://localhost:8080/http://proxiedsite.com/foo/bar
		originalURI := chain.Context.Request().URI()

		// this is the extracted URL that the client requests to proxy
		// http://proxiedsite.com/foo/bar
		reqURL := chain.Request.URL

		params := map[string]string{
			// ie: http://localhost:8080
			"{{PROXY_ORIGIN}}": fmt.Sprintf("%s://%s", originalURI.Scheme(), originalURI.Host()),
			// ie: http://proxiedsite.com
			"{{ORIGIN}}": fmt.Sprintf("%s://%s", reqURL.Scheme, reqURL.Host),
		}

		rr := rewriters.NewScriptInjectorRewriterWithParams(
			patchDynamicResourceURLsScript,
			rewriters.BeforeDOMContentLoaded,
			params,
		)

		htmlRewriter := rewriters.NewHTMLRewriter(chain.Response.Body, rr)
		chain.Response.Body = htmlRewriter

		return nil
	}
}
