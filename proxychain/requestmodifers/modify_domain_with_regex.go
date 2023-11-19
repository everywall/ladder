package requestmodifers

import (
	"ladder/proxychain"
	"regexp"
)

func ModifyDomainWithRegex(match regexp.Regexp, replacement string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.URL.Host = match.ReplaceAllString(px.Request.URL.Host, replacement)
		return nil
	}
}
