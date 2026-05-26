package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"ink/internal/config"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEditorRejectsInvalidToken(t *testing.T) {
	server := httptest.NewServer(editorServer{initialPaths: []string{filepath.Join(t.TempDir(), "note.md")}, token: "secret"}.handler())
	defer server.Close()

	res, err := http.Get(server.URL + "/api/document?token=wrong")
	if err != nil {
		t.Fatalf("GET document: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("GET document status = %d, want %d", res.StatusCode, http.StatusForbidden)
	}
}

func TestEditorGetDocument(t *testing.T) {
	mdPath := writeMarkdownFixture(t, "# Title\n")
	server := httptest.NewServer(editorServer{initialPaths: []string{mdPath}, token: "secret"}.handler())
	defer server.Close()

	res, err := http.Get(documentURL(server.URL, mdPath))
	if err != nil {
		t.Fatalf("GET document: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET document status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var got editorDocumentResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode document: %v", err)
	}
	if got.Path != mdPath || got.Markdown != "# Title\n" || got.ModTimeUnixNano == "" {
		t.Fatalf("document = %#v, want fixture document", got)
	}
}

func TestEditorGetDocumentReturnsStringModTime(t *testing.T) {
	mdPath := writeMarkdownFixture(t, "# Title\n")
	document, err := readEditorDocument(mdPath)
	if err != nil {
		t.Fatalf("read editor document: %v", err)
	}

	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	if !strings.Contains(string(raw), `"modTimeUnixNano":"`) {
		t.Fatalf("document json = %s, want modTimeUnixNano as string", string(raw))
	}
}

func TestEditorRender(t *testing.T) {
	mdPath := writeMarkdownFixture(t, "# Title\n")
	server := httptest.NewServer(editorServer{initialPaths: []string{mdPath}, token: "secret"}.handler())
	defer server.Close()

	res := postJSON(t, server.URL+"/api/render?token=secret", editorRenderRequest{Markdown: "# Preview"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST render status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var got editorRenderResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode render response: %v", err)
	}
	if got.HTML != "<h1>Preview</h1>\n" {
		t.Fatalf("render html = %q, want preview fragment", got.HTML)
	}
}

func TestEditorPageIncludesSynchronizedScrollBehavior(t *testing.T) {
	html := renderEditorPage("secret", []string{"/tmp/note.md"})

	for _, want := range []string{
		`const initialPaths = ["/tmp/note.md"];`,
		`editor.addEventListener("scroll"`,
		`preview.addEventListener("scroll"`,
		`syncScroll(editor, preview)`,
		`overflow: hidden;`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("editor page does not contain %q", want)
		}
	}
}

func TestEditorSaveDocument(t *testing.T) {
	mdPath := writeMarkdownFixture(t, "# Before\n")
	if err := os.Chmod(mdPath, 0600); err != nil {
		t.Fatalf("chmod fixture: %v", err)
	}
	document, err := readEditorDocument(mdPath)
	if err != nil {
		t.Fatalf("read editor document: %v", err)
	}

	server := httptest.NewServer(editorServer{initialPaths: []string{mdPath}, token: "secret"}.handler())
	defer server.Close()

	res := postJSON(t, documentURL(server.URL, mdPath), editorSaveRequest{
		Markdown:        "# After\n",
		ModTimeUnixNano: document.ModTimeUnixNano,
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST document status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	raw, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("read saved markdown: %v", err)
	}
	if string(raw) != "# After\n" {
		t.Fatalf("saved markdown = %q, want updated content", string(raw))
	}

	info, err := os.Stat(mdPath)
	if err != nil {
		t.Fatalf("stat saved markdown: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("saved mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestEditorSaveConflict(t *testing.T) {
	mdPath := writeMarkdownFixture(t, "# Before\n")
	document, err := readEditorDocument(mdPath)
	if err != nil {
		t.Fatalf("read editor document: %v", err)
	}

	if err := os.WriteFile(mdPath, []byte("# External\n"), 0644); err != nil {
		t.Fatalf("rewrite markdown: %v", err)
	}
	newTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(mdPath, newTime, newTime); err != nil {
		t.Fatalf("set modified time: %v", err)
	}

	server := httptest.NewServer(editorServer{initialPaths: []string{mdPath}, token: "secret"}.handler())
	defer server.Close()

	res := postJSON(t, documentURL(server.URL, mdPath), editorSaveRequest{
		Markdown:        "# After\n",
		ModTimeUnixNano: document.ModTimeUnixNano,
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Fatalf("POST document status = %d, want %d", res.StatusCode, http.StatusConflict)
	}

	var got editorErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if !strings.Contains(got.Error, "changed on disk") {
		t.Fatalf("error = %q, want conflict message", got.Error)
	}
}

func TestEditorListFilesUsesConfiguredLibrary(t *testing.T) {
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
	if err := os.Mkdir(filepath.Join(firstLibrary, "nested.md"), 0755); err != nil {
		t.Fatalf("mkdir nested fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondLibrary, "c.md"), []byte("c"), 0644); err != nil {
		t.Fatalf("write second fixture: %v", err)
	}
	if _, err := config.Set(config.KeyLibrary, []string{firstLibrary, secondLibrary}); err != nil {
		t.Fatalf("set library: %v", err)
	}

	server := httptest.NewServer(editorServer{token: "secret"}.handler())
	defer server.Close()

	res, err := http.Get(server.URL + "/api/files?token=secret")
	if err != nil {
		t.Fatalf("GET files: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET files status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var got editorFilesResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode files: %v", err)
	}

	gotPaths := make([]string, 0, len(got.Files))
	for _, file := range got.Files {
		gotPaths = append(gotPaths, file.Path)
		if file.Name == "" || file.Library == "" {
			t.Fatalf("file = %#v, want name and library", file)
		}
	}
	wantPaths := []string{
		filepath.Join(firstLibrary, "a.md"),
		filepath.Join(firstLibrary, "b.md"),
		filepath.Join(secondLibrary, "c.md"),
	}
	if strings.Join(gotPaths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("file paths = %#v, want %#v", gotPaths, wantPaths)
	}
}

func TestEditorDocumentAllowsConfiguredLibraryPath(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "library")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	mdPath := filepath.Join(library, "note.md")
	if err := os.WriteFile(mdPath, []byte("# Library\n"), 0644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if _, err := config.Set(config.KeyLibrary, []string{library}); err != nil {
		t.Fatalf("set library: %v", err)
	}

	server := httptest.NewServer(editorServer{token: "secret"}.handler())
	defer server.Close()

	res, err := http.Get(documentURL(server.URL, mdPath))
	if err != nil {
		t.Fatalf("GET document: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET document status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var got editorDocumentResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode document: %v", err)
	}
	if got.Path != mdPath || got.Markdown != "# Library\n" {
		t.Fatalf("document = %#v, want library markdown", got)
	}
}

func TestEditorDocumentRejectsPathOutsideLibrary(t *testing.T) {
	home := setupHome(t)
	library := filepath.Join(home, "library")
	if err := os.Mkdir(library, 0755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	if _, err := config.Set(config.KeyLibrary, []string{library}); err != nil {
		t.Fatalf("set library: %v", err)
	}
	outsidePath := writeMarkdownFixture(t, "# Outside\n")

	server := httptest.NewServer(editorServer{token: "secret"}.handler())
	defer server.Close()

	res, err := http.Get(documentURL(server.URL, outsidePath))
	if err != nil {
		t.Fatalf("GET document: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("GET document status = %d, want %d", res.StatusCode, http.StatusForbidden)
	}
}

func TestEditorDocumentAllowsInitialPathOutsideLibrary(t *testing.T) {
	setupHome(t)
	mdPath := writeMarkdownFixture(t, "# Initial\n")

	server := httptest.NewServer(editorServer{initialPaths: []string{mdPath}, token: "secret"}.handler())
	defer server.Close()

	res, err := http.Get(documentURL(server.URL, mdPath))
	if err != nil {
		t.Fatalf("GET document: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET document status = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestEditorDocumentRequiresPathWhenMultipleInitialFiles(t *testing.T) {
	firstPath := writeMarkdownFixture(t, "# First\n")
	secondPath := filepath.Join(t.TempDir(), "second.md")
	if err := os.WriteFile(secondPath, []byte("# Second\n"), 0644); err != nil {
		t.Fatalf("write second markdown: %v", err)
	}

	server := httptest.NewServer(editorServer{initialPaths: []string{firstPath, secondPath}, token: "secret"}.handler())
	defer server.Close()

	res, err := http.Get(server.URL + "/api/document?token=secret")
	if err != nil {
		t.Fatalf("GET document: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET document status = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
}

func TestEditDuplicateMarkdownReturnsEditCommands(t *testing.T) {
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

	err := editMarkdown(context.Background(), "note.md")
	if err == nil {
		t.Fatal("editMarkdown() error = nil, want error")
	}
	if strings.Contains(err.Error(), "ink view ") {
		t.Fatalf("editMarkdown() error = %q, want edit command recommendations", err.Error())
	}
	for _, library := range []string{firstLibrary, secondLibrary} {
		want := "ink edit " + filepath.Join(library, "note.md")
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("editMarkdown() error = %q, want recommended command %q", err.Error(), want)
		}
	}
}

func writeMarkdownFixture(t *testing.T, body string) string {
	t.Helper()

	mdPath := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(mdPath, []byte(body), 0644); err != nil {
		t.Fatalf("write markdown fixture: %v", err)
	}
	return mdPath
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("encode json: %v", err)
	}
	res, err := http.Post(url, "application/json", &buf)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return res
}

func documentURL(baseURL, mdPath string) string {
	return baseURL + "/api/document?token=secret&path=" + url.QueryEscape(mdPath)
}
