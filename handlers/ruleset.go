package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
	"net/url"
	"strings"
)

func NewRulesetSiteHandler(opts *ProxyOptions) fiber.Handler {

	return func(c *fiber.Ctx) error {
		if opts == nil {
			c.SendStatus(404)
			c.SendString("No ruleset specified. Set the RULESET environment variable or use the --ruleset flag.")
		}

		// no specific rule requested, return the entire ruleset
		if c.Params("*") == "" {
			switch c.Get("accept") {
			case "application/json":
				jsn, err := opts.Ruleset.JSON()
				if err != nil {
					return err
				}
				c.Set("content-type", "application/json")
				return c.Send([]byte(jsn))

			default:
				// TODO: the ruleset.MarshalYAML() method is currently broken and panics
				yml, err := opts.Ruleset.YAML()
				if err != nil {
					return err
				}
				c.Set("content-type", "application/yaml")
				return c.Send([]byte(yml))
			}
		}

		// a specific rule was requested by path /ruleset/https://example.com
		// return only that particular rule
		reqURL, err := extractURLFromContext(c, "ruleset/")
		if err != nil {
			return err
		}
		rule, exists := opts.Ruleset.GetRule(reqURL)
		if !exists {
			c.SendStatus(404)
			c.SendString(fmt.Sprintf("A rule that matches '%s' was not found in the ruleset.", reqURL))
		}

		switch c.Get("accept") {
		case "application/json":
			jsn, err := json.MarshalIndent(rule, "", "  ")
			if err != nil {
				return err
			}
			c.Set("content-type", "application/json")
			return c.Send(jsn)
		default:
			yml, err := yaml.Marshal(rule)
			if err != nil {
				return err
			}
			c.Set("content-type", "application/yaml")
			return c.Send(yml)
		}
	}
}

// extractURLFromContext extracts a URL from the request ctx.
func extractURLFromContext(ctx *fiber.Ctx, apiPrefix string) (*url.URL, error) {
	reqURL := ctx.Params("*")

	reqURL = strings.TrimPrefix(reqURL, apiPrefix)

	// sometimes client requests doubleroot '//'
	// there is a bug somewhere else, but this is a workaround until we find it
	if strings.HasPrefix(reqURL, "/") || strings.HasPrefix(reqURL, `%2F`) {
		reqURL = strings.TrimPrefix(reqURL, "/")
		reqURL = strings.TrimPrefix(reqURL, `%2F`)
	}

	// unescape url query
	uReqURL, err := url.QueryUnescape(reqURL)
	if err == nil {
		reqURL = uReqURL
	}

	return url.Parse(reqURL)
}
