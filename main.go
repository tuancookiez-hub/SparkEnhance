// main.go — Wails v3 entry point.
// No secrets here. All credentials come from config file or environment variables.
package main

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wuancookiez-hub/SparkEnhance/internal/config"
	"github.com/wuancookiez-hub/SparkEnhance/internal/enhance"
	"github.com/wuancookiez-hub/SparkEnhance/internal/placement"
	"github.com/wuancookiez-hub/SparkEnhance/internal/platform"
)

const (
	appName           = "SparkEnhance"
	barWidth          = 600
	barHeight         = 56
	autoHideSeconds   = 30
	enhanceTimeoutSec = 90
)

var (
	bar          application.Window
	barVisible   atomic.Bool
	autoHideCh   = make(chan struct{}, 1)
	isEnhancing  atomic.Bool
	selText      atomic.Value // string
)

// Event names emitted to frontend
const (
	evStart     = "enhance:start"
	evResult    = "enhance:result"
	evError     = "enhance:error"
	evIdle      = "enhance:idle"
	evCopied    = "clipboard:copied"
)

func main() {
	cfg := loadConfig()
	selText.Store("")
	barVisible.Store(false)

	app := application.New(&application.App{
		Name:        appName,
		Description: "Hover-enhance any text → numbered agent brief",
		Assets:      &application.AssetOptions{FS: embedAssets},
	})

	setupTray(app)
	setupBar(app)
	setupHotkey(app, cfg)

	go autoHideLoop()

	app.Run()
}

func loadConfig() *config.Config {
	cfg := config.Load()
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("MINIMAX_API_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.gmi-serving.com/v1"
		cfg.Model = "MiniMaxAI/MiniMax-M3"
	}
	return cfg
}

func runEnhance(app *application.App, cfg *config.Config) {
	if isEnhancing.Swap(true) {
		return
	}
	sel, err := platform.CaptureSelection()
	if err != nil || sel == "" {
		app.Events.EmitEvent(evError, map[string]any{"error": "No text selected"})
		showBar(barWidth, barHeight)
		return
	}
	app.Events.EmitEvent(evStart, map[string]any{"selection": sel})
	showBar(barWidth, barHeight)

	c := enhance.New(cfg.APIKey, cfg.BaseURL, cfg.Model)
	ctx, cancel := context.WithTimeout(context.Background(), enhanceTimeoutSec*time.Second)
	defer cancel()

	res, err := c.Enhance(ctx, sel)
	if err != nil {
		app.Events.EmitEvent(evError, map[string]any{"error": err.Error()})
		isEnhancing.Store(false)
		return
	}

	app.Events.EmitEvent(evResult, map[string]any{"result": res})
}

func showBar(w, h int) {
	if barVisible.Load() {
		resizeBar(w, h)
	} else {
		positionBar(w, h)
	}
	bar.Show()
	if hwnd := bar.NativeWindow(); hwnd != 0 {
		platform.SetRoundedWindowRegion(hwnd, 15)
	}
	barVisible.Store(true)
	resetAutoHide()
}

func hideBar() {
	bar.Hide()
	barVisible.Store(false)
}

func positionBar(w, h int) {
	mon, _ := platform.PrimaryMonitor()
	if mon.Width == 0 {
		bar.SetPos(100, 100)
		bar.SetSize(w, h)
		return
	}
	x, y := placement.Bar(mon, w, h)
	bar.SetPos(x, y)
	bar.SetSize(w, h)
}

func resizeBar(w, h int) {
	hwnd := bar.NativeWindow()
	x, y, ow, oh, ok := platform.WindowBounds(hwnd)
	if !ok {
		bar.SetSize(w, h)
		return
	}
	mon, _ := platform.MonitorFromPoint(x+ow/2, y+oh/2)
	if mon.Width == 0 {
		mon, _ = platform.PrimaryMonitor()
	}
	w, h = placement.LimitSize(mon, w, h, barWidth, barHeight, 0.6)
	bar.SetSize(w, h)
	platform.MoveWindowTopLeft(hwnd, x, y, w, h)
	platform.SetRoundedWindowRegion(hwnd, 15)
}

func resetAutoHide() {
	select {
	case autoHideCh <- struct{}{}:
	default:
	}
}

func autoHideLoop() {
	for {
		t := time.NewTimer(autoHideSeconds * time.Second)
		select {
		case <-t.C:
			hideBar()
		case <-autoHideCh:
			t.Stop()
		}
	}
}

func setupTray(app *application.App) {
	tray := app.NewTray()
	tray.SetTooltip(appName)

	m := app.NewMenu()
	m.Add("Show Bar (Ctrl+Shift+E)").OnClick(func() { showBar(barWidth, barHeight) })
	m.AddSeparator()
	m.Add("Settings…").OnClick(func() {
		win, _ := app.GetWebviewWindow("settings")
		if win != nil {
			win.Show()
			win.BringToTop()
		}
	})
	m.AddSeparator()
	m.Add("Quit").OnClick(func() { app.Quit() })
	tray.SetMenu(m)
}

func setupBar(app *application.App) {
	bar = app.NewWebviewWindow(&application.WebviewWindowOptions{
		Title:     "",
		Width:     barWidth,
		Height:    barHeight,
		AlwaysOnTop: true,
		Decorations: false,
		Transparent: true,
		URL:         "/",
		X: -9999,
		Y: -9990,
		Windows: &application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
		Mac: &application.MacWindow{
			TitleBar:         application.TitleBarHidden,
			FullSizeContent:  true,
		},
	})

	bar.RegisterHook(application.OnWindowShown, func(e *application.WindowEvent) {})
	bar.RegisterHook(application.OnWindowClosed, func(e *application.WindowEvent) {
		barVisible.Store(false)
	})
}

func setupHotkey(app *application.App, cfg *config.Config) {
	hotkey := app.NewHotKey(&application.HotKeyOptions{
		Accelerator: "ctrl+shift+e",
		Callback: func() { runEnhance(app, cfg) },
	})
	_ = hotkey // kept alive for lifetime of app
}

//go:embed frontend/dist
var embedAssets any // replaced by go:embed directive in real build
