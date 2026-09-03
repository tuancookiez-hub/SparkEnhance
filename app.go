// Package main is the SparkEnhance Wails backend.
//
// Architecture:
//   - app.go:        Wails-bound methods callable from the React UI
//   - enhance/:      GMI Cloud /v1/chat/completions client (M3 model)
//   - config/:       JSON-on-disk settings at %APPDATA%\SparkEnhance\config.json
//   - platform/:     OS glue — global hotkey, system tray, clipboard
//
// All long-running operations are exposed to the UI as Wails events so
// the React frontend can render progress and final output.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/tuancookiez-hub/sparkenhance/internal/config"
	"github.com/tuancookiez-hub/sparkenhance/internal/enhance"
	"github.com/tuancookiez-hub/sparkenhance/internal/platform"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is bound to the Wails frontend. Every exported method becomes a
// TypeScript function in frontend/wailsjs/go/main/App.
type App struct {
	ctx       context.Context
	cfg       *config.Config
	enhancer  *enhance.Client
	mu        sync.Mutex
	started   bool
	hotkeyCh  chan struct{}
	cancelHot context.CancelFunc
}

// NewApp constructs the app with the default config and GMI base URL.
// Per-request model + API key come from the user via Setup().
func NewApp() *App {
	cfg, err := config.Load(config.DefaultDir())
	if err != nil {
		log.Printf("config load: %v (using empty defaults)", err)
		cfg = config.NewDefault()
	}
	return &App{
		cfg:      cfg,
		enhancer: enhance.NewClient(""),
	}
}

// startup is invoked by Wails after the WebView2 window is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.SetOutput(logFile())
	log.Println("SparkEnhance starting (Wails backend)")

	// Initialize system tray.
	trayCtx, _ := context.WithCancel(ctx)
	go platform.RunTray(trayCtx, a.onTrayEnhance, a.onTrayQuit, a.onTraySettings)

	// Initialize global hotkey from saved config.
	a.installHotkey()

	a.started = true
	log.Println("SparkEnhance ready")
}

// shutdown is invoked by Wails on window close.
func (a *App) shutdown(ctx context.Context) {
	if a.cancelHot != nil {
		a.cancelHot()
	}
	log.Println("SparkEnhance shutting down")
}

// onTrayEnhance is called when the user clicks the tray's "Enhance" item.
func (a *App) onTrayEnhance() {
	a.fireEnhanceFromClipboard()
}

// onTraySettings is called when the user clicks the tray's "Settings" item.
// It shows the main Wails window so they can edit API key / model.
func (a *App) onTraySettings() {
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
}

// onTrayQuit is called when the user clicks the tray's "Quit" item.
func (a *App) onTrayQuit() {
	wailsruntime.Quit(a.ctx)
}

// ====== Bound methods (callable from React) ======

// IsConfigured reports whether an API key has been saved.
func (a *App) IsConfigured() bool { return a.cfg.IsConfigured() }

// SetupInput is the first-run form: save the API key and the chosen model.
type SetupInput struct {
	APIKey    string `json:"apiKey"`
	BaseURL   string `json:"baseUrl"`
	Model     string `json:"model"`
	Hotkey    string `json:"hotkey"`
	AutoPaste bool   `json:"autoPaste"`
}

// SaveSetup persists the configuration and (re)installs the global hotkey.
func (a *App) SaveSetup(in SetupInput) error {
	if in.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if in.BaseURL == "" {
		in.BaseURL = "https://api.gmi-serving.com/v1"
	}
	if in.Model == "" {
		in.Model = "MiniMax-M3"
	}
	if in.Hotkey == "" {
		in.Hotkey = "ctrl+shift+e"
	}
	if err := a.cfg.SetAll(in.APIKey, in.BaseURL, in.Model, in.Hotkey, in.AutoPaste); err != nil {
		return err
	}
	a.enhancer = enhance.NewClientWithBase(in.APIKey, in.BaseURL, in.Model)
	a.installHotkey()
	return nil
}

// GetConfig returns the current configuration (minus the API key for safety).
func (a *App) GetConfig() map[string]any {
	return map[string]any{
		"baseUrl":   a.cfg.GetBaseURL(),
		"model":     a.cfg.GetModel(),
		"hotkey":    a.cfg.Hotkey(),
		"autoPaste": a.cfg.AutoPaste(),
		"hasKey":    a.cfg.IsConfigured(),
	}
}

// ValidateKey pings GMI /v1/models with the saved key to confirm it works.
func (a *App) ValidateKey() error {
	if !a.cfg.IsConfigured() {
		return fmt.Errorf("no API key configured")
	}
	a.enhancer = enhance.NewClientWithBase(a.cfg.GetAPIKey(), a.cfg.GetBaseURL(), a.cfg.GetModel())
	ctx, cancel := context.WithTimeout(a.ctx, 15*1e9) // 15s
	defer cancel()
	return a.enhancer.ValidateKey(ctx)
}

