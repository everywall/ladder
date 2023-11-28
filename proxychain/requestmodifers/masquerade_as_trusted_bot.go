package requestmodifers

import (
	"ladder/proxychain"
)

// MasqueradeAsGoogleBot modifies user agent and x-forwarded for
// to appear to be a Google Bot
func MasqueradeAsGoogleBot() proxychain.RequestModification {
	const botUA string = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; Googlebot/2.1; http://www.google.com/bot.html) Chrome/79.0.3945.120 Safari/537.36"
	const botIP string = "66.249.78.8" // TODO: create a random ip pool from https://developers.google.com/static/search/apis/ipranges/googlebot.json
	// https://github.com/trisulnsm/trisul-scripts/blob/master/lua/frontend_scripts/reassembly/ja3/prints/ja3fingerprint.json
	const ja3 string = "769,49195-49199-49196-49200-52393-52392-52244-52243-49161-49171-49162-49172-156-157-47-53-10,65281-0-23-35-13-5-18-16-11-10-21,29-23-24,0"
	//	"741,49195-49199-49200-49161-49171-49162-49172-156-157-47-10-53-51-57,65281-0-23-35-13-13172-11-10,29-23-24,0"

	return masqueradeAsTrustedBot(botUA, botIP, ja3)
}

// MasqueradeAsBingBot modifies user agent and x-forwarded for
// to appear to be a Bing Bot
func MasqueradeAsBingBot() proxychain.RequestModification {
	const botUA string = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm) Chrome/79.0.3945.120 Safari/537.36"
	const botIP string = "13.66.144.9" // https://www.bing.com/toolbox/bingbot.json
	return masqueradeAsTrustedBot(botUA, botIP, "")
}

// MasqueradeAsWaybackMachineBot modifies user agent and x-forwarded for
// to appear to be a archive.org (wayback machine) Bot
func MasqueradeAsWaybackMachineBot() proxychain.RequestModification {
	const botUA string = "Mozilla/5.0 (compatible; archive.org_bot +http://www.archive.org/details/archive.org_bot)"
	const botIP string = "207.241.235.164"
	return masqueradeAsTrustedBot(botUA, botIP, "")
}

// MasqueradeAsFacebookBot modifies user agent and x-forwarded for
// to appear to be a Facebook Bot (link previews?)
func MasqueradeAsFacebookBot() proxychain.RequestModification {
	const botUA string = "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
	// 31.13.97.0/24, 31.13.99.0/24, 31.13.100.0/24, 66.220.144.0/20, 69.63.189.0/24, 69.63.190.0/24, 69.171.224.0/20, 69.171.240.0/21, 69.171.248.0/24, 173.252.73.0/24, 173.252.74.0/24, 173.252.77.0/24, 173.252.100.0/22, 173.252.104.0/21, 173.252.112.0/24, 2a03:2880:10::/48, 2a03:2880:10ff::/48, 2a03:2880:11::/48, 2a03:2880:11ff::/48, 2a03:2880:20::/48, 2a03:2880:20ff::/48, 2a03:2880:21ff::/48, 2a03:2880:30ff::/48, 2a03:2880:31ff::/48, 2a03:2880:1010::/48, 2a03:2880:1020::/48, 2a03:2880:2020::/48, 2a03:2880:2050::/48, 2a03:2880:2040::/48, 2a03:2880:2110::/48, 2a03:2880:2130::/48, 2a03:2880:3010::/48, 2a03:2880:3020::/48
	const botIP string = "31.13.99.8"
	const ja3 string = "771,49199-49195-49171-49161-49200-49196-49172-49162-51-57-50-49169-49159-47-53-10-5-4-255,0-11-10-13-13172-16,23-25-28-27-24-26-22-14-13-11-12-9-10,0-1-2"
	return masqueradeAsTrustedBot(botUA, botIP, ja3)
}

// MasqueradeAsYandexBot modifies user agent and x-forwarded for
// to appear to be a Yandex Spider Bot
func MasqueradeAsYandexBot() proxychain.RequestModification {
	const botUA string = "Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)"
	// 100.43.90.0/24, 37.9.115.0/24, 37.140.165.0/24, 77.88.22.0/25, 77.88.29.0/24, 77.88.31.0/24, 77.88.59.0/24, 84.201.146.0/24, 84.201.148.0/24, 84.201.149.0/24, 87.250.243.0/24, 87.250.253.0/24, 93.158.147.0/24, 93.158.148.0/24, 93.158.151.0/24, 93.158.153.0/32, 95.108.128.0/24, 95.108.138.0/24, 95.108.150.0/23, 95.108.158.0/24, 95.108.156.0/24, 95.108.188.128/25, 95.108.234.0/24, 95.108.248.0/24, 100.43.80.0/24, 130.193.62.0/24, 141.8.153.0/24, 178.154.165.0/24, 178.154.166.128/25, 178.154.173.29, 178.154.200.158, 178.154.202.0/24, 178.154.205.0/24, 178.154.239.0/24, 178.154.243.0/24, 37.9.84.253, 199.21.99.99, 178.154.162.29, 178.154.203.251, 178.154.211.250, 178.154.171.0/24, 178.154.200.0/24, 178.154.244.0/24, 178.154.246.0/24, 95.108.181.0/24, 95.108.246.252, 5.45.254.0/24, 5.255.253.0/24, 37.140.141.0/24, 37.140.188.0/24, 100.43.81.0/24, 100.43.85.0/24, 100.43.91.0/24, 199.21.99.0/24, 2a02:6b8:b000::/32, 2a02:6b8:b010::/32, 2a02:6b8:b011::/32, 2a02:6b8:c0e::/32
	const botIP string = "37.9.115.9"
	const ja3 string = "769,49200-49196-49192-49188-49172-49162-165-163-161-159-107-106-105-104-57-56-55-54-136-135-134-133-49202-49198-49194-49190-49167-49157-157-61-53-132-49199-49195-49191-49187-49171-49161-164-162-160-158-103-64-63-62-51-50-49-48-154-153-152-151-69-68-67-66-49201-49197-49193-49189-49166-49156-156-60-47-150-65-7-49169-49159-49164-49154-5-4-49170-49160-22-19-16-13-49165-49155-10-255,0-11-10-35-13-15,23-25-28-27-24-26-22-14-13-11-12-9-10,0-1-2"
	return masqueradeAsTrustedBot(botUA, botIP, ja3)
}

