package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"

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
				tx.DeleteIncomingCookies(),
				tx.RewriteHTMLResourceURLs(),
				tx.GenerateReadableOutline(), // <-- this response modification does the outline rendering
			).
			SetFiberCtx(c).
			Execute()

	}
}
