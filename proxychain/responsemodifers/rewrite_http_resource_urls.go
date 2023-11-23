package responsemodifers

import (
	_ "embed"
	"fmt"
	"ladder/proxychain"
	"ladder/proxychain/responsemodifers/rewriters"
	"strings"
)

// RewriteHTMLResourceURLs modifies HTTP responses
// to rewrite URLs attributes in HTML content (such as src, href)
//   - `<img src='/relative_path'>` -> `<img src='/https://proxiedsite.com/relative_path'>`
//   - This function is designed to allow the proxified page
//     to still be browsible by routing all resource URLs through the proxy.
func RewriteHTMLResourceURLs() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// don't add rewriter if it's not even html
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		// proxyURL is the URL of the ladder: http://localhost:8080 (ladder)
		originalURI := chain.Context.Request().URI()
		proxyURL := fmt.Sprintf("%s://%s", originalURI.Scheme(), originalURI.Host())

		// the rewriting actually happens in chain.Execute() as the client is streaming the response body back
		rr := rewriters.NewHTMLTokenURLRewriter(chain.Request.URL, proxyURL)
		// we just queue it up here
		chain.AddHTMLTokenRewriter(rr)

		return nil
	}
}
