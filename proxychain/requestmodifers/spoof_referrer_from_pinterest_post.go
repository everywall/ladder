package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromPinterestPost modifies the referrer header
// pretending to be from a pinterest post
func SpoofReferrerFromPinterestPost(url string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddRequestModifications(
			SpoofReferrer("https://www.pinterest.com/"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
