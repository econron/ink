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
	Library string `json:"library"`
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

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
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

func Set(key, value string) (string, error) {
	switch key {
	case KeyLibrary:
		return setLibrary(value)
	default:
		return "", unknownKeyError(key)
	}
}

func Get(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	switch key {
	case KeyLibrary:
		return cfg.Library, nil
	default:
		return "", unknownKeyError(key)
	}
}

func List() (Config, error) {
	return Load()
}

func Library() (string, error) {
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

func setLibrary(value string) (string, error) {
	library, err := ResolvePath(value)
	if err != nil {
		return "", err
	}
	if err := validateDirectory(library); err != nil {
		return "", err
	}

	cfg, err := Load()
	if err != nil {
		return "", err
	}
	cfg.Library = library
	if err := Save(cfg); err != nil {
		return "", err
	}
	return library, nil
}

func defaultConfig() (Config, error) {
	library, err := DefaultLibrary()
	if err != nil {
		return Config{}, err
	}
	return Config{Library: library}, nil
}

func normalize(cfg Config) (Config, error) {
	if cfg.Library == "" {
		return defaultConfig()
	}
	library, err := ResolvePath(cfg.Library)
	if err != nil {
		return Config{}, err
	}
	cfg.Library = library
	return cfg, nil
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
