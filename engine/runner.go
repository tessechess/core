package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// runEngine writes trfInput to a temp file, executes the binary and returns the
// contents of the output file the engine was told to write. buildArgs receives
// the input and output file paths and returns the argument list. A timeout of
// ctx gives ErrEngineTimeout, a non-zero exit gives ErrEngineExecution with the
// engine's combined stdout and stderr.
func runEngine(
	ctx context.Context,
	binaryPath string,
	buildArgs func(inputPath, outputPath string) []string,
	trfInput string,
) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pairing-engine-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.trf")
	outputPath := filepath.Join(tmpDir, "output.trf")

	if err := os.WriteFile(inputPath, []byte(trfInput), 0600); err != nil {
		return "", fmt.Errorf("failed to write TRF input: %w", err)
	}

	args := buildArgs(inputPath, outputPath)
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.WaitDelay = time.Second

	configureProcessGroup(cmd)

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	err = cmd.Run()
	if err != nil {
		if ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("%w: %s", ErrEngineTimeout, binaryPath)
		}
		detail := combined.String()
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("%w: %s — %s", ErrEngineExecution, binaryPath, detail)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read engine output file: %w", err)
	}
	return string(data), nil
}
