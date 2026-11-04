package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
	"os"
)

func Ruleset(c *fiber.Ctx) error {
	if os.Getenv("EXPOSE_RULESET") == "false" {
		return c.Status(403).SendString("Rules Disabled")
	}
	s, err := configured()
	if err != nil {
		return c.Status(500).SendString("Invalid transport configuration")
	}
	body, err := yaml.Marshal(s.LegacyRules())
	if err != nil {
		return c.Status(500).SendString("Rules unavailable")
	}
	return c.Type("yaml").Send(body)
}
