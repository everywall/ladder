package handlers

import (
	_ "embed"
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"
	"log"
	"regexp"

	"net/http"

	"github.com/gofiber/fiber/v2"
)

//go:embed playground.html
var playgroundHtml string

type ModifierQuery struct {
	RequestModifierQuery  RequestModifierQuery  `json:"requestmodifierquery"`
	ResponseModifierQuery ResponseModifierQuery `json:"responsemodifierquery"`
}

type RequestModifierQuery struct {
	ForwardRequestHeaders         bool `json:"forwardrequestheaders"`
	MasqueradeAsGoogleBot         bool `json:"masqueradeasgooglebot"`
	MasqueradeAsBingBot           bool `json:"masqueradeasbingbot"`
	MasqueradeAsWaybackMachineBot bool `json:"masqueradeaswaybackmachinebot"`
	MasqueradeAsFacebookBot       bool `json:"masqueradeasfacebookbot"`
	MasqueradeAsYandexBot         bool `json:"masqueradeasyandexbot"`
	MasqueradeAsBaiduBot          bool `json:"masqueradeasbaidubot"`
	MasqueradeAsDuckDuckBot       bool `json:"masqueradeasduckduckbot"`
	MasqueradeAsYahooBot          bool `json:"masqueradeasyahoobot"`
	ModifyDomainWithRegex         struct {
		match       string `json:"match"`
		replacement string `json:"replacement"`
	} `json:"modifydomainwithregex"`
	SetOutgoingCookie struct {
		name string `json:"name"`
		val  string `json:"val"`
	} `json:"setoutgoingcookie"`
	SetOutgoingCookies struct {
		cookies string `json:"cookies"`
	} `json:"setoutgoingcookies"`
	DeleteOutgoingCookie struct {
		name string `json:"name"`
	} `json:"deleteoutgoingcookie"`
	DeleteOutgoingCookies       bool `json:"deleteoutgoingcookies"`
	DeleteOutgoingCookiesExcept struct {
		whitelist string `json:"whitelist"`
	} `json:"deleteoutgoingcookiesexcept"`
	ModifyPathWithRegex struct {
		match       string `json:"match"`
		replacement string `json:"replacement"`
	} `json:"modifypathwithregex"`
	ModifyQueryParams struct {
		key   string `json:"key"`
		value string `json:"value"`
	} `json:"modifyqueryparams"`
	SetRequestHeader struct {
		name string `json:"name"`
		val  string `json:"val"`
	} `json:"setrequestheader"`
	DeleteRequestHeader struct {
		name string `json:"name"`
	} `json:"deleterequestheader"`
	RequestArchiveIs      bool `json:"requestarchiveis"`
	RequestGoogleCache    bool `json:"requestgooglecache"`
	RequestWaybackMachine bool `json:"requestwaybackmachine"`
	ResolveWithGoogleDoH  bool `json:"resolvewithgoogledoh"`
	SpoofOrigin           struct {
		url string `json:"url"`
	} `json:"spooforigin"`
	HideOrigin    bool `json:"hideorigin"`
	SpoofReferrer struct {
		url string `json:"url"`
	} `json:"spoofreferrer"`
	HideReferrer                   bool `json:"hidereferrer"`
	SpoofReferrerFromBaiduSearch   bool `json:"spoofreferrerfrombaidusearch"`
	SpoofReferrerFromBingSearch    bool `json:"spoofreferrerfrombingsearch"`
	SpoofReferrerFromGoogleSearch  bool `json:"spoofreferrerfromgooglesearch"`
	SpoofReferrerFromLinkedInPost  bool `json:"spoofreferrerfromlinkedinpost"`
	SpoofReferrerFromNaverSearch   bool `json:"spoofreferrerfromnaversearch"`
	SpoofReferrerFromPinterestPost bool `json:"spoofreferrerfrompinterestpost"`
	SpoofReferrerFromQQPost        bool `json:"spoofreferrerfromqqpost"`
	SpoofReferrerFromRedditPost    bool `json:"spoofreferrerfromredditpost"`
	SpoofReferrerFromTumblrPost    bool `json:"spoofreferrerfromtumblrpost"`
	SpoofReferrerFromTwitterPost   bool `json:"spoofreferrerfromtwitterpost"`
	SpoofReferrerFromVkontaktePost bool `json:"spoofreferrerfromvkontaktepost"`
	SpoofReferrerFromWeiboPost     bool `json:"spoofreferrerfromweibopost"`
	SpoofUserAgent                 struct {
		ua string `json:"ua"`
	} `json:"spoofuseragent"`
	SpoofXForwardedFor struct {
		ip string `json:"ip"`
	} `json:"spoofxforwardedfor"`
}

