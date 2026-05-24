package handler

import (
	"bytes"
	"context"
	"ink/internal/config"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v3"
)

func TestParseMdToHTMLWritesCacheHTML(t *testing.T) {
	home := setupHome(t)
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "input.md")
	if err := os.WriteFile(mdPath, []byte("# Title\nhello"), 0644); err != nil {
		t.Fatalf("write markdown fixture: %v", err)
	}

	htmlPath, err := parseMdToHTML(mdPath)
	if err != nil {
		t.Fatalf("parseMdToHTML() error = %v", err)
	}

	if htmlPath == mdPath {
		t.Fatalf("parseMdToHTML() returned markdown path, want cache html path")
	}

	wantDir := filepath.Join(home, ".ink", "cache", "pages")
	if filepath.Dir(htmlPath) != wantDir {
		t.Fatalf("parseMdToHTML() path dir = %q, want %q", filepath.Dir(htmlPath), wantDir)
	}
	if filepath.Ext(htmlPath) != ".html" {
		t.Fatalf("parseMdToHTML() path = %q, want .html file", htmlPath)
	}

	body, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read generated html: %v", err)
	}

	for _, want := range []string{"<h1>Title</h1>", "<p>hello</p>"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("generated html does not contain %q: %q", want, string(body))
		}
	}
}

func TestParseMdToHTMLReusesSameCachePathForSameMarkdown(t *testing.T) {
	setupHome(t)
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "input.md")
	if err := os.WriteFile(mdPath, []byte("# First"), 0644); err != nil {
		t.Fatalf("write markdown fixture: %v", err)
	}

	firstPath, err := parseMdToHTML(mdPath)
	if err != nil {
		t.Fatalf("first parseMdToHTML() error = %v", err)
	}

	if err := os.WriteFile(mdPath, []byte("# Second"), 0644); err != nil {
		t.Fatalf("rewrite markdown fixture: %v", err)
	}
	secondPath, err := parseMdToHTML(mdPath)
	if err != nil {
		t.Fatalf("second parseMdToHTML() error = %v", err)
	}

	if firstPath != secondPath {
		t.Fatalf("parseMdToHTML() cache path changed: first %q, second %q", firstPath, secondPath)
	}

	body, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read generated html: %v", err)
	}
	if !strings.Contains(string(body), "<h1>Second</h1>") {
		t.Fatalf("generated html = %q, want rewritten markdown content", string(body))
	}
}

func TestParseMdToHTMLUsesDifferentCachePathForDifferentMarkdown(t *testing.T) {
	setupHome(t)
	dir := t.TempDir()
	firstMD := filepath.Join(dir, "first.md")
	secondMD := filepath.Join(dir, "second.md")
	if err := os.WriteFile(firstMD, []byte("# First"), 0644); err != nil {
		t.Fatalf("write first markdown fixture: %v", err)
	}
	if err := os.WriteFile(secondMD, []byte("# Second"), 0644); err != nil {
		t.Fatalf("write second markdown fixture: %v", err)
	}

	firstPath, err := parseMdToHTML(firstMD)
	if err != nil {
		t.Fatalf("first parseMdToHTML() error = %v", err)
	}
	secondPath, err := parseMdToHTML(secondMD)
	if err != nil {
		t.Fatalf("second parseMdToHTML() error = %v", err)
	}

	if firstPath == secondPath {
		t.Fatalf("parseMdToHTML() returned same cache path for different markdown files: %q", firstPath)
	}
}

