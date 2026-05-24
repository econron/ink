# Markdown パーサー設計方針

## 目的

Markdown を HTML に変換する処理を、いきなり文字列置換で実装するのではなく、次の流れで設計する。

```txt
入力 Markdown
  ↓
改行コードを正規化する
  ↓
行単位に分割する
  ↓
Markdown の構文カテゴリに分類する
  ↓
中間表現に変換する
  ↓
HTML としてレンダリングする
```

この設計にすることで、Markdown を読む責務と HTML を出力する責務を分離できる。

---

## 基本方針

最初から `mdLine -> html string` に直接変換しない。

悪い方向の例。

```go
if isH1(line) {
    return "<h1>...</h1>"
}
```

この書き方は小さいうちはわかりやすいが、対応する記法が増えると `parseMdToHTML` が肥大化する。

目指す方向は次の形。

```txt
line
  ↓
これは heading level 1 である
  ↓
BlockNode{Kind: Heading, Level: 1, Content: "..."}
  ↓
HTML にすると h1
```

つまり、次の2つを分ける。

```txt
Markdown を読む責務
HTML を出す責務
```

---

## Markdown の大カテゴリ

Markdown の構文は、大きく分けると次の2種類。

```txt
Markdown
├── Block
│   ├── Heading
│   ├── Paragraph
│   ├── List
│   ├── Blockquote
│   ├── CodeBlock
│   ├── HorizontalRule
│   └── Table
│
└── Inline
    ├── Emphasis
    ├── Strong
    ├── InlineCode
    ├── Link
    ├── Image
    └── Strikethrough
```

---

## Block 系

Block は、行単位・段落単位で意味が決まる構文。

### Heading

```md
# h1
## h2
### h3
#### h4
##### h5
###### h6
```

HTML では次のタグに対応する。

```txt
#      -> h1
##     -> h2
###    -> h3
####   -> h4
#####  -> h5
###### -> h6
```

---

### Paragraph

```md
これは普通の文章です。
```

HTML では `p` に対応する。

```html
<p>これは普通の文章です。</p>
```

---

### Unordered List

```md
- item
- item
- item
```

HTML では `ul` と `li` に対応する。

```html
<ul>
  <li>item</li>
  <li>item</li>
  <li>item</li>
</ul>
```

`-` だけでなく、Markdown では `*` や `+` も unordered list として扱われることがある。

```md
* item
+ item
```

---

### Ordered List

```md
1. item
2. item
3. item
```

HTML では `ol` と `li` に対応する。

```html
<ol>
  <li>item</li>
  <li>item</li>
  <li>item</li>
</ol>
```

---

### Blockquote

```md
> quoted text
```

HTML では `blockquote` に対応する。

```html
<blockquote>quoted text</blockquote>
```

---

### Code Block

````md
```go
fmt.Println("hello")
```
````

HTML では、概ね `pre` と `code` に対応する。

```html
<pre><code>fmt.Println("hello")</code></pre>
```

Code Block は「今コードブロックの中にいるか」という状態管理が必要になる。

---

### Horizontal Rule

```md
---
```

または、

```md
***
```

または、

```md
___
```

HTML では `hr` に対応する。

```html
<hr>
```

---

### Table

```md
| name | age |
|---|---|
| aron | 29 |
```

HTML では `table` に対応する。

```html
<table>
  <tr>
    <th>name</th>
    <th>age</th>
  </tr>
  <tr>
    <td>aron</td>
    <td>29</td>
  </tr>
</table>
```

ただし Table は Markdown の標準仕様というより、GitHub Flavored Markdown などの拡張として扱われることが多い。

---

## Inline 系

Inline は、1行の中の一部に意味を持たせる構文。

### Emphasis

```md
*italic*
_italic_
```

HTML では `em` に対応する。

```html
<em>italic</em>
```

---

### Strong

```md
**bold**
__bold__
```

HTML では `strong` に対応する。

```html
<strong>bold</strong>
```

---

### Inline Code

```md
`code`
```

HTML では `code` に対応する。

```html
<code>code</code>
```

---

### Link

```md
[OpenAI](https://openai.com)
```

HTML では `a` に対応する。

```html
<a href="https://openai.com">OpenAI</a>
```

---

### Image

```md
![alt text](image.png)
```

HTML では `img` に対応する。

```html
<img src="image.png" alt="alt text">
```

---

### Strikethrough

```md
~~deleted~~
```

HTML では `del` に対応する。

```html
<del>deleted</del>
```

これも GitHub Flavored Markdown などの拡張として扱われることが多い。

---

## Go でのカテゴリ定義案

まず Markdown の大カテゴリを定義する。

