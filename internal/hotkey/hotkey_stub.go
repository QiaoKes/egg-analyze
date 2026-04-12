//go:build !windows

package hotkey

import "fmt"

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Register(_ string, _ func()) error {
	return fmt.Errorf("global hotkey is only implemented on Windows")
}

func (m *Manager) Close() {}
