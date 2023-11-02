package main

import (
	"ladder/handlers"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
)

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

	app.Get("/", handlers.Form)
	app.Get("raw/*", handlers.Raw)
	app.Get("api/*", handlers.Api)
	app.Get("/*", handlers.ProxySite)

	port := os.Getenv("PORT")
	if os.Getenv("PORT") == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))

}
