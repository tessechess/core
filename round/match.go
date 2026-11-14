package round

import "github.com/tessechess/core/result"

// Match is one team-vs-team match (or team bye) of a round.
type Match[TeamID, PlayerID comparable] struct {
	Round             int              // round number, 1-based
	Table             int              // table (match) number, 1-based; 0 when unknown
	Home              TeamID           // home team; for a bye, the team receiving it
	Away              *TeamID          // away team; nil for a bye
	WhiteOnFirstBoard bool             // true when Home has white on board 1 (and odd boards); false when Away does
	Boards            []Game[PlayerID] // individual board games of the match, ordered by board (1..N)
	Result            *result.Code     // match-level result for team byes, forfeits, or voided matches; nil when decided by Boards
}

// IsBye reports whether the match has no away team.
func (m Match[TeamID, PlayerID]) IsBye() bool { return m.Away == nil }

// BoardColor returns the piece color that team plays on the given 1-based
// board number according to WhiteOnFirstBoard. Odd boards (1, 3, 5, ...) give
// Home white when WhiteOnFirstBoard is true and black when false; even boards
// (2, 4, 6, ...) alternate. For a bye, a board < 1, or a team not in the
// match, BoardColor returns "".
func (m Match[TeamID, PlayerID]) BoardColor(team TeamID, board int) Color {
	if m.Away == nil || board < 1 {
		return ""
	}
	homeIsWhite := (board%2 == 1) == m.WhiteOnFirstBoard
	switch {
	case m.Home == team:
		if homeIsWhite {
			return ColorWhite
		}
		return ColorBlack
	case *m.Away == team:
		if homeIsWhite {
			return ColorBlack
		}
		return ColorWhite
	default:
		return ""
	}
}

// GamePoints returns the total board game points scored by Home and Away across
// Boards. Each board's white and black points (from result.Code.Points) are
// assigned to Home and Away according to its 1-based Board number (or its
// 1-based slice position when Board is 0) and WhiteOnFirstBoard.
//
// Unplayed boards, void boards, bye boards, a team bye (Away is nil), or a
// voided match (Result is result.Void) contribute 0.
func (m Match[TeamID, PlayerID]) GamePoints() (home, away float64) {
	if m.Away == nil {
		return 0, 0
	}
	if m.Result != nil && *m.Result == result.Void {
		return 0, 0
	}
	for i, g := range m.Boards {
		if g.IsBye() || g.Result == nil || !g.Result.Counts() {
			continue
		}
		boardNum := g.Board
		if boardNum < 1 {
			boardNum = i + 1
		}
		whitePts, blackPts := g.Result.Points()
		homeIsWhite := (boardNum%2 == 1) == m.WhiteOnFirstBoard
		if homeIsWhite {
			home += whitePts
			away += blackPts
		} else {
			home += blackPts
			away += whitePts
		}
	}
	return home, away
}

// TeamGamePoints returns the board game points scored by team in Boards, using
// GamePoints. A match without played boards, a bye, or a team that is not in
// the match scores 0.
func (m Match[TeamID, PlayerID]) TeamGamePoints(team TeamID) float64 {
	home, away := m.GamePoints()
	switch {
	case m.Home == team:
		return home
	case m.Away != nil && *m.Away == team:
		return away
	default:
		return 0
	}
}

// Settled reports whether the match has a settled outcome that counts toward
// standings: either Result is non-nil and Result.Counts() is true, or the match
// is against an opponent (Away is non-nil) with at least one non-void board and
// every non-void board has a non-nil Result where Result.Counts() is true.
func (m Match[TeamID, PlayerID]) Settled() bool {
	if m.Result != nil {
		return m.Result.Counts()
	}
	if m.Away == nil || len(m.Boards) == 0 {
		return false
	}
	counted := 0
	for _, g := range m.Boards {
		if g.Result == nil {
			return false
		}
		if *g.Result == result.Void {
			continue
		}
		if !g.Result.Counts() {
			return false
		}
		counted++
	}
	return counted > 0
}
