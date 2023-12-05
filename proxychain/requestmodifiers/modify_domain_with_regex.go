package requestmodifiers

import (
	"fmt"
	"regexp"

	"ladder/proxychain"
)

func ModifyDomainWithRegex(matchRegex string, replacement string) proxychain.RequestModification {
	match, err := regexp.Compile(matchRegex)
	return func(px *proxychain.ProxyChain) error {
		if err != nil {
			return fmt.Errorf("RequestModification :: ModifyDomainWithRegex error => invalid match regex: %s - %s", matchRegex, err.Error())
		}
		px.Request.URL.Host = match.ReplaceAllString(px.Request.URL.Host, replacement)
		return nil
	}
}
