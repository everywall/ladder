package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromTumblrPost modifies the referrer header
// pretending to be from a tumblr post
func SpoofReferrerFromTumblrPost(url string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddRequestModifications(
			SpoofReferrer("https://www.tumblr.com/"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
