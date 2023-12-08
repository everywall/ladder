package responsemodifiers

import (
	_ "embed"
	"strings"

	"github.com/everywall/ladder/proxychain"
)

// DeleteLocalStorageData deletes localstorage cookies.
// If the page works once in a fresh incognito window, but fails
// for subsequent loads, try this response modifier alongside
// DeleteSessionStorageData and DeleteIncomingCookies
func DeleteLocalStorageData() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// don't add rewriter if it's not even html
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}

		chain.AddOnceResponseModifications(
			InjectScriptBeforeDOMContentLoaded(`window.sessionStorage.clear()`),
			InjectScriptAfterDOMContentLoaded(`window.sessionStorage.clear()`),
		)
		return nil
	}
}
