package rsm // ReSponseModifers

import (
	"ladder/proxychain"
	"net/http"
)

// BlockIncomingCookies prevents ALL cookies from being sent from the proxy server
// to the client.
func BlockIncomingCookies(whitelist ...string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		px.Response.Header.Del("Set-Cookie")
		return nil
	}
}

// BlockIncomingCookiesExcept prevents non-whitelisted cookies from being sent from the proxy server
// to the client. Cookies whose names are in the whitelist are not removed.
func BlockIncomingCookiesExcept(whitelist ...string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		// Convert whitelist slice to a map for efficient lookups
		whitelistMap := make(map[string]struct{})
		for _, cookieName := range whitelist {
			whitelistMap[cookieName] = struct{}{}
		}

		// If the response has no cookies, return early
		if px.Response.Header == nil {
			return nil
		}

		// Filter the cookies in the response
		filteredCookies := []string{}
		for _, cookieStr := range px.Response.Header["Set-Cookie"] {
			cookie := parseCookie(cookieStr)
			if _, found := whitelistMap[cookie.Name]; found {
				filteredCookies = append(filteredCookies, cookieStr)
			}
		}

		// Update the Set-Cookie header with the filtered cookies
		if len(filteredCookies) > 0 {
			px.Response.Header["Set-Cookie"] = filteredCookies
		} else {
			px.Response.Header.Del("Set-Cookie")
		}

		return nil
	}
}

// parseCookie parses a cookie string and returns an http.Cookie object.
func parseCookie(cookieStr string) *http.Cookie {
	header := http.Header{}
	header.Add("Set-Cookie", cookieStr)
	request := http.Request{Header: header}
	return request.Cookies()[0]
}
