package requestmodifers

import (
	"fmt"
	"ladder/proxychain"
	"math/rand"
)

// SpoofReferrerFromWeiboPost modifies the referrer header
// pretending to be from a Weibo post (popular in China)
func SpoofReferrerFromWeiboPost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		referrer := fmt.Sprintf("http://weibo.com/u/%d", rand.Intn(90001))
		chain.AddRequestModifications(
			SpoofReferrer(referrer),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
