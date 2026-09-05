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
	apiKey  string
	baseURL string
	model   string
	hotkey  string
}

// Load returns the persisted config, or sensible defaults if no file exists.
// API key is read from disk only — never from environment or hardcoded.
func Load(path string) (*Config, error) {
	c := &Config{
		baseURL: "https://api.gmi-serving.com/v1",
		model:   "MiniMaxAI/MiniMax-M3",
		hotkey:  "ctrl+shift+e",
	}

	if path == "" {
		return c, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}

	var raw struct {
		APIKey  string `json:"apiKey"`
		BaseURL string `json:"baseURL"`
		Model   string `json:"model"`
		Hotkey  string `json:"hotkey"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return c, err
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
	return c, nil
}

// Save persists the config. Used by the settings form.
func Save(path string, c *Config) error {
	raw := struct {
		APIKey  string `json:"apiKey"`
		BaseURL string `json:"baseURL"`
		Model   string `json:"model"`
		Hotkey  string `json:"hotkey"`
	}{
		APIKey:  c.apiKey,
		BaseURL: c.baseURL,
		Model:   c.model,
		Hotkey:  c.hotkey,
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Getters — explicitly typed so we can return empty strings safely.

func (c *Config) APIKey() string  { return c.apiKey }
func (c *Config) BaseURL() string { return c.baseURL }
func (c *Config) Model() string   { return c.model }
func (c *Config) Hotkey() string  { return c.hotkey }

// Setters.

func (c *Config) SetAPIKey(v string)  { c.apiKey = v }
func (c *Config) SetBaseURL(v string) { c.baseURL = v }
func (c *Config) SetModel(v string)   { c.model = v }
func (c *Config) SetHotkey(v string)  { c.hotkey = v }
