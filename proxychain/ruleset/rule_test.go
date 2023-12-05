package ruleset_v2

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"testing"
)

// unmarshalRule is a helper function to unmarshal a Rule from a JSON string.
func unmarshalRule(t *testing.T, ruleJSON string) *Rule {
	rule := &Rule{}
	err := json.Unmarshal([]byte(ruleJSON), rule)
	if err != nil {
		t.Fatalf("expected no error in Unmarshal, got '%s'", err)
	}
	return rule
}

func TestRuleUnmarshalJSON(t *testing.T) {
	ruleJSON := `{
		"domains": ["example.com", "www.example.com"],
		"responsemodifications": [{"name": "APIContent", "params": []}, {"name": "SetContentSecurityPolicy", "params": ["foobar"]}, {"name": "SetIncomingCookie", "params": ["authorization-bearer", "hunter2"]}],
		"requestmodifications": [{"name": "ForwardRequestHeaders", "params": []}]
	}`

	rule := unmarshalRule(t, ruleJSON)

	if len(rule.Domains) != 2 {
		t.Errorf("expected number of domains to be 2")
	}
	if !(rule.Domains[0] == "example.com" || rule.Domains[1] == "example.com") {
		t.Errorf("expected domain to be example.com")
	}
	if len(rule.ResponseModifications) != 3 {
		t.Errorf("expected number of ResponseModifications to be 3, got %d", len(rule.ResponseModifications))
	}
	if len(rule.RequestModifications) != 1 {
		t.Errorf("expected number of RequestModifications to be 1, got %d", len(rule.RequestModifications))
	}
}

func TestRuleMarshalJSON(t *testing.T) {
	ruleJSON := `{
		"domains": ["example.com", "www.example.com"],
		"responsemodifications": [{"name": "APIContent", "params": []}, {"name": "SetContentSecurityPolicy", "params": ["foobar"]}, {"name": "SetIncomingCookie", "params": ["authorization-bearer", "hunter2"]}],
		"requestmodifications": [{"name": "ForwardRequestHeaders", "params": []}]
	}`

	rule := unmarshalRule(t, ruleJSON)

	jsonRule, err := json.Marshal(rule)
	if err != nil {
		t.Errorf("expected no error marshalling rule to json, got '%s'", err.Error())
	}
	fmt.Println(string(jsonRule))
}

// ===============================================

// unmarshalYAMLRule is a helper function to unmarshal a Rule from a YAML string.
func unmarshalYAMLRule(t *testing.T, ruleYAML string) *Rule {
	rule := &Rule{}
	err := yaml.Unmarshal([]byte(ruleYAML), rule)
	if err != nil {
		t.Fatalf("expected no error in Unmarshal, got '%s'", err)
	}
	return rule
}

func TestRuleUnmarshalYAML(t *testing.T) {
	ruleYAML := `
domains:
- example.com
- www.example.com
responsemodifications:
- name: APIContent
  params: []
- name: SetContentSecurityPolicy
  params:
  - foobar
- name: SetIncomingCookie
  params:
  - authorization-bearer
  - hunter2
requestmodifications:
- name: ForwardRequestHeaders
  params: []
`

	rule := unmarshalYAMLRule(t, ruleYAML)

	if len(rule.Domains) != 2 {
		t.Errorf("expected number of domains to be 2")
	}
	if !(rule.Domains[0] == "example.com" || rule.Domains[1] == "example.com") {
		t.Errorf("expected domain to be example.com")
	}
	if len(rule.ResponseModifications) != 3 {
		t.Errorf("expected number of ResponseModifications to be 3, got %d", len(rule.ResponseModifications))
	}
	if len(rule.RequestModifications) != 1 {
		t.Errorf("expected number of RequestModifications to be 1, got %d", len(rule.RequestModifications))
	}
}

func TestRuleMarshalYAML(t *testing.T) {
	ruleYAML := `
domains:
- example.com
- www.example.com
responsemodifications:
- name: APIContent
  params: []
- name: SetContentSecurityPolicy
  params:
  - foobar
- name: SetIncomingCookie
  params:
  - authorization-bearer
  - hunter2
requestmodifications:
- name: ForwardRequestHeaders
  params: []
`

	rule := unmarshalYAMLRule(t, ruleYAML)

	yamlRule, err := yaml.Marshal(rule)
	if err != nil {
		t.Errorf("expected no error marshalling rule to yaml, got '%s'", err.Error())
	}
	if yamlRule == nil {
		t.Errorf("expected marshalling rule to yaml to not be nil")
	}
}
