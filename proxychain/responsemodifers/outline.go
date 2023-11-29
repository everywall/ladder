package responsemodifers

import (
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"io"
	"ladder/proxychain"
	"strings"
)

// Outline creates an JSON representation of the article
func Outline() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// Use readability
		opts := trafilatura.Options{
			IncludeImages: true,
			IncludeLinks:  true,
			//FavorPrecision:     true,
			FallbackCandidates: nil, // TODO: https://github.com/markusmobius/go-trafilatura/blob/main/examples/chained/main.go
			// implement fallbacks from	"github.com/markusmobius/go-domdistiller" and 	"github.com/go-shiori/go-readability"
			OriginalURL: chain.Request.URL,
		}

		result, err := trafilatura.Extract(chain.Response.Body, opts)
		if err != nil {
			return err
		}

		doc := trafilatura.CreateReadableDocument(result)
		reader := io.NopCloser(strings.NewReader(dom.OuterHTML(doc)))
		chain.Response.Body = reader
		return nil
	}

}
