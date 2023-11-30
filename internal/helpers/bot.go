package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/3th1nk/cidr"
)

type Bot interface {
	UpdatePool() error
	GetRandomIdentity() string
}

type GoogleBot struct {
	UserAgent   string
	Fingerprint string
	IPPool      googleBotPool
}

type googleBotPool struct {
	Timestamp string            `json:"creationTime"`
	Prefixes  []googleBotPrefix `json:"prefixes"`
}

type googleBotPrefix struct {
	IPv6 string `json:"ipv6Prefix,omitempty"`
	IPv4 string `json:"ipv4Prefix,omitempty"`
}

// const googleBotTimestampFormat string = "2006-01-02T15:04:05.999999"

// TODO: move this thing's pointer aound, not use it as a global variable
var GlobalGoogleBot = GoogleBot{
	UserAgent: "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; Googlebot/2.1; http://www.google.com/bot.html) Chrome/79.0.3945.120 Safari/537.36",

	// https://github.com/trisulnsm/trisul-scripts/blob/master/lua/frontend_scripts/reassembly/ja3/prints/ja3fingerprint.json
	Fingerprint: "769,49195-49199-49196-49200-52393-52392-52244-52243-49161-49171-49162-49172-156-157-47-53-10,65281-0-23-35-13-5-18-16-11-10-21,29-23-24,0",

	IPPool: googleBotPool{
		Timestamp: "2023-11-28T23:00:56.000000",
		Prefixes: []googleBotPrefix{
			{
				IPv4: "34.100.182.96/28",
			},
		},
	},
}

func (bot *GoogleBot) UpdatePool() error {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
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

	err = json.Unmarshal(body, &bot.IPPool)

	return err
}

func (bot *GoogleBot) GetRandomIP() string {
	count := len(bot.IPPool.Prefixes)

	var prefix googleBotPrefix

	if count == 1 {
		prefix = bot.IPPool.Prefixes[0]
	} else {
		idx := rand.Intn(count)
		prefix = bot.IPPool.Prefixes[idx]
	}

	if prefix.IPv4 != "" {
		ip, err := randomIPFromSubnet(prefix.IPv4)
		if err == nil {
			return ip
		}
	}

	if prefix.IPv6 != "" {
		ip, err := randomIPFromSubnet(prefix.IPv6)
		if err == nil {
			return ip
		}
	}

	// fallback to default IP which is known to work
	ip, _ := randomIPFromSubnet(bot.IPPool.Prefixes[0].IPv4)

	return ip
}

func randomIPFromSubnet(c string) (string, error) {
	block, err := cidr.Parse(c)
	if err != nil {
		return "", err
	}

	// TODO: the beginning of the network is technically a viable IP to use
	// but maybe a different solution would be better here
	return block.Network().String(), nil
}
