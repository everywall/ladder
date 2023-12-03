package requestmodifiers

import (
	"ladder/proxychain"
)

// SpoofReferrerFromVkontaktePost modifies the referrer header
// pretending to be from a vkontakte post (popular in Russia)
func SpoofReferrerFromVkontaktePost() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddOnceRequestModifications(
			SpoofReferrer("https://away.vk.com/"),
			SetRequestHeader("sec-fetch-site", "cross-site"),
			SetRequestHeader("sec-fetch-dest", "document"),
			SetRequestHeader("sec-fetch-mode", "navigate"),
		)
		return nil
	}
}