func TestParseMdToHTMLMissingFileReturnsError(t *testing.T) {
	setupHome(t)

	_, err := parseMdToHTML(filepath.Join(t.TempDir(), "missing.md"))
	if err == nil {
		t.Fatal("parseMdToHTML() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "read markdown file") {
		t.Fatalf("parseMdToHTML() error = %q, want read markdown file error", err.Error())
	}
}

func TestViewWithoutFilenameReturnsError(t *testing.T) {
	err := View(context.Background(), &cli.Command{})
	if err == nil {
		t.Fatal("View() error = nil, want error")
	}
	if err.Error() != "you need filename" {
		t.Fatalf("View() error = %q, want %q", err.Error(), "you need filename")
	}
}

func TestViewUsesConfiguredLibrary(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "library")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	if err := os.WriteFile(filepath.Join(library, "note.md"), []byte("# Configured\n"), 0644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if _, err := config.Set(config.KeyLibrary, []string{library}); err != nil {
		t.Fatalf("set library: %v", err)
	}

	oldOpenFile := openFile
	t.Cleanup(func() {
		openFile = oldOpenFile
	})

	var opened string
	openFile = func(path string) error {
		opened = path
		return nil
	}

	if err := viewMarkdown("note.md"); err != nil {
		t.Fatalf("viewMarkdown() error = %v", err)
	}
	if opened == "" {
		t.Fatal("viewMarkdown() did not open generated HTML")
	}

	body, err := os.ReadFile(opened)
	if err != nil {
		t.Fatalf("read generated html: %v", err)
	}
	if !strings.Contains(string(body), "<h1>Configured</h1>") {
		t.Fatalf("generated html = %q, want configured markdown content", string(body))
	}
}

func TestViewFindsMarkdownFromMultipleLibraries(t *testing.T) {
	home := setupHome(t)
	firstLibrary := filepath.Join(home, "first")
	secondLibrary := filepath.Join(home, "second")
	for _, library := range []string{firstLibrary, secondLibrary} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
	}
	if err := os.WriteFile(filepath.Join(secondLibrary, "note.md"), []byte("# Second Library\n"), 0644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if _, err := config.Set(config.KeyLibrary, []string{firstLibrary, secondLibrary}); err != nil {
		t.Fatalf("set library: %v", err)
	}

	oldOpenFile := openFile
	t.Cleanup(func() {
		openFile = oldOpenFile
	})

	var opened string
	openFile = func(path string) error {
		opened = path
		return nil
	}

	if err := viewMarkdown("note.md"); err != nil {
		t.Fatalf("viewMarkdown() error = %v", err)
	}

	body, err := os.ReadFile(opened)
	if err != nil {
		t.Fatalf("read generated html: %v", err)
	}
	if !strings.Contains(string(body), "<h1>Second Library</h1>") {
		t.Fatalf("generated html = %q, want second library content", string(body))
	}
}

func TestViewDuplicateMarkdownReturnsError(t *testing.T) {
	home := setupHome(t)
	firstLibrary := filepath.Join(home, "first")
	secondLibrary := filepath.Join(home, "second")
	for _, library := range []string{firstLibrary, secondLibrary} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
		if err := os.WriteFile(filepath.Join(library, "note.md"), []byte("# Duplicate\n"), 0644); err != nil {
			t.Fatalf("write markdown: %v", err)
		}
	}
	if _, err := config.Set(config.KeyLibrary, []string{firstLibrary, secondLibrary}); err != nil {
		t.Fatalf("set library: %v", err)
	}

	err := viewMarkdown("note.md")
	if err == nil {
		t.Fatal("viewMarkdown() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `multiple files matched "note.md"; run one of:`) {
		t.Fatalf("viewMarkdown() error = %q, want multiple match error", err.Error())
	}
	for _, library := range []string{firstLibrary, secondLibrary} {
		want := "ink view " + filepath.Join(library, "note.md")
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("viewMarkdown() error = %q, want recommended command %q", err.Error(), want)
		}
	}
}

func TestViewAcceptsAbsoluteMarkdownPath(t *testing.T) {
	setupHome(t)
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "absolute.md")
	if err := os.WriteFile(mdPath, []byte("# Absolute\n"), 0644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	oldOpenFile := openFile
	t.Cleanup(func() {
		openFile = oldOpenFile
	})

	var opened string
	openFile = func(path string) error {
		opened = path
		return nil
	}

	if err := viewMarkdown(mdPath); err != nil {
		t.Fatalf("viewMarkdown() error = %v", err)
	}

	body, err := os.ReadFile(opened)
	if err != nil {
		t.Fatalf("read generated html: %v", err)
	}
	if !strings.Contains(string(body), "<h1>Absolute</h1>") {
		t.Fatalf("generated html = %q, want absolute path content", string(body))
	}
}

