package requestmodifers

import (
	"fmt"
	"ladder/proxychain"
)

// SpoofReferrerFromNaverSearch modifies the referrer header
// pretending to be from a Naver search (popular in South Korea)
func SpoofReferrerFromNaverSearch(url string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		referrer := fmt.Sprintf(
			"https://search.naver.com/search.naver?where=nexearch&sm=top_hty&fbm=0&ie=utf8&query=%s",
			chain.Request.URL.Host,
		)
		chain.AddRequestModifications(
			SpoofReferrer(referrer),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
