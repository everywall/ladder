package handlers

import (
	"embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed script.js
var scriptData embed.FS

func Script(c *fiber.Ctx) error {

	scriptData, err := scriptData.ReadFile("script.js")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	c.Set("Content-Type", "text/javascript")

	return c.Send(scriptData)

}
