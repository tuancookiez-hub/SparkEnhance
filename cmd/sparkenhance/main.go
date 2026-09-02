package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tuancookiez-hub/sparkenhance/internal/config"
	"github.com/tuancookiez-hub/sparkenhance/internal/enhance"
)

// Entry point. Full tray + hotkey + bar UI will replace the stub below.
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfgDir := config.DefaultDir()
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		log.Fatalf("create config dir: %v", err)
	}
	cfg, err := config.Load(cfgDir)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	fmt.Printf("SparkEnhance starting\n")
	fmt.Printf("  config dir: %s\n", cfgDir)
	fmt.Printf("  hotkey:     %s\n", cfg.Hotkey)
	fmt.Printf("  configured: %v\n", cfg.IsConfigured())

	// Handle SIGINT / SIGTERM for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("shutting down")
		cancel()
	}()

	// Verify the core pipeline works end-to-end with a real M3 call.
	// The GMI key is read from env (GMI_API_KEY) for first-run testing;
	// the GUI flow will prompt for it on first launch.
	if key := os.Getenv("GMI_API_KEY"); key != "" {
		client := enhance.NewClient(key)
		pingCtx, pingCancel := context.WithTimeout(ctx, 30*time.Second)
		err := client.ValidateKey(pingCtx)
		pingCancel()
		if err != nil {
			log.Printf("API key validation failed: %v", err)
		} else {
			log.Printf("API key validated against GMI Cloud")
		}
	} else {
		log.Printf("GMI_API_KEY not set — GUI will prompt on first launch")
	}

	<-ctx.Done()
}
