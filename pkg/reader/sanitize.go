package reader

import (
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// Source elements are never reused in the output tree. Only the tags and
// attributes in sanitize are constructed, including when input is malformed.
func prune(n *html.Node) {
	for child := n.FirstChild; child != nil; {
		next := child.NextSibling
		if discard(child) {
			n.RemoveChild(child)
		} else {
			prune(child)
		}
		child = next
	}
}

func discard(n *html.Node) bool {
	if n.Type == html.CommentNode {
		return true
	}
	if n.Type != html.ElementNode {
		return false
	}
	switch n.Data {
	case "head", "script", "style", "noscript", "template", "nav", "aside", "footer", "form", "input", "button", "select", "textarea", "option", "datalist", "iframe", "frame", "frameset", "object", "embed", "applet", "svg", "math", "canvas", "video", "audio", "source", "track", "link", "meta", "base":
		return true
	}
	if hasAttribute(n, "hidden") || hasAttribute(n, "inert") || strings.EqualFold(strings.TrimSpace(attribute(n, "aria-hidden")), "true") {
		return true
	}
	style := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, attribute(n, "style"))
	if strings.Contains(style, "display:none") || strings.Contains(style, "visibility:hidden") || strings.Contains(style, "content-visibility:hidden") {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(attribute(n, "role"))) {
	case "navigation", "banner", "contentinfo", "complementary", "search", "dialog", "alertdialog":
		return true
	}
	for _, token := range strings.Fields(strings.ToLower(attribute(n, "class") + " " + attribute(n, "id"))) {
		switch token {
		case "nav", "navbar", "navigation", "sidebar", "advertisement", "advertisements", "ad", "ad-banner", "ads", "cookie-banner", "cookie-consent", "newsletter", "newsletter-signup", "share-tools", "social-share", "comments", "comment-list", "related-posts", "related-articles", "sr-only", "visually-hidden", "hidden":
			return true
		}
	}
	return false
}

func sanitize(parent, source *html.Node, base *url.URL) {
	if source.Type == html.TextNode {
		parent.AppendChild(&html.Node{Type: html.TextNode, Data: source.Data})
		return
	}
	if source.Type != html.ElementNode || discard(source) {
		return
	}
	if source.Data == "img" {
		if alt := compact(attribute(source, "alt")); alt != "" {
			parent.AppendChild(&html.Node{Type: html.TextNode, Data: " " + alt + " "})
		}
		return
	}
	tag := source.Data
	switch tag {
	case "p", "h1", "h2", "h3", "h4", "h5", "h6", "a", "strong", "b", "em", "i", "u", "s", "del", "sub", "sup", "code", "pre", "blockquote", "ul", "ol", "li", "dl", "dt", "dd", "br", "hr", "table", "caption", "thead", "tbody", "tfoot", "tr", "th", "td":
	case "div", "section", "article", "main", "header", "figure", "figcaption", "address", "details", "summary":
		tag = "div"
	default:
		// Unknown inline wrappers are unwrapped rather than losing their text.
		for child := source.FirstChild; child != nil; child = child.NextSibling {
			sanitize(parent, child, base)
		}
		return
	}
	clean := &html.Node{Type: html.ElementNode, Data: tag}
	if tag == "a" {
		if href := safeLink(attribute(source, "href"), base); href != "" {
			clean.Attr = []html.Attribute{{Key: "href", Val: href}, {Key: "rel", Val: "noopener noreferrer"}}
		}
	}
	if tag == "ol" {
		if start, err := strconv.Atoi(attribute(source, "start")); err == nil && start >= 1 && start <= 1000000 {
			clean.Attr = []html.Attribute{{Key: "start", Val: strconv.Itoa(start)}}
		}
	}
	if tag == "td" || tag == "th" {
		for _, key := range []string{"colspan", "rowspan"} {
			if value, err := strconv.Atoi(attribute(source, key)); err == nil && value > 1 && value <= 100 {
				clean.Attr = append(clean.Attr, html.Attribute{Key: key, Val: strconv.Itoa(value)})
			}
		}
	}
	parent.AppendChild(clean)
	for child := source.FirstChild; child != nil; child = child.NextSibling {
		sanitize(clean, child, base)
	}
}

func safeLink(raw string, base *url.URL) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return ""
	}
	reference, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(reference)
	resolved.Scheme = strings.ToLower(resolved.Scheme)
	if httpURL(resolved) && resolved.User == nil {
		return resolved.String()
	}
	if resolved.Scheme == "mailto" && resolved.Opaque != "" {
		return resolved.String()
	}
	return ""
}
