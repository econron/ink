package markdown

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

func renderHTML(nodes []BlockNode) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"ja\">\n")
	b.WriteString("<head>\n")
	b.WriteString("<meta charset=\"UTF-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>ink preview</title>\n")
	b.WriteString(renderDocumentStyle())
	b.WriteString("</head>\n")
	b.WriteString("<body>\n")
	b.WriteString("<main class=\"ink-document\">\n")

	for _, node := range nodes {
		b.WriteString(renderBlock(node))
		b.WriteString("\n")
	}

	b.WriteString("</main>\n")
	if hasMermaidBlock(nodes) {
		b.WriteString(renderMermaidScript())
	}
	b.WriteString("</body>\n")
	b.WriteString("</html>\n")
	return b.String()
}

func hasMermaidBlock(nodes []BlockNode) bool {
	for _, node := range nodes {
		if node.Kind == BlockCodeBlock && isMermaidLanguage(node.Language) {
			return true
		}
		if len(node.Children) > 0 && hasMermaidBlock(node.Children) {
			return true
		}
		for _, item := range node.ListItems {
			if hasMermaidBlock(item.Blocks) {
				return true
			}
		}
	}
	return false
}

func renderMermaidScript() string {
	return `<script type="module">
import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
mermaid.initialize({ startOnLoad: true });
</script>
`
}

func renderDocumentStyle() string {
	return `<style>
:root {
  color-scheme: light;
  --ink-bg: #f5f7fa;
  --ink-paper: #ffffff;
  --ink-text: #24292f;
  --ink-muted: #667085;
  --ink-border: #d8dee4;
  --ink-soft: #f6f8fa;
  --ink-code: #eef2f6;
  --ink-accent: #1f6feb;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  background: var(--ink-bg);
  color: var(--ink-text);
  font-family: ui-serif, Georgia, Cambria, "Times New Roman", serif;
  font-size: 17px;
  line-height: 1.75;
}

.ink-document {
  width: min(760px, calc(100% - 32px));
  margin: 48px auto;
  padding: 56px 64px;
  background: var(--ink-paper);
  border: 1px solid var(--ink-border);
  border-radius: 8px;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
}

.ink-document > :first-child {
  margin-top: 0;
}

.ink-document > :last-child {
  margin-bottom: 0;
}

h1, h2, h3, h4, h5, h6 {
  margin: 1.7em 0 0.65em;
  color: #111827;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-weight: 700;
  line-height: 1.25;
}

h1 {
  padding-bottom: 0.35em;
  border-bottom: 1px solid var(--ink-border);
  font-size: 2rem;
}

h2 {
  padding-bottom: 0.25em;
  border-bottom: 1px solid #e8edf3;
  font-size: 1.55rem;
}

h3 {
  font-size: 1.25rem;
}

h4, h5, h6 {
  font-size: 1.05rem;
}

p, ul, ol, blockquote, pre, table {
  margin: 1em 0;
}

a {
  color: var(--ink-accent);
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
  text-underline-offset: 0.18em;
}

ul, ol {
  padding-left: 1.55em;
}

li + li {
  margin-top: 0.25em;
}

li > ul, li > ol {
  margin: 0.35em 0;
}

blockquote {
  margin-left: 0;
  padding: 0.75em 1em;
  border-left: 4px solid #a8b3c4;
  background: #f8fafc;
  color: #475467;
}

blockquote > :first-child {
  margin-top: 0;
}

blockquote > :last-child {
  margin-bottom: 0;
}

code {
  padding: 0.15em 0.35em;
  border-radius: 5px;
  background: var(--ink-code);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace;
  font-size: 0.9em;
}

pre {
  overflow-x: auto;
  padding: 1em;
  border: 1px solid var(--ink-border);
  border-radius: 8px;
  background: #0f172a;
  line-height: 1.6;
}

pre code {
  display: block;
  padding: 0;
  background: transparent;
  color: #e5e7eb;
  font-size: 0.88em;
}

.mermaid {
  overflow-x: auto;
  margin: 1.2em 0;
  padding: 1.25em;
  border: 1px solid var(--ink-border);
  border-radius: 8px;
  background: #ffffff;
  text-align: center;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.95em;
}

th, td {
  padding: 0.55em 0.75em;
  border: 1px solid var(--ink-border);
  text-align: left;
  vertical-align: top;
}

th {
  background: var(--ink-soft);
  font-weight: 700;
}

hr {
  height: 1px;
  margin: 2em 0;
  border: 0;
  background: var(--ink-border);
}

img {
  max-width: 100%;
  height: auto;
}

@media (max-width: 640px) {
  body {
    font-size: 16px;
  }

  .ink-document {
    width: 100%;
    margin: 0;
    padding: 28px 20px;
    border: 0;
    border-radius: 0;
    box-shadow: none;
  }
}
</style>
`
}

