package pairing_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tessechess/core/engine"
	"github.com/tessechess/core/pairing"
	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

// roundTwo is fourPlayers after a played round 1 (a–b 1-0, c–d draw).
func roundTwo(settings *pairing.Settings[string]) pairing.Input[string] {
	in := fourPlayers()
	in.CompletedRounds = 1
	in.History = []round.Game[string]{
		{Round: 1, Board: 1, White: "a", Black: strPtr("b"), Result: codePtr(result.WhiteWins)},
		{Round: 1, Board: 2, White: "c", Black: strPtr("d"), Result: codePtr(result.Draw)},
	}
	in.Settings = settings
	return in
}

// assertResults checks the Round and Result of every game; nil means no result.
func assertResults(t *testing.T, got []round.Game[string], wantRound int, want ...*result.Code) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d games, want %d", len(got), len(want))
	}
	for i, g := range got {
		if g.Round != wantRound {
			t.Errorf("game %d: round %d, want %d", i, g.Round, wantRound)
		}
		switch {
		case want[i] == nil && g.Result != nil:
			t.Errorf("game %d: result %q, want none", i, *g.Result)
		case want[i] != nil && (g.Result == nil || *g.Result != *want[i]):
			t.Errorf("game %d: result %v, want %q", i, g.Result, *want[i])
		}
	}
}

func TestPair_PreassignedByesAreReturnedWithResults(t *testing.T) {
	in := roundTwo(&pairing.Settings[string]{
		WithdrawnPlayers: []string{"c"},
		HalfPointByes:    []string{"d"},
	})

	boards, err := pairing.Pair(context.Background(), simplePairs("1\n1 2\n"), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "a", strPtr("b")), board(2, "c", nil), board(3, "d", nil))
	assertResults(t, boards, 2, nil, codePtr(result.ZeroPointBye), codePtr(result.WhiteHalf))
}

func TestPair_PreassignedByesInStartingRankOrder(t *testing.T) {
	// Withdrawn players listed out of order still get boards by starting rank.
	in := roundTwo(&pairing.Settings[string]{WithdrawnPlayers: []string{"d", "a"}})

	boards, err := pairing.Pair(context.Background(), simplePairs("1\n2 3\n"), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "b", strPtr("c")), board(2, "a", nil), board(3, "d", nil))
}

func TestPair_EngineByeKeepsNilResult(t *testing.T) {
	// d is withdrawn; the engine gives c the pairing-allocated bye.
	in := roundTwo(&pairing.Settings[string]{WithdrawnPlayers: []string{"d"}})

	boards, err := pairing.Pair(context.Background(), simplePairs("2\n1 2\n3 0\n"), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "a", strPtr("b")), board(2, "c", nil), board(3, "d", nil))
	assertResults(t, boards, 2, nil, nil, codePtr(result.ZeroPointBye))
}

func TestPair_ByesAfterFixedBoards(t *testing.T) {
	in := roundTwo(&pairing.Settings[string]{
		WithdrawnPlayers: []string{"d"},
		FixedBoards:      []pairing.FixedBoard[string]{{BoardNumber: 5, Player: "a"}},
	})

	boards, err := pairing.Pair(context.Background(), simplePairs("2\n1 2\n3 0\n"), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "c", nil), board(5, "a", strPtr("b")), board(6, "d", nil))
}

func TestPair_PlayerLevelFixedBoardSkippedForBye(t *testing.T) {
	in := roundTwo(&pairing.Settings[string]{WithdrawnPlayers: []string{"a"}})
	in.Players[0].FixedBoard = intPtr(1) // a would sit on board 1, but is withdrawn
	in.Players[3].FixedBoard = intPtr(2) // d still moves to board 2

	boards, err := pairing.Pair(context.Background(), simplePairs("2\n2 0\n3 4\n"), in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "b", nil), board(2, "c", strPtr("d")), board(3, "a", nil))
}

func TestPair_EnginePairingAPlayerWithByeFails(t *testing.T) {
	in := roundTwo(&pairing.Settings[string]{WithdrawnPlayers: []string{"c"}})

	for _, out := range []string{"2\n1 2\n3 4\n", "2\n1 2\n4 3\n"} {
		_, err := pairing.Pair(context.Background(), simplePairs(out), in)
		if !errors.Is(err, pairing.ErrTRFParsing) || !strings.Contains(err.Error(), "pre-assigned bye") {
			t.Errorf("output %q: got %v, want ErrTRFParsing about a pre-assigned bye", out, err)
		}
	}
}

func TestPair_TRFOutputEchoingPreassignedByes(t *testing.T) {
	// A TRF answer repeats the pre-assigned byes of its input (c: Z, d: H) next
	// to the new pairing a–b; they must not come back as engine byes.
	pad := func(line string) string { return line + strings.Repeat(" ", 91-len(line)) }
	out := strings.Join([]string{
		"012 Test Open",
		pad("001    1      Alpha, Anna                       2400") + "   2 w 1     2 w  ",
		pad("001    2      Bravo, Boris                      2300") + "   1 b 0     1 b  ",
		pad("001    3      Charlie, Cyril                    2200") + "   4 w =  0000 - Z",
		pad("001    4      Delta, Dana                       2100") + "   3 b =  0000 - H",
	}, "\n")
	eng := &fakeEngine{format: engine.OutputTRF, output: out}
	in := roundTwo(&pairing.Settings[string]{
		WithdrawnPlayers: []string{"c"},
		HalfPointByes:    []string{"d"},
	})

	boards, err := pairing.Pair(context.Background(), eng, in)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	assertBoards(t, boards, board(1, "a", strPtr("b")), board(2, "c", nil), board(3, "d", nil))
	assertResults(t, boards, 2, nil, codePtr(result.ZeroPointBye), codePtr(result.WhiteHalf))
}

func TestPrepare_SavedByeInHistoryIsZInTRF(t *testing.T) {
	// The bye game Pair returned for round 1 is written as Z in round 2's TRF.
	in := fourPlayers()
	in.CompletedRounds = 1
	in.History = []round.Game[string]{
		{Round: 1, Board: 1, White: "a", Black: strPtr("b"), Result: codePtr(result.WhiteWins)},
		{Round: 1, Board: 2, White: "c", Result: codePtr(result.ZeroPointBye)},
		{Round: 1, Board: 3, White: "d", Result: codePtr(result.WhiteHalf)},
	}
	in.Settings = nil

	p, err := pairing.Prepare(in)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	want := map[string]string{"3": "0000 - Z", "4": "0000 - H"}
	for _, line := range strings.Split(p.TRF(), "\n") {
		if !strings.HasPrefix(line, "001") {
			continue
		}
		rank := strings.TrimSpace(line[4:8])
		w, ok := want[rank]
		if !ok {
			continue
		}
		if len(line) < 99 || line[91:99] != w {
			t.Errorf("rank %s: want %q in round 1, line %q", rank, w, line)
		}
		delete(want, rank)
	}
	if len(want) > 0 {
		t.Errorf("TRF has no line for ranks %v\n%s", want, p.TRF())
	}
}
