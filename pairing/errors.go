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

	// ErrNotEnoughPlayers indicates fewer than two players were provided for pairing.
	ErrNotEnoughPlayers = errors.New("at least 2 players are required for pairing")

	// ErrInvalidCycles indicates a non-positive cycle count for round-robin pairing.
	ErrInvalidCycles = errors.New("round-robin cycles must be at least 1")

	// ErrInvalidCycleOrder indicates an unknown CycleOrder for round-robin pairing.
	ErrInvalidCycleOrder = errors.New("unknown round-robin cycle order")

	// ErrInvalidStartingRank indicates a missing, out-of-range, or duplicate starting rank.
	ErrInvalidStartingRank = errors.New("invalid or duplicate player starting rank")
)
