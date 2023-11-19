package responsemodifers

import (
	"bytes"
	"io"
	"ladder/proxychain"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type HTMLResourceURLRewriter struct {
	src    io.Reader
	buffer *bytes.Buffer 	// buffer to temporarily hold rewritten output for the reader
	proxyURL    *url.URL    // proxyURL is the URL of the proxy, not the upstream URL
}

func NewHTMLResourceURLRewriter(src io.Reader, proxyURL *url.URL) *HTMLResourceURLRewriter {
	return &HTMLResourceURLRewriter{
		src:    	 src,
		buffer: 	 new(bytes.Buffer),
		proxyURL:    proxyURL,
	}
}

func rewriteToken(token *html.Token, baseURL *url.URL) {
	attrsToRewrite := map[string]bool{"href": true, "src": true, "action": true, "srcset": true}
	for i := range token.Attr {
		attr := &token.Attr[i]
		if attrsToRewrite[attr.Key] && strings.HasPrefix(attr.Val, "/") {
			// Make URL absolute
			attr.Val = "/https://" + baseURL.Host + attr.Val
		}
	}
}

func (r *HTMLResourceURLRewriter) Read(p []byte) (int, error) {
	if r.buffer.Len() != 0 {
		return r.buffer.Read(p)
	}

		tokenizer := html.NewTokenizer(r.src)
		for {
			tokenType := tokenizer.Next()
			if tokenType == html.ErrorToken {
				err := tokenizer.Err()
				if err == io.EOF {
					return 0, io.EOF // End of document
				}
				return 0, err // Actual error
			}
			token := tokenizer.Token()
			if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
				rewriteToken(&token, r.url)
			}
			r.buffer.WriteString(token.String())
			if r.buffer.Len() > 0 {
				break
			}
		}
	}
}


// RewriteHTMLResourceURLs updates src/href attributes in HTML content to route through the proxy.
func RewriteHTMLResourceURLs() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		ct := chain.Response.Header.Get("content-type")
		if ct != "text/html" {
			return nil
		}

		// parse dom
		tokenizer := html.NewTokenizer(chain.Body)
		var buffer bytes.Buffer

		// traverse dom and proxify existing src/img resource links
		for {
			tokenType := tokenizer.Next()
			switch tokenType {
			case html.ErrorToken:
				// End of the document, set the new body
				chain.Body = io.ReaderFrom(buffer)
				return nil
			case html.StartTagToken, html.SelfClosingTagToken:
				token := tokenizer.Token()
				// Rewrite the necessary attributes
				token = rewriteToken(token, u)
				buffer.WriteString(token.String())
			case html.TextToken, html.CommentToken, html.DoctypeToken, html.EndTagToken:
				// Write the token to the buffer as is
				buffer.WriteString(tokenizer.Token().String())
			}
		}
	}
}

// rewriteToken rewrites the tokens with URLs to point to the proxy server.
func rewriteToken(token html.Token, u *url.URL) html.Token {
	// Define attributes to rewrite, add more as needed such as "srcset"
	rewriteAttrs := map[string]bool{"href": true, "src": true, "action": true, "srcset": true}

	for i, attr := range token.Attr {
		_, shouldRewrite := rewriteAttrs[attr.Key]
		if shouldRewrite {
			val := attr.Val
			if strings.HasPrefix(val, "/") {
				token.Attr[i].Val = "/https://" + u.Host + val
			}
		}
	}
	return token
}
