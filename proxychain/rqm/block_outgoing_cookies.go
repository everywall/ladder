package rqm // ReQuestModifier

import (
	"ladder/proxychain"
)

// BlockOutgoingCookies prevents ALL cookies from being sent from the client
// to the upstream proxy server.
func BlockOutgoingCookies() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Del("Cookie")
		return nil
	}
}

// BlockOutgoingCookiesExcept prevents non-whitelisted cookies from being sent from the client
// to the upstream proxy server. Cookies whose names are in the whitelist are not removed.
func BlockOutgoingCookiesExcept(whitelist ...string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		// Convert whitelist slice to a map for efficient lookups
		whitelistMap := make(map[string]struct{})
		for _, cookieName := range whitelist {
			whitelistMap[cookieName] = struct{}{}
		}

		// Get all cookies from the request header
		cookies := px.Request.Cookies()

		// Clear the original Cookie header
		px.Request.Header.Del("Cookie")

		// Re-add cookies that are in the whitelist
		for _, cookie := range cookies {
			if _, found := whitelistMap[cookie.Name]; found {
				px.Request.AddCookie(cookie)
			}
		}

		return nil
	}
}
