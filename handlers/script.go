package handlers

import (
	"embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed script.js
var scriptData embed.FS

//go:embed playground-script.js
var playgroundScriptData embed.FS

func Script(c *fiber.Ctx) error {
	if c.Path() == "/script.js" {
		scriptData, err := scriptData.ReadFile("script.js")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}
		c.Set("Content-Type", "text/javascript")
		return c.Send(scriptData)
	}
	if c.Path() == "/playground-script.js" {
		playgroundScriptData, err := playgroundScriptData.ReadFile("playground-script.js")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}
		c.Set("Content-Type", "text/javascript")
		return c.Send(playgroundScriptData)
	}
	return nil
}
