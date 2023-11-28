package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"

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

	return func(c *fiber.Ctx) error {
		proxychain := proxychain.
			NewProxyChain().
			SetFiberCtx(c).
			SetDebugLogging(opts.Verbose).
			SetRequestModifications(
				//rx.MasqueradeAsFacebookBot(),
				rx.MasqueradeAsGoogleBot(),
				//rx.DeleteOutgoingCookies(),
				rx.ForwardRequestHeaders(),
				rx.SetOutgoingCookie("nyt-a", " "),
				rx.SetOutgoingCookie("nyt-gdpr", "0"),
				rx.SetOutgoingCookie("nyt-gdpr", "0"),
				rx.SetOutgoingCookie("nyt-geo", "DE"),
				rx.SetOutgoingCookie("nyt-privacy", "1"),
				rx.SpoofReferrerFromGoogleSearch(),
				//rx.RequestWaybackMachine(),
				//rx.RequestArchiveIs(),
			).
			AddResponseModifications(
				tx.BypassCORS(),
				tx.BypassContentSecurityPolicy(),
				//tx.DeleteIncomingCookies(),
				tx.ForwardResponseHeaders(),
				tx.RewriteHTMLResourceURLs(),
				tx.PatchDynamicResourceURLs(),
				//tx.SetContentSecurityPolicy("default-src * 'unsafe-inline' 'unsafe-eval' data: blob:;"),
			).
			Execute()

		return proxychain
	}
}
