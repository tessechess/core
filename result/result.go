// Package result defines game result codes and the points they award.
//
// The format is always "white_score-black_score".
//
// Real-game results (both players present):
//
//	WhiteWins    — "1-0"     white wins normally
//	BlackWins    — "0-1"     black wins normally
//	Draw         — "0.5-0.5" draw
//	WhiteForfeit — "+/-"     white wins; black forfeited
//	BlackForfeit — "-/+"     black wins; white forfeited
//	WhiteHalf    — "0.5-0"   white awarded 0.5, black 0
//	BlackHalf    — "0-0.5"   black awarded 0.5, white 0
//
// Unplayed-round results (no black player — bye recipient is always white):
//
//	WhiteWins    — "1-0"   full-point bye; white receives 1.0 (reused)
//	WhiteHalf    — "0.5-0" half-point bye; white receives 0.5 (reused)
//	ZeroPointBye — "0-0"   zero-point bye or withdrawal; white receives 0.0
//
// Administrative:
//
//	Unplayed — "*"    round not yet played (no points awarded)
//	Void     — "void" pairing voided; excluded from all calculations
package result

// Code is a game result code.
type Code string

const (
	// WhiteWins is a win for white: white 1, black 0. On a bye (no black
	// player) it is the full-point bye. Reportable.
	WhiteWins Code = "1-0"
	// BlackWins is a win for black: white 0, black 1. Reportable.
	BlackWins Code = "0-1"
	// Draw gives each side 0.5. Reportable.
	Draw Code = "0.5-0.5"
	// WhiteForfeit is a win for white because black was present but did not
	// play: white 1, black 0. Reportable; IsForfeit is true.
	WhiteForfeit Code = "+/-"
	// BlackForfeit is a win for black because white was present but did not
	// play: white 0, black 1. Reportable; IsForfeit is true.
	BlackForfeit Code = "-/+"
	// WhiteHalf awards white 0.5 and black 0 (an adjudication). On a bye it is
	// the half-point bye. Reportable.
	WhiteHalf Code = "0.5-0"
	// BlackHalf awards black 0.5 and white 0 (an adjudication). Reportable.
	BlackHalf Code = "0-0.5"
	// ZeroPointBye is a zero-point bye or a withdrawal: no points. It is set
	// by the system and is not reportable.
	ZeroPointBye Code = "0-0"
	// Unplayed marks a game that has not been played yet: no points, and it
	// takes no part in scoring (Counts is false). Not reportable.
	Unplayed Code = "*"
	// Void marks a voided pairing, excluded from all calculations (Counts is
	// false). Not reportable.
	Void Code = "void"
)

// Points returns the points awarded to white and black. Unplayed, Void and
// unknown codes award nothing.
func (c Code) Points() (white, black float64) {
	switch c {
	case WhiteWins, WhiteForfeit:
		return 1, 0
	case BlackWins, BlackForfeit:
		return 0, 1
	case Draw:
		return 0.5, 0.5
	case WhiteHalf:
		return 0.5, 0
	case BlackHalf:
		return 0, 0.5
	}
	return 0, 0
}

// Valid reports whether c is one of the known result codes.
func (c Code) Valid() bool {
	switch c {
	case WhiteWins, BlackWins, Draw, WhiteForfeit, BlackForfeit,
		WhiteHalf, BlackHalf, ZeroPointBye, Unplayed, Void:
		return true
	}
	return false
}

// IsForfeit reports whether c is a forfeit (one player present but the game not played).
func (c Code) IsForfeit() bool {
	return c == WhiteForfeit || c == BlackForfeit
}

// Counts reports whether c is a settled result that takes part in scoring and
// reports, i.e. it is neither Unplayed nor Void.
func (c Code) Counts() bool {
	return c != Unplayed && c != Void
}

// Reportable returns the codes an arbiter may report for a game: the seven
// results of a played game (or a forfeit or adjudication). The bye code
// ZeroPointBye, Unplayed and Void are set by the system, not reported.
func Reportable() []Code {
	return []Code{WhiteWins, BlackWins, Draw, WhiteForfeit, BlackForfeit, WhiteHalf, BlackHalf}
}
