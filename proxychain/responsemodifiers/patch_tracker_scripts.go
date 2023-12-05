package responsemodifiers

import (
	"embed"
	"encoding/json"
	"io"
	"log"
	"regexp"

	"ladder/proxychain"
)

//go:embed vendor/ddg-tracker-surrogates/mapping.json
var mappingJSON []byte

//go:embed vendor/ddg-tracker-surrogates/surrogates/*
var surrogateFS embed.FS

var rules domainRules

func init() {
	err := json.Unmarshal([]byte(mappingJSON), &rules)
	if err != nil {
		log.Printf("[ERROR]: PatchTrackerScripts: failed to deserialize ladder/proxychain/responsemodifiers/vendor/ddg-tracker-surrogates/mapping.json")
	}
}

// mapping.json schema
type rule struct {
	RegexRule *regexp.Regexp `json:"regexRule"`
	Surrogate string         `json:"surrogate"`
	Action    string         `json:"action,omitempty"`
}

type domainRules map[string][]rule

func (r *rule) UnmarshalJSON(data []byte) error {
	type Tmp struct {
		RegexRule string `json:"regexRule"`
		Surrogate string `json:"surrogate"`
		Action    string `json:"action,omitempty"`
	}

	var tmp Tmp
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	regex := regexp.MustCompile(tmp.RegexRule)

	r.RegexRule = regex
	r.Surrogate = tmp.Surrogate
	r.Action = tmp.Action

	return nil
}

// PatchTrackerScripts replaces any request to tracker scripts such as google analytics
// with a no-op stub that mocks the API structure of the original scripts they replace.
// Some pages depend on the existence of these structures for proper loading, so this may fix
// some broken elements.
// Surrogate script code borrowed from: DuckDuckGo Privacy Essentials browser extension for Firefox, Chrome. (Apache 2.0 license)
func PatchTrackerScripts() proxychain.ResponseModification {

	return func(chain *proxychain.ProxyChain) error {

		// preflight checks
		reqURL := chain.Request.URL.String()
		isTracker := false
		//

		var surrogateScript io.ReadCloser
		for domain, domainRules := range rules {
			for _, rule := range domainRules {
				if !rule.RegexRule.MatchString(reqURL) {
					continue
				}

				// found tracker script, replacing response body with nop stub from
				// ./vendor/ddg-tracker-surrogates/surrogates/{{rule.Surrogate}}
				isTracker = true
				script, err := surrogateFS.Open("vendor/ddg-tracker-surrogates/surrogates/" + rule.Surrogate)
				if err != nil {
					panic(err)
				}
				surrogateScript = io.NopCloser(script)
				log.Printf("INFO: PatchTrackerScripts :: injecting surrogate for '%s' => 'surrogates/%s'\n", domain, rule.Surrogate)
				break

			}
		}

		if !isTracker {
			return nil
		}

		chain.Response.Body = surrogateScript
		chain.Context.Set("content-type", "text/javascript")
		return nil
	}
}
