package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	url := "http://127.0.0.1:" + port + "/healthz"
	if len(os.Args) > 1 && os.Args[1] != "" {
		url = os.Args[1]
	}

	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		os.Exit(1)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		os.Exit(1)
	}
}