// ListModels fetches the model list from a given base URL + API key.
func (a *App) ListModels(baseURL, apiKey string) ([]string, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if apiKey == "" && a.cfg.IsConfigured() {
		apiKey = a.cfg.GetAPIKey()
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	client := enhance.NewClientWithBase(apiKey, baseURL, "")
	ctx, cancel := context.WithTimeout(a.ctx, 15*1e9)
	defer cancel()
	return client.ListModels(ctx)
}

// EnhanceInput is the binding for the floating UI.
type EnhanceInput struct {
	Text string `json:"text"`
}

// EnhanceResult mirrors enhance.EnhanceResult for JSON binding.
type EnhanceResult struct {
	Output string `json:"output"`
	Score  int    `json:"score"`
}

// EnhanceText rewrites the given text using the configured model.
// Returns the cleaned output and a quality score.
func (a *App) EnhanceText(in EnhanceInput) (EnhanceResult, error) {
	if !a.cfg.IsConfigured() {
		return EnhanceResult{}, fmt.Errorf("not configured: open Settings and add an API key")
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.enhancer == nil {
		a.enhancer = enhance.NewClientWithBase(a.cfg.GetAPIKey(), a.cfg.GetBaseURL(), a.cfg.GetModel())
	}
	ctx, cancel := context.WithTimeout(a.ctx, 90*1e9)
	defer cancel()

	res, err := a.enhancer.Enhance(ctx, in.Text)
	if err != nil {
		return EnhanceResult{}, err
	}
	return EnhanceResult{Output: res.Output, Score: res.Score}, nil
}

// ScoreOnly returns just the quality score (used for the live badge as the
// user types into the floating bar).
func (a *App) ScoreOnly(text string) int {
	return enhance.Score(text)
}

// ReadClipboard returns the current clipboard text (for the auto-fill feature).
func (a *App) ReadClipboard() (string, error) {
	return platform.ReadClipboard()
}

// WriteClipboard sets the clipboard to the given text.
func (a *App) WriteClipboard(text string) error {
	return platform.WriteClipboard(text)
}

// GetSelection reads the currently selected text by sending Ctrl+C and
// grabbing the clipboard contents. Returns empty if nothing was selected.
func (a *App) GetSelection() (string, error) {
	return platform.GetSelection()
}

// SimulatePaste writes text to the clipboard and sends Ctrl+V.
func (a *App) SimulatePaste(text string) error {
	return platform.SimulatePaste(text)
}

// SetAutoPaste toggles whether the floating bar auto-pastes after enhance.
func (a *App) SetAutoPaste(v bool) error {
	return a.cfg.SetAutoPaste(v)
}

// SetHotkey updates the global hotkey.
func (a *App) SetHotkey(hotkey string) error {
	if err := a.cfg.SetHotkey(hotkey); err != nil {
		return err
	}
	a.installHotkey()
	return nil
}

// installHotkey (re)registers the global hotkey. Safe to call multiple times.
func (a *App) installHotkey() {
	if a.cancelHot != nil {
		a.cancelHot()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelHot = cancel
	go platform.ListenHotkey(ctx, a.cfg.Hotkey(), a.onHotkey)
}

// onHotkey is fired by platform.ListenHotkey in a goroutine.
func (a *App) onHotkey() {
	log.Println("hotkey fired")
	a.fireEnhanceFromClipboard()
}

// fireEnhanceFromClipboard reads the current selection (via Ctrl+C) and
// emits a "show-bar" event to the frontend with the result.
func (a *App) fireEnhanceFromClipboard() {
	prev, _ := platform.ReadClipboard()
	sel, err := platform.GetSelection()
	if err != nil || sel == "" {
		sel, _ = platform.ReadClipboard()
	}
	if sel == "" {
		wailsruntime.EventsEmit(a.ctx, "enhance:error", "No text selected")
		return
	}
	wailsruntime.EventsEmit(a.ctx, "enhance:start", sel)

	before := enhance.Score(sel)
	a.mu.Lock()
	if a.enhancer == nil {
		a.enhancer = enhance.NewClientWithBase(a.cfg.GetAPIKey(), a.cfg.GetBaseURL(), a.cfg.GetModel())
	}
	enh := a.enhancer
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(a.ctx, 90*1e9)
	defer cancel()
	res, err := enh.Enhance(ctx, sel)
	if err != nil {
		wailsruntime.EventsEmit(a.ctx, "enhance:error", err.Error())
		return
	}
	wailsruntime.EventsEmit(a.ctx, "enhance:done", map[string]any{
		"selection": sel,
		"output":    res.Output,
		"before":    before,
		"after":     res.Score,
		"prevClip":  prev,
	})
}

// ShowFloatingWindow positions and shows the floating bar window near the cursor.
func (a *App) ShowFloatingWindow(x, y int) {
	wailsruntime.WindowSetPosition(a.ctx, x, y)
	wailsruntime.WindowShow(a.ctx)
}

// HideFloatingWindow hides the floating bar.
func (a *App) HideFloatingWindow() {
	wailsruntime.WindowHide(a.ctx)
}

// ====== Logging ======

var logFileOnce sync.Once
var logFileHandle *os.File

func logFile() *os.File {
	logFileOnce.Do(func() {
		dir := config.DefaultDir()
		if err := os.MkdirAll(dir, 0700); err != nil {
			return
		}
		f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return
		}
		logFileHandle = f
		log.SetOutput(&syncFile{f})
	})
	return logFileHandle
}

type syncFile struct{ f *os.File }

func (s *syncFile) Write(p []byte) (int, error) {
	n, err := s.f.Write(p)
	if n > 0 {
		_ = s.f.Sync()
	}
	return n, err
}

// MarshalJSON helper to pretty-print the current config (debugging only).
func (a *App) dumpConfig() string {
	b, _ := json.MarshalIndent(a.cfg, "", "  ")
	return string(b)
}
