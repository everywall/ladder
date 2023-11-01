package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gojek/heimdall/v7/httpclient"
)

type loggingTransport struct{}

func (s *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {

	bytes, _ := httputil.DumpRequestOut(r, true)

	resp, err := http.DefaultTransport.RoundTrip(r)
	// err is returned after dumping the response

	respBytes, _ := httputil.DumpResponse(resp, false)
	bytes = append(bytes, respBytes...)

	fmt.Printf("%s\n", bytes)

	return resp, err
}

func FetchSite(c *fiber.Ctx) error {
	// Get the url from the URL
	urlQuery := c.Params("*")

	u, err := url.Parse(urlQuery)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(u.String())

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	// Fetch the site
	//resp, err := http.Get(url)
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	req.Header.Set("X-Forwarded-For", "66.249.66.1")
	//fmt.Println(c.GetReqHeaders()["Cookie"])
	//req.Header.Set("Cookie", fmt.Sprint(c.GetReqHeaders()["Cookie"]))

	//resp, err := http.DefaultClient.Do(req)
	//client := &http.Client{}
	//client.Timeout = time.Duration(5 * time.Second)

	client := http.Client{}
	client.Transport = &loggingTransport{}

	fmt.Println("DEBUG1")
	resp, err := client.Do(req)
	fmt.Println("DEBUG2")

	/*
		timeout := 1000 * time.Millisecond
		client := httpclient.NewClient(httpclient.WithHTTPTimeout(timeout))

		headers := http.Header{}
		headers.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
		headers.Set("X-Forwarded-For", "66.249.66.1")

		fmt.Println("DEBUG1")
		resp, err := client.Get(u.String(), headers)
		if err != nil {
			panic(err)
		}
		fmt.Println("DEBUG2")
	*/
	if err != nil {
		return c.SendString(err.Error())
	}
	defer resp.Body.Close()

	bodyB, err := io.ReadAll(resp.Body)
	if err != nil {
		//log.Fatalln(err)
		return c.SendString(err.Error())
	}
	body := string(bodyB)

	imagePattern := `<img\s+([^>]*\s+)?src="(/)([^"]*)"`
	re := regexp.MustCompile(imagePattern)
	body = re.ReplaceAllString(body, fmt.Sprintf(`<img $1 src="%s$3"`, "/proxy/https://"+u.Host+"/"))

	scriptPattern := `<script\s+([^>]*\s+)?src="(/)([^"]*)"`
	reScript := regexp.MustCompile(scriptPattern)
	body = reScript.ReplaceAllString(body, fmt.Sprintf(`<script $1 script="%s$3"`, "/proxy/https://"+u.Host+"/"))

	//body = strings.ReplaceAll(body, "srcset=\"/", "srcset=\"/proxy/https://"+u.Host+"/")

	//body = strings.ReplaceAll(body, "https://"+u.Host, "/proxy/https://"+u.Host)
	body = strings.ReplaceAll(body, "href=\"/", "href=\"/proxy/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "href=\"https://"+u.Host, "href=\"/proxy/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url('/", "url('/proxy/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url(/", "url(/proxy/https://"+u.Host+"/")

	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	return c.SendString(body)
}

func ProxySite(c *fiber.Ctx) error {
	// Get the url from the URL
	urlQuery := c.Params("*")

	u, err := url.Parse(urlQuery)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(u.String())

	// Fetch the site
	//resp, err := http.Get(u.String())
	/*
		client := &http.Client{}
		req, _ := http.NewRequest("GET", u.String(), nil)
		//req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
		//req.Header.Set("X-Forwarded-For", "66.249.66.1")
		//req.Header.Set("Referer", u.String())
		//req.Header.Set("Host", u.Host)
		resp, err := client.Do(req)
	*/

	timeout := 1000 * time.Millisecond
	client := httpclient.NewClient(httpclient.WithHTTPTimeout(timeout))

	resp, err := client.Get(u.String(), nil)
	if err != nil {
		panic(err)
	}

	if err != nil {
		return c.SendString(err.Error())
	}
	defer resp.Body.Close()

	bodyB, err := io.ReadAll(resp.Body)
	if err != nil {
		//log.Fatalln(err)
		return c.SendString(err.Error())
	}
	body := string(bodyB)

	imagePattern := `<img\s+([^>]*\s+)?src="(/)([^"]*)"`
	re := regexp.MustCompile(imagePattern)
	body = re.ReplaceAllString(body, fmt.Sprintf(`<img $1 src="%s$3"`, "/proxy/https://"+u.Host+"/"))

	scriptPattern := `<script\s+([^>]*\s+)?src="(/)([^"]*)"`
	reScript := regexp.MustCompile(scriptPattern)
	body = reScript.ReplaceAllString(body, fmt.Sprintf(`<script $1 script="%s$3"`, "/proxy/https://"+u.Host))

	//body = strings.ReplaceAll(body, "srcset=\"/", "srcset=\"/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "href=\"/", "href=\"/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url('/", "url('/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "url(/", "url(/https://"+u.Host+"/")
	body = strings.ReplaceAll(body, "href=\"https://"+u.Host, "href=\"/https://"+u.Host+"/")

	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	return c.SendString(body)
}
