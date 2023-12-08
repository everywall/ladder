package requestmodifiers

import (
	"github.com/everywall/ladder/proxychain"
)

// SpoofReferrerFromPinterestPost modifies the referrer header
// pretending to be from a pinterest post
func SpoofReferrerFromPinterestPost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Request.Header.Set("referrer", "https://www.pinterest.com/")
		chain.Request.Header.Set("sec-fetch-site", "cross-site")
		chain.Request.Header.Set("sec-fetch-dest", "document")
		chain.Request.Header.Set("sec-fetch-mode", "navigate")
		return nil
	}
}
