package main

import (
	"context"
	"fmt"
	"ink/internal/handler"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "ink",
				Usage: "command basic",
				Commands: []*cli.Command{
					{
						Name:  "ls",
						Usage: "lists up downlods folder.",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							err := handler.Ls(ctx, cmd)
							if err != nil {
								printError(err)
							}
							return nil
						},
					},
					{
						Name:  "view",
						Usage: "view the markdown on browser",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							err := handler.View(ctx, cmd)
							if err != nil {
								printError(err)
							}
							return nil
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func printError(err error) {
	if err == nil {
		return
	}
	fmt.Fprint(os.Stderr, formatError(err))
}

func formatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("ink: %v\n", err)
}
