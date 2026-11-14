package round_test

import (
	"testing"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

func TestMatchIsBye(t *testing.T) {
	if !(round.Match[string, string]{Home: "teamA"}).IsBye() {
		t.Error("match without away team should be a bye")
	}
	if (round.Match[string, string]{Home: "teamA", Away: id("teamB")}).IsBye() {
		t.Error("match with away team should not be a bye")
	}
}

func TestMatchBoardColor(t *testing.T) {
	mWhiteFirst := round.Match[string, string]{
		Home:              "A",
		Away:              id("B"),
		WhiteOnFirstBoard: true,
	}
	mBlackFirst := round.Match[string, string]{
		Home:              "A",
		Away:              id("B"),
		WhiteOnFirstBoard: false,
	}
	mBye := round.Match[string, string]{
		Home:              "A",
		WhiteOnFirstBoard: true,
	}

	cases := []struct {
		name  string
		m     round.Match[string, string]
		team  string
		board int
		want  round.Color
	}{
		{"home white on board 1", mWhiteFirst, "A", 1, round.ColorWhite},
		{"home black on board 2", mWhiteFirst, "A", 2, round.ColorBlack},
		{"home white on board 3", mWhiteFirst, "A", 3, round.ColorWhite},
		{"away black on board 1", mWhiteFirst, "B", 1, round.ColorBlack},
		{"away white on board 2", mWhiteFirst, "B", 2, round.ColorWhite},
		{"home black on board 1 when WhiteOnFirstBoard=false", mBlackFirst, "A", 1, round.ColorBlack},
		{"home white on board 2 when WhiteOnFirstBoard=false", mBlackFirst, "A", 2, round.ColorWhite},
		{"away white on board 1 when WhiteOnFirstBoard=false", mBlackFirst, "B", 1, round.ColorWhite},
		{"away black on board 2 when WhiteOnFirstBoard=false", mBlackFirst, "B", 2, round.ColorBlack},
		{"unknown team", mWhiteFirst, "C", 1, ""},
		{"invalid board 0", mWhiteFirst, "A", 0, ""},
		{"negative board", mWhiteFirst, "A", -1, ""},
		{"bye match", mBye, "A", 1, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.m.BoardColor(c.team, c.board); got != c.want {
				t.Errorf("BoardColor(%q, %d) = %q, want %q", c.team, c.board, got, c.want)
			}
		})
	}
}

func TestMatchGamePoints(t *testing.T) {
	// 4-board match where Home ("A") has White on board 1 (boards 1, 3) and Black on boards 2, 4:
	// Board 1: A1 (W) vs B1 (B) -> 1-0     (A: 1, B: 0)
	// Board 2: B2 (W) vs A2 (B) -> 0-1     (A: 1, B: 0)
	// Board 3: A3 (W) vs B3 (B) -> 0.5-0.5 (A: 0.5, B: 0.5)
	// Board 4: B4 (W) vs A4 (B) -> 1-0     (A: 0, B: 1)
	// Total: A = 2.5, B = 1.5
	boards := []round.Game[string]{
		{Round: 1, Board: 1, White: "A1", Black: id("B1"), Result: code(result.WhiteWins)},
		{Round: 1, Board: 2, White: "B2", Black: id("A2"), Result: code(result.BlackWins)},
		{Round: 1, Board: 3, White: "A3", Black: id("B3"), Result: code(result.Draw)},
		{Round: 1, Board: 4, White: "B4", Black: id("A4"), Result: code(result.WhiteWins)},
	}

	m := round.Match[string, string]{
		Round:             1,
		Table:             1,
		Home:              "A",
		Away:              id("B"),
		WhiteOnFirstBoard: true,
		Boards:            boards,
	}

	homePts, awayPts := m.GamePoints()
	if homePts != 2.5 || awayPts != 1.5 {
		t.Errorf("GamePoints() = (%v, %v), want (2.5, 1.5)", homePts, awayPts)
	}
	if got := m.TeamGamePoints("A"); got != 2.5 {
		t.Errorf("TeamGamePoints(A) = %v, want 2.5", got)
	}
	if got := m.TeamGamePoints("B"); got != 1.5 {
		t.Errorf("TeamGamePoints(B) = %v, want 1.5", got)
	}
	if got := m.TeamGamePoints("C"); got != 0 {
		t.Errorf("TeamGamePoints(C) = %v, want 0", got)
	}

	// Flip WhiteOnFirstBoard to false (with the same physical board results, now Away had White on 1, 3):
	mFlipped := m
	mFlipped.WhiteOnFirstBoard = false
	homeFlipped, awayFlipped := mFlipped.GamePoints()
	if homeFlipped != 1.5 || awayFlipped != 2.5 {
		t.Errorf("flipped GamePoints() = (%v, %v), want (1.5, 2.5)", homeFlipped, awayFlipped)
	}

	// Fallback to slice index when Board == 0, and ignore unplayed/void boards:
	mImplicit := round.Match[string, string]{
		Home:              "A",
		Away:              id("B"),
		WhiteOnFirstBoard: true,
		Boards: []round.Game[string]{
			{White: "A1", Black: id("B1"), Result: code(result.WhiteWins)}, // implicit board 1: A gets 1
			{White: "B2", Black: id("A2"), Result: code(result.Unplayed)},  // unplayed: 0
			{White: "A3", Black: id("B3"), Result: code(result.Void)},      // void: 0
			{White: "B4", Black: id("A4")},                                 // nil result: 0
		},
	}
	if h, a := mImplicit.GamePoints(); h != 1.0 || a != 0.0 {
		t.Errorf("implicit boards GamePoints() = (%v, %v), want (1.0, 0.0)", h, a)
	}

	// Voided match or bye returns (0, 0):
	mVoid := m
	mVoid.Result = code(result.Void)
	if h, a := mVoid.GamePoints(); h != 0 || a != 0 {
		t.Errorf("void match GamePoints() = (%v, %v), want (0, 0)", h, a)
	}
	mBye := round.Match[string, string]{Home: "A", Result: code(result.WhiteWins)}
	if h, a := mBye.GamePoints(); h != 0 || a != 0 {
		t.Errorf("bye match GamePoints() = (%v, %v), want (0, 0)", h, a)
	}
}

