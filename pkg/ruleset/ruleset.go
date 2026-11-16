package ruleset

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Regex struct {
	Match   string `yaml:"match"`
	Replace string `yaml:"replace"`
}
type KV struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type RuleSet []Rule

type Rule struct {
	Domain  string   `yaml:"domain,omitempty"`
	Domains []string `yaml:"domains,omitempty"`
	Paths   []string `yaml:"paths,omitempty"`
	Headers struct {
		UserAgent     string `yaml:"user-agent,omitempty"`
		XForwardedFor string `yaml:"x-forwarded-for,omitempty"`
		Referer       string `yaml:"referer,omitempty"`
		Cookie        string `yaml:"cookie,omitempty"`
		CSP           string `yaml:"content-security-policy,omitempty"`
	} `yaml:"headers,omitempty"`
	GoogleCache     bool    `yaml:"googleCache,omitempty"`
	UseFlareSolverr bool    `yaml:"useFlareSolverr,omitempty"`
	RegexRules      []Regex `yaml:"regexRules,omitempty"`

	URLMods struct {
		Domain []Regex `yaml:"domain,omitempty"`
		Path   []Regex `yaml:"path,omitempty"`
		Query  []KV    `yaml:"query,omitempty"`
	} `yaml:"urlMods,omitempty"`

	Injections []struct {
		Position string `yaml:"position,omitempty"`
		Append   string `yaml:"append,omitempty"`
		Prepend  string `yaml:"prepend,omitempty"`
		Replace  string `yaml:"replace,omitempty"`
	} `yaml:"injections,omitempty"`
}

// NewRulesetFromEnv creates a new RuleSet based on the RULESET environment variable.
// It logs a warning and returns an empty RuleSet if the RULESET environment variable is not set.
// Legacy callers receive a logged error; new callers use the error-returning NewRuleset.
func NewRulesetFromEnv() RuleSet {
	rulesPath, ok := os.LookupEnv("RULESET")
	if !ok {
		log.Printf("WARN: No ruleset specified. Set the `RULESET` environment variable to load one for a better success rate.")
		return RuleSet{}
	}

	ruleSet, err := NewRuleset(rulesPath)
	if err != nil {
		log.Println(err)
	}

	return ruleSet
}

// NewRuleset loads a RuleSet from a given string of rule paths, separated by semicolons.
// It supports loading rules from both local file paths and remote URLs.
// Returns a RuleSet and an error if any issues occur during loading.
func NewRuleset(rulePaths string) (RuleSet, error) {
	var ruleSet RuleSet

	var errs []error

	rp := strings.Split(rulePaths, ";")
	for _, rule := range rp {
		var err error

		rulePath := strings.TrimSpace(rule)
		if rulePath == "" {
			continue
		}
		isRemote := strings.HasPrefix(rulePath, "http://") || strings.HasPrefix(rulePath, "https://")

		if isRemote {
			err = ruleSet.loadRulesFromRemoteFile(rulePath)
		} else {
			err = ruleSet.loadRulesFromLocalDir(rulePath)
		}

		if err != nil {
			e := fmt.Errorf("WARN: failed to load ruleset from '%s'", rulePath)
			errs = append(errs, errors.Join(e, err))

			continue
		}
	}

	if len(errs) != 0 {
		e := fmt.Errorf("WARN: failed to load %d rulesets", len(rp))
		errs = append(errs, e)

		// else, bubble up any errors, such as syntax or remote host issues
		return ruleSet, errors.Join(errs...)
	}

	if err := ruleSet.Validate(); err != nil {
		return nil, err
	}
	return ruleSet, nil
}

// ================== RULESET loading logic ===================================

// loadRulesFromLocalDir loads rules from a local directory specified by the path.
// It walks through the directory, loading rules from YAML files.
// Returns an error if the directory cannot be accessed
// A malformed file fails the load rather than silently dropping that rule.
func (rs *RuleSet) loadRulesFromLocalDir(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}

	yamlRegex := regexp.MustCompile(`(?i)\.ya?ml$`)

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if isYaml := yamlRegex.MatchString(path); !isYaml {
			return nil
		}

		err = rs.loadRulesFromLocalFile(path)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// loadRulesFromLocalFile loads rules from a local YAML file specified by the path.
