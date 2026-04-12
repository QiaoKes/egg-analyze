package hotkey

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func TestFormatShortcut(t *testing.T) {
	value, err := FormatShortcut(fyne.KeyModifierControl|fyne.KeyModifierShift, fyne.KeyQ)
	if err != nil {
		t.Fatalf("FormatShortcut returned error: %v", err)
	}
	if value != "Ctrl+Shift+Q" {
		t.Fatalf("FormatShortcut returned %q", value)
	}
}

func TestFormatShortcutRejectsModifierOnly(t *testing.T) {
	_, err := FormatShortcut(fyne.KeyModifierControl, desktop.KeyControlLeft)
	if err == nil {
		t.Fatal("FormatShortcut should reject modifier-only shortcuts")
	}
}

func TestNormalizeShortcut(t *testing.T) {
	got := NormalizeShortcut("control + alt + escape")
	if got != "Ctrl+Alt+Esc" {
		t.Fatalf("NormalizeShortcut returned %q", got)
	}
}
