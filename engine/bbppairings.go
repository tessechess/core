package engine

import (
	"context"
	"fmt"
	"os"
	"time"
)

var _ PairingEngine = (*BBPPairings)(nil)

// BBPPairings is the PairingEngine for the bbpPairings binary. It runs
// "bbpPairings --dutch <input> -p <output>" (Dutch system, pair list output).
//
// Build one with NewBBPPairings, or fill in the fields directly; Timeout must
// then be set, because a zero Timeout makes every run time out immediately.
type BBPPairings struct {
	BinaryPath string        // path to the bbpPairings executable
	Timeout    time.Duration // longest a single run may take
}

// NewBBPPairings returns an engine that runs the bbpPairings binary at
// binaryPath, killing it after timeout. Nothing is checked until
// GeneratePairings is called.
func NewBBPPairings(binaryPath string, timeout time.Duration) *BBPPairings {
	return &BBPPairings{
		BinaryPath: binaryPath,
		Timeout:    timeout,
	}
}

// Name returns "bbppairings".
func (b *BBPPairings) Name() string { return "bbppairings" }

// OutputFormat is OutputSimplePairs: with -p, bbpPairings writes the pair list
// ("count\nwhite black\n..."), the same format as JaVaFo, not a TRF.
func (b *BBPPairings) OutputFormat() OutputFormat { return OutputSimplePairs }

// GeneratePairings pairs the next round of the TRF in trfInput and returns the
// engine's pair list. It returns ErrEngineNotFound if BinaryPath does not
// exist, ErrEngineTimeout if the run exceeds Timeout (or the deadline of ctx
// passes first) and ErrEngineExecution, with the engine's output, if it exits
// with an error.
func (b *BBPPairings) GeneratePairings(ctx context.Context, trfInput string) (string, error) {
	if _, err := os.Stat(b.BinaryPath); err != nil {
		return "", fmt.Errorf("%w: %s", ErrEngineNotFound, b.BinaryPath)
	}

	ctx, cancel := context.WithTimeout(ctx, b.Timeout)
	defer cancel()

	return runEngine(ctx, b.BinaryPath, func(inputPath, outputPath string) []string {
		return []string{"--dutch", inputPath, "-p", outputPath}
	}, trfInput)
}
