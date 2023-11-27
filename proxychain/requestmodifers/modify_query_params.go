package requestmodifers

import (
	"ladder/proxychain"
	"net/url"
)

// ModifyQueryParams replaces query parameter values in URL's query params in a ProxyChain's URL.
// If the query param key doesn't exist, it is created.
func ModifyQueryParams(key string, value string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		q := px.Request.URL.Query()
		px.Request.URL.RawQuery = modifyQueryParams(key, value, q)
		return nil
	}
}

func modifyQueryParams(key string, value string, q url.Values) string {
	if value == "" {
		q.Del(key)
		return q.Encode()
	}
	q.Set(key, value)
	return q.Encode()
}
