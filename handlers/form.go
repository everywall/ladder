package handlers

import (
	_ "embed"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

//go:embed form.html
var formHtml string

func Form(c *fiber.Ctx) error {
	if os.Getenv("DISABLE_FORM") == "true" {
		c.Set("Content-Type", "text/html")
		c.SendStatus(fiber.StatusNotFound)
		return c.SendString("Form Disabled")
	} else {
		if os.Getenv("FORM_PATH") != "" {
			dat, err := os.ReadFile(os.Getenv("FORM_PATH"))
			if err != nil {
				log.Println("ERROR: unable to load custom form", err)
			} else {
				formHtml = string(dat)
			}
		}
		c.Set("Content-Type", "text/html")
		return c.SendString(formHtml)
	}
}
