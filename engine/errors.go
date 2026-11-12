package engine

import "errors"

// Errors returned by the engines' GeneratePairings. Test for them with
// errors.Is; the errors carry details (path, engine output) after the
// sentinel's text.
var (
	// ErrEngineNotFound means the engine binary (or JaVaFo's jar or the java
	// executable) does not exist at the configured path.
	ErrEngineNotFound = errors.New("engine binary not found at configured path")
	// ErrEngineTimeout means a run exceeded the engine's Timeout, or the
	// deadline of the context passed first.
	ErrEngineTimeout = errors.New("engine exceeded timeout")
	// ErrEngineExecution means the engine exited with an error; the error text
	// contains the engine's output.
	ErrEngineExecution = errors.New("engine returned non-zero exit code")
)
