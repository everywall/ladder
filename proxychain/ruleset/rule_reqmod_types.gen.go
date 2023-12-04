package ruleset

import (
	"ladder/proxychain"
	rx "ladder/proxychain/responsemodifiers"
)

type ResponseModifierFactory func(params ...string) proxychain.ResponseModification

var resModMap map[string]ResponseModifierFactory

// TODO: create codegen using AST parsing of exported methods in ladder/proxychain/responsemodifiers/*.go
func init() {
	resModMap = make(map[string]ResponseModifierFactory)

	resModMap["ForwardRequestHeaders"] = func(_ ...string) proxychain.ResponseModification {
		return rx.ForwardRequestHeaders()
	}

	resModMap["MasqueradeAsGoogleBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsGoogleBot()
	}

	resModMap["MasqueradeAsBingBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsBingBot()
	}

	resModMap["MasqueradeAsWaybackMachineBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsWaybackMachineBot()
	}

	resModMap["MasqueradeAsFacebookBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsFacebookBot()
	}

	resModMap["MasqueradeAsYandexBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsYandexBot()
	}

	resModMap["MasqueradeAsBaiduBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsBaiduBot()
	}

	resModMap["MasqueradeAsDuckDuckBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsDuckDuckBot()
	}

	resModMap["MasqueradeAsYahooBot"] = func(_ ...string) proxychain.ResponseModification {
		return rx.MasqueradeAsYahooBot()
	}

	resModMap["ModifyDomainWithRegex"] = func(params, ...string) proxychain.ResponseModification {
		return rx.ModifyDomainWithRegex(params[0], params[1])
	}

	resModMap["SetOutgoingCookie"] = func(params, ...string) proxychain.ResponseModification {
		return rx.SetOutgoingCookie(params[0], params[1])
	}

	resModMap["SetOutgoingCookies"] = func(params, ...string) proxychain.ResponseModification {
		return rx.SetOutgoingCookies(params[0])
	}

	resModMap["DeleteOutgoingCookie"] = func(params, ...string) proxychain.ResponseModification {
		return rx.DeleteOutgoingCookie(params[0])
	}

	resModMap["DeleteOutgoingCookies"] = func(_ ...string) proxychain.ResponseModification {
		return rx.DeleteOutgoingCookies()
	}

	resModMap["DeleteOutgoingCookiesExcept"] = func(params, ...string) proxychain.ResponseModification {
		return rx.DeleteOutgoingCookiesExcept(params[0])
	}

	resModMap["ModifyPathWithRegex"] = func(params, ...string) proxychain.ResponseModification {
		return rx.ModifyPathWithRegex(params[0], params[1])
	}

	resModMap["ModifyQueryParams"] = func(params, ...string) proxychain.ResponseModification {
		return rx.ModifyQueryParams(params[0], params[1])
	}

	resModMap["SetRequestHeader"] = func(params, ...string) proxychain.ResponseModification {
		return rx.SetRequestHeader(params[0], params[1])
	}

	resModMap["DeleteRequestHeader"] = func(params, ...string) proxychain.ResponseModification {
		return rx.DeleteRequestHeader(params[0])
	}

	resModMap["RequestArchiveIs"] = func(_ ...string) proxychain.ResponseModification {
		return rx.RequestArchiveIs()
	}

	resModMap["RequestGoogleCache"] = func(_ ...string) proxychain.ResponseModification {
		return rx.RequestGoogleCache()
	}

	resModMap["RequestWaybackMachine"] = func(_ ...string) proxychain.ResponseModification {
		return rx.RequestWaybackMachine()
	}

	resModMap["NewCustomDialer"] = func(params, ...string) proxychain.ResponseModification {
		return rx.NewCustomDialer(params[0])
	}

	resModMap["ResolveWithGoogleDoH"] = func(_ ...string) proxychain.ResponseModification {
		return rx.ResolveWithGoogleDoH()
	}

	resModMap["SpoofOrigin"] = func(params, ...string) proxychain.ResponseModification {
		return rx.SpoofOrigin(params[0])
	}

	resModMap["HideOrigin"] = func(_ ...string) proxychain.ResponseModification {
		return rx.HideOrigin()
	}

	resModMap["SpoofReferrer"] = func(params, ...string) proxychain.ResponseModification {
		return rx.SpoofReferrer(params[0])
	}

	resModMap["HideReferrer"] = func(_ ...string) proxychain.ResponseModification {
		return rx.HideReferrer()
	}

	resModMap["SpoofReferrerFromBaiduSearch"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromBaiduSearch()
	}

	resModMap["SpoofReferrerFromBingSearch"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromBingSearch()
	}

	resModMap["SpoofReferrerFromGoogleSearch"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromGoogleSearch()
	}

	resModMap["SpoofReferrerFromLinkedInPost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromLinkedInPost()
	}

	resModMap["SpoofReferrerFromNaverSearch"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromNaverSearch()
	}

	resModMap["SpoofReferrerFromPinterestPost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromPinterestPost()
	}

	resModMap["SpoofReferrerFromQQPost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromQQPost()
	}

	resModMap["SpoofReferrerFromRedditPost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromRedditPost()
	}

	resModMap["SpoofReferrerFromTumblrPost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromTumblrPost()
	}

	resModMap["SpoofReferrerFromTwitterPost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromTwitterPost()
	}

	resModMap["SpoofReferrerFromVkontaktePost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromVkontaktePost()
	}

	resModMap["SpoofReferrerFromWeiboPost"] = func(_ ...string) proxychain.ResponseModification {
		return rx.SpoofReferrerFromWeiboPost()
	}

	resModMap["SpoofUserAgent"] = func(params, ...string) proxychain.ResponseModification {
		return rx.SpoofUserAgent(params[0])
	}

	resModMap["SpoofXForwardedFor"] = func(params, ...string) proxychain.ResponseModification {
		return rx.SpoofXForwardedFor(params[0])
	}

}
