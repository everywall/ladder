package responsemodifers

import (
	"bytes"
	"io"
	"ladder/proxychain"
	"log"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type HTMLResourceURLRewriter struct {
	proxyURL              *url.URL // proxyURL is the URL of the proxy, not the upstream URL; TODO: implement
	tokenizer             *html.Tokenizer
	currentToken          html.Token
	tokenBuffer           *bytes.Buffer
	currentTokenIndex     int
	currentTokenProcessed bool
}

func NewHTMLResourceURLRewriter(src io.ReadCloser, proxyURL *url.URL) *HTMLResourceURLRewriter {
	log.Println("tokenize")
	return &HTMLResourceURLRewriter{
		tokenizer:         html.NewTokenizer(src),
		currentToken:      html.Token{},
		currentTokenIndex: 0,
		tokenBuffer:       new(bytes.Buffer),
		proxyURL:          proxyURL,
	}
}

func (r *HTMLResourceURLRewriter) Close() error {
	r.tokenBuffer.Reset()
	r.currentToken = html.Token{}
	r.currentTokenIndex = 0
	r.currentTokenProcessed = false
	return nil
}

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
		r.tokenBuffer.Reset()
		r.tokenBuffer.WriteString(r.currentToken.String())
		r.currentTokenProcessed = false
		r.currentTokenIndex = 0
	}

	n, err := r.tokenBuffer.Read(p)

	if err == io.EOF || r.tokenBuffer.Len() == 0 {
		r.currentTokenProcessed = true
		err = nil // Reset error to nil because EOF in this context is expected and not an actual error
	}
	return n, err

}

// RewriteHTMLResourceURLs updates src/href attributes in HTML content to route through the proxy.
func RewriteHTMLResourceURLs() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		log.Println("rhru")
		ct := chain.Response.Header.Get("content-type")
		log.Println(ct)
		if !strings.HasPrefix(ct, "text/html") {
			return nil
		}
		log.Println("rhru2")
		// chain.Response.Body is an unread http.Response.Body
		chain.Response.Body = NewHTMLResourceURLRewriter(chain.Response.Body, chain.Request.URL)
		return nil
	}
}

func rewriteToken(token *html.Token, baseURL *url.URL) {
	log.Println(token.String())
	attrsToRewrite := map[string]bool{"href": true, "src": true, "action": true, "srcset": true}
	for i := range token.Attr {
		attr := &token.Attr[i]
		if attrsToRewrite[attr.Key] {
			attr.Val = "/" + attr.Val
		}
		/*
			if attrsToRewrite[attr.Key] && strings.HasPrefix(attr.Val, "/") {
				// Make URL absolute
				attr.Val = "/https://" + baseURL.Host + attr.Val
			}
		*/
	}
}
