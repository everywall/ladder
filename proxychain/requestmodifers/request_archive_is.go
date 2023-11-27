package requestmodifers

import (
	"fmt"
	"net/url"

	"ladder/proxychain"
)

const archivistUrl string = "https://archive.is/latest"

// RequestArchiveIs modifies a ProxyChain's URL to request an archived version from archive.is
func RequestArchiveIs() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Request.URL.RawQuery = ""
		newURL, err := url.Parse(fmt.Sprintf("%s/%s", archivistUrl, chain.Request.URL.String()))
		if err != nil {
			return err
		}

		// archivist seems to sabotage requests from cloudflare's DNS
		// bypass this just in case
		chain.AddOnceRequestModifications(ResolveWithGoogleDoH())

		chain.Request.URL = newURL
		return nil
	}
}
