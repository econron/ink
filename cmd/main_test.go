package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestFormatError(t *testing.T) {
	got := formatError(errors.New("you need filename"))
	want := "ink: you need filename\n"

	if got != want {
		t.Fatalf("formatError() = %q, want %q", got, want)
	}
}

func TestFormatErrorNil(t *testing.T) {
	got := formatError(nil)
	if got != "" {
		t.Fatalf("formatError(nil) = %q, want empty string", got)
	}
}

func TestConfigCommands(t *testing.T) {
	home := setupHome(t)
	downloads := filepath.Join(home, "Downloads")
	notes := filepath.Join(home, "notes")
	archive := filepath.Join(home, "archive")
	for _, library := range []string{downloads, notes, archive} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
	}

	got := runCommand(t, "config", "set", "library", downloads, notes)
	want := "library:\n  - " + downloads + "\n  - " + notes + "\n"
	if got != want {
		t.Fatalf("config set output = %q, want %q", got, want)
	}

	got = runCommand(t, "config", "add", "library", archive)
	want = "library:\n  - " + downloads + "\n  - " + notes + "\n  - " + archive + "\n"
	if got != want {
		t.Fatalf("config add output = %q, want %q", got, want)
	}

	got = runCommand(t, "config", "remove", "library", notes)
	want = "library:\n  - " + downloads + "\n  - " + archive + "\n"
	if got != want {
		t.Fatalf("config remove output = %q, want %q", got, want)
	}

	got = runCommand(t, "config", "get", "library")
	want = downloads + "\n" + archive + "\n"
	if got != want {
		t.Fatalf("config get output = %q, want %q", got, want)
	}

	got = runCommand(t, "config", "list")
	want = "library:\n  - " + downloads + "\n  - " + archive + "\n"
	if got != want {
		t.Fatalf("config list output = %q, want %q", got, want)
	}
}

func runCommand(t *testing.T, args ...string) string {
	t.Helper()

	var out bytes.Buffer
	cmd := newCommand()
	setCommandWriter(cmd, &out)
	if err := cmd.Run(context.Background(), append([]string{"ink"}, args...)); err != nil {
		t.Fatalf("run command %v: %v", args, err)
	}
	return out.String()
}

func setCommandWriter(cmd *cli.Command, out *bytes.Buffer) {
	cmd.Writer = out
	for _, child := range cmd.Commands {
		setCommandWriter(child, out)
	}
}

func setupHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}
