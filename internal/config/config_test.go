package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLibraryDefaultsToDownloadsWhenConfigMissing(t *testing.T) {
	home := setupHome(t)

	got, err := Library()
	if err != nil {
		t.Fatalf("Library() error = %v", err)
	}

	want := []string{filepath.Join(home, "Downloads")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Library() = %#v, want %#v", got, want)
	}
}

func TestSetGetLibraryExpandsHomeAndSavesArrayConfig(t *testing.T) {
	home := setupHome(t)
	downloads := filepath.Join(home, "Downloads")
	notes := filepath.Join(home, "notes")
	for _, library := range []string{downloads, notes} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
	}

	got, err := Set(KeyLibrary, []string{"~/Downloads", notes, downloads})
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	want := []string{downloads, notes}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Set() = %#v, want %#v", got, want)
	}

	got, err = Get(KeyLibrary)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Get() = %#v, want %#v", got, want)
	}

	configPath := filepath.Join(home, ".ink", "config.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, wantText := range []string{`"library": [`, `"` + downloads + `"`, `"` + notes + `"`} {
		if !strings.Contains(string(raw), wantText) {
			t.Fatalf("config file = %q, want %q", string(raw), wantText)
		}
	}
}

func TestLoadLegacyStringLibrary(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "legacy")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	writeConfig(t, `{"library":"`+library+`"}`)

	got, err := Library()
	if err != nil {
		t.Fatalf("Library() error = %v", err)
	}
	want := []string{library}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Library() = %#v, want %#v", got, want)
	}
}

func TestListReturnsSavedLibraries(t *testing.T) {
	home := setupHome(t)
	first := filepath.Join(home, "first")
	second := filepath.Join(home, "second")
	for _, library := range []string{first, second} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
	}
	if _, err := Set(KeyLibrary, []string{first, second}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	cfg, err := List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []string{first, second}
	if !reflect.DeepEqual(cfg.Library, want) {
		t.Fatalf("List().Library = %#v, want %#v", cfg.Library, want)
	}
}

func TestAddLibrary(t *testing.T) {
	home := setupHome(t)
	first := filepath.Join(home, "first")
	second := filepath.Join(home, "second")
	for _, library := range []string{first, second} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
	}
	if _, err := Set(KeyLibrary, []string{first}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := Add(KeyLibrary, second)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	want := []string{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Add() = %#v, want %#v", got, want)
	}

	got, err = Add(KeyLibrary, second)
	if err != nil {
		t.Fatalf("Add() duplicate error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Add() duplicate = %#v, want %#v", got, want)
	}
}

func TestRemoveLibrary(t *testing.T) {
	home := setupHome(t)
	first := filepath.Join(home, "first")
	second := filepath.Join(home, "second")
	for _, library := range []string{first, second} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
	}
	if _, err := Set(KeyLibrary, []string{first, second}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := Remove(KeyLibrary, first)
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	want := []string{second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Remove() = %#v, want %#v", got, want)
	}
}

func TestRemoveUnknownLibraryReturnsError(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "library")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	if _, err := Set(KeyLibrary, []string{library}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	_, err := Remove(KeyLibrary, filepath.Join(home, "missing"))
	if err == nil {
		t.Fatal("Remove() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "library is not configured") {
		t.Fatalf("Remove() error = %q, want not configured error", err.Error())
	}
}

func TestRemoveLastLibraryReturnsError(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "library")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	if _, err := Set(KeyLibrary, []string{library}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	_, err := Remove(KeyLibrary, library)
	if err == nil {
		t.Fatal("Remove() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "library must contain at least one path") {
		t.Fatalf("Remove() error = %q, want last library error", err.Error())
	}
}

func TestSetUnknownKeyReturnsError(t *testing.T) {
	setupHome(t)

	_, err := Set("unknown", []string{"value"})
	if err == nil {
		t.Fatal("Set() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `unknown config key "unknown"`) {
		t.Fatalf("Set() error = %q, want unknown key error", err.Error())
	}
}

func TestSetLibraryMissingDirectoryReturnsError(t *testing.T) {
	home := setupHome(t)

	_, err := Set(KeyLibrary, []string{filepath.Join(home, "missing")})
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

	_, err := Set(KeyLibrary, []string{path})
	if err == nil {
		t.Fatal("Set() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "library must be a directory") {
		t.Fatalf("Set() error = %q, want directory error", err.Error())
	}
}

func TestBrokenConfigReturnsError(t *testing.T) {
	setupHome(t)
	writeConfig(t, "{")

	_, err := Library()
	if err == nil {
		t.Fatal("Library() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("Library() error = %q, want parse config error", err.Error())
	}
}

func writeConfig(t *testing.T, body string) {
	t.Helper()

	configPath, err := Path()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(body), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func setupHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}
