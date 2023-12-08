package responsemodifiers

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"strings"

	"github.com/everywall/ladder/proxychain"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	//"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
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

		// populate template parameters
		data := map[string]interface{}{
			"Success":     true,
			"Image":       extract.Metadata.Image,
			"Description": extract.Metadata.Description,
			"Sitename":    extract.Metadata.Sitename,
			"Hostname":    extract.Metadata.Hostname,
			"Url":         "/" + chain.Request.URL.String(),
			"Title":       extract.Metadata.Title, // todo: modify CreateReadableDocument so we don't have <h1> titles duplicated?
			"Date":        extract.Metadata.Date.String(),
			"Author":      createWikipediaSearchLinks(extract.Metadata.Author),
			//"Author": extract.Metadata.Author,
			"Body": distilledHTML,
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

// createWikipediaSearchLinks takes in comma or semicolon separated terms,
// then turns them into <a> links searching for the term.
func createWikipediaSearchLinks(searchTerms string) string {
	semiColonSplit := strings.Split(searchTerms, ";")

	var links []string
	for i, termGroup := range semiColonSplit {
		commaSplit := strings.Split(termGroup, ",")
		for _, term := range commaSplit {
			trimmedTerm := strings.TrimSpace(term)
			if trimmedTerm == "" {
				continue
			}

			encodedTerm := url.QueryEscape(trimmedTerm)

			wikiURL := fmt.Sprintf("https://en.wikipedia.org/w/index.php?search=%s", encodedTerm)

			link := fmt.Sprintf("<a href=\"%s\">%s</a>", wikiURL, trimmedTerm)
			links = append(links, link)
		}

		// If it's not the last element in semiColonSplit, add a comma to the last link
		if i < len(semiColonSplit)-1 {
			links[len(links)-1] = links[len(links)-1] + ","
		}
	}

	return strings.Join(links, " ")
}
