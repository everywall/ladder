package rewriters

import (
	_ "embed"
	"fmt"
	"log"
	"net/url"
	"path"
	"regexp"
	"strings"

	"golang.org/x/net/html/atom"

	"golang.org/x/net/html"
)

var (
	rewriteAttrs        map[string]map[string]bool
	specialRewriteAttrs map[string]map[string]bool
	schemeBlacklist     map[string]bool
)

func init() {
	// define all tag/attributes which might contain URLs
	// to attempt to rewrite to point to proxy instead
	rewriteAttrs = map[string]map[string]bool{
		"img":        {"src": true, "srcset": true, "longdesc": true, "usemap": true},
		"a":          {"href": true},
		"form":       {"action": true},
		"link":       {"href": true, "manifest": true, "icon": true},
		"script":     {"src": true},
		"video":      {"src": true, "poster": true},
		"audio":      {"src": true},
		"iframe":     {"src": true, "longdesc": true},
		"embed":      {"src": true},
		"object":     {"data": true, "codebase": true},
		"source":     {"src": true, "srcset": true},
		"track":      {"src": true},
		"area":       {"href": true},
		"base":       {"href": true},
		"blockquote": {"cite": true},
		"del":        {"cite": true},
		"ins":        {"cite": true},
		"q":          {"cite": true},
		"body":       {"background": true},
		"button":     {"formaction": true},
		"input":      {"src": true, "formaction": true},
		"meta":       {"content": true},
	}

	// might contain URL but requires special handling
	specialRewriteAttrs = map[string]map[string]bool{
		"img":    {"srcset": true},
		"source": {"srcset": true},
		"meta":   {"content": true},
	}

	// define URIs to NOT rewrite
	// for example: don't overwrite <img src="data:image/png;base64;iVBORw...">"
	schemeBlacklist = map[string]bool{
		"data":       true,
		"tel":        true,
		"mailto":     true,
		"file":       true,
		"blob":       true,
		"javascript": true,
		"about":      true,
		"magnet":     true,
		"ws":         true,
		"wss":        true,
		"ftp":        true,
	}
}

// HTMLTokenURLRewriter implements HTMLTokenRewriter
// it rewrites URLs within HTML resources to use a specified proxy URL.
// <img src='/relative_path'> -> <img src='/https://proxiedsite.com/relative_path'>
type HTMLTokenURLRewriter struct {
	baseURL  *url.URL
	proxyURL string // ladder URL, not proxied site URL
}

// NewHTMLTokenURLRewriter creates a new instance of HTMLResourceURLRewriter.
// It initializes the tokenizer with the provided source and sets the proxy URL.
func NewHTMLTokenURLRewriter(baseURL *url.URL, proxyURL string) *HTMLTokenURLRewriter {
	return &HTMLTokenURLRewriter{
		baseURL:  baseURL,
		proxyURL: proxyURL,
	}
}

func (r *HTMLTokenURLRewriter) ShouldModify(token *html.Token) bool {
	//fmt.Printf("touch token: %s\n", token.String())
	attrLen := len(token.Attr)
	if attrLen == 0 {
		return false
	}

	if token.Type == html.StartTagToken {
		return true
	}

	if token.Type == html.SelfClosingTagToken {
		return true
	}
	return false
}

func (r *HTMLTokenURLRewriter) ModifyToken(token *html.Token) (string, string) {
	for i := range token.Attr {
		attr := &token.Attr[i]
		switch {
		// don't touch tag/attributes that don't contain URIs
		case !rewriteAttrs[token.Data][attr.Key]:
			continue
		// don't touch attributes with special URIs (like data:)
		case schemeBlacklist[strings.Split(attr.Val, ":")[0]]:
			continue
		// don't double-overwrite the url
		case strings.HasPrefix(attr.Val, r.proxyURL):
			continue
		case strings.HasPrefix(attr.Val, "/http://"):
			continue
		case strings.HasPrefix(attr.Val, "/https://"):
			continue
		// handle special rewrites
		case specialRewriteAttrs[token.Data][attr.Key]:
			r.handleSpecialAttr(token, attr, r.baseURL)
			continue
		default:
			// rewrite url
			handleURLPart(attr, r.baseURL)
		}
	}
	return "", ""
}

// dispatcher for ModifyURL based on URI type
func handleURLPart(attr *html.Attribute, baseURL *url.URL) {
	switch {
	case strings.HasPrefix(attr.Val, "//"):
		handleProtocolRelativePath(attr, baseURL)
	case strings.HasPrefix(attr.Val, "/"):
		handleRootRelativePath(attr, baseURL)
	case strings.HasPrefix(attr.Val, "https://"):
		handleAbsolutePath(attr, baseURL)
	case strings.HasPrefix(attr.Val, "http://"):
		handleAbsolutePath(attr, baseURL)
	default:
		handleDocumentRelativePath(attr, baseURL)
	}
}

