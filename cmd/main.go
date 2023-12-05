package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"os"

	"ladder/handlers"
	"ladder/internal/cli"
	"ladder/proxychain/requestmodifiers/bot"

	"github.com/akamensky/argparse"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

//go:embed styles.css
var cssData embed.FS

//go:embed script.js
var scriptData embed.FS

//go:embed VERSION
var version string

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

	verbose := parser.Flag("v", "verbose", &argparse.Options{
		Required: false,
		Help:     "Adds verbose logging",
	})

	randomGoogleBot := parser.Flag("", "random-googlebot", &argparse.Options{
		Required: false,
		Help:     "Update the list of trusted Googlebot IPs, and use a random one for each masqueraded request",
	})

	randomBingBot := parser.Flag("", "random-bingbot", &argparse.Options{
		Required: false,
		Help:     "Update the list of trusted Bingbot IPs, and use a random one for each masqueraded request",
	})

	// TODO: add version flag that reads from handers/VERSION

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

	if *randomGoogleBot {
		err := bot.GoogleBot.UpdatePool("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
		if err != nil {
			fmt.Println("error while retrieving list of Googlebot IPs: " + err.Error())
			fmt.Println("defaulting to known trusted Googlebot identity")
		}
	}

	if *randomBingBot {
		err := bot.BingBot.UpdatePool("https://www.bing.com/toolbox/bingbot.json")
		if err != nil {
			fmt.Println("error while retrieving list of Bingbot IPs: " + err.Error())
			fmt.Println("defaulting to known trusted Bingbot identity")
		}
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

	engine := html.New("./handlers", ".html")
	engine.AddFunc(
		// add unescape function
		"unescape", func(s string) template.HTML {
			return template.HTML(s)
		},
	)

	app := fiber.New(
		fiber.Config{
			Prefork:               *prefork,
			GETOnly:               false,
			ReadBufferSize:        4096 * 4, // increase max header size
			DisableStartupMessage: true,
			Views:                 engine,
		},
	)

	app.Use(handlers.Auth())
	app.Use(handlers.Favicon())

	if os.Getenv("NOLOGS") != "true" {
		app.Use(func(c *fiber.Ctx) error {
			log.Println(c.Method(), c.Path())

			return c.Next()
		})
	}

	app.Get("/", handlers.Form)

	app.Get("/styles.css", func(c *fiber.Ctx) error {
		cssData, err := cssData.ReadFile("styles.css")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		c.Set("Content-Type", "text/css")

		return c.Send(cssData)
	})

	// TODO: move to handlers/script.go
	app.Get("/script.js", func(c *fiber.Ctx) error {
		scriptData, err := scriptData.ReadFile("script.js")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		c.Set("Content-Type", "text/javascript")

		return c.Send(scriptData)
	})

	app.Get("ruleset", handlers.Ruleset)
	app.Get("raw/*", handlers.Raw)

	proxyOpts := &handlers.ProxyOptions{
		Verbose:     *verbose,
		RulesetPath: *ruleset,
	}

	app.Get("api/content/*", handlers.NewAPIContentHandler("api/outline/*", proxyOpts))

	app.Get("outline/*", handlers.NewOutlineHandler("outline/*", proxyOpts))

	app.All("/*", handlers.NewProxySiteHandler(proxyOpts))

	fmt.Println(cli.StartupMessage(version, *port, *ruleset))
	log.Fatal(app.Listen(":" + *port))
}
