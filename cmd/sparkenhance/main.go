package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/tuancookiez-hub/sparkenhance/internal/config"
	"github.com/tuancookiez-hub/sparkenhance/internal/enhance"
	"github.com/tuancookiez-hub/sparkenhance/internal/win"
)

//go:embed assets/icon.ico
var iconFS embed.FS

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stderr)

	cfgDir := config.DefaultDir()
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		log.Fatalf("create config dir %s: %v", cfgDir, err)
	}
	cfg, err := config.Load(cfgDir)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if !cfg.IsConfigured() {
		key, err := consolePrompt("Enter GMI Cloud API key")
		if err != nil {
			log.Fatalf("first-run setup failed: %v", err)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			log.Fatalf("API key is required")
		}
		if err := cfg.SetAPIKey(key); err != nil {
			log.Fatalf("save API key: %v", err)
		}
		log.Printf("API key saved to %s", cfgDir)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("shutting down")
		cancel()
	}()

	// End-to-end smoke test: validate the key against the real GMI Cloud.
	{
		client := enhance.NewClient(cfg.GetAPIKey())
		pingCtx, pingCancel := context.WithTimeout(ctx, 30*time.Second)
		if err := client.ValidateKey(pingCtx); err != nil {
			log.Printf("WARNING: GMI API key validation failed: %v", err)
		} else {
			log.Println("GMI API key OK")
		}
		pingCancel()
	}

	iconData, err := iconFS.ReadFile("assets/icon.ico")
	if err != nil {
		log.Fatalf("embed icon: %v", err)
	}

	if err := win.Run(ctx, cfg, iconData); err != nil {
		log.Fatalf("app error: %v", err)
	}
}

func consolePrompt(label string) (string, error) {
	fmt.Fprintf(os.Stdout, "%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
