package markdown

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeLineBreaks(t *testing.T) {
	got := normalizeLineBreaks("a\r\nb\rc\n")
	want := "a\nb\nc\n"

	if got != want {
		t.Fatalf("normalizeLineBreaks() = %q, want %q", got, want)
	}
}

func TestParseBlocksHeadingParagraphAndBlankLine(t *testing.T) {
	got := parseBlocks("# Title\n\nhello\n## SubTitle\nworld")
	want := []BlockNode{
		{Kind: BlockHeading, Level: 1, Content: "Title"},
		{Kind: BlockParagraph, Content: "hello"},
		{Kind: BlockHeading, Level: 2, Content: "SubTitle"},
		{Kind: BlockParagraph, Content: "world"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseBlocks() = %#v, want %#v", got, want)
	}
}

func TestParseBlocksBlockquoteCodeHorizontalRuleAndTable(t *testing.T) {
	markdown := strings.Join([]string{
		"> quote",
		"> second",
		"",
		"```go",
		`fmt.Println("hello")`,
		"**not strong**",
		"```",
		"",
		"---",
		"",
		"| name | age |",
		"|---|---|",
		"| aron | 29 |",
	}, "\n")

	got := parseBlocks(markdown)
	if len(got) != 4 {
		t.Fatalf("parseBlocks() returned %d nodes, want 4: %#v", len(got), got)
	}
	if got[0].Kind != BlockBlockquote || got[0].Content != "quote\nsecond" {
		t.Fatalf("blockquote node = %#v", got[0])
	}
	if got[1].Kind != BlockCodeBlock || got[1].Content != "fmt.Println(\"hello\")\n**not strong**" || got[1].Language != "go" {
		t.Fatalf("code node = %#v", got[1])
	}
	if got[2].Kind != BlockHorizontalRule {
		t.Fatalf("hr node = %#v", got[2])
	}
	wantTable := BlockNode{Kind: BlockTable, Headers: []string{"name", "age"}, Rows: [][]string{{"aron", "29"}}}
	if !reflect.DeepEqual(got[3], wantTable) {
		t.Fatalf("table node = %#v, want %#v", got[3], wantTable)
	}
}

func TestCommonMarkHeadingsAndParagraphs(t *testing.T) {
	got := renderBlocks(parseBlocks(strings.Join([]string{
		"# Title ###",
		"",
		"#",
		"",
		"Setext",
		"---",
		"",
		"hello",
		"world",
	}, "\n")))

	for _, want := range []string{
		"<h1>Title</h1>",
		"<h1></h1>",
		"<h2>Setext</h2>",
		"<p>hello\nworld</p>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered blocks do not contain %q: %q", want, got)
		}
	}
}

func TestCommonMarkListPatterns(t *testing.T) {
	got := renderBlocks(parseBlocks(strings.Join([]string{
		"3) three",
		"4) four",
		"",
		"- parent",
		"  continued",
		"  - child",
		"- second",
	}, "\n")))

	for _, want := range []string{
		`<ol start="3"><li>three</li><li>four</li></ol>`,
		"<ul><li>parent\ncontinued<ul><li>child</li></ul></li><li>second</li></ul>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered list does not contain %q: %q", want, got)
		}
	}
}

func TestCommonMarkLooseList(t *testing.T) {
	got := renderBlocks(parseBlocks("- a\n\n  b\n- c"))
	want := "<ul><li><p>a</p><p>b</p></li><li><p>c</p></li></ul>"
	if got != want {
		t.Fatalf("rendered loose list = %q, want %q", got, want)
	}
}

func TestCommonMarkCodeBlocks(t *testing.T) {
	got := renderBlocks(parseBlocks(strings.Join([]string{
		"    indented",
		"    **literal**",
		"",
		"````go",
		`fmt.Println("x")`,
		"```",
		"````",
	}, "\n")))

	for _, want := range []string{
		"<pre><code>indented\n**literal**</code></pre>",
		"<pre><code class=\"language-go\">fmt.Println(&#34;x&#34;)\n```</code></pre>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered code blocks do not contain %q: %q", want, got)
		}
	}
}

