package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibraryDefaultsToDownloadsWhenConfigMissing(t *testing.T) {
	home := setupHome(t)

	got, err := Library()
	if err != nil {
		t.Fatalf("Library() error = %v", err)
	}

	want := filepath.Join(home, "Downloads")
	if got != want {
		t.Fatalf("Library() = %q, want %q", got, want)
	}
}

func TestSetGetLibraryExpandsHomeAndSavesConfig(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "Downloads")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}

	got, err := Set(KeyLibrary, "~/Downloads")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got != library {
		t.Fatalf("Set() = %q, want %q", got, library)
	}

	got, err = Get(KeyLibrary)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != library {
		t.Fatalf("Get() = %q, want %q", got, library)
	}

	configPath := filepath.Join(home, ".ink", "config.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(raw), `"library": "`+library+`"`) {
		t.Fatalf("config file = %q, want saved library", string(raw))
	}
}

func TestListReturnsSavedLibrary(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "notes")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	if _, err := Set(KeyLibrary, library); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	cfg, err := List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if cfg.Library != library {
		t.Fatalf("List().Library = %q, want %q", cfg.Library, library)
	}
}

func TestSetUnknownKeyReturnsError(t *testing.T) {
	setupHome(t)

	_, err := Set("unknown", "value")
	if err == nil {
		t.Fatal("Set() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `unknown config key "unknown"`) {
		t.Fatalf("Set() error = %q, want unknown key error", err.Error())
	}
}

func TestSetLibraryMissingDirectoryReturnsError(t *testing.T) {
	home := setupHome(t)

	_, err := Set(KeyLibrary, filepath.Join(home, "missing"))
	if err == nil {
		t.Fatal("Set() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "library directory does not exist") {
		t.Fatalf("Set() error = %q, want missing directory error", err.Error())
	}
}

func TestSetLibraryFileReturnsError(t *testing.T) {
	home := setupHome(t)
	path := filepath.Join(home, "notes.md")
	if err := os.WriteFile(path, []byte("# notes"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := Set(KeyLibrary, path)
	if err == nil {
		t.Fatal("Set() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "library must be a directory") {
		t.Fatalf("Set() error = %q, want directory error", err.Error())
	}
}

func TestBrokenConfigReturnsError(t *testing.T) {
	home := setupHome(t)
	configDir := filepath.Join(home, ".ink")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Library()
	if err == nil {
		t.Fatal("Library() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("Library() error = %q, want parse config error", err.Error())
	}
}

func setupHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}
