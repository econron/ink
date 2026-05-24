package markdown

import (
	"strconv"
	"strings"
	"unicode"
)

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

func ToHTML(markdown string) string {
	return renderHTML(parseBlocks(markdown))
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
