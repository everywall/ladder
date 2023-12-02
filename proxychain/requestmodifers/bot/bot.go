package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"math/bits"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type Bot interface {
	UpdatePool() error
	GetRandomIdentity() string
}

type bot struct {
	UserAgent   string
	Fingerprint string
	IPPool      botPool
}

type botPool struct {
	Timestamp string      `json:"creationTime"`
	Prefixes  []botPrefix `json:"prefixes"`
}

type botPrefix struct {
	IPv6 string `json:"ipv6Prefix,omitempty"`
	IPv4 string `json:"ipv4Prefix,omitempty"`
}

// TODO: move pointers around, not global variables
var GoogleBot = bot{
	UserAgent: "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; Googlebot/2.1; http://www.google.com/bot.html) Chrome/79.0.3945.120 Safari/537.36",

	// https://github.com/trisulnsm/trisul-scripts/blob/master/lua/frontend_scripts/reassembly/ja3/prints/ja3fingerprint.json
	Fingerprint: "769,49195-49199-49196-49200-52393-52392-52244-52243-49161-49171-49162-49172-156-157-47-53-10,65281-0-23-35-13-5-18-16-11-10-21,29-23-24,0",

	IPPool: botPool{
		Timestamp: "2023-11-28T23:00:56.000000",
		Prefixes: []botPrefix{
			{
				IPv4: "34.100.182.96/28",
			},
		},
	},
}

var BingBot = bot{
	UserAgent: "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm) Chrome/79.0.3945.120 Safari/537.36",
	IPPool: botPool{
		Timestamp: "2023-03-08T10:00:00.121331",
		Prefixes: []botPrefix{
			{
				IPv4: "207.46.13.0/24",
			},
		},
	},
}

func (b *bot) UpdatePool(url string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update googlebot IP pool: status code %s", resp.Status)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &b.IPPool)

	return err
}

func (b *bot) GetRandomIP() string {
	count := len(b.IPPool.Prefixes)

	var prefix botPrefix

	if count == 1 {
		prefix = b.IPPool.Prefixes[0]
	} else {
		idx := rand.Intn(count)
		prefix = b.IPPool.Prefixes[idx]
	}

	if prefix.IPv4 != "" {
		ip, err := randomIPFromSubnet(prefix.IPv4)
		if err == nil {
			return ip.String()
		}
	}

	if prefix.IPv6 != "" {
		ip, err := randomIPFromSubnet(prefix.IPv6)
		if err == nil {
			return ip.String()
		}
	}

	// fallback to default IP which is known to work
	ip, _ := randomIPFromSubnet(b.IPPool.Prefixes[0].IPv4)

	return ip.String()
}

func randomIPFromSubnet(c string) (net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(c)
	if err != nil {
		return nil, err
	}

	// int representation of byte mask
	mask := big.NewInt(0).SetBytes(ipnet.Mask).Uint64()

	// how many unset bits there are at the end of the mask
	offset := bits.TrailingZeros8(byte(0) ^ byte(mask))

	// total number of ips available in the block
	offset *= offset

	toAdd := rand.Intn(offset)

	last := len(ip) - 1
	ip[last] = ip[last] + byte(toAdd)

	return ip, nil
}
