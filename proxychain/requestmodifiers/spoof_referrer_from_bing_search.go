package requestmodifiers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromBingSearch modifies the referrer header
// pretending to be from a bing search site
func SpoofReferrerFromBingSearch() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddOnceRequestModifications(
			SpoofReferrer("https://www.bing.com/"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
			ModifyQueryParams("utm_source", "bing"),
		)
		return nil
	}
}
