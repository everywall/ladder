package requestmodifers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	http "github.com/bogdanfinn/fhttp"

	/*
		tls_client "github.com/bogdanfinn/tls-client"
		//"net/http"
	*/

	"ladder/proxychain"
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

type CustomDialer struct {
	*net.Dialer
}

func NewCustomDialer(timeout, keepAlive time.Duration) *CustomDialer {
	return &CustomDialer{
		Dialer: &net.Dialer{
			Timeout:   timeout,
			KeepAlive: keepAlive,
		},
	}
}

func (cd *CustomDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = "443"
	}

	resolvedHost, err := resolveWithGoogleDoH(host)
	if err != nil {
		return nil, err
	}
	return cd.Dialer.DialContext(ctx, network, net.JoinHostPort(resolvedHost, port))
}

// ResolveWithGoogleDoH modifies a ProxyChain's client to make the request by resolving the URL
// using Google's DNS over HTTPs service
func ResolveWithGoogleDoH() proxychain.RequestModification {
	///customDialer := NewCustomDialer(10*time.Second, 10*time.Second)
	return func(chain *proxychain.ProxyChain) error {
		/*
			options := []tls_client.HttpClientOption{
				tls_client.WithTimeoutSeconds(30),
				tls_client.WithRandomTLSExtensionOrder(),
				tls_client.WithDialer(*customDialer.Dialer),
				//tls_client.WithClientProfile(profiles.Chrome_105),
			}

			client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
			if err != nil {
				return err
			}

			chain.SetOnceHTTPClient(client)
		*/
		return nil
	}
}
