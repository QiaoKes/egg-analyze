package hotkey

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

var modifierNames = []struct {
	mask fyne.KeyModifier
	name string
}{
	{mask: fyne.KeyModifierControl, name: "Ctrl"},
	{mask: fyne.KeyModifierShift, name: "Shift"},
	{mask: fyne.KeyModifierAlt, name: "Alt"},
	{mask: fyne.KeyModifierSuper, name: "Win"},
}

func IsModifierKey(key fyne.KeyName) bool {
	switch key {
	case desktop.KeyControlLeft, desktop.KeyControlRight,
		desktop.KeyShiftLeft, desktop.KeyShiftRight,
		desktop.KeyAltLeft, desktop.KeyAltRight,
		desktop.KeySuperLeft, desktop.KeySuperRight:
		return true
	default:
		return false
	}
}

func ModifierFromKey(key fyne.KeyName) (fyne.KeyModifier, bool) {
	switch key {
	case desktop.KeyControlLeft, desktop.KeyControlRight:
		return fyne.KeyModifierControl, true
	case desktop.KeyShiftLeft, desktop.KeyShiftRight:
		return fyne.KeyModifierShift, true
	case desktop.KeyAltLeft, desktop.KeyAltRight:
		return fyne.KeyModifierAlt, true
	case desktop.KeySuperLeft, desktop.KeySuperRight:
		return fyne.KeyModifierSuper, true
	default:
		return 0, false
	}
}

func FormatShortcut(modifiers fyne.KeyModifier, key fyne.KeyName) (string, error) {
	if modifiers == 0 {
		return "", fmt.Errorf("hotkey must include at least one modifier")
	}
	if key == "" || IsModifierKey(key) {
		return "", fmt.Errorf("hotkey must include a non-modifier key")
	}

	keyName, ok := formatKeyName(key)
	if !ok {
		return "", fmt.Errorf("unsupported hotkey key %q", key)
	}

	parts := make([]string, 0, len(modifierNames)+1)
	for _, item := range modifierNames {
		if modifiers&item.mask != 0 {
			parts = append(parts, item.name)
		}
	}
	parts = append(parts, keyName)
	return strings.Join(parts, "+"), nil
}

func NormalizeShortcut(input string) string {
	parts := strings.Split(strings.TrimSpace(input), "+")
	if len(parts) == 0 {
		return ""
	}

	normalized := make([]string, 0, len(parts))
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}

		switch strings.ToUpper(part) {
		case "CTRL", "CONTROL":
			normalized = append(normalized, "Ctrl")
		case "SHIFT":
			normalized = append(normalized, "Shift")
		case "ALT":
			normalized = append(normalized, "Alt")
		case "WIN", "WINDOWS", "SUPER", "COMMAND", "CMD":
			normalized = append(normalized, "Win")
		case "ESCAPE", "ESC":
			normalized = append(normalized, "Esc")
		case "SPACE":
			normalized = append(normalized, "Space")
		case "TAB":
			normalized = append(normalized, "Tab")
		case "ENTER", "RETURN":
			normalized = append(normalized, "Enter")
		default:
			upper := strings.ToUpper(part)
			if len(upper) == 1 {
				normalized = append(normalized, upper)
				continue
			}
			if strings.HasPrefix(upper, "F") {
				normalized = append(normalized, upper)
				continue
			}
			normalized = append(normalized, part)
		}
	}
	return strings.Join(normalized, "+")
}

func formatKeyName(key fyne.KeyName) (string, bool) {
	switch key {
	case fyne.KeySpace:
		return "Space", true
	case fyne.KeyTab:
		return "Tab", true
	case fyne.KeyEscape:
		return "Esc", true
	case fyne.KeyReturn, fyne.KeyEnter:
		return "Enter", true
	}

	name := string(key)
	if len(name) == 1 {
		return strings.ToUpper(name), true
	}
	if strings.HasPrefix(name, "F") {
		return strings.ToUpper(name), true
	}
	return "", false
}