```go
type MarkdownCategory string

const (
    CategoryBlock  MarkdownCategory = "block"
    CategoryInline MarkdownCategory = "inline"
)
```

Block 系の種類を定義する。

```go
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
)
```

Inline 系の種類を定義する。

```go
type InlineKind string

const (
    InlineEmphasis      InlineKind = "emphasis"
    InlineStrong        InlineKind = "strong"
    InlineCode          InlineKind = "inline_code"
    InlineLink          InlineKind = "link"
    InlineImage         InlineKind = "image"
    InlineStrikethrough InlineKind = "strikethrough"
)
```

---

## HTML タグ対応マップ

Heading は `#` の数で `h1` から `h6` に変わるため、level で map する。

```go
var headingHTMLTagMap = map[int]string{
    1: "h1",
    2: "h2",
    3: "h3",
    4: "h4",
    5: "h5",
    6: "h6",
}
```

Block 系の一部はそのまま HTML タグに対応できる。

```go
var blockHTMLTagMap = map[BlockKind]string{
    BlockParagraph:      "p",
    BlockBlockquote:     "blockquote",
    BlockCodeBlock:      "pre",
    BlockHorizontalRule: "hr",
    BlockTable:          "table",
}
```

List は `ul` と `ol` に分かれる。

```go
var listHTMLTagMap = map[BlockKind]string{
    BlockUnorderedList: "ul",
    BlockOrderedList:   "ol",
}
```

Inline 系のタグ対応。

```go
var inlineHTMLTagMap = map[InlineKind]string{
    InlineEmphasis:      "em",
    InlineStrong:        "strong",
    InlineCode:          "code",
    InlineLink:          "a",
    InlineImage:         "img",
    InlineStrikethrough: "del",
}
```

---

## 中間表現を持つ

いきなり HTML 文字列を作らず、まず Markdown を中間表現に変換する。

最初はこれくらいでよい。

```go
type BlockNode struct {
    Kind    BlockKind
    Level   int
    Content string
}
```

Heading の例。

```go
BlockNode{
    Kind:    BlockHeading,
    Level:   2,
    Content: "タイトル",
}
```

Paragraph の例。

```go
BlockNode{
    Kind:    BlockParagraph,
    Content: "これは本文です",
}
```

List まで扱うなら、将来的にこう拡張する。

```go
type BlockNode struct {
    Kind    BlockKind
    Level   int
    Content string
    Items   []string
}
```

Unordered List の例。

```go
BlockNode{
    Kind:  BlockUnorderedList,
    Items: []string{"item1", "item2", "item3"},
}
```

---

## 全体の関数分割

最初に目指す分割はこれ。

```go
func normalizeLineBreaks(markdown string) string

func parseBlocks(markdown string) []BlockNode

func renderHTML(nodes []BlockNode) string
```

全体の流れ。

```txt
parseMarkdown
  ↓
normalizeLineBreaks
  ↓
parseBlocks
  ↓
renderHTML
```

Inline まで扱う場合は、さらに次のように分ける。

```go
func parseInline(content string) string
```

その場合の流れ。

```txt
parseMarkdown
  ↓
normalizeLineBreaks
  ↓
parseBlocks
  ↓
各 BlockNode の Content に parseInline を適用
  ↓
renderHTML
```

---

## 実装ステップ

### Step 1: 改行コードを正規化する

対象にする改行コード。

```txt

      LF    Unix / Linux / macOS 現代

    CRLF  Windows
      CR    古い Mac
```

実装例。

```go
func normalizeLineBreaks(markdown string) string {
    normalized := strings.ReplaceAll(markdown, "
", "
")
    normalized = strings.ReplaceAll(normalized, "", "
")
    return normalized
}
```

注意点。

```go
normalized := strings.ReplaceAll(markdown, "
", "
")
normalized = strings.ReplaceAll(normalized, "", "
")
```

2回目は `markdown` ではなく `normalized` を使う。

誤り。

```go
normalized := strings.ReplaceAll(markdown, "
", "
")
normalized = strings.ReplaceAll(markdown, "", "
")
```

これだと1回目の正規化結果が捨てられる。

---

### Step 2: 行単位に分割する

```go
lines := strings.Split(normalized, "
")
```

開発中は `%q` で中身を見る。

```go
fmt.Printf("%q
", raw)
fmt.Printf("%q
", normalized)

for i, line := range lines {
    fmt.Printf("[%d] %q
", i, line)
}
```

---

### Step 3: Heading を BlockNode にする

`isH1`, `isH2`, `isH3` を増やすのではなく、`#` の数を数える関数にする。

目標。

```go
level, content, ok := parseHeading("## hello")
```

戻り値。

```txt
level = 2
content = "hello"
ok = true
```

