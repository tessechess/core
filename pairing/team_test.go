package pairing_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/tessechess/core/pairing"
	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

type testTeam = pairing.Team[string, string]

func TestTeamRating(t *testing.T) {
	players := []entrant{
		{ID: "p1", Rating: 2200},
		{ID: "p2", Rating: 2400},
		{ID: "p3", Rating: 2000},
		{ID: "p4", Rating: 2100},
		{ID: "p5", Rating: 1800}, // reserve
	}

	// Top 4 ratings: 2400, 2200, 2100, 2000 -> average = 8700 / 4 = 2175
	if got := pairing.TeamRating(players, 4); got != 2175 {
		t.Errorf("TeamRating(top 4) = %d, want 2175", got)
	}

	// Rounding: top 3 ratings 2400, 2200, 2100 -> 6700 / 3 = 2233.33 -> 2233
	if got := pairing.TeamRating(players, 3); got != 2233 {
		t.Errorf("TeamRating(top 3) = %d, want 2233", got)
	}

	// Rounding 0.5 up: 2401 + 2200 = 4601 / 2 = 2300.5 -> 2301
	halfPlayers := []entrant{{ID: "a", Rating: 2401}, {ID: "b", Rating: 2200}}
	if got := pairing.TeamRating(halfPlayers, 2); got != 2301 {
		t.Errorf("TeamRating(0.5 up) = %d, want 2301", got)
	}

	// Fewer players than boardsPerMatch: missing slots count as 0 (2400 + 2200 + 0 + 0) / 4 = 1150
	if got := pairing.TeamRating(halfPlayers, 4); got != 1150 {
		t.Errorf("TeamRating(short roster) = %d, want 1150", got)
	}

	// Unrated roster / invalid boardsPerMatch / empty slice -> 0
	unrated := []entrant{{ID: "u1", Rating: 0}, {ID: "u2", Rating: 0}}
	if got := pairing.TeamRating(unrated, 2); got != 0 {
		t.Errorf("TeamRating(unrated) = %d, want 0", got)
	}
	if got := pairing.TeamRating(players, 0); got != 0 {
		t.Errorf("TeamRating(boards=0) = %d, want 0", got)
	}
	if got := pairing.TeamRating[string](nil, 4); got != 0 {
		t.Errorf("TeamRating(nil) = %d, want 0", got)
	}
}

func TestTeamStartingOrder(t *testing.T) {
	teams := []testTeam{
		{
			ID:   "beta",
			Name: "Beta Club",
			Players: []entrant{
				{ID: "b1", Rating: 2000},
				{ID: "b2", Rating: 2000},
			},
		},
		{
			ID:   "alpha",
			Name: "Alpha Club",
			Players: []entrant{
				{ID: "a1", Rating: 2000},
				{ID: "a2", Rating: 2000},
			},
		},
		{
			ID:   "unrated_z",
			Name: "Zeta Unrated",
		},
		{
			ID:   "unrated_a",
			Name: "Apex Unrated",
		},
		{
			ID:   "kings",
			Name: "Kings",
			Players: []entrant{
				{ID: "k1", Rating: 2500},
				{ID: "k2", Rating: 2300},
				{ID: "k3", Rating: 1500}, // reserve ignored for 2-board average
			},
		},
	}

	got := pairing.TeamStartingOrder(teams, 2)
	want := []string{"kings", "alpha", "beta", "unrated_a", "unrated_z"}
	if !slices.Equal(got, want) {
		t.Errorf("TeamStartingOrder = %v, want %v", got, want)
	}

	// Verify input slice was not modified.
	if teams[0].ID != "beta" {
		t.Errorf("input slice modified: got first element %q, want beta", teams[0].ID)
	}
}

