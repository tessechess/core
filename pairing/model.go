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

	"github.com/ValkoM28/tournamentessentials/core/trf"
)

// Input is everything needed to pair the next round.
type Input[ID comparable] struct {
	Tournament Tournament
	Players    []Player[ID]        // confirmed players of the section
	History    []trf.RoundData[ID] // completed rounds before the one being paired, with their games
	Settings   *Settings[ID]       // arbiter settings for this round; nil = none
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
	Title        string
	Rating       int // 0 if unrated
	Federation   string
	FIDENumber   string
	Birthday     *time.Time
	Sex          string
	FixedBoard   *int // player-level fixed board; a round-level FixedBoard for the same player wins
}

// Settings are the arbiter's settings for the round being paired.
type Settings[ID comparable] struct {
	WithdrawnPlayers  []ID   // zero-point bye this round
	HalfPointByes     []ID   // half-point bye this round
	LateEntries       []ID   // zero-point byes for the rounds they missed
	FirstPlayerColor  string // "white" or "black"; round 1 only
	ForbiddenPairings []ForbiddenPair[ID]
	FixedBoards       []FixedBoard[ID]
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

// Board is one game of the paired round.
type Board[ID comparable] struct {
	Number int
	White  ID
	Black  *ID // nil for a bye
}
