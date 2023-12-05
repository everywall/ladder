package ruleset_v2

import (
	"encoding/json"
	"fmt"
	"ladder/proxychain"
	"reflect"
	"runtime"
	"strings"

	_ "gopkg.in/yaml.v3"
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
	for _, rqmModStr := range aux.RequestModifications {
		name, params, err := parseFuncCall(rqmModStr)
		if err != nil {
			return fmt.Errorf("Rule::UnmarshalJSON invalid function call syntax => '%s'", err)
		}
		f, exists := rqmModMap[name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalJSON => requestModifier '%s' does not exist, please check spelling", name)
		}
		rule.RequestModifications = append(rule.RequestModifications, f(params...))
	}

	// convert responseModification function call string into actual functional option
	for _, rsmModStr := range aux.ResponseModifications {
		name, params, err := parseFuncCall(rsmModStr)
		if err != nil {
			return fmt.Errorf("Rule::UnmarshalJSON invalid function call syntax => '%s'", err)
		}
		f, exists := rsmModMap[name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalJSON => responseModifier '%s' does not exist, please check spelling", name)
		}
		rule.ResponseModifications = append(rule.ResponseModifications, f(params...))
	}

	return nil
}

// not fully possible to go from rule to JSON rule because
// reflection cannot get the parameters of the functional options
// of requestmodifiers and responsemodifiers
func (r *Rule) MarshalJSON() ([]byte, error) {
	type Aux struct {
		Domains               []string `json:"domains"`
		RequestModifications  []string `json:"request_modifications"`
		ResponseModifications []string `json:"response_modifications"`
	}
	aux := &Aux{}
	aux.Domains = r.Domains

	for _, rqmMod := range r.RequestModifications {
		fnName := getFunctionName(rqmMod)
		aux.RequestModifications = append(aux.RequestModifications, fnName)
	}

	for _, rsmMod := range r.ResponseModifications {
		fnName := getFunctionName(rsmMod)
		aux.ResponseModifications = append(aux.ResponseModifications, fnName)
	}

	return json.Marshal(aux)
}

// getFunctionName returns the name of the function
func getFunctionName(i interface{}) string {
	// Get the value of the interface
	val := reflect.ValueOf(i)

	// Ensure it's a function
	if val.Kind() != reflect.Func {
		return "Not a function"
	}

	// Get the pointer to the function
	ptr := val.Pointer()

	// Get the function details from runtime
	funcForPc := runtime.FuncForPC(ptr)

	if funcForPc == nil {
		return "Unknown"
	}

	// Return the name of the function
	return extractShortName(funcForPc.Name())
}

// extractShortName extracts the short function name from the full name
func extractShortName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		// Assuming the function name is always the second last part
		return parts[len(parts)-2]
	}
	return ""
}

// == YAML
// UnmarshalYAML implements the yaml.Unmarshaler interface for Rule
func (rule *Rule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Aux struct {
		Domains               []string `yaml:"domains"`
		RequestModifications  []string `yaml:"request_modifications"`
		ResponseModifications []string `yaml:"response_modifications"`
	}

	aux := &Aux{}
	if err := unmarshal(aux); err != nil {
		return err
	}

	rule.Domains = aux.Domains

	// Process requestModifications
	for _, rqmModStr := range aux.RequestModifications {
		name, params, err := parseFuncCall(rqmModStr)
		if err != nil {
			return fmt.Errorf("Rule::UnmarshalYAML invalid function call syntax => '%s'", err)
		}
		f, exists := rqmModMap[name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalYAML => requestModifier '%s' does not exist, please check spelling", name)
		}
		rule.RequestModifications = append(rule.RequestModifications, f(params...))
	}

	// Process responseModifications
	for _, rsmModStr := range aux.ResponseModifications {
		name, params, err := parseFuncCall(rsmModStr)
		if err != nil {
			return fmt.Errorf("Rule::UnmarshalYAML invalid function call syntax => '%s'", err)
		}
		f, exists := rsmModMap[name]
		if !exists {
			return fmt.Errorf("Rule::UnmarshalYAML => responseModifier '%s' does not exist, please check spelling", name)
		}
		rule.ResponseModifications = append(rule.ResponseModifications, f(params...))
	}

	return nil
}

func (r *Rule) MarshalYAML() (interface{}, error) {
	type Aux struct {
		Domains               []string `yaml:"domains"`
		RequestModifications  []string `yaml:"request_modifications"`
		ResponseModifications []string `yaml:"response_modifications"`
	}

	aux := &Aux{
		Domains: r.Domains,
	}

	for _, rqmMod := range r.RequestModifications {
		// Assuming getFunctionName returns a string representation of the function
		fnName := getFunctionName(rqmMod)
		aux.RequestModifications = append(aux.RequestModifications, fnName)
	}

	for _, rsmMod := range r.ResponseModifications {
		fnName := getFunctionName(rsmMod)
		aux.ResponseModifications = append(aux.ResponseModifications, fnName)
	}

	return aux, nil
}
