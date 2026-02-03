package handlers

import (
	"math/rand"
	"os"
	"strings"
	"sync"
)

type BrowserProfile struct {
	Name       string
	JA3        string
	HTTP2      string
	UserAgent  string
}

var (
	chrome121 = BrowserProfile{
		Name:      "chrome",
		JA3:       "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		HTTP2:     "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	}
	firefox121 = BrowserProfile{
		Name:      "firefox",
		JA3:       "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0",
		HTTP2:     "1:65536;2:0;4:131072;5:16384|12517377|0|m,p,a,s",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	}
	safari17 = BrowserProfile{
		Name:      "safari",
		JA3:       "771,4865-4866-4867-49195-49199-52393-52392-49196-49200-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		HTTP2:     "1:65536;2:0;4:131072;5:16384|12517377|0|m,p,a,s",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	}
	profiles   = []BrowserProfile{chrome121, firefox121, safari17}
	profileMap = map[string]BrowserProfile{"chrome": chrome121, "firefox": firefox121, "safari": safari17}
	profileMu  sync.Mutex
)

var rotateFinger bool

func init() {
	rotateFinger = os.Getenv("ROTATE_FINGERPRINT") == "true"
}

func getBrowserProfile(ruleProfile string) BrowserProfile {
	name := strings.ToLower(strings.TrimSpace(os.Getenv("BROWSER_PROFILE")))
	if ruleProfile != "" {
		name = strings.ToLower(strings.TrimSpace(ruleProfile))
	}
	if name == "" {
		name = "chrome"
	}
	if p, ok := profileMap[name]; ok {
		if rotateFinger {
			profileMu.Lock()
			p = profiles[rand.Intn(len(profiles))]
			profileMu.Unlock()
		}
		return p
	}
	if rotateFinger {
		profileMu.Lock()
		p := profiles[rand.Intn(len(profiles))]
		profileMu.Unlock()
		return p
	}
	return chrome121
}
