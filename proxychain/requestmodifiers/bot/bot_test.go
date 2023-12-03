package bot

import (
	"net"
	"testing"
)

func TestRandomIPFromSubnet(t *testing.T) {
	err := GoogleBot.UpdatePool("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
	if err != nil {
		t.Error(err)
	}

	for _, prefix := range GoogleBot.IPPool.Prefixes {
		subnet := prefix.IPv4
		if prefix.IPv6 != "" {
			subnet = prefix.IPv6
		}

		t.Run(subnet, func(t *testing.T) {
			_, ipnet, err := net.ParseCIDR(subnet)
			if err != nil {
				t.Error(err)
			}

			ip, err := randomIPFromSubnet(subnet)
			if err != nil {
				t.Error(err)
			}

			if !ipnet.Contains(ip) {
				t.Fail()
			}
		})
	}
}
