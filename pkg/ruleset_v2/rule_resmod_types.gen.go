package ruleset_v2

import (
	"ladder/proxychain"
	rx "ladder/proxychain/responsemodifiers"
)

type ResponseModifierFactory func(params ...string) proxychain.ResponseModification

var resModMap map[string]ResponseModifierFactory

// TODO: create codegen using AST parsing of exported methods in ladder/proxychain/responsemodifiers/*.go
func init() {
	resModMap = make(map[string]ResponseModifierFactory)
	resModMap["APIContent"] = func(_ ...string) proxychain.ResponseModification {
		return rx.APIContent()
	}
	resModMap["SetContentSecurityPolicy"] = func(params ...string) proxychain.ResponseModification {
		return rx.SetContentSecurityPolicy(params[0])
	}
	resModMap["SetIncomingCookie"] = func(params ...string) proxychain.ResponseModification {
		return rx.SetIncomingCookie(params[0], params[1])
	}
}
