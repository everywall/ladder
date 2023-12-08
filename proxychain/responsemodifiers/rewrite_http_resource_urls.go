package responsemodifiers

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/everywall/ladder/proxychain/responsemodifiers/rewriters"

	"github.com/everywall/ladder/proxychain"
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

		// replace http.Response.Body with a readcloser that wraps the original, modifying the html attributes
		rr := rewriters.NewHTMLTokenURLRewriter(chain.Request.URL, proxyURL)
		htmlRewriter := rewriters.NewHTMLRewriter(chain.Response.Body, rr)
		chain.Response.Body = htmlRewriter

		return nil
	}
}
