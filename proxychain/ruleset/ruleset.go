package ruleset_v2

import (
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

	"gopkg.in/yaml.v3"
)

type IRuleset interface {
	HasRule(url url.URL) bool
	GetRule(url url.URL) (rule Rule, exists bool)
}

type Ruleset struct {
	rulesetPath string
	rules       map[string]Rule
}

func (rs Ruleset) GetRule(url *url.URL) (rule Rule, exists bool) {
	rule, exists = rs.rules[url.Hostname()]
	return rule, exists
}

func (rs Ruleset) HasRule(url *url.URL) bool {
	_, exists := rs.GetRule(url)
	return exists
}

// NewRuleset loads a new RuleSet from a path
func NewRuleset(path string) (Ruleset, error) {
	rs := Ruleset{
		rulesetPath: path,
		rules:       map[string]Rule{},
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

		isYAML := filepath.Ext(path) == "yaml" || filepath.Ext(path) == "yml"
		if !isYAML {
			return nil
		}

		err = rs.loadRulesFromLocalFile(path)
		if err != nil {
			log.Printf("WARN: failed to load directory ruleset '%s': %s, skipping", path, err)
			return nil
		}

		log.Printf("INFO: loaded ruleset %s\n", path)

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// loadRulesFromLocalFile loads rules from a local YAML file specified by the path.
// Returns an error if the file cannot be read or if there's a syntax error in the YAML.
func (rs *Ruleset) loadRulesFromLocalFile(path string) error {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		e := fmt.Errorf("failed to read rules from local file: '%s'", path)
		return errors.Join(e, err)
	}

	rule := Rule{}
	err = yaml.Unmarshal(yamlFile, &rule)

	if err != nil {
		e := fmt.Errorf("failed to load rules from local file, possible syntax error in '%s' - %s", path, err)
		ee := errors.Join(e, err)
		debugPrintRule(string(yamlFile), ee)
		return ee
	}

	for _, domain := range rule.Domains {
		rs.rules[domain] = rule
		if !strings.HasSuffix(domain, "www.") {
			rs.rules["www."+domain] = rule
		}
	}

	return nil
}

// loadRulesFromRemoteFile loads rules from a remote URL.
// It supports plain and gzip compressed content.
// Returns an error if there's an issue accessing the URL or if there's a syntax error in the YAML.
func (rs *Ruleset) loadRulesFromRemoteFile(rulesURL string) error {
	rule := Rule{}

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
	isGzip := strings.HasSuffix(rulesURL, ".gz") || strings.HasSuffix(rulesURL, ".gzip")
	if isGzip {
		reader, err = gzip.NewReader(resp.Body)

		if err != nil {
			return fmt.Errorf("failed to create gzip reader for URL '%s' with status code '%s': %w", rulesURL, resp.Status, err)
		}
	} else {
		reader = resp.Body
	}

	err = yaml.NewDecoder(reader).Decode(&rule)

	if err != nil {
		return fmt.Errorf("failed to load rules from remote url '%s' with status code '%s' and possible syntax error - %s", rulesURL, resp.Status, err)
	}

	if rs.rules == nil {
		fmt.Println("nilmap")
		rs.rules = make(map[string]Rule)
	}

	for _, domain := range rule.Domains {
		rs.rules[domain] = rule
		if !strings.HasSuffix(domain, "www.") {
			rs.rules["www."+domain] = rule
		}
	}

	return nil
}

// ================= utility methods ==========================

// Yaml returns the ruleset as a Yaml string
func (rs *Ruleset) Yaml() (string, error) {
	y, err := yaml.Marshal(rs)
	if err != nil {
		return "", err
	}

	return string(y), nil
}

// GzipYaml returns an io.Reader that streams the Gzip-compressed YAML representation of the RuleSet.
func (rs *Ruleset) GzipYaml() (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		gw := gzip.NewWriter(pw)
		defer gw.Close()

		if err := yaml.NewEncoder(gw).Encode(rs); err != nil {
			gw.Close() // Ensure to close the gzip writer
			pw.CloseWithError(err)
			return
		}
	}()

	return pr, nil
}

// Domains extracts and returns a slice of all domains present in the RuleSet.
func (rs *Ruleset) Domains() []string {
	var domains []string
	for domain := range rs.rules {
		domains = append(domains, domain)
	}
	return domains
}

// DomainCount returns the count of unique domains present in the RuleSet.
func (rs *Ruleset) DomainCount() int {
	return len(rs.Domains())
}

// Count returns the total number of rules in the RuleSet.
func (rs *Ruleset) Count() int {
	return len(rs.rules)
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
