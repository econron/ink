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
	cmd := newCommand()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:  "ink",
		Usage: "markdown previewer",
		Commands: []*cli.Command{
			{
				Name:   "ls",
				Usage:  "list markdown files in library.",
				Action: commandAction(handler.Ls),
			},
			{
				Name:   "view",
				Usage:  "view the markdown on browser",
				Action: commandAction(handler.View),
			},
			{
				Name:      "edit",
				Usage:     "edit markdown files in browser",
				ArgsUsage: "[filename.md ...]",
				Action:    commandAction(handler.Edit),
			},
			{
				Name:  "config",
				Usage: "manage ink config",
				Commands: []*cli.Command{
					{
						Name:      "set",
						Usage:     "set config value",
						ArgsUsage: "<key> <value>...",
						Action:    commandAction(handler.ConfigSet),
					},
					{
						Name:      "add",
						Usage:     "add config value",
						ArgsUsage: "<key> <value>",
						Action:    commandAction(handler.ConfigAdd),
					},
					{
						Name:      "remove",
						Usage:     "remove config value",
						ArgsUsage: "<key> <value>",
						Action:    commandAction(handler.ConfigRemove),
					},
					{
						Name:      "get",
						Usage:     "get config value",
						ArgsUsage: "<key>",
						Action:    commandAction(handler.ConfigGet),
					},
					{
						Name:   "list",
						Usage:  "list config values",
						Action: commandAction(handler.ConfigList),
					},
				},
			},
		},
	}
}

func commandAction(action func(context.Context, *cli.Command) error) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		if err := action(ctx, cmd); err != nil {
			printError(err)
		}
		return nil
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