type ResponseModifierQuery struct {
	APIContent          bool `json:"apicontent"`
	BlockElementRemoval struct {
		cssSelector string `json:"cssSelector"`
	} `json:"blockelementremoval"`
	BypassCORS                  bool `json:"bypasscors"`
	BypassContentSecurityPolicy bool `json:"bypasscontentsecuritypolicy"`
	SetContentSecurityPolicy    struct {
		csp string `json:"csp"`
	} `json:"setcontentsecuritypolicy"`
	ForwardResponseHeaders             bool `json:"forwardresponseheaders"`
	GenerateReadableOutline            bool `json:"generatereadableoutline"`
	InjectScriptBeforeDOMContentLoaded struct {
		js string `json:"js"`
	} `json:"injectscriptbeforedomcontentloaded"`
	InjectScriptAfterDOMContentLoaded struct {
		js string `json:"js"`
	} `json:"injectscriptafterdomcontentloaded"`
	InjectScriptAfterDOMIdle struct {
		js string `json:"js"`
	} `json:"injectscriptafterdomidle"`
	DeleteIncomingCookies       bool `json:"deleteincomingcookies"`
	DeleteIncomingCookiesExcept struct {
		whitelist string `json:"whitelist"`
	} `json:"deleteincomingcookiesexcept"`
	SetIncomingCookies struct {
		cookies string `json:"cookies"`
	} `json:"setincomingcookies"`
	SetIncomingCookie struct {
		name string `json:"name"`
		val  string `json:"val"`
	} `json:"setincomingcookie"`
	SetResponseHeader struct {
		key   string `json:"key"`
		value string `json:"value"`
	} `json:"setresponseheader"`
	DeleteResponseHeader struct {
		key string `json:"key"`
	} `json:"deleteresponseheader"`
	PatchDynamicResourceURLs bool `json:"patchdynamicresourceurls"`
	PatchGoogleAnalytics     bool `json:"patchgoogleanalytics"`
	PatchTrackerScripts      bool `json:"patchtrackerscripts"`
	RewriteHTMLResourceURLs  bool `json:"rewritehtmlresourceurls"`
}

func PlaygroundHandler(path string, opts *ProxyOptions) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodGet {
			c.Set("Content-Type", "text/html")
			return c.SendString(playgroundHtml)
		} else if c.Method() == fiber.MethodPost {
			// Parse JSON data from the POST request body
			var modificationData ModifierQuery
			if err := c.BodyParser(&modificationData); err != nil {
				return err
			}

			// Create a new proxy chain with playground modifiers
			return proxychain.
				NewProxyChain().
				WithAPIPath(path).
				SetRequestModifications(
					BuildRequestModifications(modificationData.RequestModifierQuery)...,
				).
				AddResponseModifications(
					BuildResponseModifications(modificationData.ResponseModifierQuery)...,
				).
				SetFiberCtx(c).
				Execute()
		}

		return c.Status(http.StatusMethodNotAllowed).SendString("Method not allowed")
	}
}

