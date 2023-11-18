package rqm // ReQuestModifier

import (
	"ladder/proxychain"
)

// ModifyQueryParams replaces query parameter values in URL's query params in a ProxyChain's URL.
// If the query param key doesn't exist, it is created.
func ModifyQueryParams(key string, value string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		q := px.Request.URL.Query()
		if value == "" {
			q.Del(key)
			return nil
		}
		q.Set(key, value)
		px.Request.URL.RawQuery = q.Encode()
		return nil
	}
}