// Protocol-relative URLs: These start with "//" and will use the same protocol (http or https) as the current page.
func handleProtocolRelativePath(attr *html.Attribute, baseURL *url.URL) {
	attr.Val = strings.TrimPrefix(attr.Val, "/")
	handleRootRelativePath(attr, baseURL)
	log.Printf("proto rel url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

// Root-relative URLs: These are relative to the root path and start with a "/".
func handleRootRelativePath(attr *html.Attribute, baseURL *url.URL) {
	// doublecheck this is a valid relative URL
	log.Printf("PROCESSING: key: %s val: %s\n", attr.Key, attr.Val)
	_, err := url.Parse(fmt.Sprintf("http://localhost.com%s", attr.Val))
	if err != nil {
		log.Println(err)
		return
	}

	// log.Printf("BASEURL patch:  %s\n", baseURL)

	attr.Val = fmt.Sprintf(
		"%s://%s/%s",
		baseURL.Scheme,
		baseURL.Host,
		strings.TrimPrefix(attr.Val, "/"),
	)
	attr.Val = escape(attr.Val)
	attr.Val = fmt.Sprintf("/%s", attr.Val)

	log.Printf("root rel url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

// Document-relative URLs: These are relative to the current document's path and don't start with a "/".
func handleDocumentRelativePath(attr *html.Attribute, baseURL *url.URL) {
	log.Printf("PROCESSING: key: %s val: %s\n", attr.Key, attr.Val)
	if strings.HasPrefix(attr.Val, "#") {
		return
	}
	relativePath := path.Join(strings.Trim(baseURL.RawPath, "/"), strings.Trim(attr.Val, "/"))
	attr.Val = fmt.Sprintf(
		"%s://%s/%s",
		baseURL.Scheme,
		strings.Trim(baseURL.Host, "/"),
		relativePath,
	)
	attr.Val = escape(attr.Val)
	attr.Val = fmt.Sprintf("/%s", attr.Val)
	log.Printf("doc rel url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

// full URIs beginning with https?://proxiedsite.com
func handleAbsolutePath(attr *html.Attribute, baseURL *url.URL) {
	// check if valid URL
	log.Printf("PROCESSING: key: %s val: %s\n", attr.Key, attr.Val)
	u, err := url.Parse(attr.Val)
	if err != nil {
		return
	}
	if !(u.Scheme == "http" || u.Scheme == "https") {
		return
	}
	attr.Val = fmt.Sprintf("/%s", escape(strings.TrimPrefix(attr.Val, "/")))
	log.Printf("abs url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

// handle edge cases for special attributes
func (r *HTMLTokenURLRewriter) handleSpecialAttr(token *html.Token, attr *html.Attribute, baseURL *url.URL) {
	switch {
	// srcset attribute doesn't contain a single URL but a comma-separated list of URLs, each potentially followed by a space and a descriptor (like a width, pixel density, or other conditions).
	case token.DataAtom == atom.Img && attr.Key == "srcset":
		handleSrcSet(attr, baseURL)
	case token.DataAtom == atom.Source && attr.Key == "srcset":
		handleSrcSet(attr, baseURL)
	// meta with http-equiv="refresh": The content attribute of a meta tag, when used for a refresh directive, contains a time interval followed by a URL, like content="5;url=http://example.com/".
	case token.DataAtom == atom.Meta && attr.Key == "content" && regexp.MustCompile(`^\d+;url=`).MatchString(attr.Val):
		handleMetaRefresh(attr, baseURL)
	default:
		break
	}
}

func handleMetaRefresh(attr *html.Attribute, baseURL *url.URL) {
	sec := strings.Split(attr.Val, ";url=")[0]
	url := strings.Split(attr.Val, ";url=")[1]
	f := &html.Attribute{Val: url, Key: "src"}
	handleURLPart(f, baseURL)
	attr.Val = fmt.Sprintf("%s;url=%s", sec, f.Val)
}

func handleSrcSet(attr *html.Attribute, baseURL *url.URL) {
	var srcSetBuilder strings.Builder
	srcSetItems := strings.Split(attr.Val, ",")

	for i, srcItem := range srcSetItems {
		srcParts := strings.Fields(srcItem)

		if len(srcParts) == 0 {
			continue
		}

		f := &html.Attribute{Val: srcParts[0], Key: "src"}
		handleURLPart(f, baseURL)

		if i > 0 {
			srcSetBuilder.WriteString(", ")
		}

		srcSetBuilder.WriteString(f.Val)
		if len(srcParts) > 1 {
			srcSetBuilder.WriteString(" ")
			srcSetBuilder.WriteString(strings.Join(srcParts[1:], " "))
		}
	}

	attr.Val = srcSetBuilder.String()
}

func escape(str string) string {
	//return str
	return strings.ReplaceAll(url.PathEscape(str), "%2F", "/")
}