// Returns an error if the file cannot be read or if there's a syntax error in the YAML.
func (rs *RuleSet) loadRulesFromLocalFile(path string) error {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		e := fmt.Errorf("failed to read rules from local file: '%s'", path)
		return errors.Join(e, err)
	}

	var r RuleSet
	err = yaml.Unmarshal(yamlFile, &r)

	if err != nil {
		e := fmt.Errorf("failed to load rules from local file, possible syntax error in '%s'", path)
		ee := errors.Join(e, err)

		return ee
	}

	if err := r.Validate(); err != nil {
		return err
	}
	*rs = append(*rs, r...)

	return nil
}

// loadRulesFromRemoteFile loads rules from a remote URL.
// It supports plain and gzip compressed content.
// Returns an error if there's an issue accessing the URL or if there's a syntax error in the YAML.
func (rs *RuleSet) loadRulesFromRemoteFile(rulesURL string) error {
	var r RuleSet
	if _, err := ParseURL(rulesURL); err != nil {
		return fmt.Errorf("invalid remote ruleset URL")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(rulesURL)
	if err != nil {
		e := fmt.Errorf("failed to load rules from remote url '%s'", rulesURL)
		return errors.Join(e, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := fmt.Errorf("failed to load rules from remote url (%s) on '%s'", resp.Status, rulesURL)
		return errors.Join(e, err)
	}

	var reader io.Reader

	isGzip := strings.HasSuffix(rulesURL, ".gz") || strings.HasSuffix(rulesURL, ".gzip") || resp.Header.Get("content-encoding") == "gzip"

	if isGzip {
		reader, err = gzip.NewReader(resp.Body)

		if err != nil {
			return fmt.Errorf("failed to create gzip reader for URL '%s' with status code '%s': %w", rulesURL, resp.Status, err)
		}
	} else {
		reader = resp.Body
	}

	data, readErr := io.ReadAll(io.LimitReader(reader, 4*1024*1024+1))
	if readErr != nil {
		return fmt.Errorf("remote ruleset read failed")
	}
	if len(data) > 4*1024*1024 {
		return fmt.Errorf("remote ruleset exceeds 4 MiB")
	}
	err = yaml.Unmarshal(data, &r)

	if err != nil {
		e := fmt.Errorf("failed to load rules from remote url '%s' with status code '%s' and possible syntax error", rulesURL, resp.Status)
		ee := errors.Join(e, err)

		return ee
	}

	if err := r.Validate(); err != nil {
		return err
	}
	*rs = append(*rs, r...)

	return nil
}

// ================= utility methods ==========================

// Yaml returns the ruleset as a Yaml string
func (rs *RuleSet) Yaml() (string, error) {
	y, err := yaml.Marshal(rs)
	if err != nil {
		return "", err
	}

	return string(y), nil
}

// GzipYaml returns an io.Reader that streams the Gzip-compressed YAML representation of the RuleSet.
func (rs *RuleSet) GzipYaml() (io.Reader, error) {
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
func (rs *RuleSet) Domains() []string {
	var domains []string
	for _, rule := range *rs {
		domains = append(domains, rule.Domain)
		domains = append(domains, rule.Domains...)
	}
	return domains
}

// DomainCount returns the count of unique domains present in the RuleSet.
func (rs *RuleSet) DomainCount() int {
	return len(rs.Domains())
}

// Count returns the total number of rules in the RuleSet.
func (rs *RuleSet) Count() int {
	return len(*rs)
}

// PrintStats logs the number of rules and domains loaded in the RuleSet.
func (rs *RuleSet) PrintStats() {
	log.Printf("INFO: Loaded %d rules for %d domains\n", rs.Count(), rs.DomainCount())
}

// debugPrintRule is a utility function for printing a rule and associated error for debugging purposes.
func debugPrintRule(rule string, err error) {
	log.Printf("ruleset parse failed (configuration values omitted): %v", err)
}
