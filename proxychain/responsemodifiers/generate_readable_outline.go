package responsemodifiers

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/everywall/ladder/proxychain"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

//go:embed vendor/generate_readable_outline.html
var templateFS embed.FS

// GenerateReadableOutline creates an reader-friendly distilled representation of the article.
// This is a reliable way of bypassing soft-paywalled articles, where the content is hidden, but still present in the DOM.
func GenerateReadableOutline() proxychain.ResponseModification {
	// get template only once, and resuse for subsequent calls
	f := "vendor/generate_readable_outline.html"
	tmpl, err := template.ParseFS(templateFS, f)
	if err != nil {
		panic(fmt.Errorf("tx.GenerateReadableOutline Error: %s not found", f))
	}

	return func(chain *proxychain.ProxyChain) error {
		// ===========================================================
		// 1. extract dom contents using reading mode algo
		// ===========================================================
		opts := trafilatura.Options{
			IncludeImages:      false,
			IncludeLinks:       true,
			FavorRecall:        true,
			Deduplicate:        true,
			FallbackCandidates: nil, // TODO: https://github.com/markusmobius/go-trafilatura/blob/main/examples/chained/main.go
			// implement fallbacks from	"github.com/markusmobius/go-domdistiller" and 	"github.com/go-shiori/go-readability"
			OriginalURL: chain.Request.URL,
		}

		extract, err := trafilatura.Extract(chain.Response.Body, opts)
		if err != nil {
			return err
		}

		// ============================================================================
		// 2. render generate_readable_outline.html template using metadata from step 1
		// ============================================================================

		// render DOM to string without H1 title
		removeFirstH1(extract.ContentNode)
		// rewrite all links to stay on /outline/ path
		rewriteHrefLinks(extract.ContentNode, chain.Context.BaseURL(), chain.APIPrefix)
		var b bytes.Buffer
		html.Render(&b, extract.ContentNode)
		distilledHTML := b.String()

		siteName := strings.Split(extract.Metadata.Sitename, ";")[0]
		title := strings.Split(extract.Metadata.Title, "|")[0]
		fmtDate := createWikipediaDateLink(extract.Metadata.Date)
		readingTime := formatDuration(estimateReadingTime(extract.ContentText))

		// populate template parameters
		data := map[string]interface{}{
			"Success":     true,
			"Image":       extract.Metadata.Image,
			"Description": extract.Metadata.Description,
			"Sitename":    siteName,
			"Hostname":    extract.Metadata.Hostname,
			"Url":         "/" + chain.Request.URL.String(),
			"Title":       title,
			"Date":        fmtDate,
			"Author":      createDDGFeelingLuckyLinks(extract.Metadata.Author, extract.Metadata.Hostname),
			"Body":        distilledHTML,
			"ReadingTime": readingTime,
		}

		// ============================================================================
		// 3. queue sending the response back to the client by replacing the response body
		// (the response body will be read as a stream in proxychain.Execute() later on.)
		// ============================================================================
		pr, pw := io.Pipe() // pipe io.writer contents into io.reader

		// Use a goroutine for writing to the pipe so we don't deadlock the request
		go func() {
			defer pw.Close()

			err := tmpl.Execute(pw, data) // <- render template
			if err != nil {
				log.Printf("WARN: GenerateReadableOutline template rendering error: %s\n", err)
			}
		}()

		chain.Context.Set("content-type", "text/html")
		chain.Response.Body = pr // <- replace response body reader with our new reader from pipe
		return nil
	}
}

// =============================================
// DOM Rendering helpers
// =============================================

func removeFirstH1(n *html.Node) {
	var recurse func(*html.Node) bool
	recurse = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.DataAtom == atom.H1 {
			return true // Found the first H1, return true to stop
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if recurse(c) {
				n.RemoveChild(c)
				return false // Removed first H1, no need to continue
			}
		}
		return false
	}
	recurse(n)
}

