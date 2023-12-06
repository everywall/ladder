package cli

import (
	"fmt"
	"io"
	"os"

	"ladder/proxychain/ruleset"
)

// HandleRulesetMerge merges a set of ruleset files, specified by the rulesetPath or RULESET env variable, into either YAML or Gzip format.
// Exits the program with an error message if the ruleset path is not provided or if loading the ruleset fails.
//
// Parameters:
// - rulesetPath: Specifies the path to the ruleset file.
// - mergeRulesets: Indicates if a merge operation should be performed.
// - useGzip: Indicates if the merged rulesets should be gzip-ped.
// - output: Specifies the output file. If nil, stdout will be used.
//
// Returns:
// - An error if the ruleset loading or merging process fails, otherwise nil.
func HandleRulesetMerge(rulesetPath string, mergeRulesets bool, output *os.File) error {
	if !mergeRulesets {
		return nil
	}

	if rulesetPath == "" {
		rulesetPath = os.Getenv("RULESET")
	}

	if rulesetPath == "" {
		fmt.Println("error: no ruleset provided. Try again with --ruleset <ruleset.yaml>")
		os.Exit(1)
	}

	rs, err := ruleset_v2.NewRuleset(rulesetPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return yamlMerge(rs, output)
}

// yamlMerge takes a RuleSet and an optional output file path pointer. It converts the RuleSet into YAML format.
// If the output file path is provided, the YAML data is written to this file. If not, the YAML data is printed to stdout.
//
// Parameters:
// - rs: The ruleset.RuleSet to be converted to YAML.
// - output: The output for the merged data. If nil, stdout will be used.
//
// Returns:
// - An error if YAML conversion or file writing fails, otherwise nil.
func yamlMerge(rs ruleset_v2.Ruleset, output io.Writer) error {
	yaml, err := rs.YAML()
	if err != nil {
		return err
	}

	if output == nil {
		fmt.Println(yaml)
		os.Exit(0)
	}

	_, err = io.WriteString(output, yaml)
	if err != nil {
		return fmt.Errorf("failed to write merged YAML ruleset: %v", err)
	}

	return nil
}
