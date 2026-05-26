package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"ink/internal/config"
	"ink/internal/markdown"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/econron/browser"
	"github.com/urfave/cli/v3"
)

var openURL = browser.OpenURL

type editorServer struct {
	initialPaths []string
	token        string
}

type editorFilesResponse struct {
	Files []editorFile `json:"files"`
}

type editorFile struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Library string `json:"library"`
}

type editorDocumentResponse struct {
	Path            string `json:"path"`
	Markdown        string `json:"markdown"`
	ModTimeUnixNano string `json:"modTimeUnixNano"`
}

type editorRenderRequest struct {
	Markdown string `json:"markdown"`
}

type editorRenderResponse struct {
	HTML string `json:"html"`
}

type editorSaveRequest struct {
	Markdown        string `json:"markdown"`
	ModTimeUnixNano string `json:"modTimeUnixNano"`
}

type editorErrorResponse struct {
	Error string `json:"error"`
}

func Edit(ctx context.Context, cmd *cli.Command) error {
	return editMarkdowns(ctx, commandArgs(cmd))
}

func editMarkdown(ctx context.Context, mdName string) error {
	if mdName == "" {
		return fmt.Errorf("you need filename")
	}
	return editMarkdowns(ctx, []string{mdName})
}

func editMarkdowns(ctx context.Context, mdNames []string) error {
	initialPaths, err := resolveEditorInitialPaths(mdNames)
	if err != nil {
		return err
	}

	token, err := newEditorToken()
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start editor server: %w", err)
	}

	editor := editorServer{initialPaths: initialPaths, token: token}
	server := &http.Server{
		Handler:           editor.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	url := "http://" + listener.Addr().String() + "/?token=" + token
	if err := openURL(url); err != nil {
		listener.Close()
		return fmt.Errorf("open editor in browser: %w", err)
	}

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve editor: %w", err)
	}
	return nil
}

func resolveEditorInitialPaths(mdNames []string) ([]string, error) {
	paths := make([]string, 0, len(mdNames))
	seen := make(map[string]bool, len(mdNames))
	for _, mdName := range mdNames {
		if mdName == "" {
			continue
		}
		mdPath, err := resolveMarkdownPathForCommand(mdName, "edit")
		if err != nil {
			return nil, err
		}
		mdPath, err = cleanEditorPath(mdPath)
		if err != nil {
			return nil, err
		}
		if _, err := readEditorDocument(mdPath); err != nil {
			return nil, err
		}
		if seen[mdPath] {
			continue
		}
		seen[mdPath] = true
		paths = append(paths, mdPath)
	}
	return paths, nil
}

func newEditorToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("create editor token: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func (s editorServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleEditorPage)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/document", s.handleDocument)
	mux.HandleFunc("/api/render", s.handleRender)
	return mux
}

func (s editorServer) handleEditorPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !s.authorize(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(renderEditorPage(s.token, s.initialPaths))); err != nil {
		return
	}
}

func (s editorServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	files, err := listEditorFiles()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, editorFilesResponse{Files: files})
}

func (s editorServer) handleDocument(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetDocument(w, r)
	case http.MethodPost:
		s.handleSaveDocument(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s editorServer) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	mdPath, status, err := s.documentPath(r)
	if err != nil {
		writeJSONError(w, status, err.Error())
		return
	}

	document, err := readEditorDocument(mdPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, document)
}

func (s editorServer) handleSaveDocument(w http.ResponseWriter, r *http.Request) {
	mdPath, status, err := s.documentPath(r)
	if err != nil {
		writeJSONError(w, status, err.Error())
		return
	}

	var req editorSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	document, status, err := saveEditorDocument(mdPath, req.Markdown, req.ModTimeUnixNano)
	if err != nil {
		writeJSONError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, document)
}

func (s editorServer) handleRender(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req editorRenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	writeJSON(w, http.StatusOK, editorRenderResponse{HTML: markdown.ToHTMLFragment(req.Markdown)})
}

