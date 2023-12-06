package responsemodifiers

import (
	_ "embed"
	"strings"

	"ladder/proxychain"
	"ladder/proxychain/responsemodifiers/rewriters"
)

//go:embed vendor/block_element_removal.js
var blockElementRemoval string

// BlockElementRemoval prevents paywall javascript from removing a
// particular element by detecting the removal, then immediately reinserting it.
// This is useful when a page will return a "fake" 404, after flashing the content briefly.
// If the /outline/ API works, but the regular API doesn't, try this modifier.
func BlockElementRemoval(cssSelector string) proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// don't add rewriter if it's not even html
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		params := map[string]string{
			// ie: "div.article-content"
			"{{CSS_SELECTOR}}": cssSelector,
		}

		rr := rewriters.NewScriptInjectorRewriterWithParams(
			blockElementRemoval,
			rewriters.BeforeDOMContentLoaded,
			params,
		)

		htmlRewriter := rewriters.NewHTMLRewriter(chain.Response.Body, rr)
		chain.Response.Body = htmlRewriter

		return nil
	}
}
