package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"
	"log"

	"github.com/gofiber/fiber/v2"
)

func Outline(path string, opts *ProxyOptions) fiber.Handler {

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
		result, err := proxychain.
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
				tx.APIOutline(),
			).
			SetFiberCtx(c).
			ExecuteForAPI()

		if err != nil {
			log.Fatal(err)
		}

		return c.Render("outline", fiber.Map{
			"Success": true,
			"Params":  c.Params("*"),
			"Title":   "Outline",
			"Body":    result,
		})
	}
}