func (s editorServer) authorize(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Query().Get("token") == s.token {
		return true
	}
	writeJSONError(w, http.StatusForbidden, "invalid token")
	return false
}

func (s editorServer) documentPath(r *http.Request) (string, int, error) {
	mdPath := r.URL.Query().Get("path")
	if mdPath == "" {
		if len(s.initialPaths) == 1 {
			return s.initialPaths[0], http.StatusOK, nil
		}
		return "", http.StatusBadRequest, fmt.Errorf("missing path")
	}

	mdPath, err := cleanEditorPath(mdPath)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	allowed, err := s.canEditPath(mdPath)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}
	if !allowed {
		return "", http.StatusForbidden, fmt.Errorf("path is not editable in this session")
	}
	return mdPath, http.StatusOK, nil
}

func (s editorServer) canEditPath(path string) (bool, error) {
	for _, initialPath := range s.initialPaths {
		if path == initialPath {
			return true, nil
		}
	}
	return isLibraryMarkdownPath(path)
}

func cleanEditorPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute")
	}
	return filepath.Clean(path), nil
}

func isLibraryMarkdownPath(path string) (bool, error) {
	if filepath.Ext(path) != ".md" {
		return false, nil
	}

	libraries, err := config.Library()
	if err != nil {
		return false, fmt.Errorf("get library config: %w", err)
	}
	for _, library := range libraries {
		if filepath.Dir(path) == filepath.Clean(library) {
			return true, nil
		}
	}
	return false, nil
}

func listEditorFiles() ([]editorFile, error) {
	libraries, err := config.Library()
	if err != nil {
		return nil, fmt.Errorf("get library config: %w", err)
	}

	files := make([]editorFile, 0)
	for _, library := range libraries {
		libraryFiles, err := listFiles(library)
		if err != nil {
			return nil, err
		}
		for _, file := range libraryFiles {
			files = append(files, editorFile{
				Path:    filepath.Clean(file),
				Name:    filepath.Base(file),
				Library: filepath.Clean(library),
			})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Name == files[j].Name {
			return files[i].Path < files[j].Path
		}
		return files[i].Name < files[j].Name
	})
	return files, nil
}

func readEditorDocument(path string) (editorDocumentResponse, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return editorDocumentResponse{}, fmt.Errorf("read markdown file: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return editorDocumentResponse{}, fmt.Errorf("stat markdown file: %w", err)
	}
	return editorDocumentResponse{
		Path:            path,
		Markdown:        string(raw),
		ModTimeUnixNano: modTimeVersion(info),
	}, nil
}

func saveEditorDocument(path, content string, expectedModTime string) (editorDocumentResponse, int, error) {
	if expectedModTime == "" {
		return editorDocumentResponse{}, http.StatusBadRequest, fmt.Errorf("missing modTimeUnixNano")
	}

	info, err := os.Stat(path)
	if err != nil {
		return editorDocumentResponse{}, http.StatusInternalServerError, fmt.Errorf("stat markdown file: %w", err)
	}
	if modTimeVersion(info) != expectedModTime {
		return editorDocumentResponse{}, http.StatusConflict, fmt.Errorf("markdown file changed on disk; reload before saving")
	}
	if err := writeFileAtomicallyWithMode(path, content, info.Mode()); err != nil {
		return editorDocumentResponse{}, http.StatusInternalServerError, err
	}

	document, err := readEditorDocument(path)
	if err != nil {
		return editorDocumentResponse{}, http.StatusInternalServerError, err
	}
	return document, http.StatusOK, nil
}

func modTimeVersion(info os.FileInfo) string {
	return strconv.FormatInt(info.ModTime().UnixNano(), 10)
}

func writeFileAtomicallyWithMode(path, content string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create markdown directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary markdown file: %w", err)
	}
	tmpPath := tmpFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temporary markdown file: %w", err)
	}
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temporary markdown file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary markdown file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("move markdown file into place: %w", err)
	}
	removeTemp = false
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, editorErrorResponse{Error: message})
}

