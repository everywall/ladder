package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifiers"
	tx "ladder/proxychain/responsemodifiers"
	"ladder/proxychain/ruleset"

	"github.com/gofiber/fiber/v2"
)

type ProxyOptions struct {
	Ruleset ruleset_v2.IRuleset
	Verbose bool
}

func NewProxySiteHandler(opts *ProxyOptions) fiber.Handler {
	/*
		var rs ruleset.RuleSet
		if opts.RulesetPath != "" {
			r, err := ruleset.NewRuleset(opts.RulesetPath)
			if err != nil {
				panic(err)
			}
			rs = r
		}
	*/

	return func(c *fiber.Ctx) error {
		proxychain := proxychain.
			NewProxyChain().
			SetFiberCtx(c).
			SetDebugLogging(opts.Verbose).
			SetRequestModifications(
				//rx.SpoofJA3fingerprint(ja3, "Googlebot"),
				rx.AddCacheBusterQuery(),
				//rx.MasqueradeAsGoogleBot(),
				rx.ForwardRequestHeaders(),
				rx.DeleteOutgoingCookies(),
				rx.SpoofReferrerFromRedditPost(),
				//rx.SpoofReferrerFromLinkedInPost(),
				//rx.RequestWaybackMachine(),
				//rx.RequestArchiveIs(),
			).
			AddResponseModifications(
				//tx.ForwardResponseHeaders(),
				//tx.BlockThirdPartyScripts(),
				tx.DeleteIncomingCookies(),
				tx.DeleteLocalStorageData(),
				tx.DeleteSessionStorageData(),
				tx.BypassCORS(),
				tx.BypassContentSecurityPolicy(),
				tx.RewriteHTMLResourceURLs(),
				tx.PatchDynamicResourceURLs(),
				tx.PatchTrackerScripts(),
				//tx.BlockElementRemoval(".article-content"), // techcrunch
				tx.BlockElementRemoval(".available-content"), // substack
			// tx.SetContentSecurityPolicy("default-src * 'unsafe-inline' 'unsafe-eval' data: blob:;"),
			)

		// load ruleset
		rule, exists := opts.Ruleset.GetRule(proxychain.Request.URL)
		if exists {
			proxychain.AddOnceRequestModifications(rule.RequestModifications...)
			proxychain.AddOnceResponseModifications(rule.ResponseModifications...)
		}

		return proxychain.Execute()
	}
}
