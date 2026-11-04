package sitepack

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validPack = `schemaVersion: 1
id: fixture
version: "1.0.0"
rules:
 - id: article
   domains: [example.com]
   paths: [/article]
   remove: [.ad]
   tests:
    - name: article
      url: https://example.com/article/one
      status: 200
      contains: [Article]
`

func loadText(t *testing.T, text string) (Pack, error) {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "pack.yaml")
	require.NoError(t, os.WriteFile(filename, []byte(text), 0600))
	return Load(filename)
}
func TestLoadStrictVersionIDsRegexSelectorAndTests(t *testing.T) {
	p, err := loadText(t, validPack)
	require.NoError(t, err)
	require.Equal(t, "fixture", p.ID)
	require.Len(t, p.SHA256, 64)
	cases := []string{strings.Replace(validPack, "schemaVersion: 1", "schemaVersion: 2", 1), strings.Replace(validPack, "version: \"1.0.0\"", "version: latest", 1), validPack + "unknown: value\n", strings.Replace(validPack, "remove: [.ad]", "regex: [{match: '[', replace: x}]", 1), strings.Replace(validPack, "remove: [.ad]", "remove: ['[']", 1), strings.Replace(validPack, "status: 200\n      contains: [Article]", "status: 0", 1), strings.Replace(validPack, "https://example.com/article/one", "https://badexample.com/article/one", 1), validPack + "---\n"}
	for _, text := range cases {
		_, err := loadText(t, text)
		require.Error(t, err, text)
	}
	// Append the same rule object in memory to exercise stable-id uniqueness.
	duplicatePack, err := loadText(t, validPack)
	require.NoError(t, err)
	duplicatePack.Rules = append(duplicatePack.Rules, duplicatePack.Rules[0])
	require.ErrorContains(t, duplicatePack.Validate(), "duplicate id")
	sample, err := Load("testdata/wikipedia.yaml")
	require.NoError(t, err)
	require.Equal(t, "wikipedia", sample.ID)
}
func TestDomainAndPathBoundariesAndFirstMatch(t *testing.T) {
	p, err := loadText(t, validPack)
	require.NoError(t, err)
	r := p.Rules[0]
	for _, tc := range []struct {
		raw   string
		match bool
	}{{"https://example.com/article", true}, {"http://WWW.Example.COM.:123/article/one", true}, {"https://badexample.com/article", false}, {"https://example.com.evil.test/article", false}, {"https://example.com/articles", false}, {"https://example.com/article-two", false}} {
		u, err := url.Parse(tc.raw)
		require.NoError(t, err)
		require.Equal(t, tc.match, r.Matches(u), tc.raw)
	}
}
