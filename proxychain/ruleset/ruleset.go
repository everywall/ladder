package ruleset_v2

import (
	//"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"encoding/json"
	"gopkg.in/yaml.v3"
)

type IRuleset interface {
	YAML() (string, error)
	JSON() (string, error)
	HasRule(url *url.URL) bool
	GetRule(url *url.URL) (rule *Rule, exists bool)
}

type Ruleset struct {
	Rules    []Rule           `json:"rules" yaml:"rules"`
	_rulemap map[string]*Rule // internal map for fast lookups; points at a rule in the Rules slice
}

func (rs *Ruleset) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type AuxRuleset struct {
		Rules []Rule `yaml:"rules"`
	}
	yamlRuleset := &AuxRuleset{}

	if err := unmarshal(&yamlRuleset); err != nil {
		// if there is no top-level rule key, we'll try to unmarshal as if it is just a bare rule
		recovered := false
		yamlRule := &Rule{}
		if err := unmarshal(&yamlRule); err != nil {
			yamlRuleset.Rules = append(yamlRuleset.Rules, *yamlRule)
			recovered = true
		}

		if !recovered {
			return err
		}
	}

	rs._rulemap = make(map[string]*Rule)
	rs.Rules = yamlRuleset.Rules

	// create a map of pointers to rules loaded above based on domain string keys
	// this way we don't have two copies of the rule in ruleset
	for i, rule := range rs.Rules {
		rulePtr := &rs.Rules[i]
		for _, domain := range rule.Domains {
			rs._rulemap[domain] = rulePtr
			if !strings.HasPrefix(domain, "www.") {
				rs._rulemap["www."+domain] = rulePtr
			}
		}
	}

	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
// It customizes the marshaling of a Ruleset object into YAML
func (rs *Ruleset) MarshalYAML() (interface{}, error) {

	type AuxRule struct {
		Domains               []string `yaml:"domains"`
		RequestModifications  []_rqm   `yaml:"requestmodifications"`
		ResponseModifications []_rsm   `yaml:"responsemodifications"`
	}

	type Aux struct {
		Rules []AuxRule `yaml:"rules"`
	}

	aux := Aux{}

	for _, rule := range rs.Rules {
		auxRule := AuxRule{
			Domains:               rule.Domains,
			RequestModifications:  rule._rqms,
			ResponseModifications: rule._rsms,
		}
		aux.Rules = append(aux.Rules, auxRule)
	}
	return aux, nil

	/*
		var b bytes.Buffer
		y := yaml.NewEncoder(&b)
		y.SetIndent(2)
		err := y.Encode(&aux)

		return b.String(), err
	*/
}

// ==========================================================

func (rs *Ruleset) UnmarshalJSON(data []byte) error {
	type AuxRuleset struct {
		Rules []Rule `json:"rules"`
	}
	ar := &AuxRuleset{}

	if err := json.Unmarshal(data, ar); err != nil {
		return err
	}

	rs._rulemap = make(map[string]*Rule)
	rs.Rules = ar.Rules

	for i, rule := range rs.Rules {
		rulePtr := &rs.Rules[i]
		for _, domain := range rule.Domains {
			rs._rulemap[domain] = rulePtr
			if !strings.HasPrefix(domain, "www.") {
				rs._rulemap["www."+domain] = rulePtr
			}
		}
	}

	return nil
}

func (rs *Ruleset) MarshalJSON() ([]byte, error) {
	type AuxRule struct {
		Domains               []string `json:"domains"`
		RequestModifications  []_rqm   `json:"requestmodifications"`
		ResponseModifications []_rsm   `json:"responsemodifications"`
	}

	type Aux struct {
		Rules []AuxRule `json:"rules"`
	}

	aux := Aux{}
	for _, rule := range rs.Rules {
		auxRule := AuxRule{
			Domains:               rule.Domains,
			RequestModifications:  rule._rqms,
			ResponseModifications: rule._rsms,
		}
		aux.Rules = append(aux.Rules, auxRule)
	}

	return json.Marshal(aux)
}

// ===========================================================

func (rs Ruleset) GetRule(url *url.URL) (rule *Rule, exists bool) {
	rule, exists = rs._rulemap[url.Hostname()]
	return rule, exists
}

func (rs Ruleset) HasRule(url *url.URL) bool {
	_, exists := rs.GetRule(url)
	return exists
}

// NewRuleset loads a new RuleSet from a path
func NewRuleset(path string) (Ruleset, error) {
	rs := Ruleset{
		_rulemap: map[string]*Rule{},
		Rules:    []Rule{},
	}

	switch {
	case strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://"):
		err := rs.loadRulesFromRemoteFile(path)
		return rs, err
	default:
		err := rs.loadRulesFromLocalDir(path)
		return rs, err
	}
}

// NewRulesetFromEnv creates a new RuleSet based on the RULESET environment variable.
// It logs a warning and returns an empty RuleSet if the RULESET environment variable is not set.
// If the RULESET is set but the rules cannot be loaded, it panics.
func NewRulesetFromEnv() Ruleset {
	rulesPath, ok := os.LookupEnv("RULESET")
	if !ok {
		log.Printf("WARN: No ruleset specified. Set the `RULESET` environment variable to load one for a better success rate.")
		return Ruleset{}
	}

	ruleSet, err := NewRuleset(rulesPath)
	if err != nil {
		log.Println(err)
	}

	return ruleSet
}

