package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSiteKeepsQueryValuesIntact(t *testing.T) {
	var got map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer srv.Close()

	queries := map[string]string{
		"q":    "fish & chips",
		"next": "/a?b=1#top",
		"plus": "1+1=2",
	}
	if _, _, _, err := fetchSite(srv.URL+"/search", queries); err != nil {
		t.Fatalf("fetchSite: %v", err)
	}

	for k, want := range queries {
		if vals := got[k]; len(vals) != 1 || vals[0] != want {
			t.Errorf("query %q = %q, want [%q]", k, vals, want)
		}
	}
	if len(got) != len(queries) {
		t.Errorf("upstream got query keys %v, want exactly %v", got, queries)
	}
}
