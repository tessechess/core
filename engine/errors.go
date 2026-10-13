package engine

import "errors"

var (
	ErrEngineNotFound  = errors.New("engine binary not found at configured path")
	ErrEngineTimeout   = errors.New("engine exceeded timeout")
	ErrEngineExecution = errors.New("engine returned non-zero exit code")
)
