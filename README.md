# SparkEnhance

![SparkEnhance hero banner](assets/hero-banner.jpg)

> Hover-enhance any text in any Windows app into a numbered agent brief.
> Tray app. **Ctrl+Shift+E**. MiniMax-M3 via GMI Cloud.

![Icon](build/appicon.png)

## What it is

A standalone Windows app that turns messy text into a structured, agent-ready
brief using **MiniMax-M3** on GMI Cloud. Sits in the system tray, listens for
the global hotkey, reads the current selection, and writes the enhanced
version back to the clipboard.

## Demo (3-min)

> Built with **Wails v3** (Go 1.23 + WebView2). Backend in Go, UI in React/TypeScript.

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

## What changed in v0.5 (Wails v3 rewrite)

- **True pill bar.** Win32 `SetWindowRgn` clips the window to a 15px
  rounded rectangle, so the OS only composites a pill — no rectangular
  DWM border around the CSS-drawn curve.
- **No grey flash.** `BackgroundTypeSolid` with the pill color pre-painted
  by WebView2 from t=0.
- **No secrets in source.** API key loaded from
  `%APPDATA%\SparkEnhance\config.json` (gitignored) or the
  `MINIMAX_API_KEY` env var.
- **Cleaner internals.** Wails v3 with React 18 + Vite + TypeScript on
  the frontend, native Win32 syscalls for hotkey/clipboard/window-clip.

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
git clone https://github.com/tuancookiez-hub/SparkEnhance.git
cd SparkEnhance
go install github.com/wailsapp/wails/v3/cmd/wails@latest
wails build -platform windows/amd64
# Output: build/bin/sparkenhance.exe
```

The frontend builds automatically as part of `wails build`. To build only
the frontend during development:

```bash
cd frontend && npm install && npm run build
```

### Get a GMI Cloud API key

1. Create a free account at <https://console.gmicloud.ai>
2. Go to **API Keys** → create a new key
3. Paste it into SparkEnhance on first launch (or set `MINIMAX_API_KEY`)
4. Pick the model: **MiniMax-M3** (default), or any other model on the
   GMI base URL

## Configuration

The app reads from `%APPDATA%\SparkEnhance\config.json` on Windows
(`~/.config/SparkEnhance/config.json` on Linux/macOS):

```json
{
  "apiKey": "sk-…",
  "baseURL": "https://api.gmi-serving.com/v1",
  "model": "MiniMaxAI/MiniMax-M3",
  "hotkey": "ctrl+shift+e"
}
```

The `apiKey` field is gitignored — never commit it. Use the `MINIMAX_API_KEY`
environment variable in CI or shared environments instead.

## Architecture

```
┌──────────────────────────────────────────────────┐
│  Floating bar — 600×56 WebView2 window             │
│  - Opaque pill bg: #181a26                        │
│  - Win32 SetWindowRgn clips to 15px rounded rect  │
│  - States: idle / loading / result / error        │
│  - Auto-hide after 30s of inactivity              │
└────────────────┬─────────────────────────────────┘
                 │ wails events (TypeScript ⇄ Go)
┌────────────────▼─────────────────────────────────┐
│  Go backend (main.go)                            │
│  - runEnhance:   read selection → call GMI → emit│
│  - showBar/hideBar: position + clip + show       │
│  - Hotkey:       global Ctrl+Shift+E             │
│  - Auto-hide loop: 30s timer, reset on activity   │
└────────────────┬─────────────────────────────────┘
                 │
┌────────────────▼─────────────────────────────────┐
│  internal/ (no secrets stored)                   │
│  - config:     load/save %APPDATA%\…\config.json │
│  - enhance:    MiniMax-M3 chat-completions call  │
│  - platform:   Win32 SetWindowRgn, clipboard,    │
│                monitor detection, MoveWindowPos  │
│  - placement:  bar x/y math                      │
└──────────────────────────────────────────────────┘
```

## Files

| Path | Purpose |
|---|---|
| `main.go`              | Wails v3 bootstrap, hotkey, bar lifecycle |
| `configpath.go`        | Cross-platform config dir resolver |
| `internal/config/`     | JSON config at `%APPDATA%\SparkEnhance\config.json` |
| `internal/enhance/`    | MiniMax-M3 chat-completions client + system prompt |
| `internal/platform/`   | Win32 SetWindowRgn, clipboard, monitor detection |
| `internal/placement/`  | Bar position math (centred on monitor, top of screen) |
| `frontend/src/App.tsx` | React root — bar screen + settings screen |
| `frontend/src/main.tsx` | Router — /settings vs / |
| `frontend/src/style.css` | Pill styling (matches Win32 region) |
| `build/windows/icon.ico` | 6-resolution app icon |

## Hotkey

Default: **Ctrl+Shift+E**. To change it, edit `config.json` and restart the app.

## Security & privacy

- **API key** is stored locally on disk and never leaves the machine except
  when sent to the configured `baseURL` as a Bearer token.
- **Selection text** is sent to the configured `baseURL` only when the hotkey
  fires — never logged, never persisted.
- **No telemetry, no analytics, no phone-home.**
- **`config.json` is gitignored** — make sure it stays that way if you fork.

## Tests

```bash
go test ./...
# internal/enhance:  cleaner, scorer, mocked GMI round-trip
# internal/platform: window region, monitor detection
# internal/placement: bar position bounds
```

## Compared to the Desktop plugin

| | Plugin (Hermes Desktop) | SparkEnhance (this) |
|---|---|---|
| Where it runs | Inside Hermes Desktop | Standalone Windows app |
| Where it works | Hermes composer | Any text in any app |
| Activation | Click sparkle | Ctrl+Shift+E (global) |
| Output | Replaces composer draft | Clipboard + auto-paste |
| Binary | Bundled with Desktop | 14 MB single .exe |
| UI tech | React (WebView2) inside Hermes | React (WebView2) standalone |

The rewrite logic — system prompt + cleaner — is **ported 1:1** from the
plugin's `prompts.py`.

## Hackathon submission

- **Track:** 1 (Reasoning)
- **Models used:** MiniMax-M3 (and 3.5-Speculative)
- **GMI endpoint:** `https://api.gmi-serving.com/v1/chat/completions`
- **Public repo:** <https://github.com/tuancookiez-hub/SparkEnhance>

## License

MIT — see [LICENSE](LICENSE).
