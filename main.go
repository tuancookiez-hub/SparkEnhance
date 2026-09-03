// SparkEnhance entry point.
//
// Architecture:
//   - main:     bootstraps the Wails WebView2 window
//   - app.go:   Wails bindings + global hotkey + window repositioning
//   - enhance/: GMI Cloud /v1/chat/completions client
//   - config/:  JSON-on-disk settings
//   - platform/: Win32 hook + clipboard + tray
//
// The Wails window is the floating bar. When hidden, the app keeps running
// in the system tray (background) and listens for the global hotkey.
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "SparkEnhance",
		Width:  440,
		Height: 200,
		// HideWindowOnClose: closing the X button hides instead of quits.
		// The app keeps running so the hotkey still fires.
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0x14, G: 0x14, B: 0x1c, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
