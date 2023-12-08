package handlers

import (
	_ "embed"
	"ladder/proxychain"
	ruleset_v2 "ladder/proxychain/ruleset"

	"net/http"

	"github.com/gofiber/fiber/v2"
)

//go:embed playground.html
var playgroundHtml string

func PlaygroundHandler(path string, opts *ProxyOptions) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodGet {
			c.Set("Content-Type", "text/html")

			return c.SendString(playgroundHtml)
		} else if c.Method() == fiber.MethodPost {
			var modificationData ruleset_v2.Rule
			if err := c.BodyParser(&modificationData); err != nil {
				return err
			}

			c.Method(fiber.MethodGet)

			return proxychain.
				NewProxyChain().
				SetFiberCtx(c).
				WithAPIPath(path).
				AddOnceRequestModifications(modificationData.RequestModifications...).
				AddOnceResponseModifications(modificationData.ResponseModifications...).
				Execute()
		}

		return c.Status(http.StatusMethodNotAllowed).SendString("Method not allowed")
	}
}
