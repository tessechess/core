// Package trf generates FIDE TRF 1.6 input for pairing engines (bbpPairings,
// JaVaFo) and parses their output.
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend). Inside the TRF they are referred to by starting rank; the
// returned PlayerMapping translates between the two.
package trf

import (
	"time"

	"github.com/tessechess/core/player"
	"github.com/tessechess/core/round"
)

// GeneratorInput contains all data needed to produce a TRF 1.6 file.
type GeneratorInput[ID comparable] struct {
	TournamentName string           // written as the tournament name (012)
	City           string           // optional (022)
	Federation     string           // optional, 3-letter code (032)
	StartDate      time.Time        // written as YYYY/MM/DD; zero = omitted
	EndDate        time.Time        // written as YYYY/MM/DD; zero = omitted
	TimeControl    string           // optional (122)
	TotalRounds    int              // total rounds of the tournament; must be > 0 (XXR)
	Players        []PlayerData[ID] // all players of the section, in any order; at least one
	// Games of the completed rounds, in round order. Each game's Round places
	// it in the TRF; voided games are skipped.
	Games []round.Game[ID]
	// CompletedRounds is the number of completed rounds (1..CompletedRounds);
	// the round being paired is CompletedRounds+1.
	CompletedRounds int
	Settings        *RoundSettings[ID] // nil = no special settings for the upcoming round
}

// PlayerData holds one player's details for TRF generation.
type PlayerData[ID comparable] struct {
	ID           ID           // the caller's ID, used in games and settings
	StartingRank int          // pre-assigned starting rank (pairing number), must be > 0
	Name         string       // "Lastname, Firstname"
	Title        player.Title // only FIDE titles are written; others leave the column blank
	FIDERating   int          // 0 if unrated
	Federation   string       // 3-letter code
	FIDENumber   string       // FIDE ID, optional
	BirthDate    *time.Time   // written as YYYY/MM/DD; nil = blank
	Sex          player.Sex   // written as "m" (male) or "w" (female); unknown leaves the column blank
}

// PlayerMapping provides bidirectional lookup between TRF starting ranks and player IDs.
type PlayerMapping[ID comparable] struct {
	RankToID map[int]ID // starting rank to player ID
	IDToRank map[ID]int // player ID to starting rank
}

// RoundSettings holds arbiter-configured pairing settings translated to TRF terms.
// Applied to the upcoming (not-yet-paired) round only.
type RoundSettings[ID comparable] struct {
	// WithdrawnPlayers receive a zero-point bye (Z) for the upcoming round.
	WithdrawnPlayers []ID
	// HalfPointByes receive a half-point bye (H) for the upcoming round.
	HalfPointByes []ID
	// FirstPlayerColor sets the XXC line (round.ColorWhite or round.ColorBlack;
	// empty = no line). Round 1 only.
	FirstPlayerColor round.Color
	// ForbiddenPairings emits XXF lines preventing these players from being paired together.
	ForbiddenPairings []ForbiddenPair[ID]
	// LateEntries are IDs of players joining late. The generator
	// inserts zero-point byes (Z) for all rounds these players missed.
	LateEntries []ID
}

// ForbiddenPair is a pair of player IDs that must not be paired.
type ForbiddenPair[ID comparable] struct {
	Player1 ID // one of the two players
	Player2 ID // the other; the order does not matter
}

// ParsedPairing represents a single pairing extracted from engine output.
type ParsedPairing struct {
	BoardNumber int // 1-based board number
	WhiteRank   int // TRF starting rank
	BlackRank   int // 0 = bye
}
