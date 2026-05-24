package markdown

import (
	"fmt"
	"html"
	"strings"
	"unicode"
)

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
