package requestmodifers

import (
	"ladder/proxychain"
	"strings"
)

var forwardBlacklist map[string]bool

func init() {
	forwardBlacklist = map[string]bool{
		"host":              true,
		"connection":        true,
		"keep-alive":        true,
		"content-length":    true,
		"content-encoding":  true,
		"transfer-encoding": true,
		"referer":           true,
		"x-forwarded-for":   true,
		"x-real-ip":         true,
		"forwarded":         true,
	}
}

// ForwardRequestHeaders forwards the requests headers sent from the client to the upstream server
func ForwardRequestHeaders(ua string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {

		forwardHeaders := func(key, value []byte) {
			k := strings.ToLower(string(key))
			v := string(value)
			if forwardBlacklist[k] {
				return
			}
			chain.Request.Header.Set(k, v)
		}

		chain.Context.Request().
			Header.VisitAll(forwardHeaders)

		return nil
	}
}