func renderBlock(node BlockNode) string {
	switch node.Kind {
	case BlockHeading:
		tag := headingHTMLTagMap[node.Level]
		if tag == "" {
			tag = "h1"
		}
		return fmt.Sprintf("<%s>%s</%s>", tag, renderInline(node.Content), tag)
	case BlockParagraph:
		return fmt.Sprintf("<p>%s</p>", renderInline(node.Content))
	case BlockUnorderedList, BlockOrderedList:
		return renderList(node)
	case BlockBlockquote:
		return renderBlockquote(node)
	case BlockCodeBlock:
		return renderCodeBlock(node)
	case BlockHorizontalRule:
		return "<hr>"
	case BlockTable:
		return renderTable(node)
	case BlockHTML:
		return node.Content
	default:
		return renderInline(node.Content)
	}
}

func renderList(node BlockNode) string {
	tag := listHTMLTagMap[node.Kind]
	if tag == "" {
		tag = "ul"
	}

	var b strings.Builder
	if node.Kind == BlockOrderedList && node.Start > 1 {
		b.WriteString("<")
		b.WriteString(tag)
		b.WriteString(` start="`)
		b.WriteString(strconv.Itoa(node.Start))
		b.WriteString(`">`)
	} else {
		b.WriteString("<")
		b.WriteString(tag)
		b.WriteString(">")
	}

	if len(node.ListItems) > 0 {
		for _, item := range node.ListItems {
			b.WriteString(renderListItem(item, node.Tight))
		}
	} else {
		for _, item := range node.Items {
			b.WriteString("<li>")
			b.WriteString(renderInline(item))
			b.WriteString("</li>")
		}
	}

	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
	return b.String()
}

func renderListItem(item ListItem, tight bool) string {
	var b strings.Builder
	b.WriteString("<li>")
	for _, block := range item.Blocks {
		if tight && block.Kind == BlockParagraph {
			b.WriteString(renderInline(block.Content))
			continue
		}
		b.WriteString(renderBlock(block))
	}
	b.WriteString("</li>")
	return b.String()
}

func renderBlockquote(node BlockNode) string {
	var b strings.Builder
	b.WriteString("<blockquote>")
	if len(node.Children) > 0 {
		for _, child := range node.Children {
			b.WriteString(renderBlock(child))
		}
	} else {
		b.WriteString(renderInline(node.Content))
	}
	b.WriteString("</blockquote>")
	return b.String()
}

func renderCodeBlock(node BlockNode) string {
	if isMermaidLanguage(node.Language) {
		return renderMermaidBlock(node.Content)
	}

	var b strings.Builder
	b.WriteString("<pre><code")
	if node.Language != "" {
		b.WriteString(` class="language-`)
		b.WriteString(html.EscapeString(node.Language))
		b.WriteString(`"`)
	}
	b.WriteString(">")
	b.WriteString(html.EscapeString(node.Content))
	b.WriteString("</code></pre>")
	return b.String()
}

func renderMermaidBlock(content string) string {
	var b strings.Builder
	b.WriteString(`<div class="mermaid">`)
	b.WriteString(html.EscapeString(content))
	b.WriteString("</div>")
	return b.String()
}

func isMermaidLanguage(language string) bool {
	fields := strings.Fields(strings.ToLower(language))
	return len(fields) > 0 && fields[0] == "mermaid"
}

func renderTable(node BlockNode) string {
	var b strings.Builder
	b.WriteString("<table><thead><tr>")
	for _, header := range node.Headers {
		b.WriteString("<th>")
		b.WriteString(renderInline(header))
		b.WriteString("</th>")
	}
	b.WriteString("</tr></thead><tbody>")
	for _, row := range node.Rows {
		b.WriteString("<tr>")
		for _, cell := range row {
			b.WriteString("<td>")
			b.WriteString(renderInline(cell))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}
