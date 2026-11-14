package pairing_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tessechess/core/engine"
	"github.com/tessechess/core/pairing"
	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

func fourTeamsInput() pairing.TeamInput[string, string] {
	return pairing.TeamInput[string, string]{
		Tournament: pairing.Tournament{
			Name:        "Team Open",
			City:        "Košice",
			Federation:  "SVK",
			TimeControl: "90+30",
			StartDate:   time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
			EndDate:     time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
			TotalRounds: 5,
		},
		BoardsPerMatch: 2,
		Teams: []pairing.Team[string, string]{
			{
				ID:           "A",
				StartingRank: 1,
				Name:         "Team Alpha",
				Federation:   "SVK",
				Players:      []entrant{{ID: "A1", Rating: 2400}, {ID: "A2", Rating: 2200}}, // avg 2300
			},
			{
				ID:           "B",
				StartingRank: 2,
				Name:         "Team Bravo",
				Federation:   "CZE",
				Players:      []entrant{{ID: "B1", Rating: 2300}, {ID: "B2", Rating: 2100}}, // avg 2200
			},
			{
				ID:           "C",
				StartingRank: 3,
				Name:         "Team Charlie",
				Federation:   "POL",
				Players:      []entrant{{ID: "C1", Rating: 2200}, {ID: "C2", Rating: 2000}}, // avg 2100
			},
			{
				ID:           "D",
				StartingRank: 4,
				Name:         "Team Delta",
				Federation:   "HUN",
				Players:      []entrant{{ID: "D1", Rating: 2100}, {ID: "D2", Rating: 1900}}, // avg 2000
			},
		},
		Settings: &pairing.Settings[string]{FirstPlayerColor: round.ColorWhite},
	}
}

func assertMatches(t *testing.T, got []round.Match[string, string], want ...round.Match[string, string]) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d matches %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		g, w := got[i], want[i]
		awayOK := (g.Away == nil && w.Away == nil) || (g.Away != nil && w.Away != nil && *g.Away == *w.Away)
		if g.Round != w.Round || g.Table != w.Table || g.Home != w.Home || !awayOK || g.WhiteOnFirstBoard != w.WhiteOnFirstBoard {
			t.Errorf("match %d: got {Round:%d Table:%d Home:%s Away:%s White1:%v}, want {Round:%d Table:%d Home:%s Away:%s White1:%v}",
				i, g.Round, g.Table, g.Home, deref(g.Away), g.WhiteOnFirstBoard,
				w.Round, w.Table, w.Home, deref(w.Away), w.WhiteOnFirstBoard)
		}
	}
}

func TestPairTeams_SimplePairsAndBye(t *testing.T) {
	eng := simplePairs("2\n1 3\n4 2\n")
	matches, err := pairing.PairTeams(context.Background(), eng, fourTeamsInput())
	if err != nil {
		t.Fatalf("PairTeams: %v", err)
	}
	assertMatches(t, matches,
		round.Match[string, string]{Round: 1, Table: 1, Home: "A", Away: strPtr("C"), WhiteOnFirstBoard: true},
		round.Match[string, string]{Round: 1, Table: 2, Home: "D", Away: strPtr("B"), WhiteOnFirstBoard: true},
	)

	// Odd number of teams (3 teams, rank 3 gets bye):
	inOdd := fourTeamsInput()
	inOdd.Teams = inOdd.Teams[:3]
	engBye := simplePairs("2\n1 2\n3 0\n")
	matchesOdd, err := pairing.PairTeams(context.Background(), engBye, inOdd)
	if err != nil {
		t.Fatalf("PairTeams(odd): %v", err)
	}
	assertMatches(t, matchesOdd,
		round.Match[string, string]{Round: 1, Table: 1, Home: "A", Away: strPtr("B"), WhiteOnFirstBoard: true},
		round.Match[string, string]{Round: 1, Table: 2, Home: "C", Away: nil, WhiteOnFirstBoard: false},
	)
}

