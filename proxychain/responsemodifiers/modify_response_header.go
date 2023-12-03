package responsemodifiers

import (
	"ladder/proxychain"
)

// SetResponseHeader modifies response headers from the upstream server
func SetResponseHeader(key string, value string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		px.Context.Response().Header.Set(key, value)
		return nil
	}
}

// DeleteResponseHeader removes response headers from the upstream server
func DeleteResponseHeader(key string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		px.Context.Response().Header.Del(key)
		return nil
	}
}
