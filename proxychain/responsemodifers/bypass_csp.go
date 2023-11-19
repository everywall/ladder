package responsemodifers

import (
	"ladder/proxychain"
)

// BypassContentSecurityPolicy modifies response headers to prevent the browser
// from enforcing any CSP restrictions. This should run at the end of the chain.
func BypassContentSecurityPolicy() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddResponseModifications(
			DeleteResponseHeader("Content-Security-Policy"),
			DeleteResponseHeader("Content-Security-Policy-Report-Only"),
			DeleteResponseHeader("X-Content-Security-Policy"),
			DeleteResponseHeader("X-WebKit-CSP"),
		)
		return nil
	}
}

// SetContentSecurityPolicy modifies response headers to a specific CSP
func SetContentSecurityPolicy(csp string) proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Response.Header.Set("Content-Security-Policy", csp)
		return nil
	}
}
