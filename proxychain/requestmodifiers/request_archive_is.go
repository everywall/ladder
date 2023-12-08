package requestmodifiers

import (
	"fmt"
	"net/url"
	"regexp"

	tx "github.com/everywall/ladder/proxychain/responsemodifiers"

	"github.com/everywall/ladder/proxychain"
)

const archivistUrl string = "https://archive.is/latest"

// RequestArchiveIs modifies a ProxyChain's URL to request an archived version from archive.is
func RequestArchiveIs() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		rURL := preventRecursiveArchivistURLs(chain.Request.URL.String())
		chain.Request.URL.RawQuery = ""
		newURL, err := url.Parse(fmt.Sprintf("%s/%s", archivistUrl, rURL))
		if err != nil {
			return err
		}

		// archivist seems to sabotage requests from cloudflare's DNS
		// bypass this just in case
		chain.AddOnceRequestModifications(ResolveWithGoogleDoH())

		chain.Request.URL = newURL

		// cleanup archivst headers
		script := `[...document.querySelector("body > center").childNodes].filter(e => e.id != "SOLID").forEach(e => e.remove())`
		chain.AddOnceResponseModifications(
			tx.InjectScriptAfterDOMContentLoaded(script),
		)
		return nil
	}
}

// https://archive.is/20200421201055/https://rt.live/ -> http://rt.live/
func preventRecursiveArchivistURLs(url string) string {
	re := regexp.MustCompile(`https?:\/\/archive\.is\/\d+\/(https?:\/\/.*)`)
	match := re.FindStringSubmatch(url)
	if match != nil {
		return match[1]
	}
	return url
}