func TestMatchSettled(t *testing.T) {
	cases := []struct {
		name string
		m    round.Match[string, string]
		want bool
	}{
		{
			name: "all boards settled",
			m: round.Match[string, string]{
				Home: "A",
				Away: id("B"),
				Boards: []round.Game[string]{
					{Board: 1, White: "A1", Black: id("B1"), Result: code(result.WhiteWins)},
					{Board: 2, White: "B2", Black: id("A2"), Result: code(result.Draw)},
				},
			},
			want: true,
		},
		{
			name: "one board nil result",
			m: round.Match[string, string]{
				Home: "A",
				Away: id("B"),
				Boards: []round.Game[string]{
					{Board: 1, White: "A1", Black: id("B1"), Result: code(result.WhiteWins)},
					{Board: 2, White: "B2", Black: id("A2"), Result: nil},
				},
			},
			want: false,
		},
		{
			name: "one board unplayed",
			m: round.Match[string, string]{
				Home: "A",
				Away: id("B"),
				Boards: []round.Game[string]{
					{Board: 1, White: "A1", Black: id("B1"), Result: code(result.WhiteWins)},
					{Board: 2, White: "B2", Black: id("A2"), Result: code(result.Unplayed)},
				},
			},
			want: false,
		},
		{
			name: "no boards and no match result",
			m: round.Match[string, string]{
				Home: "A",
				Away: id("B"),
			},
			want: false,
		},
		{
			name: "match-level forfeit without boards",
			m: round.Match[string, string]{
				Home:   "A",
				Away:   id("B"),
				Result: code(result.WhiteForfeit),
			},
			want: true,
		},
		{
			name: "full-point bye",
			m: round.Match[string, string]{
				Home:   "A",
				Result: code(result.WhiteWins),
			},
			want: true,
		},
		{
			name: "bye without result",
			m: round.Match[string, string]{
				Home: "A",
			},
			want: false,
		},
		{
			name: "voided match",
			m: round.Match[string, string]{
				Home:   "A",
				Away:   id("B"),
				Result: code(result.Void),
			},
			want: false,
		},
		{
			name: "all boards void",
			m: round.Match[string, string]{
				Home: "A",
				Away: id("B"),
				Boards: []round.Game[string]{
					{Board: 1, White: "A1", Black: id("B1"), Result: code(result.Void)},
				},
			},
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.m.Settled(); got != c.want {
				t.Errorf("Settled() = %v, want %v", got, c.want)
			}
		})
	}
}