// loadRulesFromLocalDir loads rules from a local directory specified by the path.
// It walks through the directory, loading rules from YAML files.
// Returns an error if the directory cannot be accessed
// If there is an issue loading any file, it will be skipped
func (rs *Ruleset) loadRulesFromLocalDir(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("loadRulesFromLocalDir: invalid path - %s", err)
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		isYAML := filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml"
		if !isYAML {
			return nil
		}

		tmpRs := Ruleset{_rulemap: make(map[string]*Rule)}
		err = tmpRs.loadRulesFromLocalFile(path)
		if err != nil {
			log.Printf("WARN: failed to load directory ruleset '%s': %s, skipping", path, err)
			return nil
		}
		rs.Rules = append(rs.Rules, tmpRs.Rules...)

		//log.Printf("INFO: loaded ruleset %s\n", path)

		return nil
	})

	// create a map of pointers to rules loaded above based on domain string keys
	// this way we don't have two copies of the rule in ruleset
	if rs._rulemap == nil {
		rs._rulemap = make(map[string]*Rule)
	}
	for i, rule := range rs.Rules {
		rulePtr := &rs.Rules[i]
		for _, domain := range rule.Domains {
			rs._rulemap[domain] = rulePtr
			if !strings.HasPrefix(domain, "www.") {
				rs._rulemap["www."+domain] = rulePtr
			}
		}
	}

	if err != nil {
		return err
	}

	return nil
}

// loadRulesFromLocalFile loads rules from a local YAML file specified by the path.
// Returns an error if the file cannot be read or if there's a syntax error in the YAML.
func (rs *Ruleset) loadRulesFromLocalFile(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		e := fmt.Errorf("failed to read rules from local file: '%s'", path)
		return errors.Join(e, err)
	}

	isJSON := strings.HasSuffix(path, ".json")
	if isJSON {
		err = json.Unmarshal(file, rs)
	} else {
		err = yaml.Unmarshal(file, rs)
	}

	if err != nil {
		e := fmt.Errorf("failed to load rules from local file, possible syntax error in '%s' - %s", path, err)
		debugPrintRule(string(file), e)
		return e
	}

	return nil
}

// loadRulesFromRemoteFile loads rules from a remote URL.
// It supports plain and gzip compressed content.
// Returns an error if there's an issue accessing the URL or if there's a syntax error in the YAML.
func (rs *Ruleset) loadRulesFromRemoteFile(rulesURL string) error {

	resp, err := http.Get(rulesURL)
	if err != nil {
		return fmt.Errorf("failed to load rules from remote url '%s' - %s", rulesURL, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to load rules from remote url (%s) on '%s' - %s", resp.Status, rulesURL, err)
	}

	var reader io.Reader

	// in case remote server did not set content-encoding gzip header
	isGzip := strings.HasSuffix(rulesURL, ".gz") || strings.HasSuffix(rulesURL, ".gzip") || resp.Header.Get("content-encoding") == "gzip"
	if isGzip {
		reader, err = gzip.NewReader(resp.Body)

		if err != nil {
			return fmt.Errorf("failed to create gzip reader for URL '%s' with status code '%s': %w", rulesURL, resp.Status, err)
		}
	} else {
		reader = resp.Body
	}

	isJSON := strings.HasSuffix(rulesURL, ".json") || resp.Header.Get("content-type") == "application/json"
	if isJSON {
		err = json.NewDecoder(reader).Decode(&rs)
	} else {
		err = yaml.NewDecoder(reader).Decode(&rs)
	}

	if err != nil {
		return fmt.Errorf("failed to load rules from remote url '%s' with status code '%s' and possible syntax error - %s", rulesURL, resp.Status, err)
	}

	return nil
}

// ================= utility methods ==========================

// YAML returns the ruleset as a Yaml string
func (rs Ruleset) YAML() (string, error) {
	yml, err := yaml.Marshal(&rs)
	if err != nil {
		return "", err
	}
	return string(yml), nil
}

// JSON returns the ruleset as a JSON string
func (rs Ruleset) JSON() (string, error) {
	jsn, err := json.Marshal(&rs)
	if err != nil {
		return "", err
	}
	return string(jsn), nil
}

// Domains extracts and returns a slice of all domains present in the RuleSet.
func (rs *Ruleset) Domains() []string {
	var domains []string
	for _, rule := range rs.Rules {
		domains = append(domains, rule.Domains...)
	}
	return domains
}

// DomainCount returns the count of unique domains present in the RuleSet.
func (rs *Ruleset) DomainCount() int {
	return len(rs.Domains())
}

// Count returns the total number of rules in the RuleSet.
func (rs *Ruleset) Count() int {
	return len(rs.Rules)
}

// PrintStats logs the number of rules and domains loaded in the RuleSet.
func (rs *Ruleset) PrintStats() {
	log.Printf("INFO: Loaded %d rules for %d domains\n", rs.Count(), rs.DomainCount())
}

// debugPrintRule is a utility function for printing a rule and associated error for debugging purposes.
func debugPrintRule(rule string, err error) {
	fmt.Println("------------------------------ BEGIN DEBUG RULESET -----------------------------")
	fmt.Printf("%s\n", err.Error())
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(rule)
	fmt.Println("------------------------------ END DEBUG RULESET -------------------------------")
}
