package responsemodifers

import (
	"bytes"
	"encoding/json"
	"github.com/markusmobius/go-trafilatura"
	"io"
	"ladder/proxychain"
	"ladder/proxychain/responsemodifers/api"
)

// APIContent creates an JSON representation of the article and returns it as an API response.
func APIContent() proxychain.ResponseModification {

	return func(chain *proxychain.ProxyChain) error {
		// we set content-type twice here, in case another response modifier
		// tries to forward over the original headers
		chain.Context.Set("content-type", "application/json")
		chain.Response.Header.Set("content-type", "application/json")

		// extract dom contents
		opts := trafilatura.Options{
			IncludeImages: true,
			IncludeLinks:  true,
			// FavorPrecision:     true,
			FallbackCandidates: nil, // TODO: https://github.com/markusmobius/go-trafilatura/blob/main/examples/chained/main.go
			// implement fallbacks from	"github.com/markusmobius/go-domdistiller" and 	"github.com/go-shiori/go-readability"
			OriginalURL: chain.Request.URL,
		}

		result, err := trafilatura.Extract(chain.Response.Body, opts)
		if err != nil {
			chain.Response.Body = api.CreateAPIErrReader(err)
			return nil
		}

		res := api.ExtractResultToAPIResponse(result)
		jsonData, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return err
		}

		chain.Response.Body = io.NopCloser(bytes.NewReader(jsonData))
		return nil
	}

}
