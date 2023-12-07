package handlers

import (
	"fmt"
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifiers"
	tx "ladder/proxychain/responsemodifiers"

	"github.com/gofiber/fiber/v2"
)

func NewRawProxySiteHandler(opts *ProxyOptions) fiber.Handler {

	return func(c *fiber.Ctx) error {
		proxychain := proxychain.
			NewProxyChain().
			SetFiberCtx(c).
			SetRequestModifications(
				rx.AddCacheBusterQuery(),
				rx.MasqueradeAsGoogleBot(),
				rx.ForwardRequestHeaders(),
				rx.HideOrigin(),
				rx.DeleteOutgoingCookies(),
				rx.SpoofReferrerFromRedditPost(),
			)

		// no options passed in, return early
		if opts == nil {
			// return as plaintext, overriding any rules
			proxychain.AddOnceResponseModifications(
				tx.SetResponseHeader("content-type", "text/plain; charset=UTF-8"),
			)

			return proxychain.Execute()
		}

		// load ruleset
		rule, exists := opts.Ruleset.GetRule(proxychain.Request.URL)
		if exists {
			proxychain.AddOnceRequestModifications(rule.RequestModifications...)
			proxychain.AddOnceResponseModifications(rule.ResponseModifications...)
		}

		// return as plaintext, overriding any rules
		proxychain.AddOnceResponseModifications(
			tx.SetResponseHeader("content-type", "text/plain; charset=UTF-8"),
		)

		return proxychain.Execute()
	}
}
