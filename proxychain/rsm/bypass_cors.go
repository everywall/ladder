package rsm // ReSponseModifers

import (
	"ladder/proxychain"
)

// BypassCORs modifies response headers to prevent the browser
// from enforcing any CORS restrictions
func BypassCORS() proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		px.AddResultModifications(
			ModifyResponseHeader("Access-Control-Allow-Origin", "*"),
			ModifyResponseHeader("Access-Control-Expose-Headers", "*"),
			ModifyResponseHeader("Access-Control-Allow-Credentials", "true"),
			ModifyResponseHeader("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, HEAD, OPTIONS, PATCH"),
			DeleteResponseHeader("X-Frame-Options"),
		)
		return nil
	}
}
