//go:build unix

package engine

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup runs the engine in its own process group and makes
// cancellation (timeout) kill the whole group, so child processes such as
// JaVaFo's JVM don't outlive the request.
func configureProcessGroup(cmd *exec.Cmd) {
	// Run the process in its own process group so we can kill all children on timeout.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Cancel kills the entire process group instead of just the leader.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
