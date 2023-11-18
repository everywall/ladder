package rqm // ReQuestModifier

import (
	"ladder/proxychain"
	"net/url"
)

const waybackUrl string = "https://web.archive.org/web/"

// RequestWaybackMachine modifies a ProxyChain's URL to request the wayback machine (archive.org) version.
func RequestWaybackMachine() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		px.Request.URL.RawQuery = ""
		newURLString := waybackUrl + px.Request.URL.String()
		newURL, err := url.Parse(newURLString)
		if err != nil {
			return err
		}
		px.Request.URL = newURL
		return nil
	}
}
