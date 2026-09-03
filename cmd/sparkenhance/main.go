// Package main is the SparkEnhance entry point.
//
// First-run flow:
//  1. Load config from %APPDATA%\SparkEnhance\config.json
//  2. If no API key, show the native setup dialog and persist the entered key
//  3. Start the message loop with tray + global hotkey + floating bar
//
// Subsequent runs skip the dialog and go straight to the tray.
//
// All logs are written to %APPDATA%\SparkEnhance\app.log AND stderr.
package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/tuancookiez-hub/sparkenhance/internal/config"
	"github.com/tuancookiez-hub/sparkenhance/internal/enhance"
	"github.com/tuancookiez-hub/sparkenhance/internal/logging"
	"github.com/tuancookiez-hub/sparkenhance/internal/setup"
	"github.com/tuancookiez-hub/sparkenhance/internal/startup"
	"github.com/tuancookiez-hub/sparkenhance/internal/win"
)

//go:embed assets/icon.ico
var iconFS embed.FS

func main() {
	// Lock the main goroutine to its OS thread for Win32 calls.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	logging.Setup()
	defer logging.Close()
	log.Println("SparkEnhance starting")

	cfgDir := config.DefaultDir()
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		log.Fatalf("create config dir %s: %v", cfgDir, err)
	}
	cfg, err := config.Load(cfgDir)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("config dir: %s", cfgDir)
	log.Printf("hotkey: %s, auto-paste: %v, configured: %v", cfg.Hotkey, cfg.AutoPaste, cfg.IsConfigured())

	// First-run setup: show the native dialog if no key is stored.
	if !cfg.IsConfigured() {
		log.Println("no API key configured; showing setup dialog")
		key, ok := setup.PromptAPIKey()
		if !ok {
			log.Println("setup cancelled by user; exiting")
			return
		}
		if err := cfg.SetAPIKey(key); err != nil {
			log.Fatalf("save API key: %v", err)
		}
		log.Println("API key saved")
	}

	// Validate the key against GMI Cloud.
	{
		client := enhance.NewClient(cfg.GetAPIKey())
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := client.ValidateKey(pingCtx); err != nil {
			log.Printf("WARNING: GMI API key validation failed: %v", err)
		} else {
			log.Println("GMI API key OK")
		}
		pingCancel()
	}

	// Register Windows startup entry (best-effort, non-fatal).
	if err := startup.Enable(); err != nil {
		log.Printf("startup registration: %v", err)
	}

	// Wire Ctrl+C / SIGTERM → shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("received signal, shutting down")
		cancel()
	}()

	iconData, err := iconFS.ReadFile("assets/icon.ico")
	if err != nil {
		log.Fatalf("embed icon: %v", err)
	}

	if err := win.Run(ctx, cfg, iconData); err != nil {
		log.Fatalf("win.Run: %v", err)
	}
	log.Println("SparkEnhance exited cleanly")
}
