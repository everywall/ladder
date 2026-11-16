package reader_test

import (
	"strings"
	"testing"

	"ladder/pkg/reader"

	"golang.org/x/net/html"
)

func extract(t *testing.T, source string) reader.Document {
	t.Helper()
	doc, err := reader.Extract([]byte(source), "https://example.org/news/2026/story.html")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return doc
}

func TestArticleSelectionAndMetadata(t *testing.T) {
	doc := extract(t, `<html><head><title>Site title</title><meta property="og:title" content="The story &amp; its meaning"><meta name="author" content="Dr. Ana Li"></head><body><nav>Top navigation</nav><main><div>Latest news promo</div><article><header><h1>The story</h1></header><p>These are the actual source facts.</p><nav>Article navigation</nav><aside>Subscribe today</aside><p hidden>Hidden fact</p><p aria-hidden="true">Invisible fact</p><div style="display: none !important">Secret promotion</div></article></main><footer>Footer links</footer></body></html>`)
	if doc.Title != "The story & its meaning" || doc.Byline != "Dr. Ana Li" {
		t.Fatalf("metadata = title %q, byline %q", doc.Title, doc.Byline)
	}
	if !strings.Contains(doc.Text, "These are the actual source facts.") {
		t.Fatalf("article facts missing: %q", doc.Text)
	}
	for _, clutter := range []string{"Top navigation", "Latest news promo", "Article navigation", "Subscribe today", "Hidden fact", "Invisible fact", "Secret promotion", "Footer links"} {
		if strings.Contains(doc.Text, clutter) || strings.Contains(doc.HTML, clutter) || strings.Contains(doc.Markdown, clutter) {
			t.Errorf("retained clutter %q", clutter)
		}
	}
}

func TestMainAndBodyFallback(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
	}{
		{"main", `<title>Main title</title><nav>Menu</nav><div role="main"><h1>Main heading</h1><p>Readable main content.</p></div><aside>Sidebar</aside>`},
		{"body", `<title>Main title</title><nav>Menu</nav><div><h1>Main heading</h1><p>Readable main content.</p></div><form>Search form</form><aside>Sidebar</aside>`},
	} {
		t.Run(test.name, func(t *testing.T) {
			doc := extract(t, test.source)
			if doc.Title != "Main title" || doc.Byline != "" || !strings.Contains(doc.Text, "Readable main content.") {
				t.Fatalf("fallback = %+v", doc)
			}
			for _, clutter := range []string{"Menu", "Sidebar", "Search form"} {
				if strings.Contains(doc.Text, clutter) {
					t.Errorf("retained %q", clutter)
				}
			}
		})
	}
}

func TestRelativeLinksUseEffectiveURL(t *testing.T) {
	doc := extract(t, `<base href="https://attacker.invalid/"><article><p>Read <a href="../guide?q=a&amp;x=b#part">the guide</a> or <a href="/help">help</a>.</p></article>`)
	if !strings.Contains(doc.HTML, `href="https://example.org/news/guide?q=a&amp;x=b#part"`) || !strings.Contains(doc.HTML, `href="https://example.org/help"`) {
		t.Fatalf("resolved HTML = %s", doc.HTML)
	}
	if !strings.Contains(doc.Markdown, `[the guide](<https://example.org/news/guide?q=a&x=b#part>)`) {
		t.Fatalf("resolved Markdown = %q", doc.Markdown)
	}
	if !strings.Contains(doc.Text, "Read the guide or help.") {
		t.Fatalf("plain text = %q", doc.Text)
	}
}

