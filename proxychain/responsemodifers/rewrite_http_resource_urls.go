package responsemodifers

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"ladder/proxychain"
	"log"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Define list of HTML attributes to try to rewrite
var AttributesToRewrite map[string]bool

func init() {
	AttributesToRewrite = map[string]bool{
		"src":         true,
		"href":        true,
		"action":      true,
		"srcset":      true,
		"poster":      true,
		"data":        true,
		"cite":        true,
		"formaction":  true,
		"background":  true,
		"usemap":      true,
		"longdesc":    true,
		"manifest":    true,
		"archive":     true,
		"codebase":    true,
		"icon":        true,
		"pluginspage": true,
	}
}

// HTMLResourceURLRewriter is a struct that rewrites URLs within HTML resources to use a specified proxy URL.
// It uses an HTML tokenizer to process HTML content and rewrites URLs in src/href attributes.
// <img src='/relative_path'> -> <img src='/https://proxiedsite.com/relative_path'>
type HTMLResourceURLRewriter struct {
	baseURL               *url.URL
	tokenizer             *html.Tokenizer
	currentToken          html.Token
	tokenBuffer           *bytes.Buffer
	currentTokenIndex     int
	currentTokenProcessed bool
	proxyURL              string // ladder URL, not proxied site URL
}

// NewHTMLResourceURLRewriter creates a new instance of HTMLResourceURLRewriter.
// It initializes the tokenizer with the provided source and sets the proxy URL.
func NewHTMLResourceURLRewriter(src io.ReadCloser, baseURL *url.URL, proxyURL string) *HTMLResourceURLRewriter {
	return &HTMLResourceURLRewriter{
		tokenizer:         html.NewTokenizer(src),
		currentToken:      html.Token{},
		currentTokenIndex: 0,
		tokenBuffer:       new(bytes.Buffer),
		baseURL:           baseURL,
		proxyURL:          proxyURL,
	}
}

// Close resets the internal state of HTMLResourceURLRewriter, clearing buffers and token data.
func (r *HTMLResourceURLRewriter) Close() error {
	r.tokenBuffer.Reset()
	r.currentToken = html.Token{}
	r.currentTokenIndex = 0
	r.currentTokenProcessed = false
	return nil
}

// Read processes the HTML content, rewriting URLs and managing the state of tokens.
// It reads HTML content, token by token, rewriting URLs to route through the specified proxy.
func (r *HTMLResourceURLRewriter) Read(p []byte) (int, error) {

	if r.currentToken.Data == "" || r.currentTokenProcessed {
		tokenType := r.tokenizer.Next()

		// done reading html, close out reader
		if tokenType == html.ErrorToken {
			if r.tokenizer.Err() == io.EOF {
				return 0, io.EOF
			}
			return 0, r.tokenizer.Err()
		}

		// flush the current token into an internal buffer
		// to handle fragmented tokens
		r.currentToken = r.tokenizer.Token()

		// patch tokens with URLs
		isTokenWithAttribute := r.currentToken.Type == html.StartTagToken || r.currentToken.Type == html.SelfClosingTagToken
		if isTokenWithAttribute {
			patchResourceURL(&r.currentToken, r.baseURL, r.proxyURL)
		}

		r.tokenBuffer.Reset()
		r.tokenBuffer.WriteString(r.currentToken.String())

		// inject <script> right after <head>
		isHeadToken := (r.currentToken.Type == html.StartTagToken || r.currentToken.Type == html.SelfClosingTagToken) && r.currentToken.Data == "head"
		if isHeadToken {
			injectScript(r.tokenBuffer, rewriteJSResourceUrlsScript)
		}

		r.currentTokenProcessed = false
		r.currentTokenIndex = 0
	}

	n, err := r.tokenBuffer.Read(p)
	if err == io.EOF || r.tokenBuffer.Len() == 0 {
		r.currentTokenProcessed = true
		err = nil // EOF in this context is expected and not an actual error
	}
	return n, err
}

// fetch("/relative_script.js") -> fetch("http://localhost:8080/relative_script.js")
//
//go:embed rewrite_js_resource_urls.js
var rewriteJSResourceUrlsScript string

func injectScript(tokenBuffer *bytes.Buffer, script string) {
	tokenBuffer.WriteString(
		fmt.Sprintf("\n<script>\n%s\n</script>\n", script),
	)
}

