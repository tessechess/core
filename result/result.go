// Package result defines game result codes and the points they award.
//
// The format is always "white_score-black_score".
//
// Real-game results (both players present):
//
//	WhiteWins    — "1-0"     white wins normally
//	BlackWins    — "0-1"     black wins normally
//	Draw         — "0.5-0.5" draw
//	WhiteForfeit — "+/-"     white wins; black forfeited (present but didn't play)
//	BlackForfeit — "-/+"     black wins; white forfeited (present but didn't play)
//	WhiteHalf    — "0.5-0"   white awarded 0.5, black 0 (edge case adjudication)
//	BlackHalf    — "0-0.5"   black awarded 0.5, white 0 (edge case adjudication)
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
	WhiteWins    Code = "1-0"
	BlackWins    Code = "0-1"
	Draw         Code = "0.5-0.5"
	WhiteForfeit Code = "+/-"
	BlackForfeit Code = "-/+"
	WhiteHalf    Code = "0.5-0"
	BlackHalf    Code = "0-0.5"
	ZeroPointBye Code = "0-0"
	Unplayed     Code = "*"
	Void         Code = "void"
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

// Submittable returns the codes an arbiter may enter for a game, in a stable order.
func Submittable() []Code {
	return []Code{WhiteWins, BlackWins, Draw, WhiteForfeit, BlackForfeit, WhiteHalf, BlackHalf}
}
