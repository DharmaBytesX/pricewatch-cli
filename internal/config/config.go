// Package config finds the Pricewatch address and the API token the CLI
// uses. Each comes from, in order: a command-line flag, an environment
// variable, the configuration file, and for the address a default.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DefaultURL is the Pricewatch website.
const DefaultURL = "https://pricewatch.exe.xyz"

// Environment variables.
const (
	EnvURL    = "PRICEWATCH_URL"    // Pricewatch address
	EnvToken  = "PRICEWATCH_TOKEN"  // API token
	EnvConfig = "PRICEWATCH_CONFIG" // configuration file path
)

// Where a setting came from.
const (
	SourceFlag    = "flag"
	SourceEnv     = "environment"
	SourceFile    = "config file"
	SourceDefault = "default"
	SourceNone    = ""
)

// File is the configuration file's content.
type File struct {
	URL   string `json:"url,omitempty"`
	Token string `json:"token,omitempty"`
}

// Path returns the configuration file: $PRICEWATCH_CONFIG, or
// pricewatch/config.json in the user's configuration directory
// (~/.config on Linux, ~/Library/Application Support on macOS,
// %AppData% on Windows).
func Path(getenv func(string) string) (string, error) {
	if p := getenv(EnvConfig); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find the configuration directory: %w", err)
	}
	return filepath.Join(dir, "pricewatch", "config.json"), nil
}

// Load reads the configuration file; a missing file is an empty File.
func Load(path string) (File, error) {
	var f File
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("read %s: %w", path, err)
	}
	return f, nil
}

// Save writes the configuration file, readable by the user only: it holds
// the API token.
func Save(path string, f File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	// Write a temporary file, then rename it: a crash never leaves half a file.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer os.Remove(tmp.Name()) // nothing to remove after the rename
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Settings are the address and token a command uses, and where each came
// from.
type Settings struct {
	URL         string
	URLSource   string
	Token       string
	TokenSource string
}

// Resolve picks each setting from the flag, the environment, then the file.
func Resolve(flagURL, flagToken string, getenv func(string) string, f File) Settings {
	var s Settings
	s.URL, s.URLSource = first(
		candidate{flagURL, SourceFlag},
		candidate{getenv(EnvURL), SourceEnv},
		candidate{f.URL, SourceFile},
		candidate{DefaultURL, SourceDefault},
	)
	s.URL = strings.TrimRight(s.URL, "/")
	s.Token, s.TokenSource = first(
		candidate{flagToken, SourceFlag},
		candidate{getenv(EnvToken), SourceEnv},
		candidate{f.Token, SourceFile},
	)
	return s
}

type candidate struct{ value, source string }

func first(cs ...candidate) (value, source string) {
	for _, c := range cs {
		if v := strings.TrimSpace(c.value); v != "" {
			return v, c.source
		}
	}
	return "", SourceNone
}
