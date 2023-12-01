package responsemodifers

import (
	"fmt"
	"net/url"
	"strings"

	"ladder/proxychain"
)

var forwardBlacklist map[string]bool

func init() {
	forwardBlacklist = map[string]bool{
		"content-length":            true,
		"content-encoding":          true,
		"transfer-encoding":         true,
		"strict-transport-security": true,
		"connection":                true,
		"keep-alive":                true,
	}
}

// ForwardResponseHeaders forwards the response headers from the upstream server to the client
func ForwardResponseHeaders() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// fmt.Println(chain.Response.Header)
		for uname, headers := range chain.Response.Header {
			name := strings.ToLower(uname)
			if forwardBlacklist[name] {
				continue
			}

			// patch location header to forward to proxy instead
			if name == "location" {
				u, err := url.Parse(chain.Context.BaseURL())
				if err != nil {
					return err
				}
				newLocation := fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, headers[0])
				chain.Context.Set("location", newLocation)
			}

			// forward headers
			for _, value := range headers {
				fmt.Println(name, value)
				chain.Context.Set(name, value)
			}
		}

		return nil
	}
}
