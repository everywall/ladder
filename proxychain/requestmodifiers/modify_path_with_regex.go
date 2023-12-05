package requestmodifiers

import (
	"fmt"
	"ladder/proxychain"
	"regexp"
)

func ModifyPathWithRegex(matchRegex string, replacement string) proxychain.RequestModification {
	match, err := regexp.Compile(matchRegex)
	return func(px *proxychain.ProxyChain) error {
		if err != nil {
			return fmt.Errorf("RequestModification :: ModifyPathWithRegex error => invalid match regex: %s - %s", matchRegex, err.Error())
		}
		px.Request.URL.Path = match.ReplaceAllString(px.Request.URL.Path, replacement)
		return nil
	}
}
