package helpers

import (
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type googlebotResp struct {
	Timestamp time.Time
	IPs       []string
}

var GooglebotIPs = googlebotResp{
	IPs: []string{"34.165.18.176"},
}

const timeFormat string = "2006-01-02T15:04:05.999999"

func UpdateGooglebotIPs() error {
	resp, err := http.Get("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("non-200 status code recieved")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	j := map[string]any{}
	json.Unmarshal(body, &j)

	timestamp, err := time.Parse(timeFormat, j["creationTime"].(string))
	if err != nil {
		return err
	}

	prefixes := j["prefixes"].([]any)

	ips := make([]string, 0, 127)

	for _, prefix := range prefixes {
		p := prefix.(map[string]any)

		if val, exists := p["ipv4Prefix"]; exists {
			v := val.(string)

			v = strings.ReplaceAll(v, "/27", "")
			v = strings.ReplaceAll(v, "/28", "")

			ips = append(ips, v)
		}

	}

	GooglebotIPs = googlebotResp{
		Timestamp: timestamp,
		IPs:       ips,
	}

	return nil
}

func RandomGooglebotIP() string {
	count := len(GooglebotIPs.IPs)
	idx := 0

	if count != 1 {
		idx = rand.Intn(count)
	}

	return GooglebotIPs.IPs[idx]
}
