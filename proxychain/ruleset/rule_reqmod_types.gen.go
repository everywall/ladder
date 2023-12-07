
package ruleset_v2
// DO NOT EDIT THIS FILE. It is automatically generated by ladder/proxychain/codegen/codegen.go
// The purpose of this is serialization of rulesets from JSON or YAML into functional options suitable
// for use in proxychains.

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifiers"
)

type RequestModifierFactory func(params ...string) proxychain.RequestModification

var rqmModMap map[string]RequestModifierFactory

func init() {
	rqmModMap = make(map[string]RequestModifierFactory)

	  rqmModMap["AddCacheBusterQuery"] = func(_ ...string) proxychain.RequestModification {
    return rx.AddCacheBusterQuery()
  }

  rqmModMap["ForwardRequestHeaders"] = func(_ ...string) proxychain.RequestModification {
    return rx.ForwardRequestHeaders()
  }

  rqmModMap["MasqueradeAsGoogleBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsGoogleBot()
  }

  rqmModMap["MasqueradeAsBingBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsBingBot()
  }

  rqmModMap["MasqueradeAsWaybackMachineBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsWaybackMachineBot()
  }

  rqmModMap["MasqueradeAsFacebookBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsFacebookBot()
  }

  rqmModMap["MasqueradeAsYandexBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsYandexBot()
  }

  rqmModMap["MasqueradeAsBaiduBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsBaiduBot()
  }

  rqmModMap["MasqueradeAsDuckDuckBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsDuckDuckBot()
  }

  rqmModMap["MasqueradeAsYahooBot"] = func(_ ...string) proxychain.RequestModification {
    return rx.MasqueradeAsYahooBot()
  }

  rqmModMap["ModifyDomainWithRegex"] = func(params ...string) proxychain.RequestModification {
    return rx.ModifyDomainWithRegex(params[0], params[1])
  }

  rqmModMap["SetOutgoingCookie"] = func(params ...string) proxychain.RequestModification {
    return rx.SetOutgoingCookie(params[0], params[1])
  }

  rqmModMap["SetOutgoingCookies"] = func(params ...string) proxychain.RequestModification {
    return rx.SetOutgoingCookies(params[0])
  }

  rqmModMap["DeleteOutgoingCookie"] = func(params ...string) proxychain.RequestModification {
    return rx.DeleteOutgoingCookie(params[0])
  }

  rqmModMap["DeleteOutgoingCookies"] = func(_ ...string) proxychain.RequestModification {
    return rx.DeleteOutgoingCookies()
  }

  rqmModMap["DeleteOutgoingCookiesExcept"] = func(params ...string) proxychain.RequestModification {
    return rx.DeleteOutgoingCookiesExcept(params[0])
  }

  rqmModMap["ModifyPathWithRegex"] = func(params ...string) proxychain.RequestModification {
    return rx.ModifyPathWithRegex(params[0], params[1])
  }

  rqmModMap["ModifyQueryParams"] = func(params ...string) proxychain.RequestModification {
    return rx.ModifyQueryParams(params[0], params[1])
  }

  rqmModMap["SetRequestHeader"] = func(params ...string) proxychain.RequestModification {
    return rx.SetRequestHeader(params[0], params[1])
  }

  rqmModMap["DeleteRequestHeader"] = func(params ...string) proxychain.RequestModification {
    return rx.DeleteRequestHeader(params[0])
  }

  rqmModMap["RequestArchiveIs"] = func(_ ...string) proxychain.RequestModification {
    return rx.RequestArchiveIs()
  }

  rqmModMap["RequestGoogleCache"] = func(_ ...string) proxychain.RequestModification {
    return rx.RequestGoogleCache()
  }

  rqmModMap["RequestWaybackMachine"] = func(_ ...string) proxychain.RequestModification {
    return rx.RequestWaybackMachine()
  }

  rqmModMap["ResolveWithGoogleDoH"] = func(_ ...string) proxychain.RequestModification {
    return rx.ResolveWithGoogleDoH()
  }

  rqmModMap["SpoofOrigin"] = func(params ...string) proxychain.RequestModification {
    return rx.SpoofOrigin(params[0])
  }

  rqmModMap["HideOrigin"] = func(_ ...string) proxychain.RequestModification {
    return rx.HideOrigin()
  }

  rqmModMap["SpoofReferrer"] = func(params ...string) proxychain.RequestModification {
    return rx.SpoofReferrer(params[0])
  }

  rqmModMap["HideReferrer"] = func(_ ...string) proxychain.RequestModification {
    return rx.HideReferrer()
  }

  rqmModMap["SpoofReferrerFromBaiduSearch"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromBaiduSearch()
  }

  rqmModMap["SpoofReferrerFromBingSearch"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromBingSearch()
  }

  rqmModMap["SpoofReferrerFromGoogleSearch"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromGoogleSearch()
  }

  rqmModMap["SpoofReferrerFromLinkedInPost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromLinkedInPost()
  }

  rqmModMap["SpoofReferrerFromNaverSearch"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromNaverSearch()
  }

  rqmModMap["SpoofReferrerFromPinterestPost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromPinterestPost()
  }

  rqmModMap["SpoofReferrerFromQQPost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromQQPost()
  }

  rqmModMap["SpoofReferrerFromRedditPost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromRedditPost()
  }

  rqmModMap["SpoofReferrerFromTumblrPost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromTumblrPost()
  }

  rqmModMap["SpoofReferrerFromTwitterPost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromTwitterPost()
  }

  rqmModMap["SpoofReferrerFromVkontaktePost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromVkontaktePost()
  }

  rqmModMap["SpoofReferrerFromWeiboPost"] = func(_ ...string) proxychain.RequestModification {
    return rx.SpoofReferrerFromWeiboPost()
  }

  rqmModMap["SpoofUserAgent"] = func(params ...string) proxychain.RequestModification {
    return rx.SpoofUserAgent(params[0])
  }

  rqmModMap["SpoofXForwardedFor"] = func(params ...string) proxychain.RequestModification {
    return rx.SpoofXForwardedFor(params[0])
  }

}