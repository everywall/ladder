package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromLinkedInPost modifies the referrer header
// pretending to be from a linkedin post
func SpoofReferrerFromLinkedInPost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddOnceRequestModifications(
			SpoofReferrer("https://www.linkedin.com/"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
			ModifyQueryParams("utm_campaign", "post"),
			ModifyQueryParams("utm_medium", "web"),
		)
		return nil
	}
}
