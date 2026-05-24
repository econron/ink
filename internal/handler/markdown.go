package handler

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"
)

type BlockKind string

const (
	BlockHeading        BlockKind = "heading"
	BlockParagraph      BlockKind = "paragraph"
	BlockUnorderedList  BlockKind = "unordered_list"
	BlockOrderedList    BlockKind = "ordered_list"
	BlockBlockquote     BlockKind = "blockquote"
	BlockCodeBlock      BlockKind = "code_block"
	BlockHorizontalRule BlockKind = "horizontal_rule"
	BlockTable          BlockKind = "table"
	BlockHTML           BlockKind = "html"
)

type InlineKind string

const (
	InlineEmphasis      InlineKind = "emphasis"
	InlineStrong        InlineKind = "strong"
	InlineCode          InlineKind = "inline_code"
	InlineLink          InlineKind = "link"
	InlineImage         InlineKind = "image"
	InlineStrikethrough InlineKind = "strikethrough"
)

type BlockNode struct {
	Kind      BlockKind
	Level     int
	Content   string
	Items     []string
	ListItems []ListItem
	Headers   []string
	Rows      [][]string
	Children  []BlockNode
	Language  string
	Start     int
	Tight     bool
}

type ListItem struct {
	Blocks []BlockNode
}

type parser struct {
	lines []string
}

type listMarker struct {
	kind          BlockKind
	start         int
	markerIndent  int
	contentIndent int
	content       string
}

type codeFence struct {
	char     byte
	length   int
	indent   int
	language string
}

var headingHTMLTagMap = map[int]string{
	1: "h1",
	2: "h2",
	3: "h3",
	4: "h4",
	5: "h5",
	6: "h6",
}

var listHTMLTagMap = map[BlockKind]string{
	BlockUnorderedList: "ul",
	BlockOrderedList:   "ol",
}

var inlineHTMLTagMap = map[InlineKind]string{
	InlineEmphasis:      "em",
	InlineStrong:        "strong",
	InlineCode:          "code",
	InlineLink:          "a",
	InlineImage:         "img",
	InlineStrikethrough: "del",
}

var htmlBlockTags = map[string]bool{
	"address": true, "article": true, "aside": true, "base": true, "basefont": true,
	"blockquote": true, "body": true, "caption": true, "center": true, "col": true,
	"colgroup": true, "dd": true, "details": true, "dialog": true, "dir": true,
	"div": true, "dl": true, "dt": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "frame": true, "frameset": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"head": true, "header": true, "hr": true, "html": true, "iframe": true,
	"legend": true, "li": true, "link": true, "main": true, "menu": true,
	"menuitem": true, "nav": true, "noframes": true, "ol": true, "optgroup": true,
	"option": true, "p": true, "param": true, "search": true, "section": true,
	"summary": true, "table": true, "tbody": true, "td": true, "tfoot": true,
	"th": true, "thead": true, "title": true, "tr": true, "track": true, "ul": true,
}

