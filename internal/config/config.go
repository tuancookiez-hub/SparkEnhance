package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config holds app settings persisted to disk at
// %APPDATA%\SparkEnhance\config.json on Windows.
type Config struct {
	mu        sync.RWMutex
	path      string
	apiKey    string `json:"api_key"`
	baseURL   string `json:"base_url"`
	model     string `json:"model"`
	hotkey    string `json:"hotkey"`
	autoPaste bool   `json:"auto_paste"`
}

// DefaultDir returns %APPDATA%\SparkEnhance on Windows.
func DefaultDir() string {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		dir = os.Getenv("XDG_CONFIG_HOME")
	}
	if dir == "" {
		dir, _ = os.UserHomeDir()
		dir = filepath.Join(dir, ".config")
	}
	return filepath.Join(dir, "SparkEnhance")
}

// NewDefault returns a Config with defaults, unsaved. Used by app.go when
// the config file cannot be loaded.
func NewDefault() *Config {
	return &Config{
		baseURL:   DefaultBaseURL,
		model:     DefaultModel,
		hotkey:    DefaultHotkey,
		autoPaste: false,
	}
}

const (
	DefaultBaseURL = "https://api.gmi-serving.com/v1"
	DefaultModel   = "MiniMax-M3"
	DefaultHotkey  = "ctrl+shift+e"
)

// Load reads the config file from dir. Returns a zero (default) config if absent.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				path:     path,
				baseURL:  DefaultBaseURL,
				model:    DefaultModel,
				hotkey:   DefaultHotkey,
				autoPaste: false,
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.path = path
	cfg.fillDefaults()
	return &cfg, nil
}

func (c *Config) fillDefaults() {
	if c.baseURL == "" {
		c.baseURL = DefaultBaseURL
	}
	if c.model == "" {
		c.model = DefaultModel
	}
	if c.hotkey == "" {
		c.hotkey = DefaultHotkey
	}
}

// Save writes cfg to its path atomically.
func (c *Config) Save() error {
	c.mu.RLock()
	path := c.path
	c.mu.RUnlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// SetAll atomically updates the full configuration and saves it.
func (c *Config) SetAll(apiKey, baseURL, model, hotkey string, autoPaste bool) error {
	c.mu.Lock()
	c.apiKey = apiKey
	if baseURL != "" {
		c.baseURL = baseURL
	}
	if model != "" {
		c.model = model
	}
	if hotkey != "" {
		c.hotkey = hotkey
	}
	c.autoPaste = autoPaste
	c.fillDefaults()
	c.mu.Unlock()
	return c.Save()
}

// SetAPIKey persists the API key.
func (c *Config) SetAPIKey(key string) error {
	c.mu.Lock()
	c.apiKey = key
	c.mu.Unlock()
	return c.Save()
}

// SetAutoPaste persists the auto-paste toggle.
func (c *Config) SetAutoPaste(v bool) error {
	c.mu.Lock()
	c.autoPaste = v
	c.mu.Unlock()
	return c.Save()
}

// SetHotkey persists a new hotkey chord.
func (c *Config) SetHotkey(hk string) error {
	if hk == "" {
		return fmt.Errorf("hotkey cannot be empty")
	}
	c.mu.Lock()
	c.hotkey = hk
	c.mu.Unlock()
	return c.Save()
}

// GetAPIKey returns the API key.
func (c *Config) GetAPIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey
}

// IsConfigured returns true if a non-empty API key is set.
func (c *Config) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey != ""
}

// GetBaseURL returns the GMI base URL.
func (c *Config) GetBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.baseURL == "" {
		return DefaultBaseURL
	}
	return c.baseURL
}

// GetModel returns the configured model name.
func (c *Config) GetModel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.model == "" {
		return DefaultModel
	}
	return c.model
}

// Hotkey returns the current hotkey string.
func (c *Config) Hotkey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hotkey
}

// AutoPaste returns the auto-paste setting.
func (c *Config) AutoPaste() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.autoPaste
}
