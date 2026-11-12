package round_test

import (
	"testing"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

func code(c result.Code) *result.Code { return &c }
func id(s string) *string             { return &s }

func TestPoints(t *testing.T) {
	cases := []struct {
		name         string
		g            round.Game[string]
		white, black float64
	}{
		{"white wins", round.Game[string]{White: "w", Black: id("b"), Result: code(result.WhiteWins)}, 1, 0},
		{"black wins", round.Game[string]{White: "w", Black: id("b"), Result: code(result.BlackWins)}, 0, 1},
		{"draw", round.Game[string]{White: "w", Black: id("b"), Result: code(result.Draw)}, 0.5, 0.5},
		{"white forfeit win", round.Game[string]{White: "w", Black: id("b"), Result: code(result.WhiteForfeit)}, 1, 0},
		{"black forfeit win", round.Game[string]{White: "w", Black: id("b"), Result: code(result.BlackForfeit)}, 0, 1},
		{"half point to black", round.Game[string]{White: "w", Black: id("b"), Result: code(result.BlackHalf)}, 0, 0.5},
		{"unplayed", round.Game[string]{White: "w", Black: id("b"), Result: code(result.Unplayed)}, 0, 0},
		{"void", round.Game[string]{White: "w", Black: id("b"), Result: code(result.Void)}, 0, 0},
		{"no result yet", round.Game[string]{White: "w", Black: id("b")}, 0, 0},
		{"full-point bye", round.Game[string]{White: "w", Result: code(result.WhiteWins)}, 1, 0},
		{"half-point bye", round.Game[string]{White: "w", Result: code(result.WhiteHalf)}, 0.5, 0},
		{"zero-point bye", round.Game[string]{White: "w", Result: code(result.ZeroPointBye)}, 0, 0},
		{"lost bye (safeguard)", round.Game[string]{White: "w", Result: code(result.BlackWins)}, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.Points("w"); got != c.white {
				t.Errorf("white: got %v, want %v", got, c.white)
			}
			if got := c.g.Points("b"); got != c.black {
				t.Errorf("black: got %v, want %v", got, c.black)
			}
			if got := c.g.Points("someone else"); got != 0 {
				t.Errorf("non-participant: got %v, want 0", got)
			}
		})
	}
}

func TestIsBye(t *testing.T) {
	if !(round.Game[string]{White: "w"}).IsBye() {
		t.Error("game without black should be a bye")
	}
	if (round.Game[string]{White: "w", Black: id("b")}).IsBye() {
		t.Error("game with black should not be a bye")
	}
}
