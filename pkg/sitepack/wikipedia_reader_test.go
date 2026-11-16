package sitepack_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"ladder/pkg/reader"
	"ladder/pkg/sitepack"
)

func TestWikipediaReaderKeepsArticleAndRemovesSiteMenus(t *testing.T) {
	pack, err := sitepack.Load("../../sitepacks/wikipedia/1.0.1.yaml")
	require.NoError(t, err)
	require.Equal(t, "1.0.1", pack.Version)
	rule := pack.Rules[0]
	// The language dropdown lives inside main without a navigation role in
	// Vector 2022. Generic reader pruning cannot distinguish it from prose.
	source := []byte(`<!doctype html><html><head><title>HTTP - Wikipedia</title></head><body>
<main id="content" class="mw-body">
  <header class="mw-body-header vector-page-titlebar">
    <h1 id="firstHeading"><span class="mw-page-title-main">HTTP</span></h1>
    <div id="p-lang-btn" class="vector-dropdown mw-portlet mw-portlet-lang">
      <div class="vector-dropdown-content"><ul>
        <li class="interlanguage-link"><a href="https://af.wikipedia.org/wiki/HTTP">Afrikaans</a></li>
        <li class="interlanguage-link"><a href="https://ar.wikipedia.org/wiki/HTTP">العربية</a></li>
      </ul><span class="wb-langlinks-edit"><a href="https://www.wikidata.org/wiki/Q8777">Edit links</a></span></div>
    </div>
  </header>
  <div class="vector-page-toolbar"><a href="/w/index.php?action=history">View history</a></div>
  <div class="vector-column-end"><div id="vector-page-tools"><a href="/wiki/Special:WhatLinksHere/HTTP">What links here</a></div></div>
  <div id="bodyContent" class="vector-body">
    <div class="vector-body-before-content"><div class="mw-indicators">Page semi-protected</div><div id="siteSub">From Wikipedia, the free encyclopedia</div></div>
    <div id="mw-content-text" class="mw-body-content"><div class="mw-parser-output">
      <table class="infobox"><caption>Protocol facts</caption><tr><th>Developed by</th><td><a href="/wiki/CERN">CERN</a></td></tr></table>
      <p>HTTP (Hypertext Transfer Protocol) is an application layer protocol.<sup class="reference"><a href="#cite_note-1">[1]</a></sup></p>
      <div class="mw-heading mw-heading2"><h2 id="Versions">Versions</h2><span class="mw-editsection"><a href="/w/index.php?action=edit">Edit section</a></span></div>
      <table class="wikitable"><caption>HTTP versions</caption><tr><th>Version</th><th>Status</th></tr><tr><td>HTTP/3</td><td>Standard</td></tr></table>
      <div class="navbox"><table><tr><td>Internet navigation menu</td></tr></table></div>
      <h2 id="References">References</h2><ol class="references"><li id="cite_note-1"><a href="https://www.rfc-editor.org/rfc/rfc9110">RFC 9110: HTTP Semantics</a></li></ol>
      <div class="printfooter">Retrieved from an old revision</div>
    </div></div>
    <div id="catlinks">Hidden categories: site maintenance</div>
  </div>
</main></body></html>`)
	before, err := reader.Extract(source, "https://en.wikipedia.org/wiki/HTTP")
	require.NoError(t, err)
	require.Contains(t, before.Text, "Afrikaans", "fixture must reproduce the observed language-menu leak")

	clean, err := rule.ApplyHTML(source)
	require.NoError(t, err)
	doc, err := reader.Extract(clean, "https://en.wikipedia.org/wiki/HTTP")
	require.NoError(t, err)
	require.Equal(t, "HTTP - Wikipedia", doc.Title)
	require.True(t, strings.HasPrefix(doc.Text, "HTTP\n\nProtocol facts"), "the article must begin immediately after its heading: %s", doc.Text)
	for _, clutter := range []string{"Afrikaans", "العربية", "Edit links", "View history", "What links here", "Page semi-protected", "Internet navigation menu", "Retrieved from an old revision", "Hidden categories"} {
		require.NotContains(t, doc.Text, clutter)
	}
	for _, article := range []string{"Hypertext Transfer Protocol", "Developed by\tCERN", "Versions", "HTTP/3\tStandard", "References", "RFC 9110: HTTP Semantics", "[1]"} {
		require.Contains(t, doc.Text, article)
	}
	require.Contains(t, doc.HTML, "<h1>HTTP</h1>")
	require.Contains(t, doc.HTML, "<h2>Versions</h2>")
	require.Contains(t, doc.HTML, "<table><caption>HTTP versions</caption>")
	require.Contains(t, doc.HTML, `href="https://en.wikipedia.org/wiki/CERN"`)
	require.Contains(t, doc.HTML, `href="https://en.wikipedia.org/wiki/HTTP#cite_note-1"`)
	require.Contains(t, doc.Markdown, "| HTTP/3 | Standard |")
	require.Contains(t, doc.Markdown, "https://www.rfc-editor.org/rfc/rfc9110")
}

func TestWikipediaReaderPackPreservesDeclaredPagesAndHonestHeaders(t *testing.T) {
	pack, err := sitepack.Load("../../sitepacks/wikipedia/1.0.1.yaml")
	require.NoError(t, err)
	rule := pack.Rules[0]
	pages := map[string]bool{}
	for _, test := range rule.Tests {
		pages[test.URL] = true
		require.NotEmpty(t, test.Contains)
		require.Contains(t, test.NotContains, "Edit links")
	}
	for _, page := range []string{"HTTP", "Proxy_server", "Encyclopedia"} {
		require.True(t, pages["https://en.wikipedia.org/wiki/"+page], "missing declared page %s", page)
	}
	headers := http.Header{"X-Forwarded-For": {"192.0.2.1"}, "Referer": {"https://example.com"}}
	rule.ApplyHeaders(headers)
	require.Equal(t, "Ladder/1.0 (https://github.com/everywall/ladder)", headers.Get("User-Agent"))
	require.Empty(t, headers.Get("X-Forwarded-For"))
	require.Empty(t, headers.Get("Referer"))
}
