# Changelog

## v0.3.0 — 2026-09-03

**Major rewrite: Wails v2 + React UI.**

- Migrated from raw Win32 to **Wails v2** (Go backend + WebView2 UI)
- React/TypeScript frontend with **Handy-inspired dark theme**
- SetupView: first-run form for GMI base URL, API key, model, hotkey, auto-paste
- DashboardView: floating enhance bar with live quality score badge
- SettingsView: edit base URL / model / hotkey without restart
- Internal Wails events: `enhance:start`, `enhance:done`, `enhance:error`
- Configurable model: MiniMax-M3, MiniMax-M3.5-Speculative, or any
  GMI-supported model via the base URL
- System tray with Enhance / Settings / Quit menu
- 6-resolution app icon (16/24/32/48/64/128/256)
- All Win32 packages deleted (win/, bar/, tray/, setup/, startup/, etc.)
- 13 enhance unit tests still pass

## v0.2.x — earlier

Raw Win32 attempt (build/sparkenhance_console.exe worked but the
floating window failed to display due to syscall quirks). Archived in
git history.
