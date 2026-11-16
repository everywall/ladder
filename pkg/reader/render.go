package reader

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

type rendered struct {
	text  string
	block bool
}

func renderDocument(root *html.Node, markdown bool) string {
	return strings.TrimSpace(renderChildren(root, markdown))
}

// Inline runs and block boundaries are joined separately, so prose whitespace
// is normalized without changing whitespace inside code blocks.
func renderChildren(n *html.Node, markdown bool) string {
	var blocks []string
	var inline strings.Builder
	flush := func() {
		if text := strings.TrimSpace(inline.String()); text != "" {
			blocks = append(blocks, text)
		}
		inline.Reset()
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		part := render(child, markdown)
		if part.block {
			flush()
			if part.text != "" {
				blocks = append(blocks, part.text)
			}
		} else {
			inline.WriteString(part.text)
		}
	}
	flush()
	return strings.Join(blocks, "\n\n")
}

func render(n *html.Node, markdown bool) rendered {
	if n.Type == html.TextNode {
		text := proseWhitespace(n.Data)
		if markdown {
			text = escapeMarkdown(text)
		}
		return rendered{text: text}
	}
	// Handle self-contained blocks before rendering children. In particular,
	// lists and code must not recursively render the same subtree twice.
	switch n.Data {
	case "code":
		text := compact(nodeText(n))
		if markdown && text != "" {
			fence := strings.Repeat("`", longestBackticks(text)+1)
			padding := ""
			if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") {
				padding = " "
			}
			text = fence + padding + text + padding + fence
		}
		return rendered{text: text}
	case "pre":
		text := strings.Trim(nodeText(n), "\n\r")
		if markdown {
			fence := strings.Repeat("`", max(3, longestBackticks(text)+1))
			text = fence + "\n" + text + "\n" + fence
		}
		return rendered{text: text, block: true}
	case "ul", "ol":
		return rendered{text: renderList(n, markdown), block: true}
	case "table":
		return rendered{text: renderTable(n, markdown), block: true}
	}
	var text string
	switch n.Data {
	case "a", "strong", "b", "em", "i", "u", "s", "del", "sub", "sup":
		var inline strings.Builder
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			inline.WriteString(render(child, markdown).text)
		}
		text = inline.String()
	default:
		text = renderChildren(n, markdown)
	}
	block := func(value string) rendered { return rendered{text: value, block: true} }
	switch n.Data {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		if markdown {
			text = strings.Repeat("#", int(n.Data[1]-'0')) + " " + text
		}
		return block(text)
	case "div", "p", "dl", "dt", "dd", "caption":
		return block(text)
	case "br":
		if markdown {
			return rendered{text: "  \n"}
		}
		return rendered{text: "\n"}
	case "hr":
		if markdown {
			return block("---")
		}
		return block("")
	case "strong", "b":
		if markdown && text != "" {
			text = wrapInline(text, "**", "**")
		}
	case "em", "i":
		if markdown && text != "" {
			text = wrapInline(text, "*", "*")
		}
	case "s", "del":
		if markdown && text != "" {
			text = wrapInline(text, "~~", "~~")
		}
	case "a":
		if href := attribute(n, "href"); markdown && href != "" && text != "" {
			// Angle delimiters keep parentheses in source URLs unambiguous.
			href = strings.NewReplacer("<", "%3C", ">", "%3E", " ", "%20", "\\", "%5C").Replace(href)
			text = wrapInline(text, "[", "](<"+href+">)")
		}
	case "blockquote":
		if markdown {
			text = "> " + strings.ReplaceAll(text, "\n", "\n> ")
		}
		return block(text)
	}
	return rendered{text: text}
}

func renderList(n *html.Node, markdown bool) string {
	index := 1
	if start, err := strconv.Atoi(attribute(n, "start")); err == nil {
		index = start
	}
	var lines []string
	for item := n.FirstChild; item != nil; item = item.NextSibling {
		if item.Type != html.ElementNode || item.Data != "li" {
			continue
		}
		marker := "- "
		if n.Data == "ol" {
			marker = strconv.Itoa(index) + ". "
		}
		index++
		text := renderChildren(item, markdown)
		line := marker + strings.ReplaceAll(text, "\n", "\n"+strings.Repeat(" ", len(marker)))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderTable(n *html.Node, markdown bool) string {
	var rows [][]string
	var captions []string
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if current.Data == "caption" {
			captions = append(captions, renderChildren(current, markdown))
			return
		}
		if current.Data == "tr" {
			var cells []string
			for cell := current.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.Data == "td" || cell.Data == "th" {
					value := strings.ReplaceAll(renderChildren(cell, markdown), "\n", " ")
					if markdown {
						value = escapeTablePipes(value)
					}
					cells = append(cells, value)
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
			return
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(n)
	var lines []string
	width := 0
	for _, row := range rows {
		width = max(width, len(row))
	}
	for i, row := range rows {
		if markdown {
			for len(row) < width {
				row = append(row, "")
			}
			lines = append(lines, "| "+strings.Join(row, " | ")+" |")
			if i == 0 {
				lines = append(lines, "| "+strings.TrimSuffix(strings.Repeat("--- | ", width), " | ")+" |")
			}
		} else {
			lines = append(lines, strings.Join(row, "\t"))
		}
	}
	if len(captions) > 0 {
		return strings.Join(captions, "\n") + "\n\n" + strings.Join(lines, "\n")
	}
	return strings.Join(lines, "\n")
}

func proseWhitespace(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
			}
			space = true
		} else {
			b.WriteRune(r)
			space = false
		}
	}
	return b.String()
}

func escapeMarkdown(s string) string {
	return strings.NewReplacer("\\", "\\\\", "&", "&amp;", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "\\<", ">", "\\>", "#", "\\#", "!", "\\!", "|", "\\|").Replace(s)
}

func wrapInline(text, before, after string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text
	}
	start := len(text) - len(strings.TrimLeftFunc(text, unicode.IsSpace))
	end := len(strings.TrimRightFunc(text, unicode.IsSpace))
	return text[:start] + before + trimmed + after + text[end:]
}

func escapeTablePipes(s string) string {
	var b strings.Builder
	slashes := 0
	for _, r := range s {
		if r == '|' && slashes%2 == 0 {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
		if r == '\\' {
			slashes++
		} else {
			slashes = 0
		}
	}
	return b.String()
}

func longestBackticks(s string) int {
	longest, run := 0, 0
	for _, r := range s {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return longest
}
