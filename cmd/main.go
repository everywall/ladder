package main

import (
	"log"
	"os"
	"paywall-ladder/handlers"
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

	//app.Static("/", "./public")
	app.Get("/", handlers.Form)

	app.Get("/proxy/*", handlers.FetchSite)
	app.Get("debug/*", handlers.Debug)
	app.Get("debug2/*", handlers.Debug2)
	app.Get("/*", handlers.ProxySite)

	port := os.Getenv("PORT")
	if os.Getenv("PORT") == "" {
		port = "2000"
	}
	log.Fatal(app.Listen(":" + port))

}
