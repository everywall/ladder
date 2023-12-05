package ruleset_v2

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"ladder/proxychain"
)

type Rule struct {
	Domains               []string
	RequestModifications  []proxychain.RequestModification
	_rqms                 []_rqm // internal represenation of RequestModifications
	ResponseModifications []proxychain.ResponseModification
	_rsms                 []_rsm // internal represenation of ResponseModifications
}

// internal represenation of ResponseModifications
type _rsm struct {
	Name   string   `json:"name" yaml:"name"`
	Params []string `json:"params" yaml:"params"`
}

// internal represenation of RequestModifications
type _rqm struct {
	Name   string   `json:"name" yaml:"name"`
	Params []string `json:"params" yaml:"params"`
}

// implement type encoding/json/Marshaler
func (rule *Rule) UnmarshalJSON(data []byte) error {
	type Aux struct {
		Domains               []string `json:"domains"`
		RequestModifications  []_rqm   `json:"requestmodifications"`
		ResponseModifications []_rsm   `json:"responsemodifications"`
	}

	aux := &Aux{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	rule.Domains = aux.Domains
	rule._rqms = aux.RequestModifications
	rule._rsms = aux.ResponseModifications

	// convert requestModification function call string into actual functional option
	for _, rqm := range aux.RequestModifications {
		f, exists := rqmModMap[rqm.Name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalJSON => requestModifier '%s' does not exist, please check spelling", rqm.Name)
		}
		rule.RequestModifications = append(rule.RequestModifications, f(rqm.Params...))
	}

	// convert responseModification function call string into actual functional option
	for _, rsm := range aux.ResponseModifications {
		f, exists := rsmModMap[rsm.Name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalJSON => responseModifier '%s' does not exist, please check spelling", rsm.Name)
		}
		rule.ResponseModifications = append(rule.ResponseModifications, f(rsm.Params...))
	}

	return nil
}

func (rule *Rule) MarshalJSON() ([]byte, error) {
	aux := struct {
		Domains               []string `json:"domains"`
		RequestModifications  []_rqm   `json:"requestmodifications"`
		ResponseModifications []_rsm   `json:"responsemodifications"`
	}{
		Domains:               rule.Domains,
		RequestModifications:  rule._rqms,
		ResponseModifications: rule._rsms,
	}

	return json.MarshalIndent(aux, "", "    ")
}

// ============================================================
// YAML

// implement type yaml marshaller
func (rule *Rule) UnmarshalYAML(unmarshal func(interface{}) error) error {

	type Aux struct {
		Domains               []string `yaml:"domains"`
		RequestModifications  []_rqm   `yaml:"requestmodifications"`
		ResponseModifications []_rsm   `yaml:"responsemodifications"`
	}

	var aux Aux
	if err := unmarshal(&aux); err != nil {
		return err
	}

	rule.Domains = aux.Domains
	rule._rqms = aux.RequestModifications
	rule._rsms = aux.ResponseModifications

	// convert requestModification function call string into actual functional option
	for _, rqm := range aux.RequestModifications {
		f, exists := rqmModMap[rqm.Name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalYAML => requestModifier '%s' does not exist, please check spelling", rqm.Name)
		}
		rule.RequestModifications = append(rule.RequestModifications, f(rqm.Params...))
	}

	// convert responseModification function call string into actual functional option
	for _, rsm := range aux.ResponseModifications {
		f, exists := rsmModMap[rsm.Name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalYAML => responseModifier '%s' does not exist, please check spelling", rsm.Name)
		}
		rule.ResponseModifications = append(rule.ResponseModifications, f(rsm.Params...))
	}

	return nil
}

func (rule *Rule) MarshalYAML() (interface{}, error) {

	type Aux struct {
		Domains               []string `yaml:"domains"`
		RequestModifications  []_rqm   `yaml:"requestmodifications"`
		ResponseModifications []_rsm   `yaml:"responsemodifications"`
	}

	aux := &Aux{
		Domains:               rule.Domains,
		RequestModifications:  rule._rqms,
		ResponseModifications: rule._rsms,
	}

	return yaml.Marshal(aux)
}