func rewriteHrefLinks(n *html.Node, baseURL string, apiPath string) {
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("GenerateReadableOutline :: rewriteHrefLinks error - %s\n", err)
	}
	apiPath = strings.Trim(apiPath, "/")
	proxyURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	newProxyURL := fmt.Sprintf("%s/%s", proxyURL, apiPath)

	var recurse func(*html.Node) bool
	recurse = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			for i := range n.Attr {
				attr := n.Attr[i]
				if attr.Key != "href" {
					continue
				}
				// rewrite url on a.href: http://localhost:8080/https://example.com -> http://localhost:8080/outline/https://example.com
				attr.Val = strings.Replace(attr.Val, proxyURL, newProxyURL, 1)
				// rewrite relative URLs too
				if strings.HasPrefix(attr.Val, "/") {
					attr.Val = fmt.Sprintf("/%s%s", apiPath, attr.Val)
				}
				n.Attr[i].Val = attr.Val
				log.Println(attr.Val)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			recurse(c)
		}
		return false
	}
	recurse(n)
}

// createWikipediaDateLink takes in a date
// and returns an <a> link pointing to the current events page for that day
func createWikipediaDateLink(t time.Time) string {
	url := fmt.Sprintf("https://en.wikipedia.org/wiki/Portal:Current_events#%s", t.Format("2006_January_2"))
	date := t.Format("January 2, 2006")
	return fmt.Sprintf("<a rel=\"noreferrer\" href=\"%s\">%s</a>", url, date)
}

// createDDGFeelingLuckyLinks takes in comma or semicolon separated terms,
// then turns them into <a> links searching for the term using DuckDuckGo's I'm
// feeling lucky feature. It will redirect the user immediately to the first search result.
func createDDGFeelingLuckyLinks(searchTerms string, siteHostname string) string {

	siteHostname = strings.TrimSpace(siteHostname)
	semiColonSplit := strings.Split(searchTerms, ";")

	var links []string
	for i, termGroup := range semiColonSplit {
		commaSplit := strings.Split(termGroup, ",")
		for _, term := range commaSplit {
			trimmedTerm := strings.TrimSpace(term)
			if trimmedTerm == "" {
				continue
			}

			ddgQuery := fmt.Sprintf(` site:%s intitle:"%s"`, strings.TrimPrefix(siteHostname, "www."), trimmedTerm)

			encodedTerm := `\%s:` + url.QueryEscape(ddgQuery)
			//ddgURL := `https://html.duckduckgo.com/html/?q=` + encodedTerm
			ddgURL := `https://www.duckduckgo.com/?q=` + encodedTerm

			link := fmt.Sprintf("<a rel=\"noreferrer\" href=\"%s\">%s</a>", ddgURL, trimmedTerm)
			links = append(links, link)
		}

		// If it's not the last element in semiColonSplit, add a comma to the last link
		if i < len(semiColonSplit)-1 {
			links[len(links)-1] = links[len(links)-1] + ","
		}
	}

	return strings.Join(links, " ")
}

// estimateReadingTime estimates how long the given text will take to read using the given configuration.
func estimateReadingTime(text string) time.Duration {
	if len(text) == 0 {
		return 0
	}

	// Init options with default values.
	WordsPerMinute := 200
	WordBound := func(b byte) bool {
		return b == ' ' || b == '\n' || b == '\r' || b == '\t'
	}

	words := 0
	start := 0
	end := len(text) - 1

	// Fetch bounds.
	for WordBound(text[start]) {
		start++
	}
	for WordBound(text[end]) {
		end--
	}

	// Calculate the number of words.
	for i := start; i <= end; {
		for i <= end && !WordBound(text[i]) {
			i++
		}

		words++

		for i <= end && WordBound(text[i]) {
			i++
		}
	}

	// Reading time stats.
	minutes := math.Ceil(float64(words) / float64(WordsPerMinute))
	duration := time.Duration(math.Ceil(minutes) * float64(time.Minute))

	return duration

}

func formatDuration(d time.Duration) string {
	// Check if the duration is less than one minute
	if d < time.Minute {
		seconds := int(d.Seconds())
		return fmt.Sprintf("%d seconds", seconds)
	}

	// Convert the duration to minutes
	minutes := int(d.Minutes())

	// Format the string for one or more minutes
	if minutes == 1 {
		return "1 minute"
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}
