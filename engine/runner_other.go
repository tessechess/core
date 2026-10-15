//go:build !unix

package engine

import "os/exec"

// configureProcessGroup is a no-op on non-Unix platforms: exec.CommandContext's
// default cancellation kills only the engine process itself, so child processes
// (e.g. JaVaFo's JVM when started via a wrapper) may outlive a timeout. Because
// the output is captured through a pipe and cmd.WaitDelay is not set, such a
// child that keeps the pipe open can also delay the return of the run past the
// timeout.
//
// TODO(#382): set WaitDelay on all platforms and kill the process tree here.
func configureProcessGroup(cmd *exec.Cmd) {}
