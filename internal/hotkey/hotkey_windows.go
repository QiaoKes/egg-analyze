//go:build windows

package hotkey

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	modAlt     = 0x0001
	modControl = 0x0002
	modShift   = 0x0004
	modWin     = 0x0008
	wmHotKey   = 0x0312
	wmQuit     = 0x0012
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessage         = user32.NewProc("GetMessageW")
	procPostThreadMessage  = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")
)

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type Manager struct {
	mu       sync.Mutex
	threadID uint32
	done     chan struct{}
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Register(hotkey string, callback func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.done != nil {
		m.stopLocked()
	}

	modifiers, key, err := parse(hotkey)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		threadID, _, _ := procGetCurrentThreadID.Call()
		m.mu.Lock()
		m.threadID = uint32(threadID)
		m.mu.Unlock()

		ret, _, callErr := procRegisterHotKey.Call(0, 1, uintptr(modifiers), uintptr(key))
		if ret == 0 {
			ready <- fmt.Errorf("register hotkey failed: %v", callErr)
			close(done)
			return
		}
		ready <- nil

		var message msg
		for {
			result, _, _ := procGetMessage.Call(uintptr(unsafePointer(&message)), 0, 0, 0)
			switch int32(result) {
			case -1, 0:
				close(done)
				return
			}

			if message.Message == wmHotKey {
				callback()
			}
			if message.Message == wmQuit {
				close(done)
				return
			}
		}
	}()

	if err := <-ready; err != nil {
		return err
	}
	m.done = done
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

func (m *Manager) stopLocked() {
	if m.done == nil {
		return
	}
	procUnregisterHotKey.Call(0, 1)
	if m.threadID != 0 {
		procPostThreadMessage.Call(uintptr(m.threadID), wmQuit, 0, 0)
	}
	<-m.done
	m.done = nil
	m.threadID = 0
}

func parse(input string) (uint, uint, error) {
	parts := strings.Split(strings.TrimSpace(input), "+")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("hotkey format should look like Ctrl+Shift+Q")
	}

	var modifiers uint
	var key uint
	for _, raw := range parts {
		part := strings.ToUpper(strings.TrimSpace(raw))
		switch part {
		case "CTRL", "CONTROL":
			modifiers |= modControl
		case "SHIFT":
			modifiers |= modShift
		case "ALT":
			modifiers |= modAlt
		case "WIN", "WINDOWS":
			modifiers |= modWin
		default:
			if key != 0 {
				return 0, 0, fmt.Errorf("multiple non-modifier keys in hotkey")
			}
			parsedKey, err := parseKey(part)
			if err != nil {
				return 0, 0, err
			}
			key = parsedKey
		}
	}
	if modifiers == 0 || key == 0 {
		return 0, 0, fmt.Errorf("hotkey must include at least one modifier and one key")
	}
	return modifiers, key, nil
}

func parseKey(input string) (uint, error) {
	if len(input) == 1 {
		switch {
		case input[0] >= 'A' && input[0] <= 'Z':
			return uint(input[0]), nil
		case input[0] >= '0' && input[0] <= '9':
			return uint(input[0]), nil
		}
	}
	if strings.HasPrefix(input, "F") {
		index, err := strconv.Atoi(strings.TrimPrefix(input, "F"))
		if err == nil && index >= 1 && index <= 24 {
			return uint(0x6F + index), nil
		}
	}
	switch input {
	case "SPACE":
		return 0x20, nil
	case "TAB":
		return 0x09, nil
	case "ESC", "ESCAPE":
		return 0x1B, nil
	case "ENTER":
		return 0x0D, nil
	}
	return 0, fmt.Errorf("unsupported hotkey key %q", input)
}

func unsafePointer(value *msg) uintptr {
	return uintptr(unsafe.Pointer(value))
}
