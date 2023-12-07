package handlers

// TODO: this is a work in progress.
// Needs codegen to populate the rqm and rsm structs from available modifiers in ladder/proxychain/*modifers/*.go

import (
	"ladder/proxychain/responsemodifiers/api"
)

type ModifiersAPIResponse struct {
	Success bool      `json:"success"`
	Error   api.Error `json:"error"`
	Result  Modifiers `json:"result"`
}

type Modifiers struct {
	RequestModifiers  []Modifier `json:"requestmodifiers"`
	ResponseModifiers []Modifier `json:"responsemodifiers"`
}

type Modifier struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Params      []Param `json:"params"`
}

type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func init() {

	rqm := []Modifier{}

	// ========= loop
	rqm = append(rqm, Modifier{
		Name:        "codegen_name",
		Description: "codegen_description",
		Params: []Param{
			{Name: "codegen_name1", Type: "codegen_type1"},
			{Name: "codegen_name2", Type: "codegen_type2"},
		},
	})
	// ========= loop end

	rsm := []Modifier{}

	// ========= loop
	rsm = append(rsm, Modifier{
		Name:        "codegen_name",
		Description: "codegen_description",
		Params: []Param{
			{Name: "codegen_name1", Type: "codegen_type1"},
			{Name: "codegen_name2", Type: "codegen_type2"},
		},
	})
	// ========= loop end

	m := &ModifiersAPIResponse{Success: true}
	m.Result.RequestModifiers = rqm
	m.Result.ResponseModifiers = rsm

}
