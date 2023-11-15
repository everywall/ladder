package cli

import (
	"fmt"
	"io"
	"io/fs"
	"ladder/pkg/ruleset"
	"os"

	"golang.org/x/term"
)

// HandleRulesetMerge merges a set of ruleset files, specified by the rulesetPath or RULESET env variable, into either YAML or Gzip format.
// Exits the program with an error message if the ruleset path is not provided or if loading the ruleset fails.
//
// Parameters:
// - rulesetPath: A pointer to a string specifying the path to the ruleset file.
// - mergeRulesets: A pointer to a boolean indicating if a merge operation should be performed.
// - mergeRulesetsGzip: A pointer to a boolean indicating if the merge should be in Gzip format.
// - mergeRulesetsOutput: A pointer to a string specifying the output file path. If empty, the output is printed to stdout.
//
// Returns:
// - An error if the ruleset loading or merging process fails, otherwise nil.
func HandleRulesetMerge(rulesetPath *string, mergeRulesets *bool, mergeRulesetsGzip *bool, mergeRulesetsOutput *string) error {
	if *rulesetPath == "" {
		*rulesetPath = os.Getenv("RULESET")
	}
	if *rulesetPath == "" {
		fmt.Println("ERROR: no ruleset provided. Try again with --ruleset <ruleset.yaml>")
		os.Exit(1)
	}

	rs, err := ruleset.NewRuleset(*rulesetPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *mergeRulesetsGzip {
		return gzipMerge(rs, mergeRulesetsOutput)
	}
	return yamlMerge(rs, mergeRulesetsOutput)
}

// gzipMerge takes a RuleSet and an optional output file path pointer. It compresses the RuleSet into Gzip format.
// If the output file path is provided, the compressed data is written to this file. Otherwise, it prints a warning
// and outputs the binary data to stdout
//
// Parameters:
// - rs: The ruleset.RuleSet to be compressed.
// - mergeRulesetsOutput: A pointer to a string specifying the output file path. If empty, the output is directed to stdout.
//
// Returns:
// - An error if compression or file writing fails, otherwise nil.
func gzipMerge(rs ruleset.RuleSet, mergeRulesetsOutput *string) error {
	gzip, err := rs.GzipYaml()
	if err != nil {
		return err
	}

	if *mergeRulesetsOutput != "" {
		out, err := os.Create(*mergeRulesetsOutput)
		defer out.Close()
		_, err = io.Copy(out, gzip)
		if err != nil {
			return err
		}
	}

	if term.IsTerminal(int(os.Stdout.Fd())) {
		println("WARNING: binary output can mess up your terminal. Use '--merge-rulesets-output <ruleset.gz>' or pipe it to a file.")
		os.Exit(1)
	}
	_, err = io.Copy(os.Stdout, gzip)
	if err != nil {
		return err
	}
	return nil
}

// yamlMerge takes a RuleSet and an optional output file path pointer. It converts the RuleSet into YAML format.
// If the output file path is provided, the YAML data is written to this file. If not, the YAML data is printed to stdout.
//
// Parameters:
// - rs: The ruleset.RuleSet to be converted to YAML.
// - mergeRulesetsOutput: A pointer to a string specifying the output file path. If empty, the output is printed to stdout.
//
// Returns:
// - An error if YAML conversion or file writing fails, otherwise nil.
func yamlMerge(rs ruleset.RuleSet, mergeRulesetsOutput *string) error {
	yaml, err := rs.Yaml()
	if err != nil {
		return err
	}
	if *mergeRulesetsOutput == "" {
		fmt.Printf(yaml)
		os.Exit(0)
	}

	err = os.WriteFile(*mergeRulesetsOutput, []byte(yaml), fs.FileMode(os.O_RDWR))
	if err != nil {
		return fmt.Errorf("ERROR: failed to write merged YAML ruleset to '%s'\n", *mergeRulesetsOutput)
	}
	return nil
}
