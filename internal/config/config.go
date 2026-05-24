package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const KeyLibrary = "library"

type Config struct {
	Library []string `json:"library"`
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".ink", "config.json"), nil
}

func CachePagesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".ink", "cache", "pages"), nil
}

func DefaultLibrary() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, "Downloads"), nil
}

func Load() (Config, error) {
	configPath, err := Path()
	if err != nil {
		return Config{}, err
	}

	raw, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return defaultConfig()
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", configPath, err)
	}
	return normalize(cfg)
}

func Save(cfg Config) error {
	cfg, err := normalize(cfg)
	if err != nil {
		return err
	}

	configPath, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	raw = append(raw, '\n')

	if err := os.WriteFile(configPath, raw, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func Set(key string, values []string) ([]string, error) {
	switch key {
	case KeyLibrary:
		return setLibrary(values)
	default:
		return nil, unknownKeyError(key)
	}
}

func Add(key, value string) ([]string, error) {
	switch key {
	case KeyLibrary:
		return addLibrary(value)
	default:
		return nil, unknownKeyError(key)
	}
}

func Remove(key, value string) ([]string, error) {
	switch key {
	case KeyLibrary:
		return removeLibrary(value)
	default:
		return nil, unknownKeyError(key)
	}
}

func Get(key string) ([]string, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	switch key {
	case KeyLibrary:
		return cfg.Library, nil
	default:
		return nil, unknownKeyError(key)
	}
}

func List() (Config, error) {
	return Load()
}

func Library() ([]string, error) {
	return Get(KeyLibrary)
}

func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	resolved := path
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory: %w", err)
		}
		if path == "~" {
			resolved = home
		} else {
			resolved = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	} else if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("only current-user home paths are supported")
	}

	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func setLibrary(values []string) ([]string, error) {
	libraries, err := resolveAndValidateLibraries(values)
	if err != nil {
		return nil, err
	}

	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	cfg.Library = libraries
	if err := Save(cfg); err != nil {
		return nil, err
	}
	return libraries, nil
}

func addLibrary(value string) ([]string, error) {
	library, err := ResolvePath(value)
	if err != nil {
		return nil, err
	}
	if err := validateDirectory(library); err != nil {
		return nil, err
	}
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	cfg.Library = appendIfMissing(cfg.Library, library)
	if err := Save(cfg); err != nil {
		return nil, err
	}
	return cfg.Library, nil
}

func removeLibrary(value string) ([]string, error) {
	library, err := ResolvePath(value)
	if err != nil {
		return nil, err
	}
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	libraries := make([]string, 0, len(cfg.Library))
	found := false
	for _, current := range cfg.Library {
		if current == library {
			found = true
			continue
		}
		libraries = append(libraries, current)
	}
	if !found {
		return nil, fmt.Errorf("library is not configured: %s", library)
	}
	if len(libraries) == 0 {
		return nil, fmt.Errorf("library must contain at least one path")
	}

	cfg.Library = libraries
	if err := Save(cfg); err != nil {
		return nil, err
	}
	return cfg.Library, nil
}

func defaultConfig() (Config, error) {
	library, err := DefaultLibrary()
	if err != nil {
		return Config{}, err
	}
	return Config{Library: []string{library}}, nil
}

func normalize(cfg Config) (Config, error) {
	if len(cfg.Library) == 0 {
		return defaultConfig()
	}
	library, err := resolveLibraries(cfg.Library)
	if err != nil {
		return Config{}, err
	}
	cfg.Library = library
	return cfg, nil
}

func decodeConfig(raw []byte) (Config, error) {
	var values struct {
		Library json.RawMessage `json:"library"`
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return Config{}, err
	}
	if len(values.Library) == 0 || string(values.Library) == "null" {
		return Config{}, nil
	}

	var library string
	if err := json.Unmarshal(values.Library, &library); err == nil {
		return Config{Library: []string{library}}, nil
	}

	var libraries []string
	if err := json.Unmarshal(values.Library, &libraries); err != nil {
		return Config{}, err
	}
	return Config{Library: libraries}, nil
}

func resolveAndValidateLibraries(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("library requires at least one path")
	}

	libraries, err := resolveLibraries(values)
	if err != nil {
		return nil, err
	}
	for _, library := range libraries {
		if err := validateDirectory(library); err != nil {
			return nil, err
		}
	}
	return libraries, nil
}

func resolveLibraries(values []string) ([]string, error) {
	libraries := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		library, err := ResolvePath(value)
		if err != nil {
			return nil, err
		}
		if seen[library] {
			continue
		}
		seen[library] = true
		libraries = append(libraries, library)
	}
	if len(libraries) == 0 {
		return nil, fmt.Errorf("library requires at least one path")
	}
	return libraries, nil
}

func appendIfMissing(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func validateDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("library directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("library must be a directory: %s", path)
	}
	return nil
}

func unknownKeyError(key string) error {
	if key == "" {
		return fmt.Errorf("config key is required")
	}
	return fmt.Errorf("unknown config key %q", key)
}
