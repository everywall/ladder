package handlers

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
)

//go:embed favicon.ico
var faviconData string

func Favicon() fiber.Handler {
	return favicon.New(favicon.Config{
		Data: []byte(faviconData),
		URL:  "/favicon.ico",
	})
}
