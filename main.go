// main.go — SparkEnhance, Wails v3 entry point.
package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/tuancookiez-hub/SparkEnhance/internal/config"
	"github.com/tuancookiez-hub/SparkEnhance/internal/enhance"
	"github.com/tuancookiez-hub/SparkEnhance/internal/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed frontend/dist
var embedAssets embed.FS

// ─── Event names ─────────────────────────────────────────────────────────────

const (
	evStart  = "enhance:start"
	evResult = "enhance:result"
	evError  = "enhance:error"
	evIdle   = "enhance:idle"
	evCopied = "clipboard:copied"
)

// ─── App state ───────────────────────────────────────────────────────────────

const (
	barWidth       = 600
	barHeight      = 56
	autoHideSec    = 30
	enhanceTimeout = 90
)

var (
	// bar is a WebviewWindow (value type) — &bar is *WebviewWindow which
	// satisfies application.Window.
	bar           application.WebviewWindow
	appInstance   *application.App
	barVisible    atomic.Bool
	isEnhancing   atomic.Bool
	autoHideCh    = make(chan struct{}, 1)
)

// ─── main ────────────────────────────────────────────────────────────────────

func main() {
	setupLogging()

	cfg := config.Load()
	if cfg.APIKey() == "" {
		if k := os.Getenv("MINIMAX_API_KEY"); k != "" {
			cfg.SetAPIKey(k)
		}
	}

	appInstance = application.New(application.Options{
		Name:        "SparkEnhance",
		Description: "Hover-enhance any text → numbered agent brief",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(embedAssets),
		},
	})

	setupBar()
	setupTray()
	setupEvents()
	setupHotkey(cfg)

	go autoHideLoop()

	appInstance.Run()
}

// ─── Bar ─────────────────────────────────────────────────────────────────────

func setupBar() {
	bar = *appInstance.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "SparkEnhance",
		Width:            barWidth,
		Height:           barHeight,
		AlwaysOnTop:      true,
		DisableResize:    true,
		Frameless:        true,
		Hidden:           true,
		URL:              "/",
		BackgroundColour: application.NewRGB(24, 26, 38),
		BackgroundType:   application.BackgroundTypeSolid,
		X:                -9999,
		Y:                -9990,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHidden,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
	})
	bar.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		_ = e
		barVisible.Store(false)
	})
}

// ─── System tray ─────────────────────────────────────────────────────────────

func setupTray() {
	tray := appInstance.SystemTray.New()
	menu := appInstance.Menu.New()
	menu.Add("Show Bar (Ctrl+Shift+E)").OnClick(func(ctx *application.Context) {
		_ = ctx
		showBar(barWidth, barHeight)
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		_ = ctx
		appInstance.Quit()
	})
	tray.SetMenu(menu)
	tray.SetTooltip("SparkEnhance")
}

// ─── Events ──────────────────────────────────────────────────────────────────

func setupEvents() {
	appInstance.Event.On("bar:activity", func(e *application.CustomEvent) {
		_ = e
		resetAutoHide()
	})

	appInstance.Event.On("bar:copy-output", func(e *application.CustomEvent) {
		if data, ok := e.Data.(map[string]any); ok {
			if text, ok := data["text"].(string); ok && text != "" {
				copyOutput(text)
			}
		}
	})

	appInstance.Event.On("bar:dismiss", func(e *application.CustomEvent) {
		_ = e
		dismissBar()
	})

	appInstance.Event.On("app:quit", func(e *application.CustomEvent) {
		_ = e
		appInstance.Quit()
	})
}

// ─── Hotkey ─────────────────────────────────────────────────────────────────

func setupHotkey(cfg *config.Config) {
	_ = appInstance.GlobalShortcut.Register(cfg.Hotkey(), func() {
		runEnhance(cfg)
	})
}

// ─── Enhance ────────────────────────────────────────────────────────────────

func runEnhance(cfg *config.Config) {
	if isEnhancing.Swap(true) {
		return
	}
	defer isEnhancing.Store(false)

	sel, err := platform.CaptureSelection()
	if err != nil || sel == "" {
		appInstance.Event.Emit(evError, map[string]any{"error": "No text selected"})
		showBar(barWidth, barHeight)
		return
	}

	appInstance.Event.Emit(evStart, map[string]any{"selection": sel})
	showBar(barWidth, barHeight)

	c := enhance.New(cfg.APIKey(), cfg.BaseURL(), cfg.Model())
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(enhanceTimeout)*time.Second)
	defer cancel()

	res, err := c.Enhance(ctx, sel)
	if err != nil {
		appInstance.Event.Emit(evError, map[string]any{"error": err.Error()})
		return
	}
	appInstance.Event.Emit(evResult, map[string]any{"result": res})
}

func copyOutput(text string) {
	if err := platform.WriteClipboard(text); err != nil {
		appInstance.Event.Emit(evError, map[string]any{"error": "Copy failed"})
		return
	}
	appInstance.Event.Emit(evCopied, map[string]any{"text": text})
	if cfg := config.Load(); cfg.AutoPaste() {
		// future: restore focus + simulate Ctrl+V
	}
	dismissBar()
}

// ─── Bar show / dismiss ──────────────────────────────────────────────────────

func showBar(w, h int) {
	if barVisible.Load() {
		resizeBar(w, h)
	} else {
		positionBar(w, h)
	}
	bar.Show()
	_ = platform.SetWindowRounded(&bar)
	barVisible.Store(true)
	resetAutoHide()
}

func dismissBar() {
	bar.Hide()
	barVisible.Store(false)
	isEnhancing.Store(false)
}

func positionBar(w, h int) {
	cx, cy := platform.CursorPos()
	wa, err := platform.WorkAreaAt(cx, cy)
	if err != nil || wa.Width == 0 {
		bar.SetPosition(100, 100)
		bar.SetSize(w, h)
		return
	}
	x := wa.X + (wa.Width-w)/2
	y := cy + 20
	if y+h > wa.Y+wa.Height-10 {
		y = wa.Y + wa.Height - h - 24
	}
	bar.SetPosition(x, y)
	bar.SetSize(w, h)
}

func resizeBar(w, h int) {
	hwnd := uintptr(bar.NativeWindow())
	x, y, oldW, oldH, ok := platform.WindowBounds(hwnd)
	if !ok {
		bar.SetSize(w, h)
		return
	}
	mon, _ := platform.MonitorAt(x+oldW/2, y+oldH/2)
	if mon.Width == 0 {
		mon, _ = platform.WorkAreaAt(x+oldW/2, y+oldH/2)
	}
	if mon.Width > 0 {
		if x+w > mon.X+mon.Width-4 {
			x = mon.X + mon.Width - w - 4
		}
		if x < mon.X+4 {
			x = mon.X + 4
		}
		if y+h > mon.Y+mon.Height-4 {
			y = mon.Y + mon.Height - h - 4
		}
	}
	bar.SetPosition(x, y)
	bar.SetSize(w, h)
	_ = platform.SetWindowRounded(&bar)
}

// ─── Auto-hide ───────────────────────────────────────────────────────────────

func resetAutoHide() {
	select {
	case autoHideCh <- struct{}{}:
	default:
	}
}

func autoHideLoop() {
	for {
		t := time.NewTimer(time.Duration(autoHideSec) * time.Second)
		select {
		case <-t.C:
			dismissBar()
		case <-autoHideCh:
			t.Stop()
		}
	}
}

// ─── Logging ─────────────────────────────────────────────────────────────────

func setupLogging() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if f, err := os.OpenFile(filepath.Join(config.DefaultDir(), "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		log.SetOutput(f)
	}
}