func renderEditorPage(token string, initialPaths []string) string {
	escapedToken, _ := json.Marshal(token)
	escapedInitialPaths, _ := json.Marshal(initialPaths)
	html := strings.ReplaceAll(editorPageHTML, "__INK_TOKEN__", string(escapedToken))
	return strings.ReplaceAll(html, "__INK_INITIAL_PATHS__", string(escapedInitialPaths))
}

const editorPageHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ink editor</title>
<style>
:root {
  color-scheme: light;
  --ink-bg: #f6f8fa;
  --ink-panel: #ffffff;
  --ink-text: #24292f;
  --ink-muted: #667085;
  --ink-border: #d8dee4;
  --ink-soft: #eef2f6;
  --ink-accent: #1f6feb;
  --ink-danger: #cf222e;
}

* {
  box-sizing: border-box;
}

html {
  height: 100%;
}

body {
  height: 100%;
  margin: 0;
  overflow: hidden;
  background: var(--ink-bg);
  color: var(--ink-text);
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

button,
input,
textarea {
  font: inherit;
}

button {
  cursor: pointer;
}

.ink-editor {
  display: grid;
  grid-template-rows: auto 1fr;
  height: 100vh;
  min-height: 0;
}

.ink-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 10px 14px;
  border-bottom: 1px solid var(--ink-border);
  background: var(--ink-panel);
}

.ink-save {
  min-width: 72px;
  padding: 7px 12px;
  border: 1px solid var(--ink-accent);
  border-radius: 6px;
  background: var(--ink-accent);
  color: #ffffff;
  font-weight: 600;
}

.ink-save:disabled {
  border-color: var(--ink-border);
  background: var(--ink-soft);
  color: var(--ink-muted);
  cursor: default;
}

.ink-path {
  overflow: hidden;
  flex: 1;
  min-width: 0;
  color: var(--ink-muted);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ink-status {
  min-width: 120px;
  color: var(--ink-muted);
  font-size: 13px;
  text-align: right;
}

.ink-shell {
  display: grid;
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
  min-height: 0;
  overflow: hidden;
}

.ink-sidebar {
  display: grid;
  grid-template-rows: auto 1fr;
  min-height: 0;
  border-right: 1px solid var(--ink-border);
  background: var(--ink-panel);
}

.ink-filter {
  width: calc(100% - 24px);
  margin: 12px;
  padding: 8px 10px;
  border: 1px solid var(--ink-border);
  border-radius: 6px;
  outline: 0;
}

.ink-files {
  overflow: auto;
  min-height: 0;
  padding: 0 8px 12px;
}

.ink-file {
  display: grid;
  gap: 2px;
  width: 100%;
  min-height: 46px;
  padding: 7px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--ink-text);
  text-align: left;
}

.ink-file:hover,
.ink-file.is-open {
  background: var(--ink-soft);
}

.ink-file.is-active {
  background: #dbeafe;
}

