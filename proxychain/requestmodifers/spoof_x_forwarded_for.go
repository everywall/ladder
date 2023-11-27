package requestmodifers

import (
	"ladder/proxychain"
)

// SpoofXForwardedFor modifies the X-Forwarded-For header
// in some cases, a forward proxy may interpret this as the source IP
func SpoofXForwardedFor(ip string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.Header.Set("X-FORWARDED-FOR", ip)
		return nil
	}
}
