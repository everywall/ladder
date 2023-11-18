package rqm // ReQuestModifier

import (
	"ladder/proxychain"
	"regexp"
)

func ModifyPathWithRegex(match regexp.Regexp, replacement string) proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.URL.Path = match.ReplaceAllString(px.Request.URL.Path, replacement)
		return nil
	}
}
