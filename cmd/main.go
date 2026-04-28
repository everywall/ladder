package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"strings"

	"ladder/handlers"
	"ladder/handlers/cli"

	"github.com/akamensky/argparse"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/favicon"
)

//go:embed favicon.ico
var faviconData string

//go:embed styles.css
var cssData embed.FS

func main() {
	parser := argparse.NewParser("ladder", "Every Wall needs a Ladder")

	portEnv := os.Getenv("PORT")
	if os.Getenv("PORT") == "" {
		portEnv = "8080"
	}

	port := parser.String("p", "port", &argparse.Options{
		Required: false,
		Default:  portEnv,
		Help:     "Port the webserver will listen on",
	})

	prefork := parser.Flag("P", "prefork", &argparse.Options{
		Required: false,
		Help:     "This will spawn multiple processes listening",
	})

	ruleset := parser.String("r", "ruleset", &argparse.Options{
		Required: false,
		Help:     "File, Directory or URL to a ruleset.yaml. Overrides RULESET environment variable.",
	})

	mergeRulesets := parser.Flag("", "merge-rulesets", &argparse.Options{
		Required: false,
		Help:     "Compiles a directory of yaml files into a single ruleset.yaml. Requires --ruleset arg.",
	})

	mergeRulesetsGzip := parser.Flag("", "merge-rulesets-gzip", &argparse.Options{
		Required: false,
		Help:     "Compiles a directory of yaml files into a single ruleset.gz Requires --ruleset arg.",
	})

	mergeRulesetsOutput := parser.String("", "merge-rulesets-output", &argparse.Options{
		Required: false,
		Help:     "Specify output file for --merge-rulesets and --merge-rulesets-gzip. Requires --ruleset and --merge-rulesets args.",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	// utility cli flag to compile ruleset directory into single ruleset.yaml
	if *mergeRulesets || *mergeRulesetsGzip {
		output := os.Stdout

		if *mergeRulesetsOutput != "" {
			output, err = os.Create(*mergeRulesetsOutput)

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		err = cli.HandleRulesetMerge(*ruleset, *mergeRulesets, *mergeRulesetsGzip, output)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if os.Getenv("PREFORK") == "true" {
		*prefork = true
	}

	// BASE_PATH allows running ladder under a sub-path, e.g. /mypath
	basePath := os.Getenv("BASE_PATH")
	if basePath != "" {
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
		basePath = strings.TrimRight(basePath, "/")
	}

	app := fiber.New(
		fiber.Config{
			Prefork:       *prefork,
			StrictRouting: true,
			ReadBufferSize: 16 * 1024,
		},
	)

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

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
		URL:  basePath + "/favicon.ico",
	}))

	if os.Getenv("NOLOGS") != "true" {
		app.Use(func(c *fiber.Ctx) error {
			log.Println(c.Method(), c.Path())

			return c.Next()
		})
	}

	// Redirect /mypath -> /mypath/
	if basePath != "" {
		app.Get(basePath, func(c *fiber.Ctx) error {
			return c.Redirect(basePath+"/", fiber.StatusMovedPermanently)
		})
	}

	router := app.Group(basePath)

	router.Get("/", handlers.Form)

	router.Get("/styles.css", func(c *fiber.Ctx) error {
		cssData, err := cssData.ReadFile("styles.css")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		c.Set("Content-Type", "text/css")

		return c.Send(cssData)
	})

	router.Get("/ruleset", handlers.Ruleset)
	router.Get("/raw/*", handlers.Raw)
	router.Post("/api", handlers.Api)
	router.Get("/api/*", handlers.Api)
	router.Get("/*", handlers.ProxySite(*ruleset))

	log.Fatal(app.Listen(":" + *port))
}
