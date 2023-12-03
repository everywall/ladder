package requestmodifiers

import (
	"ladder/proxychain"
)

// SetRequestHeader modifies a specific outgoing header
// This is the header that the upstream server will see.
func SetRequestHeader(name string, val string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Set(name, val)
		return nil
	}
}

// DeleteRequestHeader modifies a specific outgoing header
// This is the header that the upstream server will see.
func DeleteRequestHeader(name string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Del(name)
		return nil
	}
}
