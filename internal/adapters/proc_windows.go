//go:build windows

package adapters

import (
	"os/exec"
)

// configureProcAttr is a no-op on Windows since Setpgid is not supported.
func configureProcAttr(cmd *exec.Cmd) {
	// No-op for Windows
}

// killProcessGroup falls back to simple process kill on Windows.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
