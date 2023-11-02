package main

import (
	"ladder/handlers"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func main() {

	prefork, _ := strconv.ParseBool(os.Getenv("PREFORK"))
	app := fiber.New(
		fiber.Config{
			Prefork: prefork,
		},
	)

	app.Get("/", handlers.Form)
	app.Get("debug/*", handlers.Debug)
	app.Get("api/*", handlers.Api)
	app.Get("/*", handlers.ProxySite)

	port := os.Getenv("PORT")
	if os.Getenv("PORT") == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))

}
