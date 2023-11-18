package rqm // ReQuestModifier

import (
	"ladder/proxychain"
)

// SpoofUserAgent modifies the user agent
func SpoofUserAgent(ua string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Set("user-agent", ua)
		return nil
	}
}
