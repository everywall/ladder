package handlers

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainAllowed(t *testing.T) {
	allowed := []string{"nytimes.com", "", "  example.com  "}

	cases := []struct {
		host string
		want bool
		why  string
	}{
		{"nytimes.com", true, "exact match"},
		{"www.nytimes.com", true, "subdomain"},
		{"a.b.nytimes.com", true, "nested subdomain"},
		{"NYTIMES.COM", true, "case insensitive"},
		{"nytimes.com.", true, "trailing root dot"},
		{"nytimes.com:8443", true, "host with a port"},
		{"example.com", true, "entry with surrounding spaces"},

		{"nytimes.com.evil.tld", false, "attacker registered suffix, was allowed by the prefix match"},
		{"nytimes.company.com", false, "longer label starting with the allowed name"},
		{"evil-nytimes.com", false, "no dot boundary"},
		{"notnytimes.com", false, "no dot boundary"},
		{"evil.tld", false, "unrelated"},
		{"", false, "empty host"},
	}

	for _, c := range cases {
		assert.Equal(t, c.want, domainAllowed(c.host, allowed), "%s (%s)", c.host, c.why)
	}
}

// An allowlist that is empty, or that contains only blanks, must not be treated
// as "allow everything" by domainAllowed.
func TestDomainAllowedIgnoresBlankEntries(t *testing.T) {
	assert.False(t, domainAllowed("example.com", []string{""}))
	assert.False(t, domainAllowed("example.com", []string{"", "  ", ","}))
	assert.False(t, domainAllowed("example.com", nil))
}

func TestParseDomainList(t *testing.T) {
	// The unset case is the important one: strings.Split("", ",") yields [""],
	// which used to become an entry matching every host.
	assert.Empty(t, parseDomainList(""))
	assert.Empty(t, parseDomainList(",,  ,"))
	assert.Equal(t, []string{"a.com", "b.com"}, parseDomainList("a.com, b.com,"))
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",        // loopback
		"::1",              // loopback v6
		"10.1.2.3",         // private
		"172.16.9.9",       // private
		"192.168.1.1",      // private
		"169.254.169.254",  // cloud instance metadata
		"100.64.0.1",       // carrier grade NAT
		"0.0.0.0",          // unspecified
		"0.1.2.3",          // this host on this network
		"224.0.0.1",        // multicast
		"fd00::1",          // unique local v6
		"fe80::1",          // link local v6
		"::ffff:127.0.0.1", // v4 mapped loopback
	}
	for _, s := range blocked {
		assert.True(t, isBlockedIP(net.ParseIP(s)), "%s should be blocked", s)
	}

	allowed := []string{"93.184.216.34", "1.1.1.1", "8.8.8.8", "2606:2800:220:1:248:1893:25c8:1946"}
	for _, s := range allowed {
		assert.False(t, isBlockedIP(net.ParseIP(s)), "%s should be allowed", s)
	}

	assert.True(t, isBlockedIP(nil), "an unparseable address should be blocked")
}

func TestCheckRedirectReappliesAllowlist(t *testing.T) {
	original := allowedDomains
	allowedDomains = []string{"example.com"}
	defer func() { allowedDomains = original }()

	req := func(raw string) *http.Request {
		u, err := url.Parse(raw)
		assert.NoError(t, err)
		return &http.Request{URL: u}
	}

	assert.NoError(t, checkRedirect(req("https://www.example.com/a"), nil))
	assert.Error(t, checkRedirect(req("https://example.com.evil.tld/a"), nil))
	assert.Error(t, checkRedirect(req("https://evil.tld/a"), nil))

	assert.Error(t, checkRedirect(req("https://example.com/a"), make([]*http.Request, 10)),
		"redirect chains must still be bounded")
}

// fetchSite must refuse to read a server on the loopback interface. This is the
// case that matters in practice: ladder usually runs in a container next to
// other services, and the URL comes from whoever can reach the proxy.
func TestFetchSiteBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("internal-only-content"))
	}))
	defer srv.Close()

	body, _, _, err := fetchSite(srv.URL, map[string]string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-public address")
	assert.NotContains(t, body, "internal-only-content")
}

func TestFetchSiteAllowsLoopbackWhenOptedIn(t *testing.T) {
	allowPrivateAddresses = true
	defer func() { allowPrivateAddresses = false }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("internal-only-content"))
	}))
	defer srv.Close()

	body, _, _, err := fetchSite(srv.URL, map[string]string{})

	assert.NoError(t, err)
	assert.Contains(t, body, "internal-only-content")
}