func TestPairTeamRoundRobin(t *testing.T) {
	teams := []testTeam{
		{ID: "T1", StartingRank: 1, Name: "Team 1"},
		{ID: "T2", StartingRank: 2, Name: "Team 2"},
		{ID: "T3", StartingRank: 3, Name: "Team 3"},
		{ID: "T4", StartingRank: 4, Name: "Team 4"},
	}

	rounds, err := pairing.PairTeamRoundRobin(teams, 2, pairing.CycleOrderFIDE)
	if err != nil {
		t.Fatalf("PairTeamRoundRobin unexpected error: %v", err)
	}
	if len(rounds) != 6 {
		t.Fatalf("len(rounds) = %d, want 6", len(rounds))
	}

	// Round 1 of 4-team Berger table: (1 vs 4), (2 vs 3)
	r1 := rounds[0]
	if len(r1) != 2 {
		t.Fatalf("len(r1) = %d, want 2", len(r1))
	}
	if r1[0].Home != "T1" || r1[0].Away == nil || *r1[0].Away != "T4" || !r1[0].WhiteOnFirstBoard || r1[0].Table != 1 {
		t.Errorf("r1[0] = %+v, want T1 vs T4 on table 1 with WhiteOnFirstBoard=true", r1[0])
	}
	if r1[1].Home != "T2" || r1[1].Away == nil || *r1[1].Away != "T3" || !r1[1].WhiteOnFirstBoard || r1[1].Table != 2 {
		t.Errorf("r1[1] = %+v, want T2 vs T3 on table 2 with WhiteOnFirstBoard=true", r1[1])
	}

	// Cycle 2 Round 1 (round index 3, Round=4): colors reversed -> (4 vs 1), (3 vs 2)
	r4 := rounds[3]
	if r4[0].Home != "T4" || r4[0].Away == nil || *r4[0].Away != "T1" || r4[0].Round != 4 {
		t.Errorf("r4[0] = %+v, want T4 vs T1 in round 4", r4[0])
	}
}

func TestPairTeamRoundRobin_OddTeamsAndFixedTable(t *testing.T) {
	fixedTwo := 2
	teams := []testTeam{
		{ID: "T1", StartingRank: 1, FixedTable: &fixedTwo},
		{ID: "T2", StartingRank: 2},
		{ID: "T3", StartingRank: 3},
	}

	rounds, err := pairing.PairTeamRoundRobin(teams, 1, pairing.CycleOrderRepeat)
	if err != nil {
		t.Fatalf("PairTeamRoundRobin(3 teams) unexpected error: %v", err)
	}
	if len(rounds) != 3 {
		t.Fatalf("len(rounds) = %d, want 3", len(rounds))
	}

	// Round 1: Berger has (1 vs 4-bye), (2 vs 3).
	// T2 vs T3 is the only played match (table 1), T1 has a free round (bye on table 2, Result = Void).
	r1 := rounds[0]
	if len(r1) != 2 {
		t.Fatalf("len(r1) = %d, want 2", len(r1))
	}
	if r1[0].Home != "T2" || r1[0].Away == nil || *r1[0].Away != "T3" || r1[0].Table != 1 {
		t.Errorf("r1[0] = %+v, want T2 vs T3 on table 1", r1[0])
	}
	if !r1[1].IsBye() || r1[1].Home != "T1" || r1[1].Table != 2 || r1[1].Result == nil || *r1[1].Result != result.Void {
		t.Errorf("r1[1] = %+v, want T1 free-round bye on table 2 with Void result", r1[1])
	}

	// Round 2: Berger has (4-bye vs 3), (1 vs 2).
	// T1 vs T2 is pinned to FixedTable=2, so T1 vs T2 is on table 2 and T3's bye is on table 3.
	r2 := rounds[1]
	if r2[0].Home != "T1" || r2[0].Away == nil || *r2[0].Away != "T2" || r2[0].Table != 2 {
		t.Errorf("r2[0] = %+v, want T1 vs T2 on fixed table 2", r2[0])
	}
	if !r2[1].IsBye() || r2[1].Home != "T3" || r2[1].Table != 3 {
		t.Errorf("r2[1] = %+v, want T3 bye on table 3", r2[1])
	}
}

func TestPairTeamRoundRobin_Errors(t *testing.T) {
	if _, err := pairing.PairTeamRoundRobin([]testTeam{{ID: "T1", StartingRank: 1}}, 1, ""); !errors.Is(err, pairing.ErrNotEnoughPlayers) {
		t.Errorf("got %v, want ErrNotEnoughPlayers", err)
	}
	teams := []testTeam{{ID: "T1", StartingRank: 1}, {ID: "T2", StartingRank: 2}}
	if _, err := pairing.PairTeamRoundRobin(teams, 0, ""); !errors.Is(err, pairing.ErrInvalidCycles) {
		t.Errorf("got %v, want ErrInvalidCycles", err)
	}
	if _, err := pairing.PairTeamRoundRobin(teams, 1, "bad"); !errors.Is(err, pairing.ErrInvalidCycleOrder) {
		t.Errorf("got %v, want ErrInvalidCycleOrder", err)
	}
	dupRank := []testTeam{{ID: "T1", StartingRank: 1}, {ID: "T2", StartingRank: 1}}
	if _, err := pairing.PairTeamRoundRobin(dupRank, 1, ""); !errors.Is(err, pairing.ErrInvalidStartingRank) {
		t.Errorf("got %v, want ErrInvalidStartingRank", err)
	}
}

