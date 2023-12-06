package requestmodifiers

import (
	"ladder/proxychain"
	"math/rand"
)

// AddCacheBusterQuery modifies query params to add a random parameter key
// In order to get the upstream network stack to serve a fresh copy of the page.
func AddCacheBusterQuery() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddOnceRequestModifications(
			ModifyQueryParams("ord", randomString(15)),
		)

		return nil
	}
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789."

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
