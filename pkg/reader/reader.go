// Package reader extracts readable, inert article content from HTML without I/O.
package reader

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// Document contains source-derived article content in three presentation formats.
type Document struct {
	Title    string
	Byline   string
	Text     string
	Markdown string
	HTML     string
}

// Extract reads HTML using baseURL as the effective document URL. baseURL must
// be an absolute HTTP(S) URL. The result never contains resource-loading markup;
// image descriptions become text. Extract does not fetch links or run scripts.
func Extract(body []byte, baseURL string) (Document, error) {
	base, err := url.Parse(baseURL)
	if err != nil || base == nil || !httpURL(base) || strings.IndexFunc(baseURL, unicode.IsControl) >= 0 {
		return Document{}, fmt.Errorf("reader: base URL must be an absolute HTTP(S) URL")
	}
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Document{}, fmt.Errorf("reader: parse HTML: %w", err)
	}
	doc := Document{
		Title:  metadata(root, "og:title", "twitter:title"),
		Byline: metadata(root, "author", "dc.creator", "dcterms.creator"),
	}
	if doc.Title == "" {
		if title := find(root, func(n *html.Node) bool { return n.Data == "title" }); title != nil {
			doc.Title = compact(nodeText(title))
		}
	}
	prune(root)
	content := selectContent(root)
	if doc.Title == "" {
		if heading := find(content, func(n *html.Node) bool { return n.Data == "h1" }); heading != nil {
			doc.Title = compact(nodeText(heading))
		}
	}
	if doc.Byline == "" {
		if author := find(content, isByline); author != nil {
			doc.Byline = compact(nodeText(author))
		}
	}
	clean := &html.Node{Type: html.ElementNode, Data: "div"}
	for child := content.FirstChild; child != nil; child = child.NextSibling {
		sanitize(clean, child, base)
	}
	var output bytes.Buffer
	for child := clean.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&output, child); err != nil {
			return Document{}, fmt.Errorf("reader: render HTML: %w", err)
		}
	}
	doc.HTML = strings.TrimSpace(output.String())
	doc.Text = renderDocument(clean, false)
	doc.Markdown = renderDocument(clean, true)
	return doc, nil
}

func httpURL(u *url.URL) bool {
	return (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) && u.Hostname() != "" && u.Opaque == ""
}

func attribute(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasAttribute(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return true
		}
	}
	return false
}

func compact(s string) string { return strings.Join(strings.Fields(s), " ") }

func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(nodeText(child))
	}
	return b.String()
}

func find(n *html.Node, match func(*html.Node) bool) *html.Node {
	if n.Type == html.ElementNode && match(n) {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if result := find(child, match); result != nil {
			return result
		}
	}
	return nil
}

func metadata(root *html.Node, names ...string) string {
	for _, name := range names {
		meta := find(root, func(n *html.Node) bool {
			return n.Data == "meta" && (strings.EqualFold(attribute(n, "name"), name) || strings.EqualFold(attribute(n, "property"), name)) && compact(attribute(n, "content")) != ""
		})
		if meta != nil {
			return compact(attribute(meta, "content"))
		}
	}
	return ""
}

func isByline(n *html.Node) bool {
	for _, key := range []string{"itemprop", "rel", "class"} {
		for _, token := range strings.Fields(strings.ToLower(attribute(n, key))) {
			if token == "author" || token == "byline" || token == "article-author" || token == "entry-author" {
				return true
			}
		}
	}
	return false
}

func selectContent(root *html.Node) *html.Node {
	var article, main *html.Node
	var articleScore, mainScore int
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "article" {
				if score := contentScore(n); score > articleScore {
					article, articleScore = n, score
				}
			} else if n.Data == "main" || strings.EqualFold(attribute(n, "role"), "main") {
				if score := contentScore(n); score > mainScore {
					main, mainScore = n, score
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	// A substantial article is usually more precise than its enclosing main.
	// A tiny teaser article should not displace the actual page's main content.
	if article != nil && (main == nil || articleScore*2 >= mainScore) {
		return article
	}
	if main != nil {
		return main
	}
	if article != nil {
		return article
	}
	if body := find(root, func(n *html.Node) bool { return n.Data == "body" }); body != nil {
		return body
	}
	return root
}

func contentScore(n *html.Node) int {
	if n.Type == html.TextNode {
		return len([]rune(compact(n.Data)))
	}
	score := 0
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		score += contentScore(child)
	}
	if n.Data == "a" {
		return score / 4
	}
	if n.Data == "p" && score > 40 {
		score += 60
	}
	return score
}
