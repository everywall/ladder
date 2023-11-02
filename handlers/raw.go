package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func Raw(c *fiber.Ctx) error {
	// Get the url from the URL
	urlQuery := c.Params("*")

	body, _, _, err := fetchSite(urlQuery)
	if err != nil {
		log.Println("ERROR:", err)
		c.SendStatus(500)
		return c.SendString(err.Error())
	}
	return c.SendString(body)
}
