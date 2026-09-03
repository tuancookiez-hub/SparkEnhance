# SparkEnhance

> Hover-enhance any text in any Windows app into a numbered agent brief.
> Sits in the system tray. One hotkey. M3 does the work.

[![Track 1](https://img.shields.io/badge/MiniMax%20Week-Track%201-blueviolet)](https://www.gmicloud.ai/minimax-week)
[![Built with M3](https://img.shields.io/badge/M3-MiniMax--M3-ff6b9d)](https://www.gmicloud.ai)
[![Go 1.25](https://img.shields.io/badge/Go-1.25-00ADD8)](https://go.dev)
[![Single 6.7MB exe](https://img.shields.io/badge/binary-6.7MB-success)](build/sparkenhance.exe)

## What it does

SparkEnhance is a system-tray app for Windows that turns messy text into
a structured, agent-ready brief. It runs MiniMax-M3 hosted on GMI Cloud.

- 🖱️ Select text in **any** app — Notepad, Word, Chrome, VS Code, Slack
- ⌨️ Press **Ctrl+Shift+E** anywhere
- ✨ A small floating bar appears near the cursor
- 📋 The enhanced brief is on your clipboard
- (Optionally) auto-pastes into the original app

## What the rewrite looks like

**Input (messy draft):**
> fix the login flow

**Output (M3 brief):**
> **Goal:** Ship a working login flow that authenticates users against
> the existing `users` table and issues a session JWT.
>
> 1. Add a `POST /api/v1/login` endpoint that validates user credentials
> 2. Hash incoming passwords with bcrypt and compare against the `password_hash` column
> 3. Issue a JWT in the response when credentials are valid
> 4. Return `401` with a clear error message when credentials are wrong
> 5. Write 3 pytest cases covering happy path, bad password, missing user
> 6. Add rate limiting (5 attempts per minute per IP)
> 7. Update the OpenAPI spec with the new endpoint

## Why M3 (Track 1)

Track 1 is "agents that hold a plan, coding tools that finish the job,
research assistants that fact check themselves." SparkEnhance is a
**prompt-engineering layer** that uses M3 specifically because M3 is the
frontier-reasoning model — it's the one most likely to:

- Hold a coherent plan structure (Goal / Scope / Requirements / Gates)
- Resist adding goals the user didn't mention (the "anti-patterns" gate)
- Produce concrete, agent-receivable nouns (table names, endpoints, file paths)

Every output is forced through a `system` prompt that mandates the
production-grade brief structure. The quality score (40→78/100) reflects
how well the output conforms.

## Install

### Pre-built binary (Windows)

```bash
# Download sparkenhance.exe from the latest release.
# Double-click to launch. A console prompt asks for your GMI API key on
# first run; it is then stored in %APPDATA%\SparkEnhance\config.json.
```

### From source

```bash
git clone https://github.com/tuancookiez-hub/sparkenhance.git
cd sparkenhance
go build -ldflags "-H windowsgui" -o build/sparkenhance.exe ./cmd/sparkenhance/
./build/sparkenhance.exe
```

### Get a GMI Cloud API key

1. Create a free account at <https://console.gmicloud.ai>
2. Go to API Keys → create a new key
3. Paste it into SparkEnhance on first launch (or set `GMI_API_KEY` env)

## Architecture

```
┌─────────────────────────────────────────────────┐
│ System tray (Win32 NOTIFYICONDATAW)             │  ~1.2KB
├─────────────────────────────────────────────────┤
│ Message-only window (HWND_MESSAGE)              │
│  ├── Global hotkey: RegisterHotKey(Ctrl+Shift+E)│
│  ├── wndProc: WM_HOTKEY → triggerEnhance()      │
│  └── WM_USER+1: tray menu (Quit)                │
├─────────────────────────────────────────────────┤
│ Floating bar (WS_EX_NOACTIVATE | TOOLWINDOW)    │  360×80px
│  └── Renders: idle / working / done / error     │
├─────────────────────────────────────────────────┤
│ Clipboard:                                      │
│  SimulateCtrlC → Read selection → Write enhance │
│  → Simulate Ctrl+V → Restore original           │
├─────────────────────────────────────────────────┤
│ Enhance client                                  │
│  POST https://api.gmi-serving.com/v1/...        │
│  model = MiniMax-M3                             │
└─────────────────────────────────────────────────┘
```

| Component | LOC | Notes |
|---|---|---|
| `internal/enhance` | ~250 | GMI client + ported prompt + cleaner + scorer |
| `internal/clipboard` | ~120 | Win32 Unicode clipboard + keybd_event |
| `internal/hotkey` | ~95  | chord parser for `RegisterHotKey` |
| `internal/tray` | ~140 | Win32 NOTIFYICONDATAW + popup menu |
| `internal/bar` | ~260  | borderless WS_EX_NOACTIVATE floating bar |
| `internal/win` | ~200  | event loop, wndProc, glue |
| `cmd/sparkenhance` | ~85 | entry, config, single-instance |
| **Total** | **~1150** | **+ tests, 6.7MB binary, no runtime** |

## Tests

```bash
go test ./...
# ok  github.com/tuancookiez-hub/sparkenhance/internal/enhance
#    TestCleanStripsThinkBlock, TestCleanStripsFences, TestCleanStripsQuotes,
#    TestUserMessageFormat, TestCleanEmpty,
#    TestScoreEmpty, TestScoreBlankLines, TestScoreBaseline, TestScoreStrongPrompt,
#    TestScoreSpecifics, TestScoreMax100,
#    TestEnhanceRoundTrip (mocked GMI server),
#    TestValidateKey
```

## How it compares to the Desktop plugin

This is the **standalone, OS-level** version of the [Hermes Enhance
Prompt](https://github.com/tuancookiez-hub/hermes-enhance-prompt) plugin
that lives inside Hermes Desktop. The rewrite logic — system prompt,
cleaner, scorer — is **ported 1:1** from the plugin's `prompts.py`. The
**delivery surface** is the new thing:

| | Plugin (Hermes Desktop) | SparkEnhance (this) |
|---|---|---|
| Where it runs | Inside Hermes Desktop | Standalone, OS tray |
| Where it works | Hermes composer | Any text in any app |
| Activation | Click sparkle | Ctrl+Shift+E |
| Output | Replaces composer draft | Clipboard + auto-paste |
| Revert | Click discard icon | n/a (selection is replaced) |
| Binary | Bundled with Desktop | 6.7MB single .exe |
| Language | JavaScript + Python | Go |

## Hackathon submission details

- **Track:** 1 (Reasoning)
- **Models used:** MiniMax-M3
- **GMI endpoint:** `https://api.gmi-serving.com/v1/chat/completions`
- **Demo video:** <link>
- **Public repo:** <link>

## License

MIT — see [LICENSE](LICENSE).