func normalizeLineBreaks(markdown string) string {
	normalized := strings.ReplaceAll(markdown, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return normalized
}

func parseBlocks(markdown string) []BlockNode {
	normalized := normalizeLineBreaks(markdown)
	p := parser{lines: strings.Split(normalized, "\n")}
	return p.parseRange(0, len(p.lines))
}

func parseBlocksFromLines(lines []string) []BlockNode {
	p := parser{lines: lines}
	return p.parseRange(0, len(lines))
}

func (p parser) parseRange(start, end int) []BlockNode {
	var nodes []BlockNode
	for i := start; i < end; {
		line := p.lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			i++
			continue
		}

		if fence, ok := parseCodeFence(line); ok {
			node, next := p.parseFencedCodeBlock(i, end, fence)
			nodes = append(nodes, node)
			i = next
			continue
		}

		if isIndentedCodeLine(line) {
			node, next := p.parseIndentedCodeBlock(i, end)
			nodes = append(nodes, node)
			i = next
			continue
		}

		if isHTMLBlockStart(line) {
			node, next := p.parseHTMLBlock(i, end)
			nodes = append(nodes, node)
			i = next
			continue
		}

		if node, next, ok := p.parseTable(i, end); ok {
			nodes = append(nodes, node)
			i = next
			continue
		}

		if level, content, ok := parseHeading(line); ok {
			nodes = append(nodes, BlockNode{
				Kind:    BlockHeading,
				Level:   level,
				Content: content,
			})
			i++
			continue
		}

		if isHorizontalRule(line) {
			nodes = append(nodes, BlockNode{Kind: BlockHorizontalRule})
			i++
			continue
		}

		if marker, ok := parseListMarker(line); ok {
			node, next := p.parseList(i, end, marker)
			nodes = append(nodes, node)
			i = next
			continue
		}

		if _, ok := parseBlockquoteLine(line); ok {
			node, next := p.parseBlockquote(i, end)
			nodes = append(nodes, node)
			i = next
			continue
		}

		node, next := p.parseParagraph(i, end)
		nodes = append(nodes, node)
		i = next
	}

	return nodes
}

func (p parser) parseParagraph(start, end int) (BlockNode, int) {
	if start+1 < end {
		if level, ok := parseSetextHeadingLine(p.lines[start+1]); ok {
			return BlockNode{
				Kind:    BlockHeading,
				Level:   level,
				Content: strings.TrimSpace(p.lines[start]),
			}, start + 2
		}
	}

	lines := []string{strings.TrimSpace(p.lines[start])}
	i := start + 1
	for i < end {
		if strings.TrimSpace(p.lines[i]) == "" {
			break
		}
		if len(lines) == 1 {
			if level, ok := parseSetextHeadingLine(p.lines[i]); ok {
				return BlockNode{Kind: BlockHeading, Level: level, Content: lines[0]}, i + 1
			}
		}
		if isParagraphInterrupt(p.lines, i, end) {
			break
		}
		lines = append(lines, strings.TrimSpace(p.lines[i]))
		i++
	}

	return BlockNode{
		Kind:    BlockParagraph,
		Content: strings.Join(lines, "\n"),
	}, i
}

func isParagraphInterrupt(lines []string, i, end int) bool {
	if _, ok := parseCodeFence(lines[i]); ok {
		return true
	}
	if isHTMLBlockStart(lines[i]) {
		return true
	}
	if isTableStart(lines, i, end) {
		return true
	}
	if _, _, ok := parseHeading(lines[i]); ok {
		return true
	}
	if isHorizontalRule(lines[i]) {
		return true
	}
	if marker, ok := parseListMarker(lines[i]); ok {
		return marker.kind == BlockUnorderedList || marker.start == 1
	}
	if _, ok := parseBlockquoteLine(lines[i]); ok {
		return true
	}
	return false
}

func parseHeading(line string) (int, string, bool) {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return 0, "", false
	}

	trimmed := stripIndent(line, indent)
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if len(trimmed) > level && trimmed[level] != ' ' && trimmed[level] != '\t' {
		return 0, "", false
	}

	content := ""
	if len(trimmed) > level {
		content = strings.TrimSpace(trimmed[level+1:])
	}
	return level, stripClosingHeadingHashes(content), true
}

func stripClosingHeadingHashes(content string) string {
	trimmed := strings.TrimRightFunc(content, unicode.IsSpace)
	if trimmed == "" || trimmed[len(trimmed)-1] != '#' {
		return strings.TrimSpace(content)
	}

	start := len(trimmed) - 1
	for start >= 0 && trimmed[start] == '#' {
		start--
	}
	if start < 0 || trimmed[start] != ' ' && trimmed[start] != '\t' {
		return strings.TrimSpace(content)
	}
	return strings.TrimSpace(trimmed[:start])
}

