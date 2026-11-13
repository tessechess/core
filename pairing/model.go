// Package pairing pairs the next round of a Swiss section (it builds the TRF
// input from the players, the completed rounds and the round settings, runs a
// pairing engine, maps the engine's answer back to player IDs and applies fixed
// boards) or generates the full schedule of a round-robin section upfront using
// the Berger function (see Berger and PairRoundRobin).
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend). The package has no persistence, and choosing which engine to
// run for Swiss pairings is the caller's job.
//
// # Known limitations
//
// Swiss pairing requires an external engine whose program is not part of this
// module. The caller supplies it (see package engine); JaVaFo must not be
// redistributed.
//
// Acceleration is not implemented: Settings.Validate rejects anything other
// than AccelerationNone (or empty), and Pair does not write acceleration to
// the TRF.
package pairing

import (
	"time"

	"github.com/tessechess/core/player"
	"github.com/tessechess/core/round"
)

// Input is everything needed to pair the next round.
type Input[ID comparable] struct {
	Tournament Tournament       // header data written to the TRF
	Players    []Player[ID]     // confirmed players of the section
	History    []round.Game[ID] // games of the completed rounds, in round order
	// CompletedRounds is the number of completed rounds; the round being paired
	// is CompletedRounds+1.
	CompletedRounds int
	Settings        *Settings[ID] // arbiter settings for this round; nil = none (round 1 requires FirstPlayerColor)
}

// Tournament holds the header data written to the TRF.
type Tournament struct {
	Name        string    // tournament (section) name
	City        string    // optional
	Federation  string    // optional, 3-letter code
	TimeControl string    // optional, free text such as "90+30"
	StartDate   time.Time // zero = omitted from the TRF
	EndDate     time.Time // zero = omitted from the TRF
	TotalRounds int       // total rounds of the tournament; must be > 0
}

// Player is one confirmed player of the section.
type Player[ID comparable] struct {
	ID           ID           // the caller's ID, used in History, Settings and the result
	StartingRank int          // pairing number; 0 = not assigned (TRF generation then fails)
	FirstName    string       // written as "LastName, FirstName"
	LastName     string       // see FirstName
	Title        player.Title // only FIDE titles reach the TRF
	Rating       int          // 0 if unrated
	Federation   string       // 3-letter code, optional
	FIDENumber   string       // FIDE ID, optional
	Birthday     *time.Time   // nil = unknown
	Sex          player.Sex   // player.SexUnknown = not recorded
	FixedBoard   *int         // player-level fixed board; a round-level FixedBoard for the same player wins
}

// Settings are the arbiter's settings for the round being paired. Check them
// with Validate before storing or pairing.
type Settings[ID comparable] struct {
	WithdrawnPlayers  []ID                // zero-point bye this round
	HalfPointByes     []ID                // half-point bye this round
	LateEntries       []ID                // zero-point byes for the rounds they missed
	FirstPlayerColor  round.Color         // round.ColorWhite or round.ColorBlack; round 1 only (required there)
	ForbiddenPairings []ForbiddenPair[ID] // pairs of players that must not meet this round
	FixedBoards       []FixedBoard[ID]    // players that must play on a given board this round

	// Acceleration is the acceleration method for round 1; empty means none.
	// Not implemented: Validate rejects anything other than AccelerationNone
	// (or empty), and Pair ignores it.
	Acceleration Acceleration
}

// Acceleration is a pairing acceleration method, used in round 1. The empty
// value means "not set" and behaves like AccelerationNone.
//
// Not implemented: Settings.Validate rejects anything other than
// AccelerationNone (or empty).
type Acceleration string

const (
	// AccelerationNone pairs without acceleration.
	AccelerationNone Acceleration = "none"
	// AccelerationBaku is Baku acceleration (FIDE C.04.5). Not implemented
	// (rejected by Settings.Validate).
	AccelerationBaku Acceleration = "baku"
	// AccelerationDutch is the third option offered by the dashboard. Not
	// implemented (rejected by Settings.Validate).
	AccelerationDutch Acceleration = "dutch"
)

// ForbiddenPair is two players who must not be paired together.
type ForbiddenPair[ID comparable] struct {
	Player1 ID // one of the two players
	Player2 ID // the other; must differ from Player1 (see Settings.Validate)
}

// FixedBoard pins a player to a board number. It is applied after the engine
// has paired the round: the player's game moves to that board and the other
// boards are renumbered around it.
type FixedBoard[ID comparable] struct {
	BoardNumber int // 1-based; must be unique within the settings
	Player      ID  // the player who is put on that board
}
