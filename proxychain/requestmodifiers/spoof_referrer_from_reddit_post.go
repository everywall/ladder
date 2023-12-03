package requestmodifiers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromRedditPost modifies the referrer header
// pretending to be from a reddit post
func SpoofReferrerFromRedditPost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Request.Header.Set("referrer", "https://www.reddit.com/")
		chain.Request.Header.Set("sec-fetch-site", "cross-site")
		chain.Request.Header.Set("sec-fetch-dest", "document")
		chain.Request.Header.Set("sec-fetch-mode", "navigate")
		return nil
	}
}
