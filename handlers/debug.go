package handlers

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gojek/heimdall/v7/httpclient"
)

func Debug(c *fiber.Ctx) error {
	//url := c.Params("*")

	timeout := 1000 * time.Millisecond
	client := httpclient.NewClient(httpclient.WithHTTPTimeout(timeout))

	headers := http.Header{}
	headers.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	headers.Set("X-Forwarded-For", "66.249.66.1")
	res, err := client.Get("http://google.com", headers)
	if err != nil {
		panic(err)
	}

	// Heimdall returns the standard *http.Response object
	body, err := ioutil.ReadAll(res.Body)
	//fmt.Println(string(body))
	/*
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

	*/
	return c.SendString(string(body))
}
