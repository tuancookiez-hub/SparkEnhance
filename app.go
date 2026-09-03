// Package main is the SparkEnhance Wails backend.
//
// Architecture:
//   - app.go:        Wails-bound methods + global hotkey dispatch
//   - enhance/:      GMI Cloud /v1/chat/completions client
//   - config/:       JSON-on-disk settings
//   - platform/:     Win32 hook + clipboard + tray (best-effort)
//
// The Wails window is the floating bar. Closing the X button hides
// instead of quits; the app stays alive in the background and keeps
// listening for the global hotkey.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/tuancookiez-hub/sparkenhance/internal/config"
	"github.com/tuancookiez-hub/sparkenhance/internal/enhance"
	"github.com/tuancookiez-hub/sparkenhance/internal/platform"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx       context.Context
	cfg       *config.Config
	enhancer  *enhance.Client
	mu        sync.Mutex
	started   bool
	hotkeyCh  chan struct{}
	cancelHot context.CancelFunc

	// remember the last foreground HWND so we can restore focus after dismiss
	prevFocusHWND uintptr
}

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

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.SetOutput(logFile())
	log.Println("SparkEnhance starting (Wails backend)")

	// Try to install the system tray (best-effort; may fail in some WebView2 setups).
	trayCtx, _ := context.WithCancel(ctx)
	go platform.RunTray(trayCtx, a.onTrayEnhance, a.onTrayQuit, a.onTraySettings)

	// Install the global hotkey.
	a.installHotkey()

	// Hide the Wails window on first launch — app lives in the tray until the hotkey fires.
	go func() {
		// Give Wails a moment to finish its first paint.
		time.Sleep(800 * time.Millisecond)
		wailsruntime.WindowHide(a.ctx)
	}()

	a.started = true
	log.Println("SparkEnhance ready (window hidden, hotkey listening)")
}

func (a *App) shutdown(ctx context.Context) {
	if a.cancelHot != nil {
		a.cancelHot()
	}
	log.Println("SparkEnhance shutting down")
}

func (a *App) onTrayEnhance() {
	a.fireEnhanceFromClipboard()
}

func (a *App) onTraySettings() {
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
}

func (a *App) onTrayQuit() {
	wailsruntime.Quit(a.ctx)
}

// ====== Bound methods (callable from React) ======

func (a *App) IsConfigured() bool { return a.cfg.IsConfigured() }

type SetupInput struct {
	APIKey    string `json:"apiKey"`
	BaseURL   string `json:"baseUrl"`
	Model     string `json:"model"`
	Hotkey    string `json:"hotkey"`
	AutoPaste bool   `json:"autoPaste"`
}

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

func (a *App) GetConfig() map[string]any {
	return map[string]any{
		"baseUrl":   a.cfg.GetBaseURL(),
		"model":     a.cfg.GetModel(),
		"hotkey":    a.cfg.Hotkey(),
		"autoPaste": a.cfg.AutoPaste(),
		"hasKey":    a.cfg.IsConfigured(),
	}
}

func (a *App) ValidateKey() error {
	if !a.cfg.IsConfigured() {
		return fmt.Errorf("no API key configured")
	}
	a.enhancer = enhance.NewClientWithBase(a.cfg.GetAPIKey(), a.cfg.GetBaseURL(), a.cfg.GetModel())
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	return a.enhancer.ValidateKey(ctx)
}

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
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	return client.ListModels(ctx)
}

type EnhanceInput struct {
	Text string `json:"text"`
}

type EnhanceResult struct {
	Output string `json:"output"`
	Score  int    `json:"score"`
}

func (a *App) EnhanceText(in EnhanceInput) (EnhanceResult, error) {
	if !a.cfg.IsConfigured() {
		return EnhanceResult{}, fmt.Errorf("not configured: open Settings and add an API key")
	}
	a.mu.Lock()
	if a.enhancer == nil {
		a.enhancer = enhance.NewClientWithBase(a.cfg.GetAPIKey(), a.cfg.GetBaseURL(), a.cfg.GetModel())
	}
	enh := a.enhancer
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(a.ctx, 90*time.Second)
	defer cancel()
	res, err := enh.Enhance(ctx, in.Text)
	if err != nil {
		return EnhanceResult{}, err
	}
	return EnhanceResult{Output: res.Output, Score: res.Score}, nil
}

