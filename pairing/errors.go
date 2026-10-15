package pairing

import "errors"

var (
	// ErrTRFGeneration indicates a failure generating the TRF input.
	ErrTRFGeneration = errors.New("failed to generate TRF input")

	// ErrTRFParsing indicates a failure parsing the engine output.
	ErrTRFParsing = errors.New("failed to parse engine output")

	// ErrUnknownEngineRank indicates the engine returned a rank not present in the TRF input.
	ErrUnknownEngineRank = errors.New("engine returned unknown player rank")
)
