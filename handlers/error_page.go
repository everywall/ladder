package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/everywall/ladder/proxychain/responsemodifiers/api"
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
			c.Response().SetStatusCode(500)

			errReader := api.CreateAPIErrReader(err)
			if strings.HasPrefix(c.Path(), "/api/") {
				c.Set("Content-Type", "application/json")
				return c.SendStream(errReader)
			}

			errMessageBytes, err := io.ReadAll(errReader)
			if err != nil {
				return err
			}

			var errMsg api.Error
			if err := json.Unmarshal(errMessageBytes, &errMsg); err != nil {
				return err
			}

			if strings.Contains(c.Get("Accept"), "text/plain") {
				c.Set("Content-Type", "text/plain")
				return c.SendString(errMsg.Error.Message)
			}
			if strings.Contains(c.Get("Accept"), "text/html") {
				c.Set("Content-Type", "text/html")
				tmpl.Execute(c.Response().BodyWriter(), fiber.Map{
					"Status":  http.StatusText(c.Response().StatusCode()) + ": " + fmt.Sprint(c.Response().StatusCode()),
					"Message": errMsg.Error.Message,
					"Type":    errMsg.Error.Type,
					"Cause":   errMsg.Error.Cause,
				})
				return nil
			}
			c.Set("Content-Type", "application/json")
			return c.JSON(errMsg)
		}
		return err
	}
}