// Root-relative URLs: These are relative to the root path and start with a "/".
func handleRootRelativePath(attr *html.Attribute, baseURL *url.URL) {
	// doublecheck this is a valid relative URL
	_, err := url.Parse(fmt.Sprintf("http://localhost.com%s", attr.Val))
	if err != nil {
		return
	}

	//log.Printf("BASEURL patch:  %s\n", baseURL)

	attr.Val = fmt.Sprintf(
		"/%s://%s/%s",
		baseURL.Scheme,
		baseURL.Host,
		strings.TrimPrefix(attr.Val, "/"),
	)
	attr.Val = url.QueryEscape(attr.Val)
	attr.Val = fmt.Sprintf("/%s", attr.Val)

	log.Printf("root rel url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

// Document-relative URLs: These are relative to the current document's path and don't start with a "/".
func handleDocumentRelativePath(attr *html.Attribute, baseURL *url.URL) {
	attr.Val = fmt.Sprintf(
		"%s://%s/%s%s",
		baseURL.Scheme,
		strings.Trim(baseURL.Host, "/"),
		strings.Trim(baseURL.RawPath, "/"),
		strings.Trim(attr.Val, "/"),
	)
	attr.Val = url.QueryEscape(attr.Val)
	attr.Val = fmt.Sprintf("/%s", attr.Val)
	log.Printf("doc rel url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

// Protocol-relative URLs: These start with "//" and will use the same protocol (http or https) as the current page.
func handleProtocolRelativePath(attr *html.Attribute, baseURL *url.URL) {
	attr.Val = strings.TrimPrefix(attr.Val, "/")
	handleRootRelativePath(attr, baseURL)
	log.Printf("proto rel url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

func handleAbsolutePath(attr *html.Attribute, baseURL *url.URL) {
	// check if valid URL
	u, err := url.Parse(attr.Val)
	if err != nil {
		return
	}
	if !(u.Scheme == "http" || u.Scheme == "https") {
		return
	}
	attr.Val = fmt.Sprintf(
		"/%s",
		url.QueryEscape(
			strings.TrimPrefix(attr.Val, "/"),
		),
	)
	log.Printf("abs url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

func handleSrcSet(attr *html.Attribute, baseURL *url.URL) {
	for i, src := range strings.Split(attr.Val, ",") {
		src = strings.Trim(src, " ")
		for j, s := range strings.Split(src, " ") {
			s = strings.Trim(s, " ")
			if j == 0 {
				f := &html.Attribute{Val: s, Key: attr.Key}
				switch {
				case strings.HasPrefix(s, "//"):
					handleProtocolRelativePath(f, baseURL)
				case strings.HasPrefix(s, "/"):
					handleRootRelativePath(f, baseURL)
				case strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://"):
					handleAbsolutePath(f, baseURL)
				default:
					handleDocumentRelativePath(f, baseURL)
				}
				s = f.Val
			}
			if i == 0 && j == 0 {
				attr.Val = s
				continue
			}
			attr.Val = fmt.Sprintf("%s %s", attr.Val, s)
		}
		attr.Val = fmt.Sprintf("%s,", attr.Val)
	}
	attr.Val = strings.TrimSuffix(attr.Val, ",")

	log.Printf("srcset url rewritten-> '%s'='%s'", attr.Key, attr.Val)
}

func patchResourceURL(token *html.Token, baseURL *url.URL, proxyURL string) {
	for i := range token.Attr {
		attr := &token.Attr[i]

		switch {
		// don't touch attributes except for the ones we defined
		case !AttributesToRewrite[attr.Key]:
			continue
		// don't double-overwrite the url
		case strings.HasPrefix(attr.Val, proxyURL):
			continue
		case attr.Key == "srcset":
			handleSrcSet(attr, baseURL)
			continue
		case strings.HasPrefix(attr.Val, "//"):
			handleProtocolRelativePath(attr, baseURL)
			continue
		case strings.HasPrefix(attr.Val, "/"):
			handleRootRelativePath(attr, baseURL)
			continue
		case strings.HasPrefix(attr.Val, "https://") || strings.HasPrefix(attr.Val, "http://"):
			handleAbsolutePath(attr, baseURL)
			continue
		default:
			handleDocumentRelativePath(attr, baseURL)
			continue
		}

	}
}

// RewriteHTMLResourceURLs modifies HTTP responses
// to rewrite URLs attributes in HTML content (such as src, href)
//   - `<img src='/relative_path'>` -> `<img src='/https://proxiedsite.com/relative_path'>`
//   - This function is designed to allow the proxified page
//     to still be browsible by routing all resource URLs through the proxy.
//
// ---
//
//   - It works by replacing the io.ReadCloser of the http.Response.Body
//     with another io.ReaderCloser (HTMLResourceRewriter) that wraps the first one.
//
//   - This process can be done multiple times, so that the response will
//     be streamed and modified through each pass without buffering the entire response in memory.
//
//   - HTMLResourceRewriter reads the http.Response.Body stream,
//     parsing each HTML token one at a time and replacing attribute tags.
//
//   - When ProxyChain.Execute() is called, the response body will be read from the server
//     and pulled through each ResponseModification which wraps the ProxyChain.Response.Body
//     without ever buffering the entire HTTP response in memory.
func RewriteHTMLResourceURLs() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// return early if it's not HTML
		ct := chain.Response.Header.Get("content-type")
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}
		originalURI := chain.Context.Request().URI()
		proxyURL := fmt.Sprintf("%s://%s", originalURI.Scheme(), originalURI.Host())

		chain.Response.Body = NewHTMLResourceURLRewriter(chain.Response.Body, chain.Request.URL, proxyURL)
		return nil
	}
}
