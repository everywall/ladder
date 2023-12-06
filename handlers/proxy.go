package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifiers"
	tx "ladder/proxychain/responsemodifiers"
	"ladder/proxychain/ruleset"

	"github.com/gofiber/fiber/v2"
)

type ProxyOptions struct {
	RulesetPath string
	Verbose     bool
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
	rs, err := ruleset_v2.NewRuleset("ruleset_v2.yaml")
	if err != nil {
		panic(err)
	}

	return func(c *fiber.Ctx) error {
		proxychain := proxychain.
			NewProxyChain().
			SetFiberCtx(c).
			SetDebugLogging(opts.Verbose).
			SetRequestModifications(
				// rx.SpoofJA3fingerprint(ja3, "Googlebot"),
				// rx.MasqueradeAsFacebookBot(),
				// rx.MasqueradeAsGoogleBot(),
				rx.DeleteOutgoingCookies(),
				rx.ForwardRequestHeaders(),
				// rx.SpoofReferrerFromGoogleSearch(),
				rx.SpoofReferrerFromLinkedInPost(),
				// rx.RequestWaybackMachine(),
				// rx.RequestArchiveIs(),
			).
			AddResponseModifications(
				tx.ForwardResponseHeaders(),
				tx.BypassCORS(),
				tx.BypassContentSecurityPolicy(),
				// tx.DeleteIncomingCookies(),
				tx.RewriteHTMLResourceURLs(),
				tx.PatchTrackerScripts(),
				tx.PatchDynamicResourceURLs(),
				tx.BlockElementRemoval(".article-content"),
			// tx.SetContentSecurityPolicy("default-src * 'unsafe-inline' 'unsafe-eval' data: blob:;"),
			)

		// load ruleset
		rule, exists := rs.GetRule(proxychain.Request.URL)
		if exists {
			proxychain.AddOnceRequestModifications(rule.RequestModifications...)
			proxychain.AddOnceResponseModifications(rule.ResponseModifications...)
		}

		return proxychain.Execute()
	}
}
