package handlers

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/imroc/req/v3"
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

func Debug2(c *fiber.Ctx) error {
	client := req.C()        // Use C() to create a client.
	resp, err := client.R(). // Use R() to create a request.
					Get("https://httpbin.org/uuid")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp)
	return c.SendString(resp.String())
}
