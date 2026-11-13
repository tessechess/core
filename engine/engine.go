// Package engine runs external FIDE pairing programs (bbpPairings, JaVaFo) on a
// TRF input and returns their raw output; parse it with the matching parser in
// core/trf (see OutputFormat). Callers supply the binary or jar paths: the
// programs themselves are not part of core as per their licenses.
//
// Other engines plug in by implementing PairingEngine. The two built-in ones
// write the pair list (OutputSimplePairs); an engine that returns the input TRF
// with the new round appended reports OutputTRF.
//
// # Timeouts and child processes
//
// A run is limited by the engine's Timeout and by the deadline of the context,
// and cmd.WaitDelay closes the output pipe if a child process keeps it open
// past cancellation so GeneratePairings returns on time. On Unix the engine
// runs in its own process group and a timeout kills the whole group. On other
// platforms (such as Windows, where killing a process tree requires a job
// object) only the engine process itself is killed: if the configured
// executable is a wrapper that starts the real engine as a child (a shell
// script, a launcher), that child may survive in the background. Point
// BinaryPath and JavaPath at the real executable, not at a wrapper. Known
// limitation, see https://github.com/tessechess/tessentials/issues/382.
package engine

import "context"

// OutputFormat tells the caller which parser to use on the engine output.
type OutputFormat string

const (
	// OutputTRF means the engine outputs the input TRF with a new round column
	// appended to each player line; parse with trf.ParseTRFRound. It is for
	// engines that return a TRF: neither built-in engine does, but pairing
	// supports it for custom PairingEngine implementations.
	OutputTRF OutputFormat = "trf"
	// OutputSimplePairs means the engine outputs "count\nwhite black\n..."
	// (bbpPairings and JaVaFo with -p); parse with trf.ParseSimplePairs.
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
