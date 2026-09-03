package bar

// pasteEnhanced is implemented in bar_paste_windows.go using the clipboard
// + keybd_event from the win package's syscall imports.
func pasteEnhanced(text, prev string) error {
	return pasteEnhancedWindows(text, prev)
}