func TestViewCleansOldPreviewCache(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "library")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	if err := os.WriteFile(filepath.Join(library, "note.md"), []byte("# Current\n"), 0644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if _, err := config.Set(config.KeyLibrary, []string{library}); err != nil {
		t.Fatalf("set library: %v", err)
	}

	cacheDir := filepath.Join(home, ".ink", "cache", "pages")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	oldHTML := filepath.Join(cacheDir, "old.html")
	newHTML := filepath.Join(cacheDir, "new.html")
	oldText := filepath.Join(cacheDir, "old.txt")
	nestedDir := filepath.Join(cacheDir, "nested.html")
	for _, path := range []string{oldHTML, newHTML, oldText} {
		if err := os.WriteFile(path, []byte(path), 0644); err != nil {
			t.Fatalf("write cache file %s: %v", path, err)
		}
	}
	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatalf("mkdir cache directory: %v", err)
	}

	now := time.Now()
	oldTime := now.Add(-(previewCacheMaxAge + time.Hour))
	newTime := now.Add(-time.Hour)
	for _, path := range []string{oldHTML, oldText, nestedDir} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("set old time %s: %v", path, err)
		}
	}
	if err := os.Chtimes(newHTML, newTime, newTime); err != nil {
		t.Fatalf("set new time: %v", err)
	}

	oldOpenFile := openFile
	t.Cleanup(func() {
		openFile = oldOpenFile
	})
	openFile = func(path string) error {
		return nil
	}

	if err := viewMarkdown("note.md"); err != nil {
		t.Fatalf("viewMarkdown() error = %v", err)
	}

	if _, err := os.Stat(oldHTML); !os.IsNotExist(err) {
		t.Fatalf("old html still exists, err = %v", err)
	}
	for _, path := range []string{newHTML, oldText, nestedDir} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("cache path %s was removed or inaccessible: %v", path, err)
		}
	}
}

func TestListFilesReturnsMarkdownFilesOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.md", "a.md", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "nested.md"), 0755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}

	got, err := listFiles(dir)
	if err != nil {
		t.Fatalf("listFiles() error = %v", err)
	}
	want := []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "b.md"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %#v, want %#v", got, want)
	}
}

func TestLsUsesConfiguredLibrary(t *testing.T) {
	home := setupHome(t)
	firstLibrary := filepath.Join(home, "first")
	secondLibrary := filepath.Join(home, "second")
	for _, library := range []string{firstLibrary, secondLibrary} {
		if err := os.Mkdir(library, 0755); err != nil {
			t.Fatalf("mkdir library %s: %v", library, err)
		}
	}
	for _, name := range []string{"b.md", "a.md", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(firstLibrary, name), []byte(name), 0644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(secondLibrary, "c.md"), []byte("c"), 0644); err != nil {
		t.Fatalf("write second fixture: %v", err)
	}
	if _, err := config.Set(config.KeyLibrary, []string{firstLibrary, secondLibrary}); err != nil {
		t.Fatalf("set library: %v", err)
	}

	var out bytes.Buffer
	err := Ls(context.Background(), &cli.Command{Writer: &out})
	if err != nil {
		t.Fatalf("Ls() error = %v", err)
	}

	want := filepath.Join(firstLibrary, "a.md") + "\n" +
		filepath.Join(firstLibrary, "b.md") + "\n" +
		filepath.Join(secondLibrary, "c.md") + "\n"
	if out.String() != want {
		t.Fatalf("Ls() output = %q, want %q", out.String(), want)
	}
}

func TestListFilesMissingDirectoryReturnsError(t *testing.T) {
	_, err := listFiles(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("listFiles() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "list markdown files") {
		t.Fatalf("listFiles() error = %q, want list markdown files error", err.Error())
	}
}

func setupHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}