func parseSetextHeadingLine(line string) (int, bool) {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return 0, false
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return 0, false
	}

	marker := trimmed[0]
	if marker != '=' && marker != '-' {
		return 0, false
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != marker {
			return 0, false
		}
	}
	if marker == '=' {
		return 1, true
	}
	return 2, true
}

func parseListMarker(line string) (listMarker, bool) {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return listMarker{}, false
	}

	rest := stripIndent(line, indent)
	if rest == "" {
		return listMarker{}, false
	}

	if rest[0] == '-' || rest[0] == '*' || rest[0] == '+' {
		if len(rest) > 1 && rest[1] != ' ' && rest[1] != '\t' {
			return listMarker{}, false
		}
		contentIndent := indent + 1
		content := ""
		if len(rest) > 1 {
			contentIndent++
			content = strings.TrimRightFunc(rest[2:], unicode.IsSpace)
		}
		return listMarker{
			kind:          BlockUnorderedList,
			start:         0,
			markerIndent:  indent,
			contentIndent: contentIndent,
			content:       content,
		}, true
	}

	digitEnd := 0
	for digitEnd < len(rest) && digitEnd < 9 && rest[digitEnd] >= '0' && rest[digitEnd] <= '9' {
		digitEnd++
	}
	if digitEnd == 0 || digitEnd >= len(rest) {
		return listMarker{}, false
	}
	if rest[digitEnd] != '.' && rest[digitEnd] != ')' {
		return listMarker{}, false
	}
	if digitEnd+1 < len(rest) && rest[digitEnd+1] != ' ' && rest[digitEnd+1] != '\t' {
		return listMarker{}, false
	}

	start, err := strconv.Atoi(rest[:digitEnd])
	if err != nil {
		return listMarker{}, false
	}

	contentIndent := indent + digitEnd + 1
	content := ""
	if digitEnd+1 < len(rest) {
		contentIndent++
		content = strings.TrimRightFunc(rest[digitEnd+2:], unicode.IsSpace)
	}
	return listMarker{
		kind:          BlockOrderedList,
		start:         start,
		markerIndent:  indent,
		contentIndent: contentIndent,
		content:       content,
	}, true
}

func (p parser) parseList(start, end int, first listMarker) (BlockNode, int) {
	items := make([]ListItem, 0)
	flatItems := make([]string, 0)
	loose := false
	i := start

	for i < end {
		marker, ok := parseListMarker(p.lines[i])
		if !ok || marker.kind != first.kind || marker.markerIndent != first.markerIndent {
			break
		}

		itemLines := make([]string, 0)
		if marker.content != "" {
			itemLines = append(itemLines, marker.content)
		}
		i++

		itemLoose := false
		for i < end {
			line := p.lines[i]
			if strings.TrimSpace(line) == "" {
				next := i + 1
				for next < end && strings.TrimSpace(p.lines[next]) == "" {
					next++
				}
				if next >= end {
					i = next
					break
				}
				nextMarker, isNextMarker := parseListMarker(p.lines[next])
				if countLeadingSpaces(p.lines[next]) < marker.contentIndent {
					if isNextMarker && nextMarker.kind == first.kind && nextMarker.markerIndent == first.markerIndent {
						itemLoose = true
						i = next
						break
					}
					break
				}
				itemLoose = true
				itemLines = append(itemLines, "")
				i++
				continue
			}

			nextMarker, isNextMarker := parseListMarker(line)
			if isNextMarker && nextMarker.kind == first.kind && nextMarker.markerIndent == first.markerIndent {
				break
			}
			if isNextMarker && nextMarker.markerIndent < marker.contentIndent {
				break
			}

			indent := countLeadingSpaces(line)
			if indent >= marker.contentIndent {
				itemLines = append(itemLines, stripIndent(line, marker.contentIndent))
				i++
				continue
			}
			break
		}

		itemLines = trimTrailingBlankLines(itemLines)
		blocks := parseBlocksFromLines(itemLines)
		if itemLoose {
			loose = true
		}
		items = append(items, ListItem{Blocks: blocks})
		if len(blocks) == 1 && blocks[0].Kind == BlockParagraph {
			flatItems = append(flatItems, blocks[0].Content)
		}
	}

	return BlockNode{
		Kind:      first.kind,
		Items:     flatItems,
		ListItems: items,
		Start:     first.start,
		Tight:     !loose,
	}, i
}

func trimTrailingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func parseBlockquoteLine(line string) (string, bool) {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return "", false
	}
	trimmed := stripIndent(line, indent)
	if !strings.HasPrefix(trimmed, ">") {
		return "", false
	}
	content := strings.TrimPrefix(trimmed, ">")
	if strings.HasPrefix(content, " ") || strings.HasPrefix(content, "\t") {
		content = content[1:]
	}
	return content, true
}

func (p parser) parseBlockquote(start, end int) (BlockNode, int) {
	quoteLines := make([]string, 0)
	i := start
	for i < end {
		quote, ok := parseBlockquoteLine(p.lines[i])
		if !ok {
			break
		}
		quoteLines = append(quoteLines, quote)
		i++
	}

	return BlockNode{
		Kind:     BlockBlockquote,
		Content:  strings.Join(quoteLines, "\n"),
		Children: parseBlocksFromLines(quoteLines),
	}, i
}

func parseCodeFence(line string) (codeFence, bool) {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return codeFence{}, false
	}

	rest := stripIndent(line, indent)
	if len(rest) < 3 || rest[0] != '`' && rest[0] != '~' {
		return codeFence{}, false
	}

	ch := rest[0]
	length := 0
	for length < len(rest) && rest[length] == ch {
		length++
	}
	if length < 3 {
		return codeFence{}, false
	}

	language := strings.TrimSpace(rest[length:])
	if ch == '`' && strings.Contains(language, "`") {
		return codeFence{}, false
	}
	return codeFence{char: ch, length: length, indent: indent, language: language}, true
}

func (p parser) parseFencedCodeBlock(start, end int, fence codeFence) (BlockNode, int) {
	codeLines := make([]string, 0)
	i := start + 1
	for i < end {
		if isClosingCodeFence(p.lines[i], fence) {
			return BlockNode{
				Kind:     BlockCodeBlock,
				Content:  strings.Join(codeLines, "\n"),
				Language: fence.language,
			}, i + 1
		}
		codeLines = append(codeLines, stripIndent(p.lines[i], fence.indent))
		i++
	}

	return BlockNode{
		Kind:     BlockCodeBlock,
		Content:  strings.Join(codeLines, "\n"),
		Language: fence.language,
	}, i
}

func isClosingCodeFence(line string, opening codeFence) bool {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return false
	}

	rest := stripIndent(line, indent)
	count := 0
	for count < len(rest) && rest[count] == opening.char {
		count++
	}
	if count < opening.length {
		return false
	}
	return strings.TrimSpace(rest[count:]) == ""
}

func isIndentedCodeLine(line string) bool {
	return strings.TrimSpace(line) != "" && countLeadingSpaces(line) >= 4
}

func (p parser) parseIndentedCodeBlock(start, end int) (BlockNode, int) {
	codeLines := make([]string, 0)
	i := start
	for i < end {
		line := p.lines[i]
		if strings.TrimSpace(line) == "" {
			codeLines = append(codeLines, "")
			i++
			continue
		}
		if countLeadingSpaces(line) < 4 {
			break
		}
		codeLines = append(codeLines, stripIndent(line, 4))
		i++
	}

	return BlockNode{
		Kind:    BlockCodeBlock,
		Content: strings.Join(trimTrailingBlankLines(codeLines), "\n"),
	}, i
}

