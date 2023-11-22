package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromQQPost modifies the referrer header
// pretending to be from a QQ post (popular social media in China)
func SpoofReferrerFromQQPost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddRequestModifications(
			SpoofReferrer("https://new.qq.com/'"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
