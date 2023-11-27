package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromGoogleSearch modifies the referrer header
// pretending to be from a google search site
func SpoofReferrerFromGoogleSearch() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Request.Header.Set("referrer", "https://www.google.com/")
		chain.Request.Header.Set("sec-fetch-site", "cross-site")
		chain.Request.Header.Set("sec-fetch-dest", "document")
		chain.Request.Header.Set("sec-fetch-mode", "navigate")
		ModifyQueryParams("utm_source", "google")
		return nil
	}
}
