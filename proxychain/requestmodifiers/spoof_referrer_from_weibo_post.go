package requestmodifiers

import (
	"fmt"
	"math/rand"

	"ladder/proxychain"
)

// SpoofReferrerFromWeiboPost modifies the referrer header
// pretending to be from a Weibo post (popular in China)
func SpoofReferrerFromWeiboPost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		referrer := fmt.Sprintf("http://weibo.com/u/%d", rand.Intn(90001))
		chain.Request.Header.Set("referrer", referrer)
		chain.Request.Header.Set("sec-fetch-site", "cross-site")
		chain.Request.Header.Set("sec-fetch-dest", "document")
		chain.Request.Header.Set("sec-fetch-mode", "navigate")
		return nil
	}
}
