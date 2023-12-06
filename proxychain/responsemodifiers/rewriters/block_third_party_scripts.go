package rewriters

import (
	_ "embed"
	"fmt"
	"log"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// BlockThirdPartyScriptsRewriter implements HTMLTokenRewriter
// and blocks 3rd party JS in script tags by replacing the src attribute value "blocked"
type BlockThirdPartyScriptsRewriter struct {
	baseURL  *url.URL
	proxyURL string // ladder URL, not proxied site URL
}

// NewBlockThirdPartyScriptsRewriter creates a new instance of BlockThirdPartyScriptsRewriter.
// This rewriter will strip out 3rd party JS URLs from script tags.
func NewBlockThirdPartyScriptsRewriter(baseURL *url.URL, proxyURL string) *BlockThirdPartyScriptsRewriter {
	return &BlockThirdPartyScriptsRewriter{
		baseURL:  baseURL,
		proxyURL: proxyURL,
	}
}

func (r *BlockThirdPartyScriptsRewriter) ShouldModify(token *html.Token) bool {
	if token.DataAtom != atom.Script {
		return false
	}

	// check for 3p .js urls in html elements
	for i := range token.Attr {
		attr := token.Attr[i]
		switch {
		case attr.Key != "src":
			continue
		case strings.HasPrefix(attr.Val, "/"):
			return false
		case !strings.HasPrefix(attr.Val, "http"):
			return false
		case strings.HasPrefix(attr.Val, r.proxyURL):
			return false
		case strings.HasPrefix(attr.Val, fmt.Sprintf("%s://%s", r.baseURL.Scheme, r.baseURL.Hostname())):
			return false
		}
	}

	return true
}

func (r *BlockThirdPartyScriptsRewriter) ModifyToken(token *html.Token) (string, string) {
	for i := range token.Attr {
		attr := &token.Attr[i]
		if attr.Key != "src" {
			continue
		}

		if !strings.HasPrefix(attr.Val, "http") {
			continue
		}
		log.Printf("INFO: blocked 3P js: '%s' on '%s'\n", attr.Val, r.baseURL.String())
		attr.Key = "blocked"
	}
	return "", ""
}
