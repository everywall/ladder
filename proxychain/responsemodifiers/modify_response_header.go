package responsemodifiers

import (
	"github.com/everywall/ladder/proxychain"
)

// SetResponseHeader modifies response headers from the upstream server
func SetResponseHeader(key string, value string) proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Context.Set(key, value)
		return nil
	}
}

// DeleteResponseHeader removes response headers from the upstream server
func DeleteResponseHeader(key string) proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Context.Response().Header.Del(key)
		return nil
	}
}
