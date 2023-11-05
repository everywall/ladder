package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
)

func Ruleset(c *fiber.Ctx) error {

	body, err := yaml.Marshal(rulesSet)
	if err != nil {
		c.SendStatus(fiber.StatusInternalServerError)
		return c.SendString(err.Error())
	}

	return c.SendString(string(body))
}
