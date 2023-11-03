package main

import (
	_ "embed"

	"ladder/handlers"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/favicon"
)

//go:embed favicon.ico
var faviconData string

func main() {

	prefork, _ := strconv.ParseBool(os.Getenv("PREFORK"))
	app := fiber.New(
		fiber.Config{
			Prefork: prefork,
		},
	)

	userpass := os.Getenv("USERPASS")
	if userpass != "" {
		userpass := strings.Split(userpass, ":")
		app.Use(basicauth.New(basicauth.Config{
			Users: map[string]string{
				userpass[0]: userpass[1],
			},
		}))
	}

	app.Use(favicon.New(favicon.Config{
		Data: []byte(faviconData),
		URL:  "/favicon.ico",
	}))

	if os.Getenv("NOLOGS") != "true" {
		app.Use(func(c *fiber.Ctx) error {
			log.Println(c.Method(), c.Path())
			return c.Next()
		})
	}

	if os.Getenv("DISABLE_FORM") != "true" {
		app.Get("/", handlers.Form)
	} else {
		app.Get("/", handlers.NoForm)
	}

	app.Get("raw/*", handlers.Raw)
	app.Get("api/*", handlers.Api)
	app.Get("/*", handlers.ProxySite)

	port := os.Getenv("PORT")
	if os.Getenv("PORT") == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))

}