func BuildRequestModifications(requestModificationData RequestModifierQuery) []proxychain.RequestModification {
	var modifications []proxychain.RequestModification

	if requestModificationData.ForwardRequestHeaders {
		modifications = append(modifications, rx.ForwardRequestHeaders())
	}
	if requestModificationData.MasqueradeAsGoogleBot {
		modifications = append(modifications, rx.MasqueradeAsGoogleBot())
	}
	if requestModificationData.MasqueradeAsBingBot {
		modifications = append(modifications, rx.MasqueradeAsBingBot())
	}
	if requestModificationData.MasqueradeAsWaybackMachineBot {
		modifications = append(modifications, rx.MasqueradeAsWaybackMachineBot())
	}
	if requestModificationData.MasqueradeAsFacebookBot {
		modifications = append(modifications, rx.MasqueradeAsFacebookBot())
	}
	if requestModificationData.MasqueradeAsYandexBot {
		modifications = append(modifications, rx.MasqueradeAsYandexBot())
	}
	if requestModificationData.MasqueradeAsBaiduBot {
		modifications = append(modifications, rx.MasqueradeAsBaiduBot())
	}
	if requestModificationData.MasqueradeAsDuckDuckBot {
		modifications = append(modifications, rx.MasqueradeAsDuckDuckBot())
	}
	if requestModificationData.MasqueradeAsYahooBot {
		modifications = append(modifications, rx.MasqueradeAsYahooBot())
	}
	if requestModificationData.ModifyDomainWithRegex.match != "" && requestModificationData.ModifyDomainWithRegex.replacement != "" {
		regex, err := regexp.Compile(requestModificationData.ModifyDomainWithRegex.match)
		if err != nil {
			log.Fatal(err)
		}
		modifications = append(modifications, rx.ModifyDomainWithRegex(*regex, requestModificationData.ModifyDomainWithRegex.replacement))
	}
	if requestModificationData.SetOutgoingCookie.name != "" && requestModificationData.SetOutgoingCookie.val != "" {
		modifications = append(modifications, rx.SetOutgoingCookie(requestModificationData.SetOutgoingCookie.name, requestModificationData.SetOutgoingCookie.val))
	}
	if requestModificationData.SetOutgoingCookies.cookies != "" {
		modifications = append(modifications, rx.SetOutgoingCookies(requestModificationData.SetOutgoingCookies.cookies))
	}
	if requestModificationData.DeleteOutgoingCookie.name != "" {
		modifications = append(modifications, rx.DeleteOutgoingCookie(requestModificationData.DeleteOutgoingCookie.name))
	}
	if requestModificationData.DeleteOutgoingCookies {
		modifications = append(modifications, rx.DeleteOutgoingCookies())
	}
	if requestModificationData.DeleteOutgoingCookiesExcept.whitelist != "" {
		// TODO: Split comma separated values in string?
		modifications = append(modifications, rx.DeleteOutgoingCookiesExcept(requestModificationData.DeleteOutgoingCookiesExcept.whitelist))
	}
	if requestModificationData.ModifyPathWithRegex.match != "" && requestModificationData.ModifyPathWithRegex.replacement != "" {
		regex, err := regexp.Compile(requestModificationData.ModifyPathWithRegex.match)
		if err != nil {
			log.Fatal(err)
		}
		modifications = append(modifications, rx.ModifyPathWithRegex(*regex, requestModificationData.ModifyPathWithRegex.replacement))
	}
	if requestModificationData.ModifyQueryParams.key != "" && requestModificationData.ModifyQueryParams.value != "" {
		modifications = append(modifications, rx.ModifyQueryParams(requestModificationData.ModifyQueryParams.key, requestModificationData.ModifyQueryParams.value))
	}
	if requestModificationData.SetRequestHeader.name != "" && requestModificationData.SetRequestHeader.val != "" {
		modifications = append(modifications, rx.SetRequestHeader(requestModificationData.SetRequestHeader.name, requestModificationData.SetRequestHeader.val))
	}
	if requestModificationData.DeleteRequestHeader.name != "" {
		modifications = append(modifications, rx.DeleteRequestHeader(requestModificationData.DeleteRequestHeader.name))
	}
	if requestModificationData.RequestArchiveIs {
		modifications = append(modifications, rx.RequestArchiveIs())
	}
	if requestModificationData.RequestGoogleCache {
		modifications = append(modifications, rx.RequestGoogleCache())
	}
	if requestModificationData.RequestWaybackMachine {
		modifications = append(modifications, rx.RequestWaybackMachine())
	}
	if requestModificationData.ResolveWithGoogleDoH {
		modifications = append(modifications, rx.ResolveWithGoogleDoH())
	}
	if requestModificationData.SpoofOrigin.url != "" {
		modifications = append(modifications, rx.SpoofOrigin(requestModificationData.SpoofOrigin.url))
	}
	if requestModificationData.HideOrigin {
		modifications = append(modifications, rx.HideOrigin())
	}
	if requestModificationData.SpoofReferrer.url != "" {
		modifications = append(modifications, rx.SpoofReferrer(requestModificationData.SpoofReferrer.url))
	}
	if requestModificationData.HideReferrer {
		modifications = append(modifications, rx.HideReferrer())
	}
	if requestModificationData.SpoofReferrerFromBaiduSearch {
		modifications = append(modifications, rx.SpoofReferrerFromBaiduSearch())
	}
	if requestModificationData.SpoofReferrerFromBingSearch {
		modifications = append(modifications, rx.SpoofReferrerFromBingSearch())
	}
	if requestModificationData.SpoofReferrerFromGoogleSearch {
		modifications = append(modifications, rx.SpoofReferrerFromGoogleSearch())
	}
	if requestModificationData.SpoofReferrerFromLinkedInPost {
		modifications = append(modifications, rx.SpoofReferrerFromLinkedInPost())
	}
	if requestModificationData.SpoofReferrerFromNaverSearch {
		modifications = append(modifications, rx.SpoofReferrerFromNaverSearch())
	}

	if requestModificationData.SpoofReferrerFromPinterestPost {
		modifications = append(modifications, rx.SpoofReferrerFromPinterestPost())
	}
	if requestModificationData.SpoofReferrerFromQQPost {
		modifications = append(modifications, rx.SpoofReferrerFromQQPost())
	}
	if requestModificationData.SpoofReferrerFromRedditPost {
		modifications = append(modifications, rx.SpoofReferrerFromRedditPost())
	}
	if requestModificationData.SpoofReferrerFromTumblrPost {
		modifications = append(modifications, rx.SpoofReferrerFromTumblrPost())
	}
	if requestModificationData.SpoofReferrerFromTwitterPost {
		modifications = append(modifications, rx.SpoofReferrerFromTwitterPost())
	}
	if requestModificationData.SpoofReferrerFromVkontaktePost {
		modifications = append(modifications, rx.SpoofReferrerFromVkontaktePost())
	}
	if requestModificationData.SpoofReferrerFromWeiboPost {
		modifications = append(modifications, rx.SpoofReferrerFromWeiboPost())
	}
	if requestModificationData.SpoofUserAgent.ua != "" {
		modifications = append(modifications, rx.SpoofUserAgent(requestModificationData.SpoofUserAgent.ua))
	}
	if requestModificationData.SpoofXForwardedFor.ip != "" {
		modifications = append(modifications, rx.SpoofXForwardedFor(requestModificationData.SpoofXForwardedFor.ip))
	}

	return modifications
}

