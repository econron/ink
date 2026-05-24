package handler

import (
	"context"
	"fmt"
	"ink/internal/config"
	"io"
	"os"

	"github.com/urfave/cli/v3"
)

func ConfigSet(ctx context.Context, cmd *cli.Command) error {
	args := commandArgs(cmd)
	if len(args) != 2 {
		return fmt.Errorf("usage: ink config set <key> <value>")
	}

	value, err := config.Set(args[0], args[1])
	if err != nil {
		return err
	}

	return writeConfigValue(cmd, args[0], value)
}

func ConfigGet(ctx context.Context, cmd *cli.Command) error {
	args := commandArgs(cmd)
	if len(args) != 1 {
		return fmt.Errorf("usage: ink config get <key>")
	}

	value, err := config.Get(args[0])
	if err != nil {
		return err
	}

	return writeLine(cmd, value)
}

func ConfigList(ctx context.Context, cmd *cli.Command) error {
	args := commandArgs(cmd)
	if len(args) != 0 {
		return fmt.Errorf("usage: ink config list")
	}

	cfg, err := config.List()
	if err != nil {
		return err
	}

	return writeConfigValue(cmd, config.KeyLibrary, cfg.Library)
}

func commandArgs(cmd *cli.Command) []string {
	if cmd == nil || cmd.Args() == nil {
		return nil
	}
	return cmd.Args().Slice()
}

func commandWriter(cmd *cli.Command) io.Writer {
	if cmd == nil {
		return os.Stdout
	}
	if cmd.Writer != nil {
		return cmd.Writer
	}
	for _, command := range cmd.Lineage() {
		if command.Writer != nil {
			return command.Writer
		}
	}
	if root := cmd.Root(); root != nil && root.Writer != nil {
		return root.Writer
	}
	return os.Stdout
}

func writeConfigValue(cmd *cli.Command, key, value string) error {
	_, err := fmt.Fprintf(commandWriter(cmd), "%s: %s\n", key, value)
	return err
}

func writeLine(cmd *cli.Command, value string) error {
	_, err := fmt.Fprintln(commandWriter(cmd), value)
	return err
}
