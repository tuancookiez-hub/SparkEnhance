# SparkEnhance

> Hover-enhance any text in any Windows app into a numbered agent brief.
> Tray app. **Ctrl+Shift+E**. MiniMax-M3 via GMI Cloud.

![Icon](build/appicon.png)

## What it is

A standalone Windows app that turns messy text into a structured, agent-ready
brief using MiniMax-M3 on GMI Cloud. Sits in the system tray, listens for
the global hotkey, reads the current selection, and writes the enhanced
version back to the clipboard.

## Demo (3-min)

> Built with **Wails v2** (Go + WebView2). Backend in Go, UI in React/TypeScript.

## What the rewrite looks like

**Input (messy draft):**
> fix the login flow

**Output (M3 brief):**
> **Goal:** Ship a working login flow that authenticates users against
> the existing `users` table and issues a session JWT.
>
> 1. Add a `POST /api/v1/login` endpoint that validates user credentials
> 2. Hash incoming passwords with bcrypt and compare against the
>    `password_hash` column
> 3. Issue a JWT in the response when credentials are valid
> 4. Return `401` with a clear error message when credentials are wrong
> 5. Write 3 pytest cases covering happy path, bad password, missing user
> 6. Add rate limiting (5 attempts per minute per IP)
> 7. Update the OpenAPI spec with the new endpoint and example requests

## Install

### Pre-built binary (Windows 10/11 x64)

```bash
# Download sparkenhance.exe from the latest release.
# Double-click to launch. The setup window opens on first run.
```

> SparkEnhance requires **WebView2** which is preinstalled on Windows 10
> (since 2021) and Windows 11.

### From source

```bash
git clone https://github.com/tuancookiez-hub/sparkenhance.git
cd sparkenhance
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build -platform windows/amd64
# Output: build/bin/sparkenhance.exe
```

### Get a GMI Cloud API key

1. Create a free account at <https://console.gmicloud.ai>
2. Go to API Keys → create a new key
3. Paste it into SparkEnhance on first launch
4. Pick the model: **MiniMax-M3** (default), **MiniMax-M3.5-Speculative**, or
   any other model on the GMI base URL

## Architecture

```
┌──────────────────────────────────────────────────┐
│  WebView2 window (560×480)                       │
│  - SetupView: API key + model + base URL + hotkey│
│  - DashboardView: floating bar + history        │
│  - SettingsView: edit config, validation         │
└────────────────┬─────────────────────────────────┘
                 │ wails bindings (TypeScript ⇄ Go)
┌────────────────▼─────────────────────────────────┐
│  Go backend (app.go)                             │
│  - EnhanceText:   call GMI /v1/chat/completions  │
│  - GetConfig:     load %APPDATA%\SparkEnhance\…  │
│  - SaveSetup:     persist + reinstall hotkey     │
│  - ShowFloating:  position + show WebView2       │
└────────────────┬─────────────────────────────────┘
                 │
┌────────────────▼─────────────────────────────────┐
│  internal/platform (Win32)                      │
│  - RegisterHotKey: global Ctrl+Shift+E           │
│  - Clipboard:      read, write, Ctrl+C/V sim    │
│  - Shell_NotifyIcon: system tray + menu         │
└──────────────────────────────────────────────────┘
```

## Files

| Path | Purpose |
|---|---|
| `app.go`               | Wails-bound methods callable from React |
| `main.go`              | Wails bootstrap, embeds frontend/dist |
| `frontend/src/App.tsx` | Root React component, event handling |
| `frontend/src/views/SetupView.tsx`     | First-run setup form |
| `frontend/src/views/DashboardView.tsx` | Main enhance + settings UI |
| `internal/enhance/`    | GMI Cloud client + score heuristic + cleaner |
| `internal/config/`     | JSON-on-disk config at `%APPDATA%\SparkEnhance\config.json` |
| `internal/platform/`   | Win32 hotkey, clipboard, tray |
| `frontend/wailsjs/`    | Auto-generated TS bindings |
| `build/windows/icon.ico` | 6-resolution app icon |

## Tests

```bash
go test ./...
# ok  github.com/tuancookiez-hub/sparkenhance/internal/enhance
#    13 passing (cleaner, scorer, mocked GMI round-trip, key validation)
```

## Compared to the Desktop plugin

| | Plugin (Hermes Desktop) | SparkEnhance (this) |
|---|---|---|
| Where it runs | Inside Hermes Desktop | Standalone Windows app |
| Where it works | Hermes composer | Any text in any app |
| Activation | Click sparkle | Ctrl+Shift+E (global) |
| Output | Replaces composer draft | Clipboard + auto-paste |
| Binary | Bundled with Desktop | 11.2 MB single .exe |
| UI tech | React (WebView2) inside Hermes | React (WebView2) standalone |

The rewrite logic — system prompt, cleaner, scorer — is **ported 1:1** from
the plugin's `prompts.py` and `score.js`.

## Hackathon submission

- **Track:** 1 (Reasoning)
- **Models used:** MiniMax-M3 (and 3.5-Speculative)
- **GMI endpoint:** `https://api.gmi-serving.com/v1/chat/completions`
- **Public repo:** <https://github.com/tuancookiez-hub/sparkenhance>

## License

MIT — see [LICENSE](LICENSE).
