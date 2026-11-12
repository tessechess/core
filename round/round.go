// Package round defines the games of a tournament round. The same Game type is
// used for completed rounds (input to trf, pairing and standings) and for a
// freshly paired round (output of pairing).
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend).
package round

import "github.com/tessechess/core/result"

// Color is a side in a game.
type Color string

const (
	// ColorWhite is the side that moves first.
	ColorWhite Color = "white"
	// ColorBlack is the side that moves second.
	ColorBlack Color = "black"
)

// Game is one game (or bye) of a round.
type Game[ID comparable] struct {
	Round  int          // round number, 1-based
	Board  int          // board number; 0 when unknown
	White  ID           // for a bye, the player receiving it
	Black  *ID          // nil for a bye
	Result *result.Code // nil while not played
}

// IsBye reports whether the game has no opponent.
func (g Game[ID]) IsBye() bool { return g.Black == nil }

// Points returns the points player scored in the game, using result.Code.Points:
// the white side for White, the black side for Black. A game without a result,
// and a player who isn't in the game, score 0.
func (g Game[ID]) Points(player ID) float64 {
	if g.Result == nil {
		return 0
	}
	white, black := g.Result.Points()
	switch {
	case g.White == player:
		return white
	case g.Black != nil && *g.Black == player:
		return black
	default:
		return 0
	}
}