func (a *App) ScoreOnly(text string) int { return enhance.Score(text) }

func (a *App) ReadClipboard() (string, error) { return platform.ReadClipboard() }
func (a *App) WriteClipboard(text string) error { return platform.WriteClipboard(text) }
func (a *App) GetSelection() (string, error)    { return platform.GetSelection() }
func (a *App) SimulatePaste(text string) error  { return platform.SimulatePaste(text) }

func (a *App) SetAutoPaste(v bool) error { return a.cfg.SetAutoPaste(v) }

func (a *App) SetHotkey(hotkey string) error {
	if err := a.cfg.SetHotkey(hotkey); err != nil {
		return err
	}
	a.installHotkey()
	return nil
}

// installHotkey (re)registers the global hotkey via WH_KEYBOARD_LL.
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

// fireEnhanceFromClipboard is the main hotkey flow:
//  1. remember foreground window
//  2. Ctrl+C to capture selection
//  3. read clipboard
//  4. show the floating bar
//  5. call M3
//  6. emit result event to the React UI
func (a *App) fireEnhanceFromClipboard() {
	// Remember which app was in front so we can restore focus on dismiss.
	a.prevFocusHWND = getForegroundWindow()

	// Get cursor position for the floating window placement.
	cx, cy := getCursorPos()

	// Resize the Wails window to bar dimensions and position near cursor.
	wailsruntime.WindowSetSize(a.ctx, 440, 200)
	wailsruntime.WindowSetPosition(a.ctx, cx, cy+20)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)

	// Grab the selection (Ctrl+C into clipboard).
	sel, err := platform.GetSelection()
	if err != nil || sel == "" {
		wailsruntime.EventsEmit(a.ctx, "enhance:error", "No text selected. Highlight some text, then try again.")
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

	ctx, cancel := context.WithTimeout(a.ctx, 90*time.Second)
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
	})
}

// DismissBar hides the bar and restores focus to the previous app.
func (a *App) DismissBar() {
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, false)
	wailsruntime.WindowHide(a.ctx)
	if a.prevFocusHWND != 0 {
		setForegroundWindow(a.prevFocusHWND)
		a.prevFocusHWND = 0
	}
}

// CopyOutput copies the enhanced text to the clipboard and (if enabled)
// auto-pastes it at the original selection site, then dismisses the bar.
func (a *App) CopyOutput(text string) {
	if err := platform.WriteClipboard(text); err != nil {
		wailsruntime.EventsEmit(a.ctx, "enhance:error", "Copy failed: "+err.Error())
		return
	}
	if a.cfg.AutoPaste() {
		// Restore focus to the source app, then send Ctrl+V.
		if a.prevFocusHWND != 0 {
			setForegroundWindow(a.prevFocusHWND)
			time.Sleep(80 * time.Millisecond)
		}
		_ = platform.SimulatePaste(text)
	}
	a.DismissBar()
}

// QuitApp is called from the UI to fully exit.
func (a *App) QuitApp() {
	wailsruntime.Quit(a.ctx)
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

// MarshalJSON helper (debugging only).
func (a *App) dumpConfig() string {
	b, _ := json.MarshalIndent(a.cfg, "", "  ")
	return string(b)
}

// ====== Win32 helpers (no extra deps) ======

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	user32           = syscall.NewLazyDLL("user32.dll")
	procGetCursorPos = user32.NewProc("GetCursorPos")
	procGetForeground = user32.NewProc("GetForegroundWindow")
	procSetForeground = user32.NewProc("SetForegroundWindow")
	procSleep        = kernel32.NewProc("Sleep")
)

func getCursorPos() (int, int) {
	type pt struct{ X, Y int32 }
	var p pt
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 100, 100
	}
	return int(p.X), int(p.Y)
}

func getForegroundWindow() uintptr {
	h, _, _ := procGetForeground.Call()
	return h
}

func setForegroundWindow(h uintptr) {
	if h == 0 {
		return
	}
	_, _, _ = procSetForeground.Call(h)
}

func timeSleepMillis(ms int) {
	if ms <= 0 {
		return
	}
	_, _, _ = procSleep.Call(uintptr(ms))
}

// avoid "imported and not used" if we ever drop one of these
var _ = time.Second