func isHorizontalRule(line string) bool {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return false
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	var marker byte
	count := 0
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == ' ' || trimmed[i] == '\t' {
			continue
		}
		if marker == 0 {
			marker = trimmed[i]
			if marker != '-' && marker != '*' && marker != '_' {
				return false
			}
		}
		if trimmed[i] != marker {
			return false
		}
		count++
	}
	return count >= 3
}

func (p parser) parseTable(start, end int) (BlockNode, int, bool) {
	if !isTableStart(p.lines, start, end) {
		return BlockNode{}, start, false
	}

	headers := parseTableRow(p.lines[start])
	rows := make([][]string, 0)
	i := start + 2
	for i < end && isTableRow(p.lines[i]) {
		row := parseTableRow(p.lines[i])
		if len(row) > 0 {
			rows = append(rows, row)
		}
		i++
	}

	return BlockNode{
		Kind:    BlockTable,
		Headers: headers,
		Rows:    rows,
	}, i, true
}

func isTableStart(lines []string, start, end int) bool {
	if start+1 >= end || !isTableRow(lines[start]) || !isTableSeparatorRow(lines[start+1]) {
		return false
	}
	return len(parseTableRow(lines[start])) > 0
}

func isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, "|")
}

func parseTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	rawCells := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(rawCells))
	for _, cell := range rawCells {
		cells = append(cells, strings.TrimSpace(cell))
	}
	return cells
}

func isTableSeparatorRow(line string) bool {
	cells := parseTableRow(line)
	if len(cells) == 0 {
		return false
	}

	for _, cell := range cells {
		trimmed := strings.TrimSpace(cell)
		trimmed = strings.TrimPrefix(trimmed, ":")
		trimmed = strings.TrimSuffix(trimmed, ":")
		if len(trimmed) < 3 {
			return false
		}
		for _, r := range trimmed {
			if r != '-' {
				return false
			}
		}
	}
	return true
}

func isHTMLBlockStart(line string) bool {
	indent := countLeadingSpaces(line)
	if indent > 3 {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] != '<' {
		return false
	}
	if strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "<?") || strings.HasPrefix(trimmed, "<!") {
		return true
	}
	tag, ok := htmlTagName(trimmed)
	if !ok {
		return false
	}
	lower := strings.ToLower(tag)
	return lower == "script" || lower == "pre" || lower == "style" || htmlBlockTags[lower] || strings.Contains(trimmed, ">")
}

func (p parser) parseHTMLBlock(start, end int) (BlockNode, int) {
	lines := []string{p.lines[start]}
	trimmed := strings.TrimSpace(p.lines[start])
	lower := strings.ToLower(trimmed)

	if strings.HasPrefix(trimmed, "<!--") {
		i := start + 1
		for !strings.Contains(strings.Join(lines, "\n"), "-->") && i < end {
			lines = append(lines, p.lines[i])
			i++
		}
		return BlockNode{Kind: BlockHTML, Content: strings.Join(lines, "\n")}, i
	}

	for _, tag := range []string{"script", "pre", "style"} {
		if strings.HasPrefix(lower, "<"+tag) {
			closeTag := "</" + tag + ">"
			i := start + 1
			for !strings.Contains(strings.ToLower(strings.Join(lines, "\n")), closeTag) && i < end {
				lines = append(lines, p.lines[i])
				i++
			}
			return BlockNode{Kind: BlockHTML, Content: strings.Join(lines, "\n")}, i
		}
	}

	i := start + 1
	for i < end && strings.TrimSpace(p.lines[i]) != "" {
		lines = append(lines, p.lines[i])
		i++
	}
	return BlockNode{Kind: BlockHTML, Content: strings.Join(lines, "\n")}, i
}

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
		b.WriteString(fmt.Sprintf(`<%s start="%d">`, tag, node.Start))
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

