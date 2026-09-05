// SparkEnhance — User-configurable path.
// The config file lives at %APPDATA%\SparkEnhance\config.json (Windows) or
// ~/.config/SparkEnhance/config.json (Linux/macOS). It is NOT committed
// to git (.gitignore'd as config.json).
package main

import (
	"path/filepath"
	"os"
)

// configPath returns the platform-appropriate config file location.
func configPath() string {
	if p := os.Getenv("MINIMAX_CONFIG_DIR"); p != "" {
		return filepath.Join(p, "config.json")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "SparkEnhance", "config.json")
}