.ink-file-name,
.ink-file-library {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ink-file-name {
  font-size: 13px;
  font-weight: 600;
}

.ink-file-library {
  color: var(--ink-muted);
  font-size: 11px;
}

.ink-main {
  display: grid;
  grid-template-rows: auto 1fr;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.ink-tabs {
  display: flex;
  overflow-x: auto;
  min-height: 42px;
  border-bottom: 1px solid var(--ink-border);
  background: var(--ink-panel);
}

.ink-tab {
  display: flex;
  align-items: center;
  max-width: 240px;
  min-width: 120px;
  border-right: 1px solid var(--ink-border);
  background: transparent;
}

.ink-tab.is-active {
  background: #f8fafc;
}

.ink-tab-main {
  overflow: hidden;
  flex: 1;
  min-width: 0;
  padding: 11px 10px;
  border: 0;
  background: transparent;
  color: var(--ink-text);
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ink-tab-close {
  width: 30px;
  height: 30px;
  margin-right: 4px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--ink-muted);
}

.ink-tab-close:hover {
  background: var(--ink-soft);
  color: var(--ink-danger);
}

.ink-workspace {
  display: grid;
  grid-template-columns: minmax(320px, 1fr) minmax(320px, 1fr);
  min-height: 0;
  overflow: hidden;
}

textarea {
  width: 100%;
  height: 100%;
  min-height: 0;
  padding: 20px;
  border: 0;
  border-right: 1px solid var(--ink-border);
  outline: 0;
  resize: none;
  color: #111827;
  background: #ffffff;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace;
  font-size: 14px;
  line-height: 1.65;
  overflow: auto;
}

textarea:disabled {
  background: #f8fafc;
}

.ink-preview {
  overflow: auto;
  min-width: 0;
  padding: 28px 34px;
  background: #ffffff;
}

.ink-preview h1 {
  padding-bottom: 0.35em;
  border-bottom: 1px solid var(--ink-border);
}

.ink-preview pre {
  overflow-x: auto;
  padding: 1em;
  border-radius: 8px;
  background: #0f172a;
  color: #e5e7eb;
}

.ink-preview code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace;
}

.ink-preview table {
  width: 100%;
  border-collapse: collapse;
}

.ink-preview th,
.ink-preview td {
  padding: 0.55em 0.75em;
  border: 1px solid var(--ink-border);
}

@media (max-width: 900px) {
  .ink-shell {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(160px, 28vh) 1fr;
  }

  .ink-sidebar {
    border-right: 0;
    border-bottom: 1px solid var(--ink-border);
  }

  .ink-workspace {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(320px, 45vh) 1fr;
  }

  textarea {
    border-right: 0;
    border-bottom: 1px solid var(--ink-border);
  }
}
</style>
</head>
<body>
<div class="ink-editor">
  <header class="ink-toolbar">
    <button id="save" class="ink-save" type="button" disabled>Save</button>
    <div id="path" class="ink-path"></div>
    <div id="status" class="ink-status"></div>
  </header>
  <main class="ink-shell">
    <aside class="ink-sidebar">
      <input id="file-filter" class="ink-filter" type="search" placeholder="Filter files">
      <div id="files" class="ink-files"></div>
    </aside>
    <section class="ink-main">
      <div id="tabs" class="ink-tabs"></div>
      <div class="ink-workspace">
        <textarea id="markdown" spellcheck="false" disabled></textarea>
        <section id="preview" class="ink-preview"></section>
      </div>
    </section>
  </main>
</div>
<script type="module">
const token = __INK_TOKEN__;
const initialPaths = __INK_INITIAL_PATHS__;
const editor = document.getElementById("markdown");
const preview = document.getElementById("preview");
const statusEl = document.getElementById("status");
const pathEl = document.getElementById("path");
const saveButton = document.getElementById("save");
const filesEl = document.getElementById("files");
const tabsEl = document.getElementById("tabs");
const filterEl = document.getElementById("file-filter");
let fileCatalog = [];
let activePath = "";
let renderTimer = 0;
let mermaid = null;
let syncingScroll = false;

const openDocs = new Map();
const mermaidReady = initializeMermaid();

async function initializeMermaid() {
  try {
    const module = await import("https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs");
    mermaid = module.default;
    mermaid.initialize({ startOnLoad: false });
  } catch (_err) {
    mermaid = null;
  }
}

function endpoint(path, params = {}) {
  const url = new URL(path, window.location.origin);
  url.searchParams.set("token", token);
  for (const [key, value] of Object.entries(params)) {
    url.searchParams.set(key, value);
  }
  return url.toString();
}

function setStatus(message) {
  statusEl.textContent = message;
}

function fileLabel(path) {
  return path.split(/[\\/]/).pop() || path;
}

function activeDocument() {
  if (!activePath) return null;
  return openDocs.get(activePath) || null;
}

function commitActiveDraft() {
  const doc = activeDocument();
  if (!doc) return;
  doc.markdown = editor.value;
  doc.dirty = doc.markdown !== doc.savedMarkdown;
  doc.editorScrollTop = editor.scrollTop;
  doc.previewScrollTop = preview.scrollTop;
  doc.selectionStart = editor.selectionStart;
  doc.selectionEnd = editor.selectionEnd;
}

function scrollRatio(element) {
  const max = element.scrollHeight - element.clientHeight;
  if (max <= 0) return 0;
  return element.scrollTop / max;
}

function setScrollRatio(element, ratio) {
  const max = element.scrollHeight - element.clientHeight;
  element.scrollTop = max <= 0 ? 0 : max * ratio;
}

function syncScroll(source, target) {
  if (syncingScroll) return;
  syncingScroll = true;
  setScrollRatio(target, scrollRatio(source));
  window.requestAnimationFrame(() => {
    syncingScroll = false;
  });
}

function restoreDocumentViewport(doc) {
  window.requestAnimationFrame(() => {
    editor.scrollTop = doc.editorScrollTop || 0;
    preview.scrollTop = doc.previewScrollTop || 0;
    const start = doc.selectionStart || 0;
    const end = doc.selectionEnd || start;
    try {
      editor.setSelectionRange(start, end);
    } catch (_err) {
    }
  });
}

function renderFileList() {
  const query = filterEl.value.trim().toLowerCase();
  filesEl.innerHTML = "";
  for (const file of fileCatalog) {
    const searchable = (file.name + " " + file.path).toLowerCase();
    if (query && !searchable.includes(query)) continue;

    const button = document.createElement("button");
    button.type = "button";
    button.className = "ink-file";
    if (openDocs.has(file.path)) button.classList.add("is-open");
    if (activePath === file.path) button.classList.add("is-active");
    button.title = file.path;
    button.addEventListener("click", () => {
      openDocument(file.path, true).catch((err) => setStatus(err.message));
    });

    const name = document.createElement("span");
    name.className = "ink-file-name";
    name.textContent = file.name;
    const library = document.createElement("span");
    library.className = "ink-file-library";
    library.textContent = file.library;
    button.append(name, library);
    filesEl.append(button);
  }
}

function renderTabs() {
  tabsEl.innerHTML = "";
  for (const doc of openDocs.values()) {
    const tab = document.createElement("div");
    tab.className = "ink-tab";
    if (doc.path === activePath) tab.classList.add("is-active");

    const main = document.createElement("button");
    main.type = "button";
    main.className = "ink-tab-main";
    main.title = doc.path;
    main.textContent = fileLabel(doc.path) + (doc.dirty ? " *" : "");
    main.addEventListener("click", () => setActiveDocument(doc.path));

    const close = document.createElement("button");
    close.type = "button";
    close.className = "ink-tab-close";
    close.setAttribute("aria-label", "Close " + fileLabel(doc.path));
    close.innerHTML = "&times;";
    close.addEventListener("click", () => closeDocument(doc.path));

    tab.append(main, close);
    tabsEl.append(tab);
  }
}

function renderEmpty() {
  activePath = "";
  editor.value = "";
  editor.disabled = true;
  saveButton.disabled = true;
  pathEl.textContent = "";
  preview.innerHTML = "";
  editor.scrollTop = 0;
  preview.scrollTop = 0;
  renderTabs();
  renderFileList();
}

function setActiveDocument(path) {
  commitActiveDraft();
  const doc = openDocs.get(path);
  if (!doc) return;
  activePath = path;
  editor.disabled = false;
  saveButton.disabled = false;
  editor.value = doc.markdown;
  pathEl.textContent = doc.path;
  renderTabs();
  renderFileList();
  render()
    .then(() => restoreDocumentViewport(doc))
    .catch((err) => setStatus(err.message));
  setStatus(doc.dirty ? "Editing" : "Loaded");
}

function closeDocument(path) {
  commitActiveDraft();
  const doc = openDocs.get(path);
  if (!doc) return;
  if (doc.dirty && !window.confirm("Discard unsaved changes?")) return;

  const paths = Array.from(openDocs.keys());
  const index = paths.indexOf(path);
  openDocs.delete(path);
  if (activePath !== path) {
    renderTabs();
    renderFileList();
    return;
  }

  const nextPath = paths[index + 1] || paths[index - 1] || "";
  if (nextPath && openDocs.has(nextPath)) {
    setActiveDocument(nextPath);
  } else {
    renderEmpty();
    setStatus("Ready");
  }
}

async function loadFiles() {
  const res = await fetch(endpoint("/api/files"));
  const body = await res.json();
  if (!res.ok) throw new Error(body.error || "failed to load files");
  fileCatalog = body.files || [];
  renderFileList();
}

async function openDocument(path, activate) {
  if (openDocs.has(path)) {
    if (activate) setActiveDocument(path);
    return;
  }

  const res = await fetch(endpoint("/api/document", { path }));
  const doc = await res.json();
  if (!res.ok) throw new Error(doc.error || "failed to load document");
  openDocs.set(doc.path, {
    path: doc.path,
    markdown: doc.markdown,
    savedMarkdown: doc.markdown,
    modTimeUnixNano: doc.modTimeUnixNano,
    dirty: false,
    editorScrollTop: 0,
    previewScrollTop: 0,
    selectionStart: 0,
    selectionEnd: 0
  });
  renderTabs();
  renderFileList();
  if (activate || !activePath) {
    setActiveDocument(doc.path);
  }
}

async function render() {
  const doc = activeDocument();
  if (!doc) {
    preview.innerHTML = "";
    return;
  }
  const markdown = editor.value;
  const res = await fetch(endpoint("/api/render"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ markdown })
  });
  const body = await res.json();
  if (!res.ok) throw new Error(body.error || "failed to render document");
  preview.innerHTML = body.html;
  await mermaidReady;
  if (mermaid) {
    await mermaid.run({ nodes: preview.querySelectorAll(".mermaid") });
  }
  syncScroll(editor, preview);
}

