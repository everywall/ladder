package requestmodifiers

import (
	//"fmt"
	"net/url"

	"github.com/everywall/ladder/proxychain"
)

// ModifyQueryParams replaces query parameter values in URL's query params in a ProxyChain's URL.
// If the query param key doesn't exist, it is created.
func ModifyQueryParams(key string, value string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		q := chain.Request.URL.Query()
		chain.Request.URL.RawQuery = modifyQueryParams(key, value, q)
		//fmt.Println(chain.Request.URL.String())
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
