package service

import (
	"context"
	"html"
	"strings"

	"ladder/pkg/reader"
)

// TransformHTML applies only explicitly enabled pack document operations.
// Legacy regex/injection fields were never called by the proxy and remain
// inactive. The raw Fetch and Raw adapter always retain original bytes.
func (s *Service) TransformHTML(f FetchResult) ([]byte, error) {
	if !IsHTML(f.Metadata.MediaType) {
		return append([]byte{}, f.Body...), nil
	}
	u, err := s.validate(f.Metadata.SourceURL)
	if err != nil {
		return nil, err
	}
	body := append([]byte{}, f.Body...)
	for _, r := range s.selectRules(u) {
		body, err = r.rule.ApplyHTML(body)
		if err != nil {
			return nil, failure("rule_transform", 422, "pack HTML transformation failed")
		}
	}
	return body, nil
}
func IsHTML(media string) bool { return media == "text/html" || media == "application/xhtml+xml" }
func (s *Service) Read(ctx context.Context, request Request) (Document, error) {
	fetched, err := s.Fetch(ctx, request)
	if err != nil {
		return Document{}, err
	}
	doc := Document{Metadata: fetched.Metadata, Problems: []Problem{}}
	if fetched.Metadata.UpstreamStatus >= 400 {
		doc.Problems = append(doc.Problems, Problem{Code: "upstream_status", Message: "upstream returned an HTTP error"})
	}
	if fetched.Metadata.MediaType == "text/plain" {
		text := strings.TrimSpace(string(fetched.Body))
		doc.Content = Content{Markdown: plainMarkdown(text), Text: text, HTML: "<pre>" + html.EscapeString(text) + "</pre>"}
	} else if IsHTML(fetched.Metadata.MediaType) {
		body, err := s.TransformHTML(fetched)
		if err != nil {
			return doc, err
		}
		extracted, err := reader.Extract(body, fetched.Metadata.FinalURL)
		if err != nil {
			return doc, failure("extraction_failed", 422, "reader could not extract the document")
		}
		doc.Title = extracted.Title
		doc.Byline = extracted.Byline
		doc.Content = Content{Markdown: extracted.Markdown, Text: extracted.Text, HTML: extracted.HTML}
	} else {
		return doc, failure("unsupported_media_type", 415, "reader supports HTML and plain text; use raw fetch for this media type")
	}
	if strings.TrimSpace(doc.Content.Text) == "" {
		doc.Problems = append(doc.Problems, Problem{Code: "empty_content", Message: "no readable content was extracted"})
	}
	return doc, nil
}
func (s *Service) TestURL(ctx context.Context, request Request) (TestResult, error) {
	f, err := s.Fetch(ctx, request)
	if err != nil {
		return TestResult{}, err
	}
	out := TestResult{Metadata: f.Metadata, Matched: f.Metadata.Rules, Checks: []CheckResult{}, Passed: true}
	add := func(name string, pass bool, detail string) {
		out.Checks = append(out.Checks, CheckResult{Name: name, Passed: pass, Detail: detail})
		out.Passed = out.Passed && pass
	}
	add("upstream status", f.Metadata.UpstreamStatus >= 200 && f.Metadata.UpstreamStatus < 300, "upstream HTTP response must be successful")
	add("matched rules", len(f.Metadata.Rules) > 0, "at least one configured pack rule must match the requested URL")
	body, err := s.TransformHTML(f)
	if err != nil {
		return out, err
	}
	u, _ := s.validate(request.URL)
	assertions := 0
	for _, selected := range s.selectRules(u) {
		for _, test := range selected.rule.Tests {
			if test.URL != f.Metadata.SourceURL && test.URL != f.Metadata.FinalURL {
				continue
			}
			assertions++
			prefix := selected.pack.ID + "/" + selected.rule.ID + "/" + test.Name
			if test.Status != 0 {
				add(prefix+": status", f.Metadata.UpstreamStatus == test.Status, "expected declared HTTP status")
			}
			for _, text := range test.Contains {
				add(prefix+": contains", strings.Contains(string(body), text), text)
			}
			for _, text := range test.NotContains {
				add(prefix+": notContains", !strings.Contains(string(body), text), text)
			}
		}
	}
	if len(f.Metadata.Rules) > 0 {
		add("declared test coverage", assertions > 0, "requested or final URL must have a declared pack test")
	}
	return out, nil
}

// PackTests lists declared test URLs in configured pack/rule order. A command
// may call TestURL for each; validation alone never retrieves public pages.
func (s *Service) PackTests() []string {
	out := []string{}
	seen := map[string]bool{}
	for _, p := range s.packs {
		for _, r := range p.Rules {
			for _, t := range r.Tests {
				if !seen[t.URL] {
					out = append(out, t.URL)
					seen[t.URL] = true
				}
			}
		}
	}
	return out
}

// A fenced block preserves plain text even when a consumer renders Markdown.
func plainMarkdown(text string) string {
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}
	return fence + "\n" + text + "\n" + fence
}
