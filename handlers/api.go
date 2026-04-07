package handlers

import (
	_ "embed"
	"log"

	"github.com/gofiber/fiber/v2"
)

type JsonRequest struct {
	URL string `json:"url"`
}

//nolint:all
//go:embed VERSION
var version string

func Api(c *fiber.Ctx) error {
	var url string
	queries := c.Queries()

	// Check content type to determine if it's JSON
	contentType := c.Get("Content-Type")
	if contentType == "application/json" {
		// Parse JSON body
		var jsonReq JsonRequest
		if err := c.BodyParser(&jsonReq); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid JSON request",
			})
		}
		url = jsonReq.URL
	} else {
		// Get the url from the URL params
		url = c.Params("*")
	}

	body, req, resp, err := fetchSite(url, queries)
	if err != nil {
		log.Println("ERROR:", err)
		c.SendStatus(500)
		return c.SendString(err.Error())
	}

	response := Response{
		Version: version,
		Body:    body,
	}

	response.Request.Headers = make([]any, 0, len(req.Header))
	for k, v := range req.Header {
		response.Request.Headers = append(response.Request.Headers, map[string]string{
			"key":   k,
			"value": v[0],
		})
	}

	response.Response.Headers = make([]any, 0, len(resp.Header))
	for k, v := range resp.Header {
		response.Response.Headers = append(response.Response.Headers, map[string]string{
			"key":   k,
			"value": v[0],
		})
	}

	return c.JSON(response)
}

type Response struct {
	Version string `json:"version"`
	Body    string `json:"body"`
	Request struct {
		Headers []interface{} `json:"headers"`
	} `json:"request"`
	Response struct {
		Headers []interface{} `json:"headers"`
	} `json:"response"`
}
