package ruleset_v2

import (
	"encoding/json"
	"fmt"
	"ladder/proxychain"
)

type Rule struct {
	Domains               []string
	RequestModifications  []proxychain.RequestModification
	ResponseModifications []proxychain.ResponseModification
}

// implement type encoding/json/Marshaler
func (rule *Rule) UnmarshalJSON(data []byte) error {
	type Aux struct {
		Domains               []string `json:"domains"`
		RequestModifications  []string `json:"request_modifications"`
		ResponseModifications []string `json:"response_modifications"`
	}

	aux := &Aux{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	//fmt.Println(aux.Domains)
	rule.Domains = aux.Domains

	// convert requestModification function call string into actual functional option
	for _, resModStr := range aux.RequestModifications {
		name, params, err := parseFuncCall(resModStr)
		if err != nil {
			return fmt.Errorf("Rule::UnmarshalJSON invalid function call syntax => '%s'", err)
		}
		f, exists := rsmModMap[name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalJSON => responseModifer '%s' does not exist, please check spelling", err)
		}
		rule.ResponseModifications = append(rule.ResponseModifications, f(params...))
	}

	// convert responseModification function call string into actual functional option
	for _, rqmModStr := range aux.RequestModifications {
		name, params, err := parseFuncCall(rqmModStr)
		if err != nil {
			return fmt.Errorf("Rule::UnmarshalJSON invalid function call syntax => '%s'", err)
		}
		f, exists := rqmModMap[name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalJSON => requestModifier '%s' does not exist, please check spelling", err)
		}
		rule.RequestModifications = append(rule.RequestModifications, f(params...))
	}

	return nil
}

func (r *Rule) MarshalJSON() ([]byte, error) {
	return []byte{}, nil
}