async function save() {
  commitActiveDraft();
  const doc = activeDocument();
  if (!doc) {
    setStatus("No file open");
    return;
  }

  saveButton.disabled = true;
  setStatus("Saving...");
  try {
    const res = await fetch(endpoint("/api/document", { path: doc.path }), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        markdown: doc.markdown,
        modTimeUnixNano: doc.modTimeUnixNano
      })
    });
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || "failed to save document");
    doc.markdown = body.markdown;
    doc.savedMarkdown = body.markdown;
    doc.modTimeUnixNano = body.modTimeUnixNano;
    doc.dirty = false;
    editor.value = doc.markdown;
    doc.editorScrollTop = editor.scrollTop;
    doc.previewScrollTop = preview.scrollTop;
    renderTabs();
    renderFileList();
    setStatus("Saved");
  } catch (err) {
    setStatus(err.message);
  } finally {
    saveButton.disabled = false;
  }
}

editor.addEventListener("input", () => {
  commitActiveDraft();
  renderTabs();
  renderFileList();
  setStatus("Editing");
  window.clearTimeout(renderTimer);
  renderTimer = window.setTimeout(() => {
    render().catch((err) => setStatus(err.message));
  }, 180);
});

filterEl.addEventListener("input", renderFileList);
saveButton.addEventListener("click", () => {
  save();
});
editor.addEventListener("scroll", () => {
  const doc = activeDocument();
  if (doc) doc.editorScrollTop = editor.scrollTop;
  syncScroll(editor, preview);
});
preview.addEventListener("scroll", () => {
  const doc = activeDocument();
  if (doc) doc.previewScrollTop = preview.scrollTop;
  syncScroll(preview, editor);
});

window.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
    event.preventDefault();
    save();
  }
});

window.addEventListener("beforeunload", (event) => {
  commitActiveDraft();
  for (const doc of openDocs.values()) {
    if (!doc.dirty) continue;
    event.preventDefault();
    event.returnValue = "";
    return "";
  }
});

async function boot() {
  try {
    await loadFiles();
  } catch (err) {
    setStatus(err.message);
  }

  for (let i = 0; i < initialPaths.length; i += 1) {
    try {
      await openDocument(initialPaths[i], i === 0);
    } catch (err) {
      setStatus(err.message);
    }
  }

  if (!activePath) {
    renderEmpty();
    setStatus("Ready");
  }
}

boot();
</script>
</body>
</html>
`
