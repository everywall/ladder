package handlers

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
)

func Ruleset(c *fiber.Ctx) error {
	if os.Getenv("EXPOSE_RULESET") == "false" {
		c.SendStatus(fiber.StatusForbidden)
		return c.SendString("Rules Disabled")
	}

	body, err := yaml.Marshal(rulesSet)
	if err != nil {
		c.SendStatus(fiber.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	return c.SendString(string(body))
}
