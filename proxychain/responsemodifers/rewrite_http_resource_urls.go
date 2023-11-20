package responsemodifers

import (
	"bytes"
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
		"src":  true,
		"href": true,
		/*
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
		*/
	}
}

// HTMLResourceURLRewriter is a struct that rewrites URLs within HTML resources to use a specified proxy URL.
// It uses an HTML tokenizer to process HTML content and rewrites URLs in src/href attributes.
// <img src='/relative_path'> -> <img src='/https://proxiedsite.com/relative_path'>
type HTMLResourceURLRewriter struct {
	baseURL               string // eg: https://proxiedsite.com  (note, no trailing '/')
	tokenizer             *html.Tokenizer
	currentToken          html.Token
	tokenBuffer           *bytes.Buffer
	currentTokenIndex     int
	currentTokenProcessed bool
}

// NewHTMLResourceURLRewriter creates a new instance of HTMLResourceURLRewriter.
// It initializes the tokenizer with the provided source and sets the proxy URL.
func NewHTMLResourceURLRewriter(src io.ReadCloser, baseURL string) *HTMLResourceURLRewriter {
	return &HTMLResourceURLRewriter{
		tokenizer:         html.NewTokenizer(src),
		currentToken:      html.Token{},
		currentTokenIndex: 0,
		tokenBuffer:       new(bytes.Buffer),
		baseURL:           baseURL,
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
			patchResourceURL(&r.currentToken, r.baseURL)
		}

		r.tokenBuffer.Reset()
		r.tokenBuffer.WriteString(r.currentToken.String())
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

func patchResourceURL(token *html.Token, baseURL string) {
	for i := range token.Attr {
		attr := &token.Attr[i]
		// dont touch attributes except for the ones we defined
		_, exists := AttributesToRewrite[attr.Key]
		if !exists {
			continue
		}

		isRelativePath := strings.HasPrefix(attr.Val, "/")
		//log.Printf("PRE '%s'='%s'", attr.Key, attr.Val)

		// double check if attribute is valid http URL before modifying
		if isRelativePath {
			_, err := url.Parse(fmt.Sprintf("http://localhost%s", attr.Val))
			if err != nil {
				return
			}
		} else {
			u, err := url.Parse(attr.Val)
			if err != nil {
				return
			}
			if !(u.Scheme == "http" || u.Scheme == "https") {
				return
			}
		}

		// patch relative paths
		// <img src="/favicon.png"> -> <img src="/http://images.cdn.proxiedsite.com/favicon.png">
		if isRelativePath {
			log.Printf("BASEURL patch:  %s\n", baseURL)

			attr.Val = fmt.Sprintf(
				"/%s/%s",
				baseURL,
				//url.QueryEscape(
				strings.TrimPrefix(attr.Val, "/"),
				//),
			)

			log.Printf("url rewritten-> '%s'='%s'", attr.Key, attr.Val)
			continue
		}

		// patch absolute paths to relative path pointing to ladder proxy
		// <img src="http://images.cdn.proxiedsite.com/favicon.png"> -> <img src="/http://images.cdn.proxiedsite.com/favicon.png">

		//log.Printf("abolute patch:  %s\n", attr.Val)
		attr.Val = fmt.Sprintf(
			"/%s",
			//url.QueryEscape(attr.Val),
			//url.QueryEscape(
			strings.TrimPrefix(attr.Val, "/"),
			//),
			//attr.Val,
		)
		log.Printf("url rewritten-> '%s'='%s'", attr.Key, attr.Val)
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

		// should be site being requested to proxy
		baseUrl := fmt.Sprintf("%s://%s", chain.Request.URL.Scheme, chain.Request.URL.Host)
		/*
			log.Println("--------------------")
			log.Println(baseUrl)
			log.Println("--------------------")
		*/

		chain.Response.Body = NewHTMLResourceURLRewriter(chain.Response.Body, baseUrl)
		return nil
	}
}
