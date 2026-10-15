package pairing_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/ValkoM28/tournamentessentials/core/pairing"
)

type settings = pairing.Settings[string]

func TestValidate_ValidSettings(t *testing.T) {
	cases := map[string]settings{
		"empty": {},
		"full": {
			WithdrawnPlayers:  []string{"a"},
			HalfPointByes:     []string{"b"},
			LateEntries:       []string{"c"},
			FirstPlayerColor:  "black",
			Acceleration:      "baku",
			ForbiddenPairings: []pairing.ForbiddenPair[string]{{Player1: "c", Player2: "d"}},
			FixedBoards:       []pairing.FixedBoard[string]{{BoardNumber: 1, Player: "d"}, {BoardNumber: 3, Player: "e"}},
		},
		"explicit none values": {FirstPlayerColor: "", Acceleration: "none"},
		"white and dutch":      {FirstPlayerColor: "white", Acceleration: "dutch"},
	}
	for name, s := range cases {
		if err := s.Validate(); err != nil {
			t.Errorf("%s: got %v, want nil", name, err)
		}
	}
}

func TestValidate_Errors(t *testing.T) {
	fb := func(n int, p string) pairing.FixedBoard[string] {
		return pairing.FixedBoard[string]{BoardNumber: n, Player: p}
	}
	cases := []struct {
		name string
		s    settings
		kind error
		msg  string
	}{
		{"invalid colour", settings{FirstPlayerColor: "red"},
			pairing.ErrInvalidSettings, "invalid pairing settings: first_player_color must be 'white' or 'black'"},
		{"invalid acceleration", settings{Acceleration: "fast"},
			pairing.ErrInvalidSettings, "invalid pairing settings: acceleration must be 'none', 'baku', or 'dutch'"},
		{"forbidden pair with one player", settings{ForbiddenPairings: []pairing.ForbiddenPair[string]{{Player1: "a", Player2: "a"}}},
			pairing.ErrInvalidSettings, "invalid pairing settings: forbidden pairing cannot have the same player on both sides"},
		{"fixed board below 1", settings{FixedBoards: []pairing.FixedBoard[string]{fb(0, "a")}},
			pairing.ErrInvalidSettings, "invalid pairing settings: fixed board number must be >= 1"},
		{"duplicate fixed board number", settings{FixedBoards: []pairing.FixedBoard[string]{fb(2, "a"), fb(2, "b")}},
			pairing.ErrInvalidSettings, "invalid pairing settings: duplicate fixed board number 2"},
		{"player on two fixed boards", settings{FixedBoards: []pairing.FixedBoard[string]{fb(1, "a"), fb(2, "a")}},
			pairing.ErrInvalidSettings, "invalid pairing settings: player appears in multiple fixed boards"},
		{"duplicate withdrawn", settings{WithdrawnPlayers: []string{"a", "a"}},
			pairing.ErrInvalidSettings, "invalid pairing settings: duplicate player in withdrawn list"},
		{"duplicate half-point bye", settings{HalfPointByes: []string{"a", "a"}},
			pairing.ErrInvalidSettings, "invalid pairing settings: duplicate player in half-point bye list"},
		{"duplicate late entry", settings{LateEntries: []string{"a", "a"}},
			pairing.ErrInvalidSettings, "invalid pairing settings: duplicate player in late entries list"},
		{"withdrawn and half-point bye", settings{WithdrawnPlayers: []string{"a"}, HalfPointByes: []string{"a"}},
			pairing.ErrPlayerConflict, "player appears in conflicting settings lists: player cannot be both withdrawn and on half-point bye"},
		{"withdrawn and fixed board", settings{WithdrawnPlayers: []string{"a"}, FixedBoards: []pairing.FixedBoard[string]{fb(1, "a")}},
			pairing.ErrPlayerConflict, "player appears in conflicting settings lists: player cannot be both withdrawn and on a fixed board"},
		{"half-point bye and fixed board", settings{HalfPointByes: []string{"a"}, FixedBoards: []pairing.FixedBoard[string]{fb(1, "a")}},
			pairing.ErrPlayerConflict, "player appears in conflicting settings lists: player cannot be both on half-point bye and on a fixed board"},

		// Order: earlier checks win when several fail.
		{"colour before acceleration", settings{FirstPlayerColor: "red", Acceleration: "fast"},
			pairing.ErrInvalidSettings, "invalid pairing settings: first_player_color must be 'white' or 'black'"},
		{"duplicates before overlaps", settings{WithdrawnPlayers: []string{"a", "a"}, HalfPointByes: []string{"a"}},
			pairing.ErrInvalidSettings, "invalid pairing settings: duplicate player in withdrawn list"},
		{"fixed boards before duplicates", settings{FixedBoards: []pairing.FixedBoard[string]{fb(0, "b")}, WithdrawnPlayers: []string{"a", "a"}},
			pairing.ErrInvalidSettings, "invalid pairing settings: fixed board number must be >= 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.s.Validate()
			if !errors.Is(err, c.kind) {
				t.Fatalf("got %v, want %v", err, c.kind)
			}
			if err.Error() != c.msg {
				t.Errorf("message:\n got  %q\n want %q", err.Error(), c.msg)
			}
		})
	}
}

func TestReferencedPlayers(t *testing.T) {
	s := settings{
		WithdrawnPlayers:  []string{"w"},
		HalfPointByes:     []string{"h", "w"},
		LateEntries:       []string{"l"},
		ForbiddenPairings: []pairing.ForbiddenPair[string]{{Player1: "f1", Player2: "h"}, {Player1: "f2", Player2: "f1"}},
		FixedBoards:       []pairing.FixedBoard[string]{{BoardNumber: 1, Player: "x"}, {BoardNumber: 2, Player: "l"}},
	}
	want := []string{"w", "h", "l", "f1", "f2", "x"}
	if got := s.ReferencedPlayers(); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := (settings{}).ReferencedPlayers(); len(got) != 0 {
		t.Errorf("empty settings: got %v, want none", got)
	}
}
