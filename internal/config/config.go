package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config holds app settings persisted to disk.
type Config struct {
	mu        sync.RWMutex
	path      string
	APIKey    string `json:"api_key"`
	Hotkey    string `json:"hotkey"`
	AutoPaste bool   `json:"auto_paste"`
}

// DefaultDir returns %APPDATA%\SparkEnhance on Windows.
func DefaultDir() string {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	return filepath.Join(dir, "SparkEnhance")
}

// Load reads the config file from dir. Returns a zero config if absent.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{path: path, Hotkey: "ctrl+shift+e", AutoPaste: false}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.path = path
	if cfg.Hotkey == "" {
		cfg.Hotkey = "ctrl+shift+e"
	}
	return &cfg, nil
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

// GetAPIKey returns the API key (read-only snapshot).
func (c *Config) GetAPIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.APIKey
}

// SetAPIKey stores and persists the key.
func (c *Config) SetAPIKey(key string) error {
	c.mu.Lock()
	c.APIKey = key
	c.mu.Unlock()
	return c.Save()
}

// IsConfigured returns true if a non-empty API key is set.
func (c *Config) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.APIKey != ""
}
