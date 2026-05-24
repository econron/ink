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
								fmt.Printf("err: %#v", err)
							}
							return nil
						},
					},
					{
						Name:  "render",
						Usage: "render the markdown on browser",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							err := handler.Render(ctx, cmd)
							if err != nil {
								fmt.Printf("err: %#v", err)
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
