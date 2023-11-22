package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromRedditPost modifies the referrer header
// pretending to be from a reddit post
func SpoofReferrerFromRedditPost(url string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddRequestModifications(
			SpoofReferrer("https://www.reddit.com/"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
