package handler

import (
	"context"
	"fmt"
	"path"
	"github.com/econron/browser"
	"github.com/urfave/cli/v3"
)

// 一時的なもの
// const URL1 = "index.html"

func Render(ctx context.Context, cmd *cli.Command) error {
	// 対象ファイル名を取得してくる
	if cmd.Args().First() == "" {
		fmt.Println("you need filename.")
		return nil
	}
	mdName := cmd.Args().First()
	mdPath := path.Join(DOWNLOADPATH, mdName)
	// 対象ファイルをmarkdown -> html変換した新規ファイルを作成する
	htmlPath, err := parseMdToHTML(mdPath)
	if err != nil {
		fmt.Printf("an error occured while parsing Md to html. err: #%v", err)
		return nil
	}
	// 作成したhtmlファイルをブラウザでレンダーする
	if err := browser.OpenURL(htmlPath); err != nil {
		fmt.Printf("an error occured while opening file in browser. err: %#v", err)
	}
	return nil
}

func parseMdToHTML(mdPath string) (string, error) {
	fmt.Println(mdPath)
	// htmlPath := URL1

	// mdPathのファイルの中身を読み込む
	// WIP
	// 

	// return htmlPath, nil
	return mdPath, nil
}
