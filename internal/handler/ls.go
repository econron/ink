package handler

import (
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"io/fs"
	"os"
	"path"
)

const DOWNLOADPATH = "/Users/hoge/Downloads"

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

	root := os.DirFS(dir)
	mdFiles, err := fs.Glob(root, "*.md")
	if err != nil {
		return files, fmt.Errorf("An error occured while listing files: %#v", err)
	}

	for _, file := range mdFiles {
		files = append(files, path.Join(dir, file))
	}
	return files, nil
}
