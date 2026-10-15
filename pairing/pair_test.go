package pairing_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ValkoM28/tournamentessentials/core/engine"
	"github.com/ValkoM28/tournamentessentials/core/pairing"
	"github.com/ValkoM28/tournamentessentials/core/player"
	"github.com/ValkoM28/tournamentessentials/core/result"
	"github.com/ValkoM28/tournamentessentials/core/round"
)

// fakeEngine records the TRF it receives and returns a canned answer.
type fakeEngine struct {
	format engine.OutputFormat
	output string
	err    error
	input  string
}

func (f *fakeEngine) Name() string                      { return "fake" }
func (f *fakeEngine) OutputFormat() engine.OutputFormat { return f.format }
func (f *fakeEngine) GeneratePairings(_ context.Context, in string) (string, error) {
	f.input = in
	return f.output, f.err
}

func simplePairs(out string) *fakeEngine {
	return &fakeEngine{format: engine.OutputSimplePairs, output: out}
}

func intPtr(v int) *int                  { return &v }
func strPtr(s string) *string            { return &s }
func codePtr(c result.Code) *result.Code { return &c }

// fourPlayers is a minimal input: players a..d with starting ranks 1..4.
func fourPlayers() pairing.Input[string] {
	return pairing.Input[string]{
		Tournament: pairing.Tournament{
			Name:        "Test Open",
			City:        "Bratislava",
			Federation:  "SVK",
			TimeControl: "90+30",
			StartDate:   time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			EndDate:     time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
			TotalRounds: 5,
		},
		Players: []pairing.Player[string]{
			{ID: "a", StartingRank: 1, FirstName: "Anna", LastName: "Alpha", Rating: 2400},
			{ID: "b", StartingRank: 2, FirstName: "Boris", LastName: "Bravo", Rating: 2300},
			{ID: "c", StartingRank: 3, FirstName: "Cyril", LastName: "Charlie", Rating: 2200},
			{ID: "d", StartingRank: 4, FirstName: "Dana", LastName: "Delta", Rating: 2100},
		},
	}
}

func board(n int, white string, black *string) round.Game[string] {
	return round.Game[string]{Board: n, White: white, Black: black}
}

func assertBoards(t *testing.T, got []round.Game[string], want ...round.Game[string]) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d boards %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		g, w := got[i], want[i]
		blackOK := (g.Black == nil && w.Black == nil) || (g.Black != nil && w.Black != nil && *g.Black == *w.Black)
		if g.Board != w.Board || g.White != w.White || !blackOK {
			t.Errorf("board %d: got {%d %s %v}, want {%d %s %v}", i, g.Board, g.White, deref(g.Black), w.Board, w.White, deref(w.Black))
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return "<bye>"
	}
	return *s
}

// ── Success paths ───────────────────────────────────────────

func TestPair_SimplePairsMapsRanksToIDs(t *testing.T) {
	eng := simplePairs("2\n1 3\n4 2\n")

	boards, err := pairing.Pair(context.Background(), eng, fourPlayers())
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "a", strPtr("c")), board(2, "d", strPtr("b")))
}

func TestPair_ByeHasNilBlack(t *testing.T) {
	in := fourPlayers()
	in.Players = in.Players[:3]
	eng := simplePairs("2\n1 2\n3 0\n")

	boards, err := pairing.Pair(context.Background(), eng, in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "a", strPtr("b")), board(2, "c", nil))
}

func TestPair_TRFOutputFormat(t *testing.T) {
	// A TRF output with round 1 appended: 1 (white) vs 2, 3 (white) vs 4.
	pad := func(line string) string { return line + strings.Repeat(" ", 91-len(line)) }
	out := strings.Join([]string{
		"012 Test Open",
		pad("001    1      Alpha, Anna                       2400") + "   2 w  ",
		pad("001    2      Bravo, Boris                      2300") + "   1 b  ",
		pad("001    3      Charlie, Cyril                    2200") + "   4 w  ",
		pad("001    4      Delta, Dana                       2100") + "   3 b  ",
	}, "\n")
	eng := &fakeEngine{format: engine.OutputTRF, output: out}

	boards, err := pairing.Pair(context.Background(), eng, fourPlayers())
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "a", strPtr("b")), board(2, "c", strPtr("d")))
}