func TestCodeListsHeadingsAndTables(t *testing.T) {
	doc := extract(t, "<article><h1>Build guide</h1><h2>Steps</h2><ol start=3><li>Install tools<ul><li>Keep versions</li></ul></li><li>Run <code>a &lt; b</code></li></ol><pre><code>if a &lt; b {\n\tprint(`ok`)\n}\n</code></pre><blockquote><p>Use care.</p></blockquote><table><caption>Results</caption><thead><tr><th>Name</th><th>Score</th></tr></thead><tbody><tr><td>Ada</td><td>42</td></tr></tbody></table></article>")
	for _, fragment := range []string{"## Steps", "3. Install tools", "  - Keep versions", "4. Run `a < b`", "```\nif a < b {\n\tprint(`ok`)\n}\n```", "> Use care.", "| Name | Score |", "| --- | --- |", "| Ada | 42 |"} {
		if !strings.Contains(doc.Markdown, fragment) {
			t.Errorf("Markdown missing %q: %q", fragment, doc.Markdown)
		}
	}
	for _, fragment := range []string{"if a < b {\n\tprint(`ok`)\n}", "3. Install tools", "4. Run a < b", "Name\tScore", "Ada\t42"} {
		if !strings.Contains(doc.Text, fragment) {
			t.Errorf("text missing %q: %q", fragment, doc.Text)
		}
	}
}

func TestUnsafeMarkupAndURLsAreInert(t *testing.T) {
	doc := extract(t, `<article onclick="attack()" style="color:red"><h1>Safe</h1><p>Visible <a href="javascript:alert(1)" onmouseover="attack()">bad</a> <a href="data:text/html,attack">data</a> <a href="vbscript:attack">vb</a> <a href="https://good.example/" ping="https://tracker.invalid">good</a>.</p><script>alert(1)</script><style>bad {}</style><iframe src="https://tracker.invalid">frame</iframe><object data="https://tracker.invalid">object</object><svg><script>attack()</script></svg><form><input value="secret">Form clutter</form><img src="https://tracker.invalid/pixel" srcset="https://tracker.invalid/pixel2 2x" onerror="attack()" alt="A meaningful diagram"><video poster="https://tracker.invalid/poster" autoplay>media</video></article>`)
	allowedTags := map[string]bool{"html": true, "head": true, "body": true, "h1": true, "p": true, "a": true}
	node, err := html.Parse(strings.NewReader(doc.HTML))
	if err != nil {
		t.Fatal(err)
	}
	var inspect func(*html.Node)
	inspect = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if !allowedTags[n.Data] {
				t.Errorf("unexpected tag %q in %s", n.Data, doc.HTML)
			}
			for _, a := range n.Attr {
				if n.Data != "a" || (a.Key != "href" && a.Key != "rel") {
					t.Errorf("unsafe attribute %s=%q", a.Key, a.Val)
				}
				if a.Key == "href" && a.Val != "https://good.example/" {
					t.Errorf("unsafe href %q", a.Val)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			inspect(child)
		}
	}
	inspect(node)
	for _, bad := range []string{"attack()", "alert(1)", "tracker.invalid", "Form clutter", "javascript:", "data:text", "vbscript:"} {
		if strings.Contains(doc.HTML, bad) || strings.Contains(doc.Markdown, bad) || strings.Contains(doc.Text, bad) {
			t.Errorf("retained unsafe content %q", bad)
		}
	}
	if !strings.Contains(doc.Text, "A meaningful diagram") || !strings.Contains(doc.HTML, "A meaningful diagram") {
		t.Fatalf("lost useful image description: %+v", doc)
	}
}

func TestUnicodeAndLiteralMarkupStayText(t *testing.T) {
	doc := extract(t, `<article><h1>Café 東京 🪜</h1><p>Body marker: &lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt; &amp; &lt;img src=x onerror=bad&gt;. Literal *stars* and [brackets].</p></article>`)
	if doc.Title != "Café 東京 🪜" || !strings.Contains(doc.Text, `<script>alert("x")</script> & <img src=x onerror=bad>`) {
		t.Fatalf("Unicode or literal text lost: %+v", doc)
	}
	if strings.Contains(doc.HTML, "<script>") || strings.Contains(doc.HTML, "<img ") || !strings.Contains(doc.HTML, "&lt;script&gt;") {
		t.Fatalf("literal text became markup: %s", doc.HTML)
	}
	if !strings.Contains(doc.Markdown, `\<script\>`) || !strings.Contains(doc.Markdown, `\*stars\*`) || !strings.Contains(doc.Markdown, `\[brackets\]`) {
		t.Fatalf("Markdown did not escape literal text: %q", doc.Markdown)
	}
}

