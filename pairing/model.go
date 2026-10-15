// Package pairing pairs the next round of a Swiss section: it builds the TRF
// input from the players, the completed rounds and the round settings, runs a
// pairing engine, maps the engine's answer back to player IDs and applies fixed
// boards.
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend). The package has no persistence, and choosing which engine to
// run is the caller's job.
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
	FirstPlayerColor  round.Color // round.ColorWhite or round.ColorBlack; round 1 only
	ForbiddenPairings []ForbiddenPair[ID]
	FixedBoards       []FixedBoard[ID]

	// Acceleration is "none", "baku", "dutch" or "" (round 1 only).
	// TODO(#377): validated, but not written to the TRF yet, so Pair ignores it.
	Acceleration string
}

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
