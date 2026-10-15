// Package pairing pairs the next round of a Swiss section: it builds the TRF
// input from the players, the completed rounds and the round settings, runs a
// pairing engine, maps the engine's answer back to player IDs and applies fixed
// boards.
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend). The package has no persistence, and choosing which engine to
// run is the caller's job.
//
// # Known limitations
//
// Swiss pairing only, and only through an external engine; there is no
// round-robin (Berger tables) generator.
//
// The engine programs are not part of this module. The caller supplies them
// (see package engine); JaVaFo must not be redistributed.
//
// Round 1 needs Settings.FirstPlayerColor with bbpPairings (#376): without
// the colour line, bbpPairings refuses to pair a round that has no previous
// games ("Please configure the initial piece colors") and Pair returns the
// engine's error (engine.ErrEngineExecution). Later rounds do not need it.
//
// Settings.Acceleration is validated but has no effect (#377): it is not
// written to the TRF, so engines pair without acceleration.
package pairing

import (
	"time"

	"github.com/ValkoM28/tournamentessentials/core/player"
	"github.com/ValkoM28/tournamentessentials/core/round"
)

// Input is everything needed to pair the next round.
type Input[ID comparable] struct {
	Tournament Tournament
	Players    []Player[ID]     // confirmed players of the section
	History    []round.Game[ID] // games of the completed rounds, in round order
	// CompletedRounds is the number of completed rounds; the round being paired
	// is CompletedRounds+1.
	CompletedRounds int
	Settings        *Settings[ID] // arbiter settings for this round; nil = none
}

// Tournament holds the header data written to the TRF.
type Tournament struct {
	Name        string
	City        string
	Federation  string
	TimeControl string
	StartDate   time.Time
	EndDate     time.Time
	TotalRounds int
}

// Player is one confirmed player of the section.
type Player[ID comparable] struct {
	ID           ID
	StartingRank int // pairing number; 0 = not assigned (TRF generation then fails)
	FirstName    string
	LastName     string
	Title        player.Title
	Rating       int // 0 if unrated
	Federation   string
	FIDENumber   string
	Birthday     *time.Time
	Sex          player.Sex
	FixedBoard   *int // player-level fixed board; a round-level FixedBoard for the same player wins
}

// Settings are the arbiter's settings for the round being paired. Check them
// with Validate before storing or pairing.
type Settings[ID comparable] struct {
	WithdrawnPlayers  []ID        // zero-point bye this round
	HalfPointByes     []ID        // half-point bye this round
	LateEntries       []ID        // zero-point byes for the rounds they missed
	FirstPlayerColor  round.Color // round.ColorWhite or round.ColorBlack; round 1 only, and required then with bbpPairings (#376)
	ForbiddenPairings []ForbiddenPair[ID]
	FixedBoards       []FixedBoard[ID]

	// Acceleration is the acceleration method for round 1; empty means none.
	// It is validated but currently has no effect: it is not written to the
	// TRF, so Pair ignores it (#377).
	Acceleration Acceleration
}

// Acceleration is a pairing acceleration method, used in round 1. The empty
// value means "not set" and behaves like AccelerationNone.
type Acceleration string

const (
	// AccelerationNone pairs without acceleration.
	AccelerationNone Acceleration = "none"
	// AccelerationBaku is Baku acceleration (FIDE C.04.5).
	AccelerationBaku Acceleration = "baku"
	// AccelerationDutch is the third option offered by the dashboard.
	AccelerationDutch Acceleration = "dutch"
)

// ForbiddenPair is two players who must not be paired together.
type ForbiddenPair[ID comparable] struct {
	Player1 ID
	Player2 ID
}

// FixedBoard pins a player to a board number. It is applied after the engine
// has paired the round: the player's game moves to that board and the other
// boards are renumbered around it.
type FixedBoard[ID comparable] struct {
	BoardNumber int
	Player      ID
}
