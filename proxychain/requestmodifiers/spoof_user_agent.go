package requestmodifiers

import (
	_ "embed"
	"strings"

	"ladder/proxychain"
	tx "ladder/proxychain/responsemodifiers"
)

// https://github.com/faisalman/ua-parser-js/tree/master
// update using:
// git submodule update --remote --merge
//
//go:embed vendor/ua-parser-js/dist/ua-parser.min.js
var UAParserJS string

// note: spoof_user_agent.js has a dependency on ua-parser.min.js
// ua-parser.min.js should be loaded first.
//
//go:embed spoof_user_agent.js
var spoofUserAgentJS string

// SpoofUserAgent modifies the user agent
func SpoofUserAgent(ua string) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		// modify ua headers
		chain.AddOnceRequestModifications(
			SetRequestHeader("user-agent", ua),
		)

		script := strings.ReplaceAll(spoofUserAgentJS, "{{USER_AGENT}}", ua)
		chain.AddOnceResponseModifications(
			tx.InjectScriptBeforeDOMContentLoaded(script),
			tx.InjectScriptBeforeDOMContentLoaded(UAParserJS),
		)

		return nil
	}
}
