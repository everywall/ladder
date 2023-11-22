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
//
// ---
//
//   - It works by replacing the io.ReadCloser of the http.Response.Body
//     with another io.ReaderCloser (HTMLResourceRewriter) that wraps the first one.
//
//   - This process can be done multiple times, so that the response will
//     be streamed and modified through each pass without buffering the entire response in memory.
//
//   - HTMLResourceRewriter reads the http.Response.Body stream,
//     parsing each HTML token one at a time and replacing attribute tags.
//
//   - When ProxyChain.Execute() is called, the response body will be read from the server
//     and pulled through each ResponseModification which wraps the ProxyChain.Response.Body
//     without ever buffering the entire HTTP response in memory.
func RewriteHTMLResourceURLs() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// return early if it's not HTML
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		// proxyURL is the URL of the ladder: http://localhost:8080 (ladder)
		originalURI := chain.Context.Request().URI()
		proxyURL := fmt.Sprintf("%s://%s", originalURI.Scheme(), originalURI.Host())

		chain.Response.Body = rewriters.
			NewHTMLResourceURLRewriter(
				chain.Response.Body,
				chain.Request.URL,
				proxyURL,
			)

		return nil
	}
}
