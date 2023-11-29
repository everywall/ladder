package handlers

import (
	"ladder/proxychain"
	rx "ladder/proxychain/requestmodifers"
	tx "ladder/proxychain/responsemodifers"

	"github.com/gofiber/fiber/v2"
)

type ProxyOptions struct {
	RulesetPath string
	Verbose     bool
}

func NewProxySiteHandler(opts *ProxyOptions) fiber.Handler {
	/*
		var rs ruleset.RuleSet
		if opts.RulesetPath != "" {
			r, err := ruleset.NewRuleset(opts.RulesetPath)
			if err != nil {
				panic(err)
			}
			rs = r
		}
	*/
	const botUA string = "Mozilla/5.0 (compatible; Yahoo! Slurp; http://help.yahoo.com/help/us/ysearch/slurp)"
	// 5.255.250.0/24, 37.9.87.0/24, 67.195.37.0/24, 67.195.50.0/24, 67.195.110.0/24, 67.195.111.0/24, 67.195.112.0/23, 67.195.114.0/24, 67.195.115.0/24, 68.180.224.0/21, 72.30.132.0/24, 72.30.142.0/24, 72.30.161.0/24, 72.30.196.0/24, 72.30.198.0/24, 74.6.254.0/24, 74.6.8.0/24, 74.6.13.0/24, 74.6.17.0/24, 74.6.18.0/24, 74.6.22.0/24, 74.6.27.0/24, 74.6.168.0/24, 77.88.5.0/24, 77.88.47.0/24, 93.158.161.0/24, 98.137.72.0/24, 98.137.206.0/24, 98.137.207.0/24, 98.139.168.0/24, 114.111.95.0/24, 124.83.159.0/24, 124.83.179.0/24, 124.83.223.0/24, 141.8.144.0/24, 183.79.63.0/24, 183.79.92.0/24, 203.216.255.0/24, 211.14.11.0/24
	//const ja3 string = "769,49195-49199-49200-49161-49171-49162-49172-156-157-47-10-53-51-57,65281-0-23-35-13-13172-11-10,29-23-24,0"
	const ja3 string = "771,49199-49195-49171-49161-49200-49196-49172-49162-51-57-50-49169-49159-47-53-10-5-4-255,0-11-10-13-13172-16,23-25-28-27-24-26-22-14-13-11-12-9-10,0-1-2"

	return func(c *fiber.Ctx) error {
		proxychain := proxychain.
			NewProxyChain().
			SetFiberCtx(c).
			SetDebugLogging(opts.Verbose).
			SetRequestModifications(
				//rx.SpoofJA3fingerprint(ja3, "Googlebot"),
				//rx.MasqueradeAsFacebookBot(),
				rx.MasqueradeAsGoogleBot(),
				//rx.DeleteOutgoingCookies(),
				rx.ForwardRequestHeaders(),
				rx.SetOutgoingCookie("nyt-a", " "),
				rx.SetOutgoingCookie("nyt-gdpr", "0"),
				rx.SetOutgoingCookie("nyt-gdpr", "0"),
				rx.SetOutgoingCookie("nyt-geo", "DE"),
				rx.SetOutgoingCookie("nyt-privacy", "1"),
				rx.SpoofReferrerFromGoogleSearch(),
				//rx.RequestWaybackMachine(),
				//rx.RequestArchiveIs(),
			).
			AddResponseModifications(
				//tx.ForwardResponseHeaders(),
				tx.BypassCORS(),
				tx.BypassContentSecurityPolicy(),
				//tx.DeleteIncomingCookies(),
				tx.RewriteHTMLResourceURLs(),
				//tx.PatchDynamicResourceURLs(),
				tx.APIOutline(),
			//tx.SetContentSecurityPolicy("default-src * 'unsafe-inline' 'unsafe-eval' data: blob:;"),
			).
			Execute()

		return proxychain
	}
}
