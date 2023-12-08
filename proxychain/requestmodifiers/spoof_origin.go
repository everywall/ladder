package requestmodifiers

import (
	"github.com/everywall/ladder/proxychain"
)

// SpoofOrigin modifies the origin header
// if the upstream server returns a Vary header
// it means you might get a different response if you change this
func SpoofOrigin(url string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Set("origin", url)
		return nil
	}
}

// HideOrigin modifies the origin header
// so that it is the original origin, not the proxy
func HideOrigin() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Set("origin", px.Request.URL.String())
		return nil
	}
}