func renderInline(content string) string {
	var b strings.Builder
	for i := 0; i < len(content); {
		if rendered, next, ok := parseInlineCodeAt(content, i); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseInlineImageAt(content, i); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseInlineLinkAt(content, i); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseAutoLinkAt(content, i); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseBareURLAt(content, i); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if raw, next, ok := parseInlineHTMLAt(content, i); ok {
			b.WriteString(raw)
			i = next
			continue
		}
		if rendered, next, ok := parseInlineDelimitedAt(content, i, "**", InlineStrong); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseInlineDelimitedAt(content, i, "__", InlineStrong); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseInlineDelimitedAt(content, i, "~~", InlineStrikethrough); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseInlineDelimitedAt(content, i, "*", InlineEmphasis); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if rendered, next, ok := parseInlineDelimitedAt(content, i, "_", InlineEmphasis); ok {
			b.WriteString(rendered)
			i = next
			continue
		}
		if content[i] == '\\' {
			if i+1 < len(content) && isEscapablePunctuation(content[i+1]) {
				b.WriteString(html.EscapeString(content[i+1 : i+2]))
				i += 2
				continue
			}
			b.WriteString("\\")
			i++
			continue
		}

		next := nextInlineSpecial(content, i)
		if next == i {
			b.WriteString(html.EscapeString(content[i : i+1]))
			i++
			continue
		}
		b.WriteString(html.EscapeString(content[i:next]))
		i = next
	}
	return b.String()
}

func nextInlineSpecial(content string, start int) int {
	for i := start; i < len(content); i++ {
		if strings.HasPrefix(content[i:], "http://") || strings.HasPrefix(content[i:], "https://") {
			return i
		}
		switch content[i] {
		case '`', '!', '[', '*', '_', '~', '<', '\\':
			return i
		}
	}
	return len(content)
}

func parseInlineCodeAt(content string, start int) (string, int, bool) {
	if start >= len(content) || content[start] != '`' {
		return "", 0, false
	}

	run := 0
	for start+run < len(content) && content[start+run] == '`' {
		run++
	}
	marker := strings.Repeat("`", run)
	closing := strings.Index(content[start+run:], marker)
	if closing == -1 {
		return "", 0, false
	}

	codeStart := start + run
	codeEnd := codeStart + closing
	code := strings.ReplaceAll(content[codeStart:codeEnd], "\n", " ")
	if len(code) >= 2 && code[0] == ' ' && code[len(code)-1] == ' ' && strings.TrimSpace(code) != "" {
		code = code[1 : len(code)-1]
	}

	tag := inlineHTMLTagMap[InlineCode]
	return fmt.Sprintf("<%s>%s</%s>", tag, html.EscapeString(code), tag), codeEnd + run, true
}

func parseInlineImageAt(content string, start int) (string, int, bool) {
	if !strings.HasPrefix(content[start:], "![") {
		return "", 0, false
	}

	alt, href, title, end, ok := parseInlineTargetAt(content, start, 2)
	if !ok {
		return "", 0, false
	}

	var b strings.Builder
	b.WriteString(`<img src="`)
	b.WriteString(html.EscapeString(href))
	b.WriteString(`" alt="`)
	b.WriteString(html.EscapeString(alt))
	b.WriteString(`"`)
	if title != "" {
		b.WriteString(` title="`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`"`)
	}
	b.WriteString(">")
	return b.String(), end, true
}

func parseAutoLinkAt(content string, start int) (string, int, bool) {
	if start >= len(content) || content[start] != '<' {
		return "", 0, false
	}

	endRel := strings.IndexByte(content[start:], '>')
	if endRel == -1 {
		return "", 0, false
	}

	url := content[start+1 : start+endRel]
	if !isHTTPURL(url) {
		return "", 0, false
	}

	return renderPlainAnchor(url, url), start + endRel + 1, true
}

