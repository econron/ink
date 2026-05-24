package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/econron/browser"
	"github.com/urfave/cli/v3"
)

func View(ctx context.Context, cmd *cli.Command) error {
	// 対象ファイル名を取得してくる
	mdName := ""
	if cmd != nil && cmd.Args() != nil {
		mdName = cmd.Args().First()
	}
	if mdName == "" {
		return fmt.Errorf("you need filename")
	}
	mdPath := filepath.Join(DOWNLOADPATH, mdName)

	// 対象ファイルをmarkdown -> html変換した新規ファイルを作成する
	htmlPath, err := parseMdToHTML(mdPath)
	if err != nil {
		return fmt.Errorf("an error occurred while parsing markdown to html: %w", err)
	}

	// 作成したhtmlファイルをブラウザでレンダーする
	if err := browser.OpenFile(htmlPath); err != nil {
		return fmt.Errorf("an error occurred while opening file in browser: %w", err)
	}
	return nil
}

func parseMdToHTML(mdPath string) (string, error) {
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return "", fmt.Errorf("read markdown file: %w", err)
	}

	nodes := parseBlocks(string(raw))
	htmlContent := renderHTML(nodes)

	htmlFile, err := os.CreateTemp("", "ink-*.html")
	if err != nil {
		return "", fmt.Errorf("create temporary html file: %w", err)
	}
	if _, err := htmlFile.WriteString(htmlContent); err != nil {
		htmlFile.Close()
		return "", fmt.Errorf("write temporary html file: %w", err)
	}
	if err := htmlFile.Close(); err != nil {
		return "", fmt.Errorf("close temporary html file: %w", err)
	}

	return htmlFile.Name(), nil
}
