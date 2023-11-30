package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"

	"github.com/gofiber/fiber/v2"
)

func NewAPIOutlineHandler(path string, opts *ProxyOptions) fiber.Handler {
	// TODO: implement ruleset logic
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
			WithAPIPath(path).
			SetDebugLogging(opts.Verbose).
			SetRequestModifications(
				rx.MasqueradeAsGoogleBot(),
				rx.ForwardRequestHeaders(),
				rx.SpoofReferrerFromGoogleSearch(),
			).
			AddResponseModifications(
				tx.DeleteIncomingCookies(),
				tx.RewriteHTMLResourceURLs(),
				tx.APIContent(),
			).
			SetFiberCtx(c).
			Execute()

		return proxychain
	}
}