func TestActualContentBylineAndAbsentByline(t *testing.T) {
	doc := extract(t, `<article><h1>A title</h1><p class="byline">By Renée Example</p><p>Reporting from the source.</p></article>`)
	if doc.Byline != "By Renée Example" {
		t.Fatalf("byline = %q", doc.Byline)
	}
	if got := extract(t, `<article><h1>A title</h1><p>Written content with no identified author.</p></article>`).Byline; got != "" {
		t.Fatalf("invented byline %q", got)
	}
}

func TestInvalidBaseURL(t *testing.T) {
	for _, base := range []string{"", "/relative", "https://", "javascript:alert(1)", "file:///etc/passwd", "https://example.org:bad/", "https://example.org/\n"} {
		t.Run(base, func(t *testing.T) {
			if _, err := reader.Extract([]byte(`<p>Content</p>`), base); err == nil {
				t.Fatalf("accepted invalid base URL %q", base)
			}
		})
	}
}

func TestInlineWhitespaceAndNestedListOrder(t *testing.T) {
	doc := extract(t, `<article><p>Keep<em> these </em>words<strong> together </strong>please.</p><ul><li>Before<ul><li>Nested</li></ul>After</li></ul></article>`)
	if !strings.Contains(doc.Text, "Keep these words together please.") || !strings.Contains(doc.Markdown, "Keep *these* words **together** please.") {
		t.Fatalf("inline spaces changed: %+v", doc)
	}
	for _, output := range []string{doc.Text, doc.Markdown} {
		before, nested, after := strings.Index(output, "Before"), strings.Index(output, "Nested"), strings.Index(output, "After")
		if before < 0 || nested <= before || after <= nested {
			t.Fatalf("list source order changed: %q", output)
		}
	}
}

func TestMarkdownCannotTurnSourceTextIntoImagesOrHTML(t *testing.T) {
	doc := extract(t, `<article><p>![tracker](https://tracker.invalid/pixel) &lt;iframe src="https://tracker.invalid"&gt; a &amp; b</p><table><tr><td>A|B</td><td><code>C|D</code></td></tr></table><pre><code>first


last
`+"```"+`
&lt;img src=x&gt;</code></pre></article>`)
	if !strings.Contains(doc.Markdown, `\!\[tracker\]`) || !strings.Contains(doc.Markdown, `\<iframe`) {
		t.Fatalf("source Markdown/HTML was active: %q", doc.Markdown)
	}
	if !strings.Contains(doc.Markdown, `| A\|B | `+"`C\\|D`"+` |`) {
		t.Fatalf("table pipes changed: %q", doc.Markdown)
	}
	if !strings.Contains(doc.Markdown, "````\nfirst\n\n\nlast\n```\n<img src=x>\n````") || !strings.Contains(doc.Text, "first\n\n\nlast") {
		t.Fatalf("code contents or fence boundary changed: %+v", doc)
	}
}

func TestEmptyDocument(t *testing.T) {
	doc := extract(t, `<html><head><title>Empty page</title></head><body><script>hidden()</script></body></html>`)
	if doc.Title != "Empty page" || doc.Text != "" || doc.Markdown != "" || doc.HTML != "" || doc.Byline != "" {
		t.Fatalf("empty page = %+v", doc)
	}
}

func TestDeeplyNestedFormattingPreservesOneCopy(t *testing.T) {
	doc := extract(t, "<article><p>"+strings.Repeat("<strong>", 60)+"A single source fact"+strings.Repeat("</strong>", 60)+"</p></article>")
	if doc.Text != "A single source fact" || strings.Count(doc.Markdown, "A single source fact") != 1 {
		t.Fatalf("nested content changed: %+v", doc)
	}
}
