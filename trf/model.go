// Package trf generates FIDE TRF 1.6 input for pairing engines (bbpPairings,
// JaVaFo) and parses their output.
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend). Inside the TRF they are referred to by starting rank; the
// returned PlayerMapping translates between the two.
package trf

import "github.com/ValkoM28/tournamentessentials/core/result"

// GeneratorInput contains all data needed to produce a TRF 1.6 file.
type GeneratorInput[ID comparable] struct {
	TournamentName string
	City           string
	Federation     string
	StartDate      string // "YYYY/MM/DD"
	EndDate        string // "YYYY/MM/DD"
	TimeControl    string
	TotalRounds    int
	Players        []PlayerData[ID]
	Rounds         []RoundData[ID]    // completed rounds only, ordered by number
	Settings       *RoundSettings[ID] // nil = no special settings for the upcoming round
}

// PlayerData holds one player's details for TRF generation.
type PlayerData[ID comparable] struct {
	ID           ID
	StartingRank int    // pre-assigned starting rank (pairing number), must be > 0
	Name         string // "Lastname, Firstname"
	Title        string // "GM", "IM", "FM", "WGM", "WIM", "WFM", ""
	FIDERating   int    // 0 if unrated
	Federation   string // 3-letter code
	FIDENumber   string
	BirthDate    string // "YYYY/MM/DD"
	Sex          string // "m", "w", ""
}

// RoundData holds the pairings for one completed round.
type RoundData[ID comparable] struct {
	RoundNumber int
	Pairings    []PairingData[ID]
}

// PairingData holds one pairing from a completed round.
type PairingData[ID comparable] struct {
	White  ID
	Black  *ID          // nil for byes
	Result *result.Code // nil = not yet played
}

// PlayerMapping provides bidirectional lookup between TRF starting ranks and player IDs.
type PlayerMapping[ID comparable] struct {
	RankToID map[int]ID
	IDToRank map[ID]int
}

// RoundSettings holds arbiter-configured pairing settings translated to TRF terms.
// Applied to the upcoming (not-yet-paired) round only.
type RoundSettings[ID comparable] struct {
	// WithdrawnPlayers receive a zero-point bye (Z) for the upcoming round.
	WithdrawnPlayers []ID
	// HalfPointByes receive a half-point bye (H) for the upcoming round.
	HalfPointByes []ID
	// FirstPlayerColor sets the XXC line ("white" or "black"). Round 1 only.
	FirstPlayerColor string
	// ForbiddenPairings emits XXF lines preventing these players from being paired together.
	ForbiddenPairings []ForbiddenPairTRF[ID]
	// LateEntries are IDs of players joining late. The generator
	// inserts zero-point byes (Z) for all rounds these players missed.
	LateEntries []ID
}

// ForbiddenPairTRF is a pair of player IDs that must not be paired.
type ForbiddenPairTRF[ID comparable] struct {
	Player1 ID
	Player2 ID
}

// ParsedPairing represents a single pairing extracted from engine output.
type ParsedPairing struct {
	BoardNumber int
	WhiteRank   int // TRF starting rank
	BlackRank   int // 0 = bye
}
