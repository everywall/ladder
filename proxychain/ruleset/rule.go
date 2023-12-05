package ruleset_v2

import (
	"encoding/json"
	"fmt"
	"ladder/proxychain"
	// _ "gopkg.in/yaml.v3"
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
	Name   string   `json:"name"`
	Params []string `json:"params"`
}

// internal represenation of RequestModifications
type _rqm struct {
	Name   string   `json:"name"`
	Params []string `json:"params"`
}

// implement type encoding/json/Marshaler
func (rule *Rule) UnmarshalJSON(data []byte) error {
	type Aux struct {
		Domains               []string `json:"domains"`
		RequestModifications  []_rqm   `json:"request_modifications"`
		ResponseModifications []_rsm   `json:"response_modifications"`
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

func (r *Rule) MarshalJSON() ([]byte, error) {
	aux := struct {
		Domains               []string `json:"domains"`
		RequestModifications  []_rqm   `json:"request_modifications"`
		ResponseModifications []_rsm   `json:"response_modifications"`
	}{
		Domains:               r.Domains,
		RequestModifications:  r._rqms,
		ResponseModifications: r._rsms,
	}

	return json.Marshal(aux)
}
