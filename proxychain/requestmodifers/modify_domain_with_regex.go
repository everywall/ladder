package requestmodifers

import (
	"regexp"

	"ladder/proxychain"
)

func ModifyDomainWithRegex(match regexp.Regexp, replacement string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.URL.Host = match.ReplaceAllString(px.Request.URL.Host, replacement)
		return nil
	}
}
