package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"ink/internal/config"
	"ink/internal/markdown"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/econron/browser"
	"github.com/urfave/cli/v3"
)

var openFile = browser.OpenFile

const previewCacheMaxAge = 7 * 24 * time.Hour

func View(ctx context.Context, cmd *cli.Command) error {
	// 対象ファイル名を取得してくる
	mdName := ""
	if cmd != nil && cmd.Args() != nil {
		mdName = cmd.Args().First()
	}
	return viewMarkdown(mdName)
}

func viewMarkdown(mdName string) error {
	if mdName == "" {
		return fmt.Errorf("you need filename")
	}

	if err := cleanupPreviewCache(time.Now()); err != nil {
		return fmt.Errorf("clean preview cache: %w", err)
	}

	mdPath, err := resolveMarkdownPath(mdName)
	if err != nil {
		return err
	}

	// 対象ファイルをmarkdown -> html変換した新規ファイルを作成する
	htmlPath, err := parseMdToHTML(mdPath)
	if err != nil {
		return fmt.Errorf("an error occurred while parsing markdown to html: %w", err)
	}

	// 作成したhtmlファイルをブラウザでレンダーする
	if err := openFile(htmlPath); err != nil {
		return fmt.Errorf("an error occurred while opening file in browser: %w", err)
	}
	return nil
}

func resolveMarkdownPath(mdName string) (string, error) {
	return resolveMarkdownPathForCommand(mdName, "view")
}

func resolveMarkdownPathForCommand(mdName, commandName string) (string, error) {
	if filepath.IsAbs(mdName) {
		return filepath.Clean(mdName), nil
	}

	libraries, err := config.Library()
	if err != nil {
		return "", fmt.Errorf("get library config: %w", err)
	}

	matches := make([]string, 0)
	for _, library := range libraries {
		candidate := filepath.Join(library, mdName)
		info, err := os.Stat(candidate)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("find markdown file: %w", err)
		}
		if info.IsDir() {
			continue
		}
		matches = append(matches, candidate)
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("markdown file not found: %s", mdName)
	case 1:
		return matches[0], nil
	default:
		return "", multipleMarkdownMatchesError(commandName, mdName, matches)
	}
}

func multipleMarkdownMatchesError(commandName, mdName string, matches []string) error {
	var b strings.Builder
	b.WriteString("multiple files matched ")
	b.WriteString(strconv.Quote(mdName))
	b.WriteString("; run one of:")
	for _, match := range matches {
		b.WriteString("\nink ")
		b.WriteString(commandName)
		b.WriteString(" ")
		b.WriteString(match)
	}
	return errors.New(b.String())
}

func parseMdToHTML(mdPath string) (string, error) {
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return "", fmt.Errorf("read markdown file: %w", err)
	}

	htmlContent := markdown.ToHTML(string(raw))

	htmlPath, err := previewHTMLPath(mdPath)
	if err != nil {
		return "", err
	}
	if err := writeFileAtomically(htmlPath, htmlContent); err != nil {
		return "", err
	}

	return htmlPath, nil
}

func previewHTMLPath(mdPath string) (string, error) {
	absolutePath, err := filepath.Abs(mdPath)
	if err != nil {
		return "", fmt.Errorf("resolve markdown path: %w", err)
	}

	cacheDir, err := config.CachePagesDir()
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(filepath.Clean(absolutePath)))
	return filepath.Join(cacheDir, hex.EncodeToString(hash[:])+".html"), nil
}

func writeFileAtomically(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create preview cache directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary html file: %w", err)
	}
	tmpPath := tmpFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temporary html file: %w", err)
	}
	if err := tmpFile.Chmod(0644); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temporary html file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary html file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("move preview html into cache: %w", err)
	}
	removeTemp = false
	return nil
}

func cleanupPreviewCache(now time.Time) error {
	cacheDir, err := config.CachePagesDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(cacheDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read preview cache directory: %w", err)
	}

	cutoff := now.Add(-previewCacheMaxAge)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".html" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read preview cache file info: %w", err)
		}
		if !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(cacheDir, entry.Name())); err != nil {
			return fmt.Errorf("remove stale preview cache: %w", err)
		}
	}
	return nil
}
