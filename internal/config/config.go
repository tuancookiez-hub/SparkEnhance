// Package config handles loading and saving user settings from a JSON file
// stored under %APPDATA%\SparkEnhance\config.json (Windows) or the equivalent
// on other platforms.
//
// IMPORTANT: this package never logs or exposes the API key.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	apiKey   string
	baseURL  string
	model    string
	hotkey   string
	autoPaste bool
}

// DefaultDir returns the platform-appropriate config directory.
// Returns ~/.config/SparkEnhance on Unix, %APPDATA%\SparkEnhance on Windows.
func DefaultDir() string {
	if p := os.Getenv("MINIMAX_CONFIG_DIR"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "SparkEnhance")
}

// DefaultPath returns the full path to the config file.
func DefaultPath() string {
	return filepath.Join(DefaultDir(), "config.json")
}

// Load returns the persisted config, or sensible defaults if no file exists.
// API key is read from disk only — never from environment or hardcoded.
func Load() *Config {
	c := &Config{
		baseURL:  "https://api.gmi-serving.com/v1",
		model:    "MiniMaxAI/MiniMax-M3",
		hotkey:   "ctrl+shift+e",
		autoPaste: false,
	}

	path := DefaultPath()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c
	}
	if err != nil {
		return c
	}

	var raw struct {
		APIKey    string `json:"apiKey"`
		BaseURL   string `json:"baseURL"`
		Model     string `json:"model"`
		Hotkey    string `json:"hotkey"`
		AutoPaste bool   `json:"autoPaste"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return c
	}
	c.apiKey = raw.APIKey
	if raw.BaseURL != "" {
		c.baseURL = raw.BaseURL
	}
	if raw.Model != "" {
		c.model = raw.Model
	}
	if raw.Hotkey != "" {
		c.hotkey = raw.Hotkey
	}
	c.autoPaste = raw.AutoPaste
	return c
}

// Save persists the config. Used by the settings form.
func Save(c *Config) error {
	raw := struct {
		APIKey    string `json:"apiKey"`
		BaseURL   string `json:"baseURL"`
		Model     string `json:"model"`
		Hotkey    string `json:"hotkey"`
		AutoPaste bool   `json:"autoPaste"`
	}{
		APIKey:    c.apiKey,
		BaseURL:   c.baseURL,
		Model:     c.model,
		Hotkey:    c.hotkey,
		AutoPaste: c.autoPaste,
	}
	path := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Getters.

func (c *Config) APIKey() string  { return c.apiKey }
func (c *Config) BaseURL() string { return c.baseURL }
func (c *Config) Model() string   { return c.model }
func (c *Config) Hotkey() string  { return c.hotkey }
func (c *Config) AutoPaste() bool { return c.autoPaste }

// Setters.

func (c *Config) SetAPIKey(v string)   { c.apiKey = v }
func (c *Config) SetBaseURL(v string)  { c.baseURL = v }
func (c *Config) SetModel(v string)    { c.model = v }
func (c *Config) SetHotkey(v string)   { c.hotkey = v }
func (c *Config) SetAutoPaste(v bool)  { c.autoPaste = v }
