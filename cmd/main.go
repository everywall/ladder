package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"paywall-ladder/handlers"
	"runtime"
	"syscall"

	"github.com/elazarl/goproxy"
	"github.com/gofiber/fiber/v2"
)

func main() {

	//os.Setenv("HTTP_PROXY", "http://localhost:3000")

	proxyForkID := uintptr(0)
	if runtime.GOOS == "darwin" {
		_, proxyForkID, _ = syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	} else if runtime.GOOS == "linux" {
		proxyForkID, _, _ = syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	}

	fmt.Println("Proxy fork id", proxyForkID)

	if proxyForkID == 0 {

		app := fiber.New(
			fiber.Config{
				Prefork: false,
			},
		)

		//app.Static("/", "./public")
		app.Get("/", handlers.Form)

		app.Get("/proxy/*", handlers.FetchSite)
		app.Get("debug/*", handlers.Debug)
		app.Get("debug2/*", handlers.Debug2)
		app.Get("/*", handlers.ProxySite)

		port := os.Getenv("PORT")
		if os.Getenv("PORT") == "" {
			port = "2000"
		}
		app.Listen(":" + port)
	} else {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = true
		proxy.OnRequest().DoFunc(
			func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
				req.Header.Set("X-Forwarded-For", "66.249.66.1")
				return req, nil
			})

		proxyport := os.Getenv("PORT_PROXY")
		if os.Getenv("PORT_PROXY") == "" {
			proxyport = "3000"
		}
		log.Println("Proxy listening on port " + proxyport)
		log.Fatal(http.ListenAndServe(":"+proxyport, proxy))
	}

}
