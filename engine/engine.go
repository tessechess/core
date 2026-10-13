// Package engine runs external FIDE pairing programs (bbpPairings, JaVaFo) on a
// TRF input and returns their raw output; parse it with the matching parser in
// core/trf (see OutputFormat). Callers supply the binary or jar paths: the
// programs themselves are not part of core (JaVaFo must not be redistributed).
package engine

import "context"

// OutputFormat tells the caller which parser to use on the engine output.
type OutputFormat string

const (
	// OutputTRF means the engine outputs a modified TRF with a new round appended.
	// Neither built-in engine uses it; parse with trf.ParseBBPPairingsOutput.
	OutputTRF OutputFormat = "trf"
	// OutputSimplePairs means the engine outputs "count\nwhite black\n..."
	// (bbpPairings and JaVaFo with -p); parse with trf.ParseJaVaFoOutput.
	OutputSimplePairs OutputFormat = "simple_pairs"
)

// PairingEngine generates pairings from a TRF input string.
type PairingEngine interface {
	// Name returns the engine identifier ("bbppairings" or "javafo").
	Name() string
	// GeneratePairings runs the engine and returns its raw output.
	GeneratePairings(ctx context.Context, trfInput string) (output string, err error)
	// OutputFormat returns which parser should be used on the output.
	OutputFormat() OutputFormat
}
