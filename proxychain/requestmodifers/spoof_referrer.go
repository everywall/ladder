package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrer modifies the referrer header
// useful if the page can be accessed from a search engine
// or social media site, but not by browsing the website itself
// if url is "", then the referrer header is removed
func SpoofReferrer(url string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		if url == "" {
			px.Request.Header.Del("referrer")
			return nil
		}
		px.Request.Header.Set("referrer", url)
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