考え方。

```txt
行頭から # を数える
# の数が 1 以上 6 以下である
次の文字が space である
space の後ろに本文がある
```

---

### Step 4: Paragraph を扱う

Heading でも List でも Blockquote でも CodeBlock でもない普通の行は Paragraph とみなす。

```go
BlockNode{
    Kind:    BlockParagraph,
    Content: line,
}
```

ただし、空行は Paragraph にしない。

---

### Step 5: HTML にレンダリングする

`BlockNode` を受け取って HTML に変換する。

```go
func renderBlock(node BlockNode) string {
    switch node.Kind {
    case BlockHeading:
        tag := headingHTMLTagMap[node.Level]
        return fmt.Sprintf("<%s>%s</%s>", tag, node.Content, tag)

    case BlockParagraph:
        return fmt.Sprintf("<p>%s</p>", node.Content)

    default:
        return node.Content
    }
}
```

複数のノードを HTML にする。

```go
func renderHTML(nodes []BlockNode) string {
    var b strings.Builder

    b.WriteString("<html><body>")

    for _, node := range nodes {
        b.WriteString(renderBlock(node))
    }

    b.WriteString("</body></html>")

    return b.String()
}
```

---

## 最初に対応する範囲

最初はこれだけでよい。

```txt
1. 改行正規化
2. Heading
3. Paragraph
4. 空行スキップ
5. HTML レンダリング
```

この時点で、次の Markdown を HTML にできればよい。

```md
# Title
hello
## SubTitle
world
```

中間表現。

```go
[]BlockNode{
    {Kind: BlockHeading, Level: 1, Content: "Title"},
    {Kind: BlockParagraph, Content: "hello"},
    {Kind: BlockHeading, Level: 2, Content: "SubTitle"},
    {Kind: BlockParagraph, Content: "world"},
}
```

HTML。

```html
<html><body><h1>Title</h1><p>hello</p><h2>SubTitle</h2><p>world</p></body></html>
```

---

## 次に対応する範囲

次はこの順番で進める。

```txt
1. Unordered List
2. Ordered List
3. Blockquote
4. CodeBlock
5. InlineCode
6. Strong
7. Emphasis
8. Link
```

特に List と CodeBlock は状態管理が必要。

List の例。

```md
- a
- b
- c
```

これは1行ずつ単独で処理すると、正しい HTML にならない。

期待する HTML。

```html
<ul>
  <li>a</li>
  <li>b</li>
  <li>c</li>
</ul>
```

そのため、`parseBlocks` の中で次のような状態を持つ必要がある。

```txt
今リストの中か？
リストが続いているか？
リストが終わったか？
```

CodeBlock も同様。

````md
```go
fmt.Println("hello")
```
````

必要な状態。

```txt
今コードブロックの中か？
開始フェンスは ``` か ~~~ か？
言語指定はあるか？
どこでコードブロックが終わるか？
```

---

## 重要な設計判断

### Markdown 構文と HTML タグを混ぜすぎない

Markdown の `## title` は、直接 `h2` なのではない。

まずこれは、

```txt
Heading level 2
```

である。

その後、HTML にするときに、

```txt
Heading level 2 -> h2
```

になる。

この分離が重要。

---

### parse と render を分ける

避けたい形。

```go
func parseMdToHTML(markdown string) string {
    // Markdown を読みながら、その場で HTML を連結する
}
```

目指す形。

```go
func parseBlocks(markdown string) []BlockNode {
    // Markdown を読み、中間表現にする
}

func renderHTML(nodes []BlockNode) string {
    // 中間表現を HTML にする
}
```

この形だとテストしやすい。

例えば、parse のテストは HTML を見なくてもよい。

```go
nodes := parseBlocks("# Title")

want := []BlockNode{
    {Kind: BlockHeading, Level: 1, Content: "Title"},
}
```

render のテストは Markdown を見なくてもよい。

```go
html := renderHTML([]BlockNode{
    {Kind: BlockHeading, Level: 1, Content: "Title"},
})

want := "<html><body><h1>Title</h1></body></html>"
```

---

## まとめ

この設計で大事なのは、次の3段階に分けること。

```txt
Markdown 文字列
  ↓
Markdown 構文として分類された中間表現
  ↓
HTML 文字列
```

最初は小さく始める。

```txt
Heading
Paragraph
Blank line
```

その後に、状態管理が必要な構文へ進む。

```txt
List
CodeBlock
Blockquote
```

さらにその後、Inline 系へ進む。

```txt
InlineCode
Strong
Emphasis
Link
Image
```

この順番で進めると、Markdown パーサーを作りながら、文字列処理・状態管理・中間表現・レンダリング分離を自然に学べる。
