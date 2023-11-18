package rsm // ReSponseModifers

import (
	"ladder/proxychain"
)

// ModifyResponseHeader modifies response headers from the upstream server
// if value is "", then the response header is deleted.
func ModifyResponseHeader(key string, value string) proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		if value == "" {
			px.Context.Response().Header.Del(key)
			return nil
		}
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
