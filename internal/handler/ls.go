package handler

import (
	"context"
	"fmt"
	"ink/internal/config"
	"os"
	"path/filepath"
	"sort"

	"github.com/urfave/cli/v3"
)

func Ls(ctx context.Context, cmd *cli.Command) error {
	library, err := config.Library()
	if err != nil {
		return fmt.Errorf("get library config: %w", err)
	}

	files, err := listFiles(library)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := writeLine(cmd, file); err != nil {
			return err
		}
	}
	return nil
}

func listFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files, fmt.Errorf("list markdown files: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}
