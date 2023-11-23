package ruleset

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

var (
	validYAML = `
- domain: example.com
  regexRules:
    - match: "^http:"
      replace: "https:"`

	invalidYAML = `
- domain: [thisIsATestYamlThatIsMeantToFail.example]
  regexRules:
    - match: "^http:"
      replace: "https:"
    - match: "[incomplete"`
)

func TestLoadRulesFromRemoteFile(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	app.Get("/valid-config.yml", func(c *fiber.Ctx) error {
		c.SendString(validYAML)
		return nil
	})

	app.Get("/invalid-config.yml", func(c *fiber.Ctx) error {
		c.SendString(invalidYAML)
		return nil
	})

	app.Get("/valid-config.gz", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/octet-stream")

		rs, err := loadRuleFromString(validYAML)
		if err != nil {
			t.Errorf("failed to load valid yaml from string: %s", err.Error())
		}

		s, err := rs.GzipYaml()
		if err != nil {
			t.Errorf("failed to load gzip serialize yaml: %s", err.Error())
		}

		err = c.SendStream(s)
		if err != nil {
			t.Errorf("failed to stream gzip serialized yaml: %s", err.Error())
		}
		return nil
	})

	// Start the server in a goroutine
	go func() {
		if err := app.Listen("127.0.0.1:9999"); err != nil {
			t.Errorf("Server failed to start: %s", err.Error())
		}
	}()

	// Wait for the server to start
	time.Sleep(time.Second * 1)

	rs, err := NewRuleset("http://127.0.0.1:9999/valid-config.yml")
	if err != nil {
		t.Errorf("failed to load plaintext ruleset from http server: %s", err.Error())
	}

	assert.Equal(t, rs[0].Domain, "example.com")

	rs, err = NewRuleset("http://127.0.0.1:9999/valid-config.gz")
	if err != nil {
		t.Errorf("failed to load gzipped ruleset from http server: %s", err.Error())
	}

	assert.Equal(t, rs[0].Domain, "example.com")

	os.Setenv("RULESET", "http://127.0.0.1:9999/valid-config.gz")

	rs = NewRulesetFromEnv()
	if !assert.Equal(t, rs[0].Domain, "example.com") {
		t.Error("expected no errors loading ruleset from gzip url using environment variable, but got one")
	}
}

func loadRuleFromString(yaml string) (RuleSet, error) {
	// Create a temporary file and load it
	tmpFile, _ := os.CreateTemp("", "ruleset*.yaml")

	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(yaml)

	rs := RuleSet{}
	err := rs.loadRulesFromLocalFile(tmpFile.Name())

	return rs, err
}

// TestLoadRulesFromLocalFile tests the loading of rules from a local YAML file.
func TestLoadRulesFromLocalFile(t *testing.T) {
	rs, err := loadRuleFromString(validYAML)
	if err != nil {
		t.Errorf("Failed to load rules from valid YAML: %s", err)
	}

	assert.Equal(t, rs[0].Domain, "example.com")
	assert.Equal(t, rs[0].RegexRules[0].Match, "^http:")
	assert.Equal(t, rs[0].RegexRules[0].Replace, "https:")

	_, err = loadRuleFromString(invalidYAML)
	if err == nil {
		t.Errorf("Expected an error when loading invalid YAML, but got none")
	}
}

// TestLoadRulesFromLocalDir tests the loading of rules from a local nested directory full of yaml rulesets
func TestLoadRulesFromLocalDir(t *testing.T) {
	// Create a temporary directory
	baseDir, err := os.MkdirTemp("", "ruleset_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %s", err)
	}

	defer os.RemoveAll(baseDir)

	// Create a nested subdirectory
	nestedDir := filepath.Join(baseDir, "nested")
	err = os.Mkdir(nestedDir, 0o755)

	if err != nil {
		t.Fatalf("Failed to create nested directory: %s", err)
	}

	// Create a nested subdirectory
	nestedTwiceDir := filepath.Join(nestedDir, "nestedTwice")
	err = os.Mkdir(nestedTwiceDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create twice-nested directory: %s", err)
	}

	testCases := []string{"test.yaml", "test2.yaml", "test-3.yaml", "test 4.yaml", "1987.test.yaml.yml", "foobar.example.com.yaml", "foobar.com.yml"}
	for _, fileName := range testCases {
		filePath := filepath.Join(nestedDir, "2x-"+fileName)
		os.WriteFile(filePath, []byte(validYAML), 0o644)

		filePath = filepath.Join(nestedDir, fileName)
		os.WriteFile(filePath, []byte(validYAML), 0o644)

		filePath = filepath.Join(baseDir, "base-"+fileName)
		os.WriteFile(filePath, []byte(validYAML), 0o644)
	}

	rs := RuleSet{}
	err = rs.loadRulesFromLocalDir(baseDir)

	assert.NoError(t, err)
	assert.Equal(t, rs.Count(), len(testCases)*3)

	for _, rule := range rs {
		assert.Equal(t, rule.Domain, "example.com")
		assert.Equal(t, rule.RegexRules[0].Match, "^http:")
		assert.Equal(t, rule.RegexRules[0].Replace, "https:")
	}
}
