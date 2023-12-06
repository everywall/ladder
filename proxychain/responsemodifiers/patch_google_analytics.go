package responsemodifiers

import (
	_ "embed"
	"io"
	"strings"

	"ladder/proxychain"
)

//go:embed vendor/patch_google_analytics.js
var gaPatch string

// PatchGoogleAnalytics replaces any request to google analytics with a no-op stub function.
// Some sites will not display content until GA is loaded, so we fake one instead.
// Credit to Raymond Hill @ github.com/gorhill/uBlock
func PatchGoogleAnalytics() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {

		// preflight check
		isGADomain := chain.Request.URL.Host == "www.google-analytics.com" || chain.Request.URL.Host == "google-analytics.com"
		isGAPath := strings.HasSuffix(chain.Request.URL.Path, "analytics.js")
		if !(isGADomain || isGAPath) {
			return nil
		}

		// send modified js payload to client containing
		// stub functions from patch_google_analytics.js
		gaPatchReader := io.NopCloser(strings.NewReader(gaPatch))
		chain.Response.Body = gaPatchReader
		chain.Context.Set("content-type", "text/javascript")
		return nil
	}
}