func TestPairMatchBoards(t *testing.T) {
	awayB := "B"
	matches := []round.Match[string, string]{
		{
			Round:             2,
			Table:             1,
			Home:              "A",
			Away:              &awayB,
			WhiteOnFirstBoard: true,
		},
		{
			Round: 2,
			Table: 2,
			Home:  "C", // bye team needs no lineup
		},
	}

	lineups := map[string][]string{
		"A": {"A1", "A2", "A3", "A4"},
		"B": {"B1", "B2", "B3", "B4"},
	}

	paired, err := pairing.PairMatchBoards(matches, lineups, 4)
	if err != nil {
		t.Fatalf("PairMatchBoards unexpected error: %v", err)
	}
	if len(paired) != 2 {
		t.Fatalf("len(paired) = %d, want 2", len(paired))
	}

	boards := paired[0].Boards
	if len(boards) != 4 {
		t.Fatalf("len(boards) = %d, want 4", len(boards))
	}

	// Board 1: A1 (W) vs B1 (B)
	// Board 2: B2 (W) vs A2 (B)
	// Board 3: A3 (W) vs B3 (B)
	// Board 4: B4 (W) vs A4 (B)
	wantWhite := []string{"A1", "B2", "A3", "B4"}
	wantBlack := []string{"B1", "A2", "B3", "A4"}
	for i := 0; i < 4; i++ {
		g := boards[i]
		if g.Round != 2 || g.Board != i+1 {
			t.Errorf("board %d: Round=%d Board=%d, want Round=2 Board=%d", i+1, g.Round, g.Board, i+1)
		}
		if g.White != wantWhite[i] || g.Black == nil || *g.Black != wantBlack[i] {
			t.Errorf("board %d: got %s vs %v, want %s vs %s", i+1, g.White, g.Black, wantWhite[i], wantBlack[i])
		}
	}

	// Bye match remains untouched.
	if !paired[1].IsBye() || len(paired[1].Boards) != 0 {
		t.Errorf("bye match modified: %+v", paired[1])
	}

	// Also verify WhiteOnFirstBoard = false reverses colors on every board.
	matches[0].WhiteOnFirstBoard = false
	pairedFlipped, err := pairing.PairMatchBoards(matches[:1], lineups, 4)
	if err != nil {
		t.Fatalf("PairMatchBoards(WhiteOnFirstBoard=false) error: %v", err)
	}
	for i := 0; i < 4; i++ {
		g := pairedFlipped[0].Boards[i]
		if g.White != wantBlack[i] || g.Black == nil || *g.Black != wantWhite[i] {
			t.Errorf("flipped board %d: got %s vs %v, want %s vs %s", i+1, g.White, g.Black, wantBlack[i], wantWhite[i])
		}
	}
}

func TestPairMatchBoards_Errors(t *testing.T) {
	awayB := "B"
	match := []round.Match[string, string]{
		{Round: 1, Table: 1, Home: "A", Away: &awayB, WhiteOnFirstBoard: true},
	}

	if _, err := pairing.PairMatchBoards(match, nil, 0); !errors.Is(err, pairing.ErrInvalidBoardsPerMatch) {
		t.Errorf("boardsPerMatch=0: got %v, want ErrInvalidBoardsPerMatch", err)
	}

	cases := []struct {
		name    string
		lineups map[string][]string
	}{
		{"missing home lineup", map[string][]string{"B": {"B1", "B2"}}},
		{"missing away lineup", map[string][]string{"A": {"A1", "A2"}}},
		{"wrong home lineup length", map[string][]string{"A": {"A1"}, "B": {"B1", "B2"}}},
		{"wrong away lineup length", map[string][]string{"A": {"A1", "A2"}, "B": {"B1", "B2", "B3"}}},
		{"duplicate in home lineup", map[string][]string{"A": {"A1", "A1"}, "B": {"B1", "B2"}}},
		{"duplicate in away lineup", map[string][]string{"A": {"A1", "A2"}, "B": {"B1", "B1"}}},
		{"same player in both lineups", map[string][]string{"A": {"A1", "X"}, "B": {"B1", "X"}}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := pairing.PairMatchBoards(match, c.lineups, 2); !errors.Is(err, pairing.ErrInvalidLineup) {
				t.Errorf("got %v, want ErrInvalidLineup", err)
			}
		})
	}
}
