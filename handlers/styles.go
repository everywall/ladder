package handlers

import (
	"embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed styles.css
var cssData embed.FS

func Styles(c *fiber.Ctx) error {

	cssData, err := cssData.ReadFile("styles.css")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	c.Set("Content-Type", "text/css")

	return c.Send(cssData)

}
