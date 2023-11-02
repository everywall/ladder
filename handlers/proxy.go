package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ProxySite(c *fiber.Ctx) error {
	// Get the url from the URL
	urlQuery := c.Params("*")

	u, err := url.Parse(urlQuery)
	if err != nil {
		log.Println("ERROR", err)
		c.SendStatus(500)
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
		c.SendStatus(500)
		return c.SendString(err.Error())
	}

	body := rewriteHtml(bodyB, u)
	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	return c.SendString(body)
}

func rewriteHtml(bodyB []byte, u *url.URL) string {
	// Rewrite the HTML
	body := string(bodyB)

	// images
	imagePattern := `<img\s+([^>]*\s+)?src="(/)([^"]*)"`
	re := regexp.MustCompile(imagePattern)
	body = re.ReplaceAllString(body, fmt.Sprintf(`<img $1 src="%s$3"`, "/https://"+u.Host+"/"))

	// scripts
	scriptPattern := `<script\s+([^>]*\s+)?src="(/)([^"]*)"`
	reScript := regexp.MustCompile(scriptPattern)
	body = reScript.ReplaceAllString(body, fmt.Sprintf(`<script $1 script="%s$3"`, "/https://"+u.Host+"/"))

	//body = strings.ReplaceAll(body, "srcset=\"/", "srcset=\"/https://"+u.Host+"/") // TODO: Needs a regex to rewrite the URL's
	body = strings.ReplaceAll(body, "href=\"/", "href=\"/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url('/", "url('/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url(/", "url(/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "href=\"https://"+u.Host, "href=\"/https://"+u.Host+"/")

	return body
}