func parseBareURLAt(content string, start int) (string, int, bool) {
	if !strings.HasPrefix(content[start:], "http://") && !strings.HasPrefix(content[start:], "https://") {
		return "", 0, false
	}
	if isInsideUnclosedMarkdownLinkTarget(content, start) {
		return "", 0, false
	}
	if start > 0 && (isASCIIAlnum(content[start-1]) || content[start-1] == '_' || content[start-1] == '-') {
		return "", 0, false
	}

	end := start
	for end < len(content) && !unicode.IsSpace(rune(content[end])) && content[end] != '<' {
		end++
	}

	urlEnd := trimBareURLTrailingPunctuation(content[start:end]) + start
	if urlEnd <= start {
		return "", 0, false
	}

	url := content[start:urlEnd]
	return renderPlainAnchor(url, url), urlEnd, true
}

func isInsideUnclosedMarkdownLinkTarget(content string, start int) bool {
	return start >= 2 && content[start-1] == '(' && content[start-2] == ']'
}

func renderAnchor(href string, text string, title string) string {
	var b strings.Builder
	b.WriteString(`<a href="`)
	b.WriteString(html.EscapeString(href))
	b.WriteString(`"`)
	if title != "" {
		b.WriteString(` title="`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`"`)
	}
	b.WriteString(">")
	b.WriteString(renderInline(text))
	b.WriteString("</a>")
	return b.String()
}

func renderPlainAnchor(href string, text string) string {
	var b strings.Builder
	b.WriteString(`<a href="`)
	b.WriteString(html.EscapeString(href))
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(text))
	b.WriteString("</a>")
	return b.String()
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func trimBareURLTrailingPunctuation(url string) int {
	end := len(url)
	for end > 0 {
		switch url[end-1] {
		case '.', ',', ':', ';', '!', '?':
			end--
			continue
		case ')':
			if strings.Count(url[:end], "(") < strings.Count(url[:end], ")") {
				end--
				continue
			}
		case ']':
			if strings.Count(url[:end], "[") < strings.Count(url[:end], "]") {
				end--
				continue
			}
		}
		break
	}
	return end
}

func parseInlineLinkAt(content string, start int) (string, int, bool) {
	if start >= len(content) || content[start] != '[' {
		return "", 0, false
	}

	text, href, title, end, ok := parseInlineTargetAt(content, start, 1)
	if !ok {
		return "", 0, false
	}

	return renderAnchor(href, text, title), end, true
}

func parseInlineTargetAt(content string, start int, textOffset int) (string, string, string, int, bool) {
	textStart := start + textOffset
	textEndRel := strings.Index(content[textStart:], "]")
	if textEndRel == -1 {
		return "", "", "", 0, false
	}
	textEnd := textStart + textEndRel
	if textEnd+1 >= len(content) || content[textEnd+1] != '(' {
		return "", "", "", 0, false
	}

	targetStart := textEnd + 2
	targetEnd := strings.Index(content[targetStart:], ")")
	if targetEnd == -1 {
		return "", "", "", 0, false
	}
	targetEnd += targetStart

	href, title, ok := parseLinkDestinationAndTitle(content[targetStart:targetEnd])
	if !ok {
		return "", "", "", 0, false
	}
	return content[textStart:textEnd], href, title, targetEnd + 1, true
}

func parseLinkDestinationAndTitle(target string) (string, string, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", true
	}

	var href string
	var rest string
	if strings.HasPrefix(target, "<") {
		end := strings.Index(target, ">")
		if end == -1 {
			return "", "", false
		}
		href = target[1:end]
		rest = strings.TrimSpace(target[end+1:])
	} else {
		split := firstWhitespaceIndex(target)
		if split == -1 {
			return target, "", true
		}
		href = target[:split]
		rest = strings.TrimSpace(target[split:])
	}

	if rest == "" {
		return href, "", true
	}
	title, ok := parseLinkTitle(rest)
	if !ok {
		return "", "", false
	}
	return href, title, true
}

