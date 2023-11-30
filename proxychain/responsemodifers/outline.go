package responsemodifers

import (
	"io"
	"strings"

	//"github.com/go-shiori/dom"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"

	//"golang.org/x/net/html"

	"ladder/proxychain"
	"ladder/proxychain/responsemodifers/api"
)

// APIOutline creates an JSON representation of the article and returns it as an API response.
func APIOutline() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// we set content-type twice here, in case another response modifier
		// tries to forward over the original headers
		chain.Context.Set("content-type", "application/json")
		chain.Response.Header.Set("content-type", "application/json")

		// extract dom contents
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
			chain.Response.Body = api.CreateAPIErrReader(err)
			return nil
		}

		doc := trafilatura.CreateReadableDocument(result)
		reader := io.NopCloser(strings.NewReader(dom.OuterHTML(doc)))
		chain.Response.Body = reader
		return nil
	}
}
