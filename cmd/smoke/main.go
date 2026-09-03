// Smoke test: verifies hotkey parsing, startup registry, and enhance client
// all link and function. Run with: go run ./cmd/smoke/
package main

import (
	"fmt"
	"os"

	"github.com/tuancookiez-hub/sparkenhance/internal/config"
	"github.com/tuancookiez-hub/sparkenhance/internal/enhance"
	"github.com/tuancookiez-hub/sparkenhance/internal/hotkey"
	"github.com/tuancookiez-hub/sparkenhance/internal/startup"
)

func main() {
	fmt.Println("SparkEnhance smoke test")
	fmt.Println("=======================")

	// 1. Hotkey parsing
	hmod, vk, err := hotkey.Parse("ctrl+shift+e")
	if err != nil {
		fmt.Printf("FAIL hotkey parse: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK   hotkey ctrl+shift+e: hmod=0x%x vk=0x%x\n", hmod, vk)

	// 2. Config
	dir := config.DefaultDir()
	fmt.Printf("OK   config dir: %s\n", dir)

	// 3. Startup registry read
	fmt.Printf("OK   startup.IsEnabled: %v\n", startup.IsEnabled())

	// 4. Enhance client (does not call API without a real key)
	c := enhance.NewClient("dummy-key")
	fmt.Printf("OK   enhance client: baseURL=%s model=%s\n", c.BaseURL(), c.Model())

	fmt.Println("\nAll components link and initialize correctly.")
}
