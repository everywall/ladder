package ruleset_v2

import (
	"encoding/json"
	"fmt"
	yaml "gopkg.in/yaml.v3"
	"testing"
)

func TestRuleUnmarshalJSON(t *testing.T) {
	ruleJSON := `{
    "domains": [
        "example.com",
        "www.example.com"
    ],
    "request_modifications": [
    	"SpoofUserAgent(\"googlebot\")"
    ],
    "response_modifications": [
      "APIContent()",
      "SetContentSecurityPolicy(\"foobar\")",
      "SetIncomingCookie(\"authorization-bearer\", \"hunter2\")"
    ]
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
	if len(rule.ResponseModifications) != 3 {
		t.Errorf("expected number of ResponseModifications to be 3, got %d", len(rule.ResponseModifications))
	}
	if len(rule.RequestModifications) != 1 {
		t.Errorf("expected number of RequestModifications to be 1, got %d", len(rule.RequestModifications))
	}

	// test marshal
	jsonRule, err := json.Marshal(rule)
	if err != nil {
		t.Errorf("expected no error marshalling rule to json, got '%s'", err.Error())
	}
	fmt.Println(string(jsonRule))
}

func TestRuleUnmarshalYAML(t *testing.T) {
	ruleYAML := `
domains:
  - example.com
  - www.example.com
request_modifications:
  - SpoofUserAgent("googlebot")
response_modifications:
  - APIContent()
  - SetContentSecurityPolicy("foobar")
  - SetIncomingCookie("authorization-bearer", "hunter2")
`

	rule := &Rule{}
	err := yaml.Unmarshal([]byte(ruleYAML), rule)
	if err != nil {
		t.Errorf("expected no error in Unmarshal, got '%s'", err)
		return
	}

	if len(rule.Domains) != 2 {
		t.Errorf("expected number of domains to be 2, got %d", len(rule.Domains))
		return
	}
	if !(rule.Domains[0] == "example.com" || rule.Domains[1] == "example.com") {
		t.Errorf("expected domain to be example.com")
		return
	}
	if len(rule.ResponseModifications) != 3 {
		t.Errorf("expected number of ResponseModifications to be 3, got %d", len(rule.ResponseModifications))
	}
	if len(rule.RequestModifications) != 1 {
		t.Errorf("expected number of RequestModifications to be 1, got %d", len(rule.RequestModifications))
	}

	yamlRule, err := yaml.Marshal(rule)
	if err != nil {
		t.Errorf("expected no error marshalling rule to yaml, got '%s'", err.Error())
	}
	fmt.Println(string(yamlRule))
}