func TestPrepareTeams_TRFHistoryAndRatings(t *testing.T) {
	in := fourTeamsInput()
	in.CompletedRounds = 1
	in.Settings = nil
	// Round 1 history:
	// Table 1: A vs C (WhiteOnFirstBoard=true). A wins 1.5 - 0.5 -> A gets 'w 1', C gets 'b 0'.
	// Table 2: B vs D (WhiteOnFirstBoard=false, so D had white on board 1).
	//          Match is 1.0 - 1.0 draw -> D gets 'w =', B gets 'b ='.
	in.History = []round.Match[string, string]{
		{
			Round:             1,
			Table:             1,
			Home:              "A",
			Away:              strPtr("C"),
			WhiteOnFirstBoard: true,
			Boards: []round.Game[string]{
				{Round: 1, Board: 1, White: "A1", Black: strPtr("C1"), Result: codePtr(result.WhiteWins)},
				{Round: 1, Board: 2, White: "C2", Black: strPtr("A2"), Result: codePtr(result.Draw)},
			},
		},
		{
			Round:             1,
			Table:             2,
			Home:              "B",
			Away:              strPtr("D"),
			WhiteOnFirstBoard: false,
			Boards: []round.Game[string]{
				{Round: 1, Board: 1, White: "D1", Black: strPtr("B1"), Result: codePtr(result.WhiteWins)},
				{Round: 1, Board: 2, White: "B2", Black: strPtr("D2"), Result: codePtr(result.WhiteWins)},
			},
		},
	}

	prep, err := pairing.PrepareTeams(in)
	if err != nil {
		t.Fatalf("PrepareTeams: %v", err)
	}
	trfStr := prep.TRF()

	// Check team names and computed team ratings (2300, 2200, 2100, 2000).
	for _, want := range []string{
		"012 Team Open",
		"Team Alpha", "2300",
		"Team Bravo", "2200",
		"Team Charlie", "2100",
		"Team Delta", "2000",
	} {
		if !strings.Contains(trfStr, want) {
			t.Errorf("TRF missing %q\n%s", want, trfStr)
		}
	}

	lines := map[string]string{}
	for _, line := range strings.Split(trfStr, "\n") {
		if strings.HasPrefix(line, "001") {
			lines[strings.TrimSpace(line[4:8])] = line
		}
	}

	checks := []struct {
		rank string
		want string
	}{
		{"1", "   3 w 1"}, // A played white vs C and won
		{"2", "   4 b ="}, // B played black (WhiteOnFirstBoard=false) vs D and drew
		{"3", "   1 b 0"}, // C played black vs A and lost
		{"4", "   2 w ="}, // D played white (WhiteOnFirstBoard=false) vs B and drew
	}
	for _, c := range checks {
		line := lines[c.rank]
		if len(line) < 99 || line[91:99] != c.want {
			t.Errorf("rank %s: got %q in line %q, want %q", c.rank, line, line, c.want)
		}
	}
}

func TestPairTeams_FixedTables(t *testing.T) {
	in := fourTeamsInput()
	in.Teams[0].FixedTable = intPtr(1) // team-level: A on table 1
	in.Settings = &pairing.Settings[string]{
		FirstPlayerColor: round.ColorWhite,
		FixedBoards: []pairing.FixedBoard[string]{
			{BoardNumber: 1, Player: "C"}, // round-level: C on table 1
			{BoardNumber: 4, Player: "A"}, // round-level overrides A's team-level FixedTable to table 4
		},
	}

	// Engine outputs 1 vs 2 on table 1, 3 vs 4 on table 2.
	matches, err := pairing.PairTeams(context.Background(), simplePairs(twoBoards), in)
	if err != nil {
		t.Fatalf("PairTeams: %v", err)
	}
	assertMatches(t, matches,
		round.Match[string, string]{Round: 1, Table: 1, Home: "C", Away: strPtr("D"), WhiteOnFirstBoard: true},
		round.Match[string, string]{Round: 1, Table: 4, Home: "A", Away: strPtr("B"), WhiteOnFirstBoard: true},
	)
}

func TestPrepareTeams_Errors(t *testing.T) {
	in := fourTeamsInput()
	in.BoardsPerMatch = 0
	if _, err := pairing.PrepareTeams(in); !errors.Is(err, pairing.ErrInvalidBoardsPerMatch) {
		t.Errorf("BoardsPerMatch=0: got %v, want ErrInvalidBoardsPerMatch", err)
	}

	in2 := fourTeamsInput()
	in2.Settings = nil
	if _, err := pairing.PrepareTeams(in2); !errors.Is(err, pairing.ErrInvalidSettings) {
		t.Errorf("round 1 nil settings: got %v, want ErrInvalidSettings", err)
	}

	in3 := fourTeamsInput()
	in3.Teams[1].StartingRank = 0
	if _, err := pairing.PrepareTeams(in3); !errors.Is(err, pairing.ErrTRFGeneration) {
		t.Errorf("missing starting rank: got %v, want ErrTRFGeneration", err)
	}

	engErr := &fakeEngine{format: engine.OutputSimplePairs, err: engine.ErrEngineTimeout}
	if _, err := pairing.PairTeams(context.Background(), engErr, fourTeamsInput()); !errors.Is(err, engine.ErrEngineTimeout) {
		t.Errorf("engine timeout: got %v, want ErrEngineTimeout", err)
	}
}

