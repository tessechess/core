package result

import (
	"slices"
	"testing"
)

func TestCodes(t *testing.T) {
	tests := []struct {
		code         Code
		white, black float64
		valid        bool
		forfeit      bool
		counts       bool
	}{
		{WhiteWins, 1, 0, true, false, true},
		{BlackWins, 0, 1, true, false, true},
		{Draw, 0.5, 0.5, true, false, true},
		{WhiteForfeit, 1, 0, true, true, true},
		{BlackForfeit, 0, 1, true, true, true},
		{WhiteHalf, 0.5, 0, true, false, true},
		{BlackHalf, 0, 0.5, true, false, true},
		{ZeroPointBye, 0, 0, true, false, true},
		{Unplayed, 0, 0, true, false, false},
		{Void, 0, 0, true, false, false},
		{Code("2-0"), 0, 0, false, false, true},
		{Code(""), 0, 0, false, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			w, b := tt.code.Points()
			if w != tt.white || b != tt.black {
				t.Errorf("Points() = (%v, %v), want (%v, %v)", w, b, tt.white, tt.black)
			}
			if got := tt.code.Valid(); got != tt.valid {
				t.Errorf("Valid() = %v, want %v", got, tt.valid)
			}
			if got := tt.code.IsForfeit(); got != tt.forfeit {
				t.Errorf("IsForfeit() = %v, want %v", got, tt.forfeit)
			}
			if got := tt.code.Counts(); got != tt.counts {
				t.Errorf("Counts() = %v, want %v", got, tt.counts)
			}
		})
	}
}

func TestReportable(t *testing.T) {
	got := Reportable()
	want := []Code{WhiteWins, BlackWins, Draw, WhiteForfeit, BlackForfeit, WhiteHalf, BlackHalf}
	if !slices.Equal(got, want) {
		t.Errorf("Reportable() = %v, want %v", got, want)
	}
	for _, c := range got {
		if !c.Valid() {
			t.Errorf("submittable code %q is not Valid()", c)
		}
		if !c.Counts() {
			t.Errorf("submittable code %q does not count", c)
		}
	}
}

func TestReportableReturnsCopy(t *testing.T) {
	s := Reportable()
	s[0] = Void
	if Reportable()[0] != WhiteWins {
		t.Error("modifying the returned slice changed later results")
	}
}

func TestPointsSumForPlayedGames(t *testing.T) {
	for _, c := range []Code{WhiteWins, BlackWins, Draw, WhiteForfeit, BlackForfeit} {
		w, b := c.Points()
		if w+b != 1 {
			t.Errorf("%q: white+black = %v, want 1", c, w+b)
		}
	}
}
