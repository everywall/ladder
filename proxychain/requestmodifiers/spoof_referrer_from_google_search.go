package requestmodifiers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromGoogleSearch modifies the referrer header
// pretending to be from a google search site
func SpoofReferrerFromGoogleSearch() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddOnceRequestModifications(
			SpoofReferrer("https://www.google.com"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
			ModifyQueryParams("utm_source", "google"),
		)
		return nil
	}
}
