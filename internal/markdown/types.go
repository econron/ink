package markdown

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
