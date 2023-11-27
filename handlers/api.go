package handlers

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

//nolint:all
//go:embed VERSION
var version string

func Api(c *fiber.Ctx) error {
	return nil
}
