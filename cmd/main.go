package main

import (
	_ "embed"
	"fmt"

	"ladder/handlers"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/favicon"
)

//go:embed favicon.ico
var faviconData string

func main() {

	parser := argparse.NewParser("ladder", "Every Wall needs a Ladder")

	p := os.Getenv("PORT")
	if os.Getenv("PORT") == "" {
		p = "8080"
	}
	port := parser.String("p", "port", &argparse.Options{
		Required: false,
		Default:  p,
		Help:     "Port the webserver will listen on"})

	pf, _ := strconv.ParseBool(os.Getenv("PREFORK"))
	prefork := parser.Flag("P", "prefork", &argparse.Options{
		Required: false,
		Default:  pf,
		Help:     "This will spawn multiple processes listening"})

	r := os.Getenv("RULESET")
	ruleset := parser.String("r", "ruleset", &argparse.Options{
		Required: false,
		Default:  r,
		Help:     "Path or URL to your ruleset"})

	handlers.LoadRules(*ruleset)

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	app := fiber.New(
		fiber.Config{
			Prefork: *prefork,
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

	app.Get("/", handlers.Form)
	app.Get("ruleset", handlers.Ruleset)

	app.Get("raw/*", handlers.Raw)
	app.Get("api/*", handlers.Api)
	app.Get("ruleset", handlers.Raw)
	app.Get("/*", handlers.ProxySite)

	log.Fatal(app.Listen(":" + *port))

}
