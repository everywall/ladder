package ruleset_v2

import (
	"encoding/json"
	"fmt"
	//"io"
	"testing"
)

func TestRuleUnmarshalJSON(t *testing.T) {
	ruleJSON := `{
    "domains": [
        "example.com",
        "www.example.com"
    ],
    "response_modifiers": [
        "APIContent()",
        "SetContentSecurityPolicy(\"foobar\")",
        "SetIncomingCookie(\"authorization-bearer\", \"hunter2\")"
    ],
    "response_modifiers": []
}`

	//fmt.Println(ruleJSON)
	rule := &Rule{}
	err := json.Unmarshal([]byte(ruleJSON), rule)
	if err != nil {
		t.Errorf("expected no error in Unmarshal, got '%s'", err)
		return
	}

	if len(rule.Domains) != 2 {
		t.Errorf("expected number of domains to be 2")
		return
	}
	if !(rule.Domains[0] == "example.com" || rule.Domains[1] == "example.com") {
		t.Errorf("expected domain to be example.com")
		return
	}
	if len(rule.ResponseModifications) == 3 {
		t.Errorf("expected number of ResponseModifications to be 3")
	}
	fmt.Println(rule.ResponseModifications)

}
