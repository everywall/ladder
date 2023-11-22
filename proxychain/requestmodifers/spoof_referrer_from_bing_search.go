package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromBingSearch modifies the referrer header
// pretending to be from a bing search site
func SpoofReferrerFromBingSearch(url string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddRequestModifications(
			SpoofReferrer("https://www.bing.com/"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