// MasqueradeAsBaiduBot modifies user agent and x-forwarded for
// to appear to be a Baidu Spider Bot
func MasqueradeAsBaiduBot() proxychain.RequestModification {
	const botUA string = "Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)"
	// 180.76.15.0/24, 119.63.196.0/24, 115.239.212./24, 119.63.199.0/24, 122.81.208.0/22, 123.125.71.0/24, 180.76.4.0/24, 180.76.5.0/24, 180.76.6.0/24, 185.10.104.0/24, 220.181.108.0/24, 220.181.51.0/24, 111.13.102.0/24, 123.125.67.144/29, 123.125.67.152/31, 61.135.169.0/24, 123.125.68.68/30, 123.125.68.72/29, 123.125.68.80/28, 123.125.68.96/30, 202.46.48.0/20, 220.181.38.0/24, 123.125.68.80/30, 123.125.68.84/31, 123.125.68.0/24
	const botIP string = "180.76.15.7"
	return masqueradeAsTrustedBot(botUA, botIP, "")
}

// MasqueradeAsDuckDuckBot modifies user agent and x-forwarded for
// to appear to be a DuckDuckGo Bot
func MasqueradeAsDuckDuckBot() proxychain.RequestModification {
	const botUA string = "DuckDuckBot/1.0; (+http://duckduckgo.com/duckduckbot.html)"
	// 46.51.197.88, 46.51.197.89, 50.18.192.250, 50.18.192.251, 107.21.1.61, 176.34.131.233, 176.34.135.167, 184.72.106.52, 184.72.115.86
	const botIP string = "46.51.197.88"
	return masqueradeAsTrustedBot(botUA, botIP, "")
}

// MasqueradeAsYahooBot modifies user agent and x-forwarded for
// to appear to be a Yahoo Bot
func MasqueradeAsYahooBot() proxychain.RequestModification {
	const botUA string = "Mozilla/5.0 (compatible; Yahoo! Slurp; http://help.yahoo.com/help/us/ysearch/slurp)"
	// 5.255.250.0/24, 37.9.87.0/24, 67.195.37.0/24, 67.195.50.0/24, 67.195.110.0/24, 67.195.111.0/24, 67.195.112.0/23, 67.195.114.0/24, 67.195.115.0/24, 68.180.224.0/21, 72.30.132.0/24, 72.30.142.0/24, 72.30.161.0/24, 72.30.196.0/24, 72.30.198.0/24, 74.6.254.0/24, 74.6.8.0/24, 74.6.13.0/24, 74.6.17.0/24, 74.6.18.0/24, 74.6.22.0/24, 74.6.27.0/24, 74.6.168.0/24, 77.88.5.0/24, 77.88.47.0/24, 93.158.161.0/24, 98.137.72.0/24, 98.137.206.0/24, 98.137.207.0/24, 98.139.168.0/24, 114.111.95.0/24, 124.83.159.0/24, 124.83.179.0/24, 124.83.223.0/24, 141.8.144.0/24, 183.79.63.0/24, 183.79.92.0/24, 203.216.255.0/24, 211.14.11.0/24
	const ja3 = "769,49200-49196-49192-49188-49172-49162-163-159-107-106-57-56-136-135-49202-49198-49194-49190-49167-49157-157-61-53-132-49199-49195-49191-49187-49171-49161-162-158-103-64-51-50-49170-49160-154-153-69-68-22-19-49201-49197-49193-49189-49166-49156-49165-49155-156-60-47-150-65-10-7-49169-49159-49164-49154-5-4-255,0-11-10-13-15,25-24-23,0-1-2"
	const botIP string = "37.9.87.5"
	return masqueradeAsTrustedBot(botUA, botIP, ja3)
}

func masqueradeAsTrustedBot(botUA string, botIP string, ja3 string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		chain.AddOnceRequestModifications(
			SpoofUserAgent(botUA),
			SetRequestHeader("x-forwarded-for", botIP),
			DeleteRequestHeader("referrer"),
			DeleteRequestHeader("origin"),
		)

		if ja3 != "" {
			chain.AddOnceRequestModifications(
				SpoofJA3fingerprint(ja3, botUA),
			)
		}

		return nil
	}
}
