package requestmodifiers

import (
	"net/url"

	"github.com/everywall/ladder/proxychain"
)

const googleCacheUrl string = "https://webcache.googleusercontent.com/search?q=cache:"

// RequestGoogleCache modifies a ProxyChain's URL to request its Google Cache version.
func RequestGoogleCache() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		encodedURL := url.QueryEscape(px.Request.URL.String())
		newURL, err := url.Parse(googleCacheUrl + encodedURL)
		if err != nil {
			return err
		}
		px.Request.URL = newURL
		return nil
	}
}
