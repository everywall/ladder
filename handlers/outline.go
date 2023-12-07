package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifiers"
	tx "ladder/proxychain/responsemodifiers"

	"github.com/gofiber/fiber/v2"
)

func NewOutlineHandler(path string, opts *ProxyOptions) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return proxychain.
			NewProxyChain().
			WithAPIPath(path).
			SetDebugLogging(opts.Verbose).
			SetRequestModifications(
				rx.MasqueradeAsGoogleBot(),
				rx.ForwardRequestHeaders(),
				rx.SpoofReferrerFromGoogleSearch(),
			).
			AddResponseModifications(
				tx.SetResponseHeader("content-type", "text/html"),
				tx.DeleteIncomingCookies(),
				tx.RewriteHTMLResourceURLs(),
				tx.GenerateReadableOutline(), // <-- this response modification does the outline rendering
			).
			SetFiberCtx(c).
			Execute()
	}
}
