package requestmodifers

import (
	"context"
	"encoding/json"
	"fmt"
	"ladder/proxychain"
	"net"
	"net/http"
	"time"
)

// resolveWithGoogleDoH resolves DNS using Google's DNS-over-HTTPS
func resolveWithGoogleDoH(host string) (string, error) {
	url := "https://dns.google/resolve?name=" + host + "&type=A"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Answer []struct {
			Data string `json:"data"`
		} `json:"Answer"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	// Get the first A record
	if len(result.Answer) > 0 {
		return result.Answer[0].Data, nil
	}
	return "", fmt.Errorf("no DoH DNS record found for %s", host)
}

// ResolveWithGoogleDoH modifies a ProxyChain's client to make the request but resolve the URL
// using Google's DNS over HTTPs service
func ResolveWithGoogleDoH() proxychain.RequestModification {
	return func(px *proxychain.ProxyChain) error {
		client := &http.Client{
			Timeout: px.Client.Timeout,
		}

		dialer := &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		}

		customDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				// If the addr doesn't include a port, determine it based on the URL scheme
				if px.Request.URL.Scheme == "https" {
					port = "443"
				} else {
					port = "80"
				}
				host = addr // assume the entire addr is the host
			}

			resolvedHost, err := resolveWithGoogleDoH(host)
			if err != nil {
				return nil, err
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(resolvedHost, port))
		}

		patchedTransportWithDoH := &http.Transport{
			DialContext: customDialContext,
		}

		client.Transport = patchedTransportWithDoH
		px.Client = client // Assign the modified client to the ProxyChain
		return nil
	}
}
