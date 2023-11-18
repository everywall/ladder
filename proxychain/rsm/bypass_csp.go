package rsm // ReSponseModifers

import (
	"ladder/proxychain"
)

// BypassCSP modifies response headers to prevent the browser
// from enforcing any CORS restrictions
func BypassCSP() proxychain.ResponseModification {
	return func(px *proxychain.ProxyChain) error {
		px.AddResultModifications(
			ModifyResponseHeader("Access-Control-Allow-Origin", "*"),
			ModifyResponseHeader("Access-Control-Expose-Headers", "*"),
			ModifyResponseHeader("Access-Control-Allow-Credentials", "true"),
			ModifyResponseHeader("Access-Control-Allow-Methods", ""),
		)
		return nil
	}
}