func TestRawHTMLBlock(t *testing.T) {
	got := renderBlocks(parseBlocks("<div>\n<strong>raw</strong>\n</div>\n\ntext"))
	want := "<div>\n<strong>raw</strong>\n</div><p>text</p>"
	if got != want {
		t.Fatalf("rendered html block = %q, want %q", got, want)
	}
}

func TestRenderInline(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "inline code",
			content: "`code`",
			want:    "<code>code</code>",
		},
		{
			name:    "strong and emphasis",
			content: "**bold** and *em*",
			want:    "<strong>bold</strong> and <em>em</em>",
		},
		{
			name:    "strikethrough",
			content: "~~gone~~",
			want:    "<del>gone</del>",
		},
		{
			name:    "link",
			content: `[OpenAI](https://openai.com "OpenAI site")`,
			want:    `<a href="https://openai.com" title="OpenAI site">OpenAI</a>`,
		},
		{
			name:    "bare url",
			content: `docs: https://cli.urfave.org/v3/examples/subcommands/basics/`,
			want:    `docs: <a href="https://cli.urfave.org/v3/examples/subcommands/basics/">https://cli.urfave.org/v3/examples/subcommands/basics/</a>`,
		},
		{
			name:    "angle autolink",
			content: `<https://openai.com>`,
			want:    `<a href="https://openai.com">https://openai.com</a>`,
		},
		{
			name:    "bare url trims punctuation",
			content: `see https://openai.com.`,
			want:    `see <a href="https://openai.com">https://openai.com</a>.`,
		},
		{
			name:    "image",
			content: `![alt](image.png "Image title")`,
			want:    `<img src="image.png" alt="alt" title="Image title">`,
		},
		{
			name:    "escape non tag html",
			content: "2 < 3",
			want:    "2 &lt; 3",
		},
		{
			name:    "raw inline html",
			content: "a <span>ok</span>",
			want:    "a <span>ok</span>",
		},
		{
			name:    "backslash escapes punctuation",
			content: `\*literal\*`,
			want:    "*literal*",
		},
		{
			name:    "underscore inside word",
			content: "foo_bar_baz",
			want:    "foo_bar_baz",
		},
		{
			name:    "do not parse inside inline code",
			content: "`**bold**`",
			want:    "<code>**bold**</code>",
		},
		{
			name:    "unmatched marker",
			content: "**bold",
			want:    "**bold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderInline(tt.content)
			if got != tt.want {
				t.Fatalf("renderInline(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestRenderBlock(t *testing.T) {
	tests := []struct {
		name string
		node BlockNode
		want string
	}{
		{
			name: "heading",
			node: BlockNode{Kind: BlockHeading, Level: 2, Content: "SubTitle"},
			want: "<h2>SubTitle</h2>",
		},
		{
			name: "paragraph",
			node: BlockNode{Kind: BlockParagraph, Content: "hello **world**"},
			want: "<p>hello <strong>world</strong></p>",
		},
		{
			name: "unordered list",
			node: BlockNode{Kind: BlockUnorderedList, Items: []string{"a", "b"}},
			want: "<ul><li>a</li><li>b</li></ul>",
		},
		{
			name: "ordered list",
			node: BlockNode{Kind: BlockOrderedList, Items: []string{"one", "two"}},
			want: "<ol><li>one</li><li>two</li></ol>",
		},
		{
			name: "blockquote",
			node: BlockNode{Kind: BlockBlockquote, Children: []BlockNode{{Kind: BlockParagraph, Content: "quote\nsecond"}}},
			want: "<blockquote><p>quote\nsecond</p></blockquote>",
		},
		{
			name: "code block",
			node: BlockNode{Kind: BlockCodeBlock, Content: "**not strong** <tag>", Language: "go"},
			want: `<pre><code class="language-go">**not strong** &lt;tag&gt;</code></pre>`,
		},
		{
			name: "mermaid code block",
			node: BlockNode{Kind: BlockCodeBlock, Content: "graph TD\nA[<start>] --> B[end]", Language: "mermaid"},
			want: `<div class="mermaid">graph TD
A[&lt;start&gt;] --&gt; B[end]</div>`,
		},
		{
			name: "horizontal rule",
			node: BlockNode{Kind: BlockHorizontalRule},
			want: "<hr>",
		},
		{
			name: "table",
			node: BlockNode{Kind: BlockTable, Headers: []string{"name", "age"}, Rows: [][]string{{"aron", "29"}}},
			want: "<table><thead><tr><th>name</th><th>age</th></tr></thead><tbody><tr><td>aron</td><td>29</td></tr></tbody></table>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBlock(tt.node)
			if got != tt.want {
				t.Fatalf("renderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderHTML(t *testing.T) {
	got := renderHTML([]BlockNode{
		{Kind: BlockHeading, Level: 1, Content: "Title"},
		{Kind: BlockParagraph, Content: "hello"},
	})

	for _, want := range []string{
		"<!DOCTYPE html>",
		`<meta charset="UTF-8">`,
		`<meta name="viewport" content="width=device-width, initial-scale=1">`,
		"<style>",
		".ink-document",
		".mermaid",
		`<main class="ink-document">`,
		"<h1>Title</h1>",
		"<p>hello</p>",
		"</main>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderHTML() does not contain %q: %q", want, got)
		}
	}

	if strings.Contains(got, "mermaid.esm.min.mjs") {
		t.Fatalf("renderHTML() included Mermaid script without Mermaid block: %q", got)
	}
}

func TestToHTMLFragment(t *testing.T) {
	got := ToHTMLFragment("# Title\n\nhello")
	want := "<h1>Title</h1>\n<p>hello</p>\n"
	if got != want {
		t.Fatalf("ToHTMLFragment() = %q, want %q", got, want)
	}
	if strings.Contains(got, "<!DOCTYPE html>") || strings.Contains(got, "<main") {
		t.Fatalf("ToHTMLFragment() returned full document HTML: %q", got)
	}
}

func TestRenderHTMLWithMermaid(t *testing.T) {
	got := renderHTML([]BlockNode{
		{Kind: BlockCodeBlock, Language: "mermaid", Content: "graph TD\nA --> B"},
	})

	for _, want := range []string{
		`<div class="mermaid">graph TD
A --&gt; B</div>`,
		`https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs`,
		"mermaid.initialize({ startOnLoad: true });",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderHTML() does not contain %q: %q", want, got)
		}
	}
}

func TestRenderTestdataBasicMarkdown(t *testing.T) {
	html := renderTestdataMarkdown(t, "basic.md")

	for _, want := range []string{
		"<h1>Basic Document</h1>",
		`<a href="https://openai.com">OpenAI</a>`,
		"<code>inline code</code>",
		"<ul><li>first item</li><li>second item</li></ul>",
		`<pre><code class="language-go">fmt.Println(&#34;hello&#34;)</code></pre>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("basic.md rendered HTML does not contain %q: %q", want, html)
		}
	}
}

func TestRenderTestdataAdvancedMarkdown(t *testing.T) {
	html := renderTestdataMarkdown(t, "advanced.md")

	for _, want := range []string{
		"<h1>Advanced Document</h1>",
		"<blockquote><p>This is a quoted paragraph.\nIt spans two lines.</p></blockquote>",
		"<table><thead><tr><th>name</th><th>role</th></tr></thead><tbody><tr><td>ink</td><td>previewer</td></tr></tbody></table>",
		`<div class="note">`,
		`<div class="mermaid">flowchart LR`,
		"APIGW[&#34;API Gateway&#34;] --&gt; LambdaA[&#34;Lambda&#34;]",
		`https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("advanced.md rendered HTML does not contain %q: %q", want, html)
		}
	}
}

func TestBrokenMarkdownFallsBackWithoutError(t *testing.T) {
	html := renderTestdataMarkdown(t, "broken.md")

	for _, want := range []string{
		"<h1>Broken Input</h1>",
		"<p>This has **unclosed strong markup.</p>",
		"<p>This link is broken: [OpenAI](https://openai.com</p>",
		"<p>This image is broken: ![alt](image.png</p>",
		"<p>| name | value |\n| not a separator |</p>",
		"<p>&lt; 3 should not become raw HTML.</p>",
		`<pre><code class="language-go">fmt.Println(&#34;still rendered as code&#34;)
**not parsed**
</code></pre>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("broken.md rendered HTML does not contain %q: %q", want, html)
		}
	}
}

func renderTestdataMarkdown(t *testing.T, name string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}

	return renderHTML(parseBlocks(string(raw)))
}

func renderBlocks(nodes []BlockNode) string {
	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(renderBlock(node))
	}
	return b.String()
}
