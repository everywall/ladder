package handlers

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

func Raw(c *fiber.Ctx) error {
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
	return c.SendString(body)
}
