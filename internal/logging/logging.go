// Package logging writes to both stderr and a rotating file in the
// SparkEnhance app data dir.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tuancookiez-hub/sparkenhance/internal/config"
)

var (
	once   sync.Once
	closer io.Closer
)

// Setup redirects the default logger to also write to app.log.
func Setup() {
	once.Do(func() {
		dir := config.DefaultDir()
		if err := os.MkdirAll(dir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "logging: mkdir %s: %v\n", dir, err)
			return
		}
		logFile := filepath.Join(dir, "app.log")
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logging: open %s: %v\n", logFile, err)
			return
		}
		closer = f
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		log.SetOutput(io.MultiWriter(os.Stderr, f))
		fmt.Fprintf(f, "\n=== SparkEnhance started %s ===\n", time.Now().Format(time.RFC3339))
	})
}

// Close flushes the log file.
func Close() {
	if closer != nil {
		_ = closer.Close()
		closer = nil
	}
}
