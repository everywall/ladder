package requestmodifers

import (
	"fmt"

	"ladder/proxychain"
	tx "ladder/proxychain/responsemodifers"
)

// SpoofReferrer modifies the referrer header.
// It is useful if the page can be accessed from a search engine
// or social media site, but not by browsing the website itself.
// if url is "", then the referrer header is removed.
func SpoofReferrer(url string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		// change refer on client side js
		script := fmt.Sprintf(`document.referrer = "%s"`, url)
		chain.AddOnceResponseModifications(
			tx.InjectScriptBeforeDOMContentLoaded(script),
		)

		if url == "" {
			chain.Request.Header.Del("referrer")
			return nil
		}
		chain.Request.Header.Set("referrer", url)
		return nil
	}
}

// HideReferrer modifies the referrer header
// so that it is the original referrer, not the proxy
func HideReferrer() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Set("referrer", px.Request.URL.String())
		return nil
	}
}
