package handlers

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"ladder/proxychain/responsemodifiers/api"
)

func NewAPIModifersListHandler(opts *ProxyOptions) fiber.Handler {
	payload := ModifiersAPIResponse{
		Success: true,
		Result:  AllMods,
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}

	return func(c *fiber.Ctx) error {
		c.Set("content-type", "application/json")
		if err != nil {
			c.SendStatus(500)
			return c.SendStream(api.CreateAPIErrReader(err))
		}

		return c.Send(body)
	}
}
