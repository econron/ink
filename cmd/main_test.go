package main

import (
	"errors"
	"testing"
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
