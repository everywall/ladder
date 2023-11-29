package responsemodifers

import (
	"ladder/proxychain"
)

// BypassCORS modifies response headers to prevent the browser
// from enforcing any CORS restrictions. This should run at the end of the chain.
func BypassCORS() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddOnceResponseModifications(
			SetResponseHeader("Access-Control-Allow-Origin", "*"),
			SetResponseHeader("Access-Control-Expose-Headers", "*"),
			SetResponseHeader("Access-Control-Allow-Credentials", "true"),
			SetResponseHeader("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, HEAD, OPTIONS, PATCH"),
			SetResponseHeader("Access-Control-Allow-Headers", "*"),
			DeleteResponseHeader("X-Frame-Options"),
		)
		return nil
	}
}
