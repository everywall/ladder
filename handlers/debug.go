package handlers

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

func Debug(c *fiber.Ctx) error {
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	req.Header.Set("X-Forwarded-For", "66.249.66.1")
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
