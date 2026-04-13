//go:build windows

package capture

import "os/exec"

func configureCommand(cmd *exec.Cmd) {
	// Flameshot is a GUI app on Windows. Hiding its window can prevent the
	// selection overlay from appearing when capture is triggered from the app launcher window.
	cmd.SysProcAttr = nil
}
