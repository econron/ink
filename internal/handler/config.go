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
	if len(args) < 2 {
		return fmt.Errorf("usage: ink config set <key> <value> [value ...]")
	}

	values, err := config.Set(args[0], args[1:])
	if err != nil {
		return err
	}

	return writeConfigValues(cmd, args[0], values)
}

func ConfigAdd(ctx context.Context, cmd *cli.Command) error {
	args := commandArgs(cmd)
	if len(args) != 2 {
		return fmt.Errorf("usage: ink config add <key> <value>")
	}

	values, err := config.Add(args[0], args[1])
	if err != nil {
		return err
	}

	return writeConfigValues(cmd, args[0], values)
}

func ConfigRemove(ctx context.Context, cmd *cli.Command) error {
	args := commandArgs(cmd)
	if len(args) != 2 {
		return fmt.Errorf("usage: ink config remove <key> <value>")
	}

	values, err := config.Remove(args[0], args[1])
	if err != nil {
		return err
	}

	return writeConfigValues(cmd, args[0], values)
}

func ConfigGet(ctx context.Context, cmd *cli.Command) error {
	args := commandArgs(cmd)
	if len(args) != 1 {
		return fmt.Errorf("usage: ink config get <key>")
	}

	values, err := config.Get(args[0])
	if err != nil {
		return err
	}

	return writeLines(cmd, values)
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

	return writeConfigValues(cmd, config.KeyLibrary, cfg.Library)
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

func writeLine(cmd *cli.Command, value string) error {
	_, err := fmt.Fprintln(commandWriter(cmd), value)
	return err
}

func writeLines(cmd *cli.Command, values []string) error {
	for _, value := range values {
		if err := writeLine(cmd, value); err != nil {
			return err
		}
	}
	return nil
}

func writeConfigValues(cmd *cli.Command, key string, values []string) error {
	writer := commandWriter(cmd)
	if _, err := fmt.Fprintf(writer, "%s:\n", key); err != nil {
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintf(writer, "  - %s\n", value); err != nil {
			return err
		}
	}
	return nil
}
