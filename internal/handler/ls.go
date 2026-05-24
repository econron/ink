package handler

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"os"
)

const DOWNLOADPATH = "/Users/okuyamaaron/Downloads"

func Ls(ctx context.Context, cmd *cli.Command) error {
	files, err := listFiles(DOWNLOADPATH)
	if err != nil {
		return err
	}
	for _, file := range files {
		fmt.Println(file)
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
	return files, nil
}
