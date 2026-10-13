//go:build !unix

package engine

import "os/exec"

// configureProcessGroup is a no-op on non-Unix platforms: exec.CommandContext's
// default cancellation kills only the engine process itself, so child processes
// (e.g. JaVaFo's JVM when started via a wrapper) may outlive a timeout.
func configureProcessGroup(cmd *exec.Cmd) {}
