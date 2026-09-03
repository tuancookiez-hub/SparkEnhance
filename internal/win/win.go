// Package win hosts the Windows event loop for SparkEnhance.
// It runs a message-only window with: tray icon, global hotkey, and a
// floating bar that shows enhance results near the cursor.
package win

import (
	"context"
	"log"
	"syscall"
	"time"
	"unsafe"

	"github.com/tuancookiez-hub/sparkenhance/internal/bar"
	"github.com/tuancookiez-hub/sparkenhance/internal/clipboard"
	"github.com/tuancookiez-hub/sparkenhance/internal/config"
	"github.com/tuancookiez-hub/sparkenhance/internal/enhance"
	"github.com/tuancookiez-hub/sparkenhance/internal/hotkey"
	"github.com/tuancookiez-hub/sparkenhance/internal/tray"
)

const (
	wndClass     = "SparkEnhanceMsgWindow"
	trayCallback = 0x0401 // WM_USER + 1
	hotkeyID     = 0x0001
)

var (
	appCtx    context.Context
	appCfg    *config.Config
	barWindow *bar.Window
	trayIcon  *tray.Icon
)

// Run is the entry point. It blocks until ctx is cancelled.
func Run(ctx context.Context, cfg *config.Config, iconData []byte) error {
	appCtx = ctx
	appCfg = cfg

	if err := acquireSingleInstance(); err != nil {
		log.Printf("single-instance check: %v (continuing)", err)
	}

	hmod, vk, err := hotkey.Parse(cfg.Hotkey)
	if err != nil {
		log.Printf("invalid hotkey %q: %v (using default ctrl+shift+e)", cfg.Hotkey, err)
		hmod, vk, _ = hotkey.Parse("ctrl+shift+e")
	}

	className := mustUTF16(wndClass)
	windowName := mustUTF16("SparkEnhance")
	hInst, _, _ := procGetModuleHandleW.Call(0)

	cls := wndclassexw{
		CbSize:        uint32(unsafe.Sizeof(wndclassexw{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInst,
		LpszClassName: className,
	}
	_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&cls)))

	hwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)),
		0, 0, 0, 0, 0, 0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return syscall.GetLastError()
	}
	log.Printf("message window: hwnd=%v", hwnd)

	if ret, _, _ := procRegisterHotKey.Call(hwnd, hotkeyID, uintptr(hmod), uintptr(vk)); ret == 0 {
		log.Printf("RegisterHotKey failed for %s", cfg.Hotkey)
	} else {
		log.Printf("hotkey %s registered", cfg.Hotkey)
	}

	ti, err := tray.New(hwnd, trayCallback, iconData, cfg, exitFn)
	if err != nil {
		log.Printf("tray init failed: %v", err)
	} else {
		trayIcon = ti
		defer trayIcon.Remove()
	}

	bw, err := bar.New(cfg.AutoPaste)
	if err != nil {
		log.Printf("bar init failed: %v", err)
	} else {
		barWindow = bw
		defer barWindow.Destroy()
	}

	go func() {
		<-ctx.Done()
		log.Println("ctx cancelled, posting WM_QUIT")
		_, _, _ = procPostQuitMessage.Call(0)
	}()

	runMessageLoop()
	log.Println("message loop exited")
	return nil
}

func exitFn() { _, _, _ = procPostQuitMessage.Call(0) }

// wndProc handles messages for the message-only window.
func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case 0x0312: // WM_HOTKEY
		log.Printf("hotkey fired")
		go triggerEnhance()
	case trayCallback:
		switch lParam {
		case 0x0205: // WM_RBUTTONUP
			if trayIcon != nil {
				trayIcon.ShowMenu()
			}
		case 0x0202: // WM_LBUTTONUP
			go triggerEnhance()
		}
	case 0x0011: // WM_COMMAND
		if trayIcon != nil {
			trayIcon.HandleCommand(uint32(wParam & 0xFFFF))
		}
	}
	return 0
}

// triggerEnhance reads the current selection (via Ctrl+C), calls M3, shows
// the result in the floating bar.
func triggerEnhance() {
	if barWindow == nil {
		log.Printf("bar not initialized")
		return
	}
	prev, _ := clipboard.Read()
	if !clipboard.SimulateCtrlC() {
		barWindow.ShowError("Failed to copy selection")
		return
	}
	time.Sleep(150 * time.Millisecond)
	selection, err := clipboard.Read()
	if err != nil || selection == "" {
		barWindow.ShowError("No text selected. Highlight text, then try again.")
		if prev != "" {
			_ = clipboard.Write(prev)
		}
		return
	}
	if len(selection) < 8 {
		barWindow.ShowError("Selection too short. Highlight at least a few words.")
		if prev != "" {
			_ = clipboard.Write(prev)
		}
		return
	}
	log.Printf("selection: %d bytes", len(selection))

	x, y := currentCursorPos()
	barWindow.Show(x, y, bar.StateEnhancing)
	barWindow.SetSelection(selection, enhance.Score(selection))

	client := enhance.NewClient(appCfg.GetAPIKey())
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result, err := client.Enhance(ctx, selection)
	if err != nil {
		log.Printf("enhance error: %v", err)
		barWindow.ShowError("Enhance failed: " + err.Error())
		return
	}
	barWindow.ShowResult(selection, result.Output, result.Score, prev, appCfg.AutoPaste)
}
