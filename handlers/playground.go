package handlers

import (
	_ "embed"
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"

	"net/http"

	"github.com/gofiber/fiber/v2"
)

//go:embed playground.html
var playgroundHtml string

type ModificationQuery struct {
	ForwardRequestHeaders   bool `json:"forward_request_headers"`
	MasqueradeAsFacebookBot bool `json:"masquerade_as_facebook_bot"`
	MasqueradeAsGoogleBot   bool `json:"masquerade_as_google_bot"`
	ModifyDomainWithRegex   bool `json:"modify_domain_with_regex"`
	ModifyOutgoingCookies   bool `json:"modify_outgoing_cookies"`
	ModifyPathWithRegex     bool `json:"modify_path_with_regex"`
	GenerateReadableOutline bool `json:"generate_readable_outline"`
}

func PlaygroundHandler(path string, opts *ProxyOptions) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodGet {
			c.Set("Content-Type", "text/html")
			return c.SendString(playgroundHtml)
		} else if c.Method() == fiber.MethodPost {
			// Parse JSON data from the POST request body
			var modificationData ModificationQuery
			if err := c.BodyParser(&modificationData); err != nil {
				return err
			}

			// Create a new proxy chain with playground modifiers
			return proxychain.
				NewProxyChain().
				WithAPIPath(path).
				SetRequestModifications(
					rx.MasqueradeAsGoogleBot(),
					rx.ForwardRequestHeaders(),
					rx.SpoofReferrerFromGoogleSearch(),
				).
				AddResponseModifications(
					BuildResponseModifications(modificationData)...,
				).
				SetFiberCtx(c).
				Execute()
		}

		return c.Status(http.StatusMethodNotAllowed).SendString("Method not allowed")
	}
}

func BuildResponseModifications(modificationData ModificationQuery) []proxychain.ResponseModification {
	modifications := []proxychain.ResponseModification{
		tx.DeleteIncomingCookies(),
		tx.RewriteHTMLResourceURLs()}
	if modificationData.GenerateReadableOutline {
		modifications = append(modifications, tx.GenerateReadableOutline())
	}
	return modifications
}
