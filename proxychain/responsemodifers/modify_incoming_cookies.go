package responsemodifers

import (
	"fmt"
	"ladder/proxychain"
	"net/http"
)

// DeleteIncomingCookies prevents ALL cookies from being sent from the proxy server
// back down to the client.
func DeleteIncomingCookies(whitelist ...string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		px.Response.Header.Del("Set-Cookie")
		return nil
	}
}

// DeleteIncomingCookiesExcept prevents non-whitelisted cookies from being sent from the proxy server
// to the client. Cookies whose names are in the whitelist are not removed.
func DeleteIncomingCookiesExcept(whitelist ...string) proxychain.ResponseModification {
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

// SetIncomingCookies adds a raw cookie string being sent from the proxy server down to the client
func SetIncomingCookies(cookies string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		px.Response.Header.Set("Set-Cookie", cookies)
		return nil
	}
}

// SetIncomingCookie modifies a specific cookie in the response from the proxy server to the client.
func SetIncomingCookie(name string, val string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		if px.Response.Header == nil {
			return nil
		}

		updatedCookies := []string{}
		found := false

		// Iterate over existing cookies and modify the one that matches the cookieName
		for _, cookieStr := range px.Response.Header["Set-Cookie"] {
			cookie := parseCookie(cookieStr)
			if cookie.Name == name {
				// Replace the cookie with the new value
				updatedCookies = append(updatedCookies, fmt.Sprintf("%s=%s", name, val))
				found = true
			} else {
				// Keep the cookie as is
				updatedCookies = append(updatedCookies, cookieStr)
			}
		}

		// If the specified cookie wasn't found, add it
		if !found {
			updatedCookies = append(updatedCookies, fmt.Sprintf("%s=%s", name, val))
		}

		// Update the Set-Cookie header
		px.Response.Header["Set-Cookie"] = updatedCookies

		return nil
	}
}
