package handlers

import (
	_ "embed"

	"log"

	"github.com/gofiber/fiber/v2"
)

//go:embed VERSION
var version string

func Api(c *fiber.Ctx) error {
	// Get the url from the URL
	urlQuery := c.Params("*")

	queries := c.Queries()
	body, req, resp, err := fetchSite(urlQuery, queries)
	if err != nil {
		log.Println("ERROR:", err)
		c.SendStatus(500)
		return c.SendString(err.Error())
	}

	response := Response{
		Version: version,
		Body:    body,
	}
	response.Request.Headers = make([]interface{}, 0)
	for k, v := range req.Header {
		response.Request.Headers = append(response.Request.Headers, map[string]string{
			"key":   k,
			"value": v[0],
		})
	}

	response.Response.Headers = make([]interface{}, 0)
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
