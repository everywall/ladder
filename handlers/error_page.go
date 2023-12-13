package handlers

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/gofiber/fiber/v2"
)

//go:embed error_page.html
var errorHTML embed.FS

func RenderErrorPage() fiber.Handler {
	f := "error_page.html"
	tmpl, err := template.ParseFS(errorHTML, f)
	if err != nil {
		panic(fmt.Errorf("RenderErrorPage Error: %s not found", f))
	}
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil {
			if strings.Contains(c.Get("Accept"), "text/html") {
				c.Set("Content-Type", "text/html")
				tmpl.Execute(c.Response().BodyWriter(), err.Error())
				return nil
			}
			return c.SendStream(bytes.NewBufferString(err.Error()))
		}
		return err
	}
}
