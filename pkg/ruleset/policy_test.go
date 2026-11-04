package ruleset

import (
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestDomainBoundariesIncludingIP(t *testing.T) {
	for _, tc := range []struct {
		host, domain string
		want         bool
	}{{"www.example.com", "example.com", true}, {"EXAMPLE.COM.", "example.com", true}, {"badexample.com", "example.com", false}, {"example.com.evil.test", "example.com", false}, {"127.0.0.1", "0.0.1", false}, {"::1", "::1", true}, {"2001:db8::1", "db8::1", false}} {
		require.Equal(t, tc.want, MatchDomain(tc.host, tc.domain), tc.host)
	}
}
func TestModifyURLPreservesUnrelatedDuplicateQueryValues(t *testing.T) {
	u, err := url.Parse("http://example.com/a%2Fb?dup=one&dup=two&literal=a%26b%3Dc&tracking=x")
	require.NoError(t, err)
	modified, err := ModifyURL(u, nil, nil, []KV{{Key: "tracking", Value: ""}, {Key: "new", Value: "a+b"}}, false)
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two"}, modified.Query()["dup"])
	require.Equal(t, "a&b=c", modified.Query().Get("literal"))
	require.Equal(t, "a+b", modified.Query().Get("new"))
	require.Equal(t, "/a%2Fb", modified.EscapedPath())
}
