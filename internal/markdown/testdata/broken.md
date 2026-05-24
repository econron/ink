# Broken Input

This has **unclosed strong markup.

This link is broken: [OpenAI](https://openai.com

This image is broken: ![alt](image.png

| name | value |
| not a separator |

< 3 should not become raw HTML.

```go
fmt.Println("still rendered as code")
**not parsed**
