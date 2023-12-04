package ruleset_v2

import (
	"net/url"
)

type IRuleset interface {
	HasRule(url url.URL) bool
	GetRule(url url.URL) (rule Rule, exists bool)
}

type Ruleset struct {
	rulesetPath string
	rules       map[string]Rule
}

func (rs Ruleset) GetRule(url url.URL) (rule Rule, exists bool) {
	rule, exists = rs.rules[url.Hostname()]
	return rule, exists
}

func (rs Ruleset) HasRule(url url.URL) bool {
	_, exists := rs.GetRule(url)
	return exists
}

func NewRuleset(path string) (Ruleset, error) {
	rs := Ruleset{
		rulesetPath: path,
	}
	return rs, nil
}
