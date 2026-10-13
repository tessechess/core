package engine

import (
	"context"
	"fmt"
	"os"
	"time"
)

// BBPPairings implements PairingEngine for the bbpPairings binary.
type BBPPairings struct {
	BinaryPath string
	Timeout    time.Duration
}

func NewBBPPairings(binaryPath string, timeout time.Duration) PairingEngine {
	return &BBPPairings{
		BinaryPath: binaryPath,
		Timeout:    timeout,
	}
}

func (b *BBPPairings) Name() string { return "bbppairings" }

func (b *BBPPairings) OutputFormat() OutputFormat { return OutputTRF }

func (b *BBPPairings) GeneratePairings(ctx context.Context, trfInput string) (string, error) {
	if _, err := os.Stat(b.BinaryPath); err != nil {
		return "", fmt.Errorf("%w: %s", ErrEngineNotFound, b.BinaryPath)
	}

	ctx, cancel := context.WithTimeout(ctx, b.Timeout)
	defer cancel()

	return RunEngine(ctx, b.BinaryPath, func(inputPath, outputPath string) []string {
		return []string{"--dutch", inputPath, "-p", outputPath}
	}, trfInput, true)
}
