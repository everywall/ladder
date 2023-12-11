package responsemodifiers

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/everywall/ladder/proxychain"
)

// ModifyIncomingScriptsWithRegex modifies all incoming javascript (application/javascript and inline <script> in text/html) using a regex match and replacement.
func ModifyIncomingScriptsWithRegex(matchRegex string, replacement string) proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		path := chain.Request.URL.Path
		ct := chain.Response.Header.Get("content-type")
		isJavascript := strings.HasSuffix(path, ".js") || ct == "text/javascript" || ct == "application/javascript"
		isHTML := strings.HasSuffix(chain.Request.URL.Path, ".html") || ct == "text/html"

		switch {
		case isJavascript:
			rBody, err := modifyResponse(chain.Response.Body, matchRegex, replacement)
			if err != nil {
				return err
			}
			chain.Response.Body = rBody
		case isHTML:
		default:
			return nil
		}
		return nil
	}
}

func modifyResponse(body io.ReadCloser, matchRegex, replacement string) (io.ReadCloser, error) {
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(matchRegex)
	if err != nil {
		return nil, err
	}
	err = body.Close()
	if err != nil {
		return body, err
	}

	modifiedContent := re.ReplaceAll(content, []byte(replacement))

	return io.NopCloser(bytes.NewReader(modifiedContent)), nil
}
