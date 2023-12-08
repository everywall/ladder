package requestmodifiers

import (
	//"net/http"
	//http "github.com/Danny-Dasilva/fhttp"
	http "github.com/bogdanfinn/fhttp"

	"github.com/everywall/ladder/proxychain"
)

// SetOutgoingCookie modifes a specific cookie name
// by modifying the request cookie headers going to the upstream server.
// If the cookie name does not already exist, it is created.
func SetOutgoingCookie(name string, val string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		cookies := chain.Request.Cookies()
		hasCookie := false
		for _, cookie := range cookies {
			if cookie.Name != name {
				continue
			}
			hasCookie = true
			cookie.Value = val
		}

		if hasCookie {
			return nil
		}

		chain.Request.AddCookie(&http.Cookie{
			Domain: chain.Request.URL.Host,
			Name:   name,
			Value:  val,
		})

		return nil
	}
}

// SetOutgoingCookies modifies a client request's cookie header
// to a raw Cookie string, overwriting existing cookies
func SetOutgoingCookies(cookies string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Request.Header.Set("Cookies", cookies)
		return nil
	}
}

// DeleteOutgoingCookie modifies the http request's cookies header to
// delete a specific request cookie going to the upstream server.
// If the cookie does not exist, it does not do anything.
func DeleteOutgoingCookie(name string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		cookies := chain.Request.Cookies()
		chain.Request.Header.Del("Cookies")

		for _, cookie := range cookies {
			if cookie.Name == name {
				chain.Request.AddCookie(cookie)
			}
		}
		return nil
	}
}

// DeleteOutgoingCookies removes the cookie header entirely,
// preventing any cookies from reaching the upstream server.
func DeleteOutgoingCookies() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Del("Cookie")
		return nil
	}
}

// DeleteOutGoingCookiesExcept prevents non-whitelisted cookies from being sent from the client
// to the upstream proxy server. Cookies whose names are in the whitelist are not removed.
func DeleteOutgoingCookiesExcept(whitelist ...string) proxychain.RequestModification {
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
