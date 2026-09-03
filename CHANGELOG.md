# Changelog

## v0.1.0 — 2026-09-03

Initial submission for MiniMax Week, Track 1.

- Tray-resident Windows app, 6.7MB single `.exe`, no runtime
- Global hotkey Ctrl+Shift+E triggers enhance flow
- Floating bar with idle/working/done/error states
- Auto-paste: writes to clipboard, simulates Ctrl+V, restores original
- MiniMax-M3 via GMI Cloud OpenAI-compatible endpoint
- System prompt + cleaner + scorer ported 1:1 from
  [hermes-enhance-prompt](https://github.com/tuancookiez-hub/hermes-enhance-prompt)
- 13 unit tests pass (cleaner, scorer, full enhance round-trip)
- First-run config at `%APPDATA%\SparkEnhance\config.json`
- Single-instance lock via named mutex
