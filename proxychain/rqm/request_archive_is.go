package rqm

import (
	"ladder/proxychain"
	"net/url"
)

const archivistUrl string = "https://archive.is/latest/"

// RequestArchiveIs modifies a ProxyChain's URL to request an archived version from archive.is
func RequestArchiveIs() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.URL.RawQuery = ""
		newURLString := archivistUrl + px.Request.URL.String()
		newURL, err := url.Parse(newURLString)
		if err != nil {
			return err
		}

		// archivist seems to sabotage requests from cloudflare's DNS
		// bypass this just in case
		px.AddRequestModifications(ResolveWithGoogleDoH())

		px.Request.URL = newURL
		return nil
	}
}
