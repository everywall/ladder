package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromQQPost modifies the referrer header
// pretending to be from a QQ post (popular social media in China)
func SpoofReferrerFromQQPost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Request.Header.Set("referrer", "https://new.qq.com/")
		chain.Request.Header.Set("sec-fetch-site", "cross-site")
		chain.Request.Header.Set("sec-fetch-dest", "document")
		return nil
	}
}