func BuildResponseModifications(responseModificationData ResponseModifierQuery) []proxychain.ResponseModification {
	var modifications []proxychain.ResponseModification

	if responseModificationData.APIContent {
		modifications = append(modifications, tx.APIContent())
	}
	if responseModificationData.BlockElementRemoval.cssSelector != "" {
		modifications = append(modifications, tx.BlockElementRemoval(responseModificationData.BlockElementRemoval.cssSelector))
	}
	if responseModificationData.BypassCORS {
		modifications = append(modifications, tx.BypassCORS())
	}
	if responseModificationData.BypassContentSecurityPolicy {
		modifications = append(modifications, tx.BypassContentSecurityPolicy())
	}
	if responseModificationData.SetContentSecurityPolicy.csp != "" {
		modifications = append(modifications, tx.SetContentSecurityPolicy(responseModificationData.SetContentSecurityPolicy.csp))
	}
	if responseModificationData.ForwardResponseHeaders {
		modifications = append(modifications, tx.ForwardResponseHeaders())
	}
	if responseModificationData.GenerateReadableOutline {
		modifications = append(modifications, tx.GenerateReadableOutline())
	}
	if responseModificationData.InjectScriptBeforeDOMContentLoaded.js != "" {
		modifications = append(modifications, tx.InjectScriptBeforeDOMContentLoaded(responseModificationData.InjectScriptBeforeDOMContentLoaded.js))
	}
	if responseModificationData.InjectScriptAfterDOMContentLoaded.js != "" {
		modifications = append(modifications, tx.InjectScriptBeforeDOMContentLoaded(responseModificationData.InjectScriptAfterDOMContentLoaded.js))
	}
	if responseModificationData.InjectScriptAfterDOMIdle.js != "" {
		modifications = append(modifications, tx.InjectScriptBeforeDOMContentLoaded(responseModificationData.InjectScriptAfterDOMIdle.js))
	}
	if responseModificationData.DeleteIncomingCookies {
		modifications = append(modifications, tx.DeleteIncomingCookies())
	}
	if responseModificationData.DeleteIncomingCookiesExcept.whitelist != "" {
		// TODO: Split comma separated values in string?
		modifications = append(modifications, tx.DeleteIncomingCookiesExcept(responseModificationData.DeleteIncomingCookiesExcept.whitelist))
	}
	if responseModificationData.SetIncomingCookies.cookies != "" {
		modifications = append(modifications, tx.SetIncomingCookies(responseModificationData.SetIncomingCookies.cookies))
	}
	if responseModificationData.SetIncomingCookie.name != "" && responseModificationData.SetIncomingCookie.val != "" {
		modifications = append(modifications, tx.SetIncomingCookie(responseModificationData.SetIncomingCookie.name, responseModificationData.SetIncomingCookie.val))
	}
	if responseModificationData.SetResponseHeader.key != "" && responseModificationData.SetResponseHeader.value != "" {
		modifications = append(modifications, tx.SetResponseHeader(responseModificationData.SetResponseHeader.key, responseModificationData.SetResponseHeader.value))
	}
	if responseModificationData.DeleteResponseHeader.key != "" {
		modifications = append(modifications, tx.DeleteResponseHeader(responseModificationData.DeleteResponseHeader.key))
	}
	if responseModificationData.PatchDynamicResourceURLs {
		modifications = append(modifications, tx.PatchDynamicResourceURLs())
	}
	if responseModificationData.PatchGoogleAnalytics {
		modifications = append(modifications, tx.PatchGoogleAnalytics())
	}
	if responseModificationData.PatchTrackerScripts {
		modifications = append(modifications, tx.PatchTrackerScripts())
	}
	if responseModificationData.RewriteHTMLResourceURLs {
		modifications = append(modifications, tx.RewriteHTMLResourceURLs())
	}

	return modifications
}
