package handlers

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

func Api(c *fiber.Ctx) error {
	// Get the url from the URL
	urlQuery := c.Params("*")

	u, err := url.Parse(urlQuery)
	if err != nil {
		return c.SendString(err.Error())
	}

	log.Println(u.String())

	// Fetch the site
	client := &http.Client{}
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("X-Forwarded-For", ForwardedFor)
	req.Header.Set("Referer", u.String())
	req.Header.Set("Host", u.Host)
	resp, err := client.Do(req)

	if err != nil {
		return c.SendString(err.Error())
	}
	defer resp.Body.Close()

	bodyB, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("ERROR", err)
		return c.SendString(err.Error())
	}
	body := rewriteHtml(bodyB, u)
	response := Response{
		Body: body,
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
	Body    string `json:"body"`
	Request struct {
		Headers []interface{} `json:"headers"`
	} `json:"request"`
	Response struct {
		Headers []interface{} `json:"headers"`
	} `json:"response"`
}
