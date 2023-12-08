package requestmodifiers

import (
	"net/url"
	"regexp"

	tx "github.com/everywall/ladder/proxychain/responsemodifiers"

	"github.com/everywall/ladder/proxychain"
)

const waybackUrl string = "https://web.archive.org/web/"

// RequestWaybackMachine modifies a ProxyChain's URL to request the wayback machine (archive.org) version.
func RequestWaybackMachine() proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.Request.URL.RawQuery = ""
		rURL := preventRecursiveWaybackURLs(chain.Request.URL.String())
		newURLString := waybackUrl + rURL
		newURL, err := url.Parse(newURLString)
		if err != nil {
			return err
		}
		chain.Request.URL = newURL

		// cleanup wayback headers
		script := `["wm-ipp-print", "wm-ipp-base"].forEach(id => { try { document.getElementById(id).remove() } catch{ } })`
		chain.AddOnceResponseModifications(
			tx.InjectScriptAfterDOMContentLoaded(script),
		)

		return nil
	}
}

func preventRecursiveWaybackURLs(url string) string {
	re := regexp.MustCompile(`https:\/\/web\.archive\.org\/web\/\d+\/\*(https?:\/\/.*)`)

	match := re.FindStringSubmatch(url)
	if match != nil {
		return match[1]
	}
	return url
}
