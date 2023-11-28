package requestmodifers

import (
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	http "github.com/Danny-Dasilva/fhttp"
	"golang.org/x/net/proxy"
	"ladder/proxychain"
)

// SpoofJA3fingerprint modifies the TLS client and user agent to spoof a particular JA3 fingerprint
// Some anti-bot WAFs such as cloudflare can fingerprint the fields of the TLS hello packet, and the order in which they appear
// https://web.archive.org/web/20231126224326/https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967/
// https://web.archive.org/web/20231119065253/https://developers.cloudflare.com/bots/concepts/ja3-fingerprint/
func SpoofJA3fingerprint(ja3 string, userAgent string) proxychain.RequestModification {
	//fmt.Println(ja3)
	return func(chain *proxychain.ProxyChain) error {
		// deep copy existing client while modifying http transport
		ja3SpoofClient := &http.Client{
			Transport:     cycletls.NewTransport(ja3, userAgent),
			Timeout:       chain.Client.Timeout,
			CheckRedirect: chain.Client.CheckRedirect,
		}

		chain.SetOnceHTTPClient(ja3SpoofClient)
		return nil
	}
}

// SpoofJA3fingerprintWithProxy modifies the TLS client and user agent to spoof a particular JA3 fingerprint and use a proxy.ContextDialer from the "golang.org/x/net/proxy"
// Some anti-bot WAFs such as cloudflare can fingerprint the fields of the TLS hello packet, and the order in which they appear
// https://web.archive.org/web/20231126224326/https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967/
// https://web.archive.org/web/20231119065253/https://developers.cloudflare.com/bots/concepts/ja3-fingerprint/
func SpoofJA3fingerprintWithProxy(ja3 string, userAgent string, proxy proxy.ContextDialer) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {

		// deep copy existing client while modifying http transport
		ja3SpoofClient := &http.Client{
			Transport:     cycletls.NewTransportWithProxy(ja3, userAgent, proxy),
			Timeout:       chain.Client.Timeout,
			CheckRedirect: chain.Client.CheckRedirect,
		}

		chain.SetOnceHTTPClient(ja3SpoofClient)
		return nil
	}
}
