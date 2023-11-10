package handlers

type Regex struct {
	Match   string `yaml:"match"`
	Replace string `yaml:"replace"`
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
	GoogleCache bool    `yaml:"googleCache,omitempty"`
	RegexRules  []Regex `yaml:"regexRules"`
	Injections  []struct {
		Position string `yaml:"position"`
		Append   string `yaml:"append"`
		Prepend  string `yaml:"prepend"`
		Replace  string `yaml:"replace"`
	} `yaml:"injections"`
}
