package pairing

import "errors"

var (
	// ErrInvalidSettings indicates invalid pairing settings (see Settings.Validate and Prepare).
	ErrInvalidSettings = errors.New("invalid pairing settings")

	// ErrPlayerConflict indicates a player appears in conflicting settings lists.
	ErrPlayerConflict = errors.New("player appears in conflicting settings lists")

	// ErrTRFGeneration indicates a failure generating the TRF input.
	ErrTRFGeneration = errors.New("failed to generate TRF input")

	// ErrTRFParsing indicates a failure parsing the engine output.
	ErrTRFParsing = errors.New("failed to parse engine output")

	// ErrUnknownEngineRank indicates the engine returned a rank not present in the TRF input.
	ErrUnknownEngineRank = errors.New("engine returned unknown player rank")
)
