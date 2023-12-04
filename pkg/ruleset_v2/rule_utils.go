package ruleset_v2

import (
	"errors"
	"strings"
)

// parseFuncCall takes a string that look like foo("bar", "baz") and breaks it down
// into funcName = "foo" and params = []string{"bar", "baz"}]
func parseFuncCall(funcCall string) (funcName string, params []string, err error) {
	// Splitting the input string into two parts: functionName and parameters
	parts := strings.SplitN(funcCall, "(", 2)
	if len(parts) != 2 {
		return "", nil, errors.New("invalid function call format")
	}

	// get function name
	funcName = strings.TrimSpace(parts[0])

	// Removing the closing parenthesis from the parameters part
	paramsPart := strings.TrimSuffix(parts[1], ")")
	if len(paramsPart) == 0 {
		// No parameters
		return funcName, []string{}, nil
	}

	inQuote := false
	inEscape := false
	param := ""
	for _, r := range paramsPart {
		switch {
		case inQuote && !inEscape && r == '\\':
			inEscape = true
			continue
		case inEscape && inQuote && r == '"':
			param += string(r)
			inEscape = false
			continue
		case inEscape:
			param += string(r)
			inEscape = false
			continue
		case r == '"':
			inQuote = !inQuote
			if !inQuote {
				params = append(params, param)
				param = ""
			}
			continue
		case !inQuote && r == ',':
			continue
		case inQuote:
			param += string(r)
			continue
		}
	}

	return funcName, params, nil
}