// ── TRF input ───────────────────────────────────────────────

func TestPrepare_TRFContents(t *testing.T) {
	in := fourPlayers()
	bday := time.Date(1990, 11, 30, 0, 0, 0, 0, time.UTC)
	in.Players[0].Birthday = &bday
	in.Players[0].Title = player.GM
	in.Players[0].Federation = "NOR"
	in.Players[0].FIDENumber = "1503014"
	in.Players[0].Sex = player.SexMale
	in.Players[1].Sex = player.SexFemale
	in.Settings = &pairing.Settings[string]{
		FirstPlayerColor:  "white",
		ForbiddenPairings: []pairing.ForbiddenPair[string]{{Player1: "b", Player2: "d"}},
	}

	p, err := pairing.Prepare(in)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	got := p.TRF()
	for _, want := range []string{
		"012 Test Open", "022 Bratislava", "032 SVK", "042 2026/03/20", "052 2026/03/22",
		"062 4", "122 90+30", "XXR 5", "XXC white1", "XXF 2 4",
		"Alpha, Anna", "1990/11/30", "Delta, Dana",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("TRF missing %q\n%s", want, got)
		}
	}

	// The sex column (position 10) must use TRF's m/w, not the stored M/F.
	for rank, want := range map[string]byte{"   1": 'm', "   2": 'w', "   3": ' '} {
		var line string
		for _, l := range strings.Split(got, "\n") {
			if len(l) > 10 && l[:3] == "001" && l[4:8] == rank {
				line = l
			}
		}
		if line == "" {
			t.Fatalf("TRF has no 001 line for rank %q\n%s", rank, got)
		}
		if line[9] != want {
			t.Errorf("rank %q: sex column = %q, want %q", rank, line[9], want)
		}
	}
}

func TestPrepare_ByeSettingsAndHistory(t *testing.T) {
	in := fourPlayers()
	in.History = []round.Game[string]{
		{Round: 1, White: "a", Black: strPtr("b"), Result: codePtr(result.WhiteWins)},
		{Round: 1, White: "c", Black: strPtr("d"), Result: codePtr(result.Draw)},
	}
	in.CompletedRounds = 1
	in.Settings = &pairing.Settings[string]{
		WithdrawnPlayers: []string{"c"},
		HalfPointByes:    []string{"d"},
	}

	p, err := pairing.Prepare(in)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	lines := map[string]string{}
	for _, line := range strings.Split(p.TRF(), "\n") {
		if strings.HasPrefix(line, "001") {
			lines[strings.TrimSpace(line[4:8])] = line
		}
	}
	// Round 1 (history) then round 2 (pre-assigned byes); TRF round blocks start at index 91.
	checks := []struct {
		rank, round string
		want        string
	}{
		{"1", "r1", "   2 w 1"},
		{"3", "r1", "   4 w ="},
		{"3", "r2", "0000 - Z"},
		{"4", "r2", "0000 - H"},
	}
	for _, c := range checks {
		line := lines[c.rank]
		offset := 91
		if c.round == "r2" {
			offset = 101
		}
		if len(line) < offset+8 || line[offset:offset+8] != c.want {
			t.Errorf("rank %s %s: want %q in line %q", c.rank, c.round, c.want, line)
		}
	}
}

// ── Errors ──────────────────────────────────────────────────

func TestPair_UnknownRank(t *testing.T) {
	for _, out := range []string{"1\n99 2\n", "1\n1 99\n"} {
		_, err := pairing.Pair(context.Background(), simplePairs(out), fourPlayers())
		if !errors.Is(err, pairing.ErrUnknownEngineRank) {
			t.Errorf("output %q: got %v, want ErrUnknownEngineRank", out, err)
		}
	}
}

func TestPair_EngineErrorPassedThrough(t *testing.T) {
	eng := &fakeEngine{format: engine.OutputSimplePairs, err: engine.ErrEngineTimeout}
	_, err := pairing.Pair(context.Background(), eng, fourPlayers())
	if !errors.Is(err, engine.ErrEngineTimeout) {
		t.Fatalf("got %v, want ErrEngineTimeout", err)
	}
	if errors.Is(err, pairing.ErrTRFParsing) {
		t.Error("engine errors must not be wrapped as parse errors")
	}
}