func parseLinkTitle(rest string) (string, bool) {
	if len(rest) < 2 {
		return "", false
	}
	open := rest[0]
	var close byte
	switch open {
	case '"':
		close = '"'
	case '\'':
		close = '\''
	case '(':
		close = ')'
	default:
		return "", false
	}
	if rest[len(rest)-1] != close {
		return "", false
	}
	return rest[1 : len(rest)-1], true
}

func firstWhitespaceIndex(s string) int {
	for i, r := range s {
		if unicode.IsSpace(r) {
			return i
		}
	}
	return -1
}

func parseInlineHTMLAt(content string, start int) (string, int, bool) {
	if start >= len(content) || content[start] != '<' {
		return "", 0, false
	}
	if strings.HasPrefix(content[start:], "<!--") {
		end := strings.Index(content[start:], "-->")
		if end == -1 {
			return "", 0, false
		}
		return content[start : start+end+3], start + end + 3, true
	}

	end := strings.Index(content[start:], ">")
	if end == -1 {
		return "", 0, false
	}
	raw := content[start : start+end+1]
	if _, ok := htmlTagName(raw); ok || strings.HasPrefix(raw, "<!") || strings.HasPrefix(raw, "<?") {
		return raw, start + end + 1, true
	}
	return "", 0, false
}

func parseInlineDelimitedAt(content string, start int, marker string, kind InlineKind) (string, int, bool) {
	if !strings.HasPrefix(content[start:], marker) {
		return "", 0, false
	}
	if strings.HasPrefix(marker, "_") && isIntraWordDelimiter(content, start, len(marker)) {
		return "", 0, false
	}

	searchStart := start + len(marker)
	for searchStart < len(content) {
		endRel := strings.Index(content[searchStart:], marker)
		if endRel == -1 {
			return "", 0, false
		}
		end := searchStart + endRel
		text := content[start+len(marker) : end]
		if text == "" {
			searchStart = end + len(marker)
			continue
		}
		if strings.HasPrefix(marker, "_") && isIntraWordDelimiter(content, end, len(marker)) {
			searchStart = end + len(marker)
			continue
		}

		tag := inlineHTMLTagMap[kind]
		return fmt.Sprintf("<%s>%s</%s>", tag, renderInline(text), tag), end + len(marker), true
	}
	return "", 0, false
}

func isIntraWordDelimiter(content string, start int, markerLen int) bool {
	prevIsWord := start > 0 && isASCIIAlnum(content[start-1])
	nextIndex := start + markerLen
	nextIsWord := nextIndex < len(content) && isASCIIAlnum(content[nextIndex])
	return prevIsWord && nextIsWord
}

func isASCIIAlnum(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func isEscapablePunctuation(b byte) bool {
	return strings.ContainsRune(`!"#$%&'()*+,-./:;<=>?@[\]^_`+"`"+`{|}~`, rune(b))
}

func htmlTagName(raw string) (string, bool) {
	if len(raw) < 3 || raw[0] != '<' {
		return "", false
	}

	i := 1
	if i < len(raw) && raw[i] == '/' {
		i++
	}
	if i >= len(raw) || !isASCIILetter(raw[i]) {
		return "", false
	}

	start := i
	for i < len(raw) && (isASCIILetter(raw[i]) || raw[i] >= '0' && raw[i] <= '9' || raw[i] == '-') {
		i++
	}
	if i >= len(raw) {
		return "", false
	}

	switch raw[i] {
	case ' ', '\t', '\n', '/', '>':
		return raw[start:i], true
	default:
		return "", false
	}
}

func isASCIILetter(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func countLeadingSpaces(line string) int {
	count := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ':
			count++
		case '\t':
			count += 4
		default:
			return count
		}
	}
	return count
}

func stripIndent(line string, indent int) string {
	column := 0
	i := 0
	for i < len(line) && column < indent {
		switch line[i] {
		case ' ':
			column++
			i++
		case '\t':
			column += 4
			i++
		default:
			return line[i:]
		}
	}
	return line[i:]
}
