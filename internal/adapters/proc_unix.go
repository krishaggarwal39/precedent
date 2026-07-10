//go:build unix

package adapters

import (
	"os/exec"
	"syscall"
)

// configureProcAttr sets up the process to run in a new process group
// so we can kill it and all its children.
func configureProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup kills the entire process group.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Kill the entire process group by negating the PID
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