func TestPair_UnparsableOutput(t *testing.T) {
	_, err := pairing.Pair(context.Background(), simplePairs("not a number\n"), fourPlayers())
	if !errors.Is(err, pairing.ErrTRFParsing) {
		t.Fatalf("got %v, want ErrTRFParsing", err)
	}
}

func TestPair_UnknownOutputFormat(t *testing.T) {
	eng := &fakeEngine{format: "xml", output: "<pairs/>"}
	_, err := pairing.Pair(context.Background(), eng, fourPlayers())
	if !errors.Is(err, pairing.ErrTRFParsing) || !strings.Contains(err.Error(), "unknown engine output format") {
		t.Fatalf("got %v, want ErrTRFParsing about the output format", err)
	}
}

func TestPrepare_MissingStartingRank(t *testing.T) {
	in := fourPlayers()
	in.Players[2].StartingRank = 0
	_, err := pairing.Prepare(in)
	if !errors.Is(err, pairing.ErrTRFGeneration) {
		t.Fatalf("got %v, want ErrTRFGeneration", err)
	}
}

// ── Fixed boards ────────────────────────────────────────────

// The engine pairs 1–2 on board 1 and 3–4 on board 2 in all fixed-board tests.
const twoBoards = "2\n1 2\n3 4\n"

func TestPair_RoundLevelFixedBoard(t *testing.T) {
	in := fourPlayers()
	in.Settings = &pairing.Settings[string]{FixedBoards: []pairing.FixedBoard[string]{{BoardNumber: 1, Player: "c"}}}

	boards, err := pairing.Pair(context.Background(), simplePairs(twoBoards), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "c", strPtr("d")), board(2, "a", strPtr("b")))
}

func TestPair_PlayerLevelFixedBoard(t *testing.T) {
	in := fourPlayers()
	in.Players[3].FixedBoard = intPtr(1) // d plays black on board 2 by default

	boards, err := pairing.Pair(context.Background(), simplePairs(twoBoards), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "c", strPtr("d")), board(2, "a", strPtr("b")))
}

func TestPair_RoundLevelFixedBoardWinsOverPlayerLevel(t *testing.T) {
	in := fourPlayers()
	in.Players[0].FixedBoard = intPtr(1)                                                                              // player-level: a on board 1
	in.Settings = &pairing.Settings[string]{FixedBoards: []pairing.FixedBoard[string]{{BoardNumber: 5, Player: "a"}}} // round-level: a on board 5

	boards, err := pairing.Pair(context.Background(), simplePairs(twoBoards), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "c", strPtr("d")), board(5, "a", strPtr("b")))
}

func TestPrepare_DoesNotModifyInputSettings(t *testing.T) {
	in := fourPlayers()
	in.Players[1].FixedBoard = intPtr(3)
	settings := &pairing.Settings[string]{FixedBoards: []pairing.FixedBoard[string]{{BoardNumber: 1, Player: "c"}}}
	in.Settings = settings

	if _, err := pairing.Prepare(in); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(settings.FixedBoards) != 1 {
		t.Errorf("caller's settings changed: %+v", settings.FixedBoards)
	}
}

func TestApplyFixedBoards_RenumbersAroundFixed(t *testing.T) {
	boards := []round.Game[string]{
		board(1, "a", strPtr("b")),
		board(2, "c", strPtr("d")),
		board(3, "e", strPtr("f")),
		board(4, "g", nil),
	}
	got := pairing.ApplyFixedBoards(boards, []pairing.FixedBoard[string]{{BoardNumber: 2, Player: "f"}})
	assertBoards(t, got,
		board(1, "a", strPtr("b")),
		board(2, "e", strPtr("f")),
		board(3, "c", strPtr("d")),
		board(4, "g", nil),
	)
}

func TestApplyFixedBoards_NoFixedPlayersPresent(t *testing.T) {
	boards := []round.Game[string]{board(1, "a", strPtr("b")), board(2, "c", nil)}
	got := pairing.ApplyFixedBoards(boards, []pairing.FixedBoard[string]{{BoardNumber: 1, Player: "zz"}})
	assertBoards(t, got, board(1, "a", strPtr("b")), board(2, "c", nil))
}
