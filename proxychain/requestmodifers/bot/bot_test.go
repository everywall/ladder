package bot

import (
	"net"
	"testing"
)

func TestRandomIPFromSubnet(t *testing.T) {
	subnets := []string{"34.100.182.96/28", "207.46.13.0/24", "2001:4860:4801:10::/64", "2001:4860:4801:c::/64"}

	for _, subnet := range subnets {
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
