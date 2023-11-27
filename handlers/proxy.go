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
				rx.DeleteOutgoingCookies(),
				//rx.RequestArchiveIs(),
				rx.MasqueradeAsGoogleBot(),
			).
			AddResponseModifications(
				tx.BypassCORS(),
				tx.BypassContentSecurityPolicy(),
				tx.DeleteIncomingCookies(),
				tx.RewriteHTMLResourceURLs(),
				tx.PatchDynamicResourceURLs(),
			).
			Execute()

		return proxychain
	}
}
