package service

import (
	"context"
	"ladder/pkg/ruleset"
	"net/http"
	"time"
)

type Request struct {
	URL   string `json:"url"`
	Cache string `json:"cache,omitempty"`
}
type CacheInfo struct {
	Hit        bool      `json:"hit"`
	AgeSeconds int64     `json:"ageSeconds"`
	ExpiresAt  time.Time `json:"expiresAt"`
	TTLSeconds int64     `json:"ttlSeconds"`
}
type RuleMatch struct {
	PackID      string `json:"packId"`
	PackVersion string `json:"packVersion"`
	RuleID      string `json:"ruleId"`
}
type Metadata struct {
	SourceURL      string      `json:"sourceUrl"`
	FinalURL       string      `json:"finalUrl"`
	SourceHost     string      `json:"sourceHost"`
	UpstreamStatus int         `json:"upstreamStatus"`
	MediaType      string      `json:"mediaType"`
	FetchedAt      time.Time   `json:"fetchedAt"`
	ServedAt       time.Time   `json:"servedAt"`
	Cache          CacheInfo   `json:"cache"`
	Rules          []RuleMatch `json:"rules"`
}
type FetchResult struct {
	Metadata      Metadata
	Header        http.Header
	RequestHeader http.Header
	Body          []byte
}
type Content struct {
	Markdown string `json:"markdown"`
	Text     string `json:"text"`
	HTML     string `json:"html"`
}
type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Document struct {
	Metadata Metadata  `json:"metadata"`
	Title    string    `json:"title"`
	Byline   string    `json:"byline,omitempty"`
	Content  Content   `json:"content"`
	Problems []Problem `json:"problems"`
}
type PackInfo struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Rules   int    `json:"rules"`
}
type CheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}
type TestResult struct {
	Metadata Metadata      `json:"metadata"`
	Matched  []RuleMatch   `json:"matched"`
	Checks   []CheckResult `json:"checks"`
	Passed   bool          `json:"passed"`
}
type Config struct {
	Client          *http.Client
	Rules           ruleset.RuleSet
	PackPaths       []string
	UserAgent       string
	ForwardedFor    string
	AllowedDomains  []string
	Timeout         time.Duration
	MaxBodyBytes    int64
	CacheTTL        time.Duration
	CacheMaxEntries int
	CacheMaxBytes   int64
	FlareSolverrURL string
	Now             func() time.Time
}

// The implementation is shared by the web server, CLI and stdio connector.
type Reader interface {
	Read(context.Context, Request) (Document, error)
}
