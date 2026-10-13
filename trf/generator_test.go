package trf_test

import (
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ValkoM28/tournamentessentials/core/result"
	"github.com/ValkoM28/tournamentessentials/core/trf"
)

func codePtr(c result.Code) *result.Code { return &c }

// idSeq backs newID, which gives each test player a unique string ID.
var idSeq atomic.Int64

func newID() string { return "player-" + strconv.FormatInt(idSeq.Add(1), 10) }

func TestGenerate_HeadersAndPlayerCount(t *testing.T) {
	input := &trf.GeneratorInput[string]{
		TournamentName: "Test Open 2026",
		City:           "Bratislava",
		Federation:     "SVK",
		StartDate:      "2026/03/20",
		EndDate:        "2026/03/22",
		TimeControl:    "90/40+30+30",
		TotalRounds:    7,
		Players: []trf.PlayerData[string]{
			{ID: newID(), StartingRank: 1, Name: "Rated, A", FIDERating: 2400},
			{ID: newID(), StartingRank: 2, Name: "Unrated, B", FIDERating: 0},
			{ID: newID(), StartingRank: 3, Name: "Rated, C", FIDERating: 1800},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	for _, want := range []string{
		"012 Test Open 2026",
		"022 Bratislava",
		"032 SVK",
		"042 2026/03/20",
		"052 2026/03/22",
		"062 3",
		"072 2",
		"122 90/40+30+30",
		"XXR 7",
	} {
		if !containsLine(out, want) {
			t.Errorf("missing header line %q", want)
		}
	}
}

func TestGenerate_MappingUsesStartingRank(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "Alpha, A", FIDERating: 2000}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "Bravo, B", FIDERating: 2500}
	p3 := trf.PlayerData[string]{ID: newID(), StartingRank: 3, Name: "Charlie, C", FIDERating: 2000}

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p3, p1, p2}, // deliberately out of order
	}

	_, mapping, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Mapping must use the pre-assigned StartingRank, not re-sort.
	wantRanks := map[string]int{
		p1.ID: 1,
		p2.ID: 2,
		p3.ID: 3,
	}
	for id, want := range wantRanks {
		if got := mapping.IDToRank[id]; got != want {
			t.Errorf("rank for %s: got %d, want %d", id, got, want)
		}
		if mapping.RankToID[want] != id {
			t.Errorf("reverse mapping for rank %d broken", want)
		}
	}
}

func TestGenerate_PlayerLineFixedWidth(t *testing.T) {
	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players: []trf.PlayerData[string]{
			{
				ID:           newID(),
				StartingRank: 1,
				Name:         "Carlsen, Magnus",
				Title:        "GM",
				FIDERating:   2830,
				Federation:   "NOR",
				FIDENumber:   "1503014",
				BirthDate:    "1990/11/30",
				Sex:          "m",
			},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	line := padTo(findPlayerLine(t, out, 1), 109)

	checks := []struct {
		start, end int
		want       string
		label      string
	}{
		{0, 3, "001", "data ID"},
		{4, 8, "   1", "starting rank"},
		{9, 10, "m", "sex"},
		{10, 12, "GM", "title prefix"},
		{48, 52, "2830", "rating"},
		{53, 56, "NOR", "federation"},
		{80, 84, " 0.0", "points (no rounds)"},
	}
	for _, c := range checks {
		got := line[c.start:c.end]
		if got != c.want {
			t.Errorf("%s [%d:%d]: got %q, want %q", c.label, c.start, c.end, got, c.want)
		}
	}

	// Name starts at index 14, left-justified in 33 chars.
	if !strings.HasPrefix(line[14:47], "Carlsen, Magnus") {
		t.Errorf("name: got %q", line[14:47])
	}

	// FIDE number right-justified in indices 57-67.
	if strings.TrimSpace(line[57:68]) != "1503014" {
		t.Errorf("FIDE number: got %q", line[57:68])
	}

	// Birth date at indices 69-78.
	if line[69:79] != "1990/11/30" {
		t.Errorf("birth date: got %q", line[69:79])
	}
}

func TestGenerate_RoundDataAndPoints(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	p3 := trf.PlayerData[string]{ID: newID(), StartingRank: 3, Name: "C, C", FIDERating: 2300}

	win := result.WhiteWins
	draw := result.Draw
	halfBye := result.WhiteHalf

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2, p3},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &p2.ID, Result: &win},
					{White: p3.ID, Black: nil, Result: &halfBye},
				},
			},
			{
				RoundNumber: 2,
				Pairings: []trf.PairingData[string]{
					{White: p2.ID, Black: &p3.ID, Result: &draw},
					{White: p1.ID, Black: nil, Result: &halfBye},
				},
			},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	line1 := padTo(findPlayerLine(t, out, 1), 109)
	line2 := padTo(findPlayerLine(t, out, 2), 109)
	line3 := padTo(findPlayerLine(t, out, 3), 109)

	// Round 1 (offset 91): p1 vs p2 white win
	assertRound(t, line1, 91, "   2", 'w', '1', "p1 r1")
	assertRound(t, line2, 91, "   1", 'b', '0', "p2 r1")
	assertRound(t, line3, 91, "0000", '-', 'H', "p3 r1 half-bye")

	// Round 2 (offset 101): p2 vs p3 draw, p1 half-bye
	assertRound(t, line1, 101, "0000", '-', 'H', "p1 r2 half-bye")
	assertRound(t, line2, 101, "   3", 'w', '=', "p2 r2")
	assertRound(t, line3, 101, "   2", 'b', '=', "p3 r2")

	// Cumulative points: p1=1.5, p2=0.5, p3=1.0
	assertPoints(t, line1, " 1.5", "p1")
	assertPoints(t, line2, " 0.5", "p2")
	assertPoints(t, line3, " 1.0", "p3")
}

func TestGenerate_VoidPairingsSkipped(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2000}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 1900}

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    3,
		Players:        []trf.PlayerData[string]{p1, p2},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &p2.ID, Result: codePtr(result.Void)},
				},
			},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	line1 := findPlayerLine(t, out, 1)
	assertPoints(t, padTo(line1, 109), " 0.0", "p1 after void")

	// Round data should be blank.
	if len(line1) > 91 && strings.TrimSpace(line1[91:]) != "" {
		t.Errorf("expected blank round data for void, got %q", line1[91:])
	}
}

func TestGenerate_MissingPlayerInHistoricalPairing(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	ghost := newID() // player who was paired but is no longer in input.Players

	win := result.WhiteWins
	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					// ghost was the black player in round 1 but is no longer confirmed
					{White: p1.ID, Black: &ghost, Result: &win},
				},
			},
		},
	}

	_, _, err := trf.Generate(input)
	if err == nil {
		t.Error("expected error when a historical pairing references a player not in input.Players")
	}
}

func TestGenerate_Errors(t *testing.T) {
	// No players.
	if _, _, err := trf.Generate(&trf.GeneratorInput[string]{TotalRounds: 5}); err == nil {
		t.Error("expected error for no players")
	}
	// Zero total rounds.
	if _, _, err := trf.Generate(&trf.GeneratorInput[string]{
		Players: []trf.PlayerData[string]{{ID: newID(), StartingRank: 1, Name: "A, A"}},
	}); err == nil {
		t.Error("expected error for zero total rounds")
	}
	// Missing starting rank.
	if _, _, err := trf.Generate(&trf.GeneratorInput[string]{
		TotalRounds: 5,
		Players:     []trf.PlayerData[string]{{ID: newID(), Name: "A, A"}},
	}); err == nil {
		t.Error("expected error for missing starting rank")
	}
}

func TestGenerate_WithdrawnPlayersGetZeroBye(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	p3 := trf.PlayerData[string]{ID: newID(), StartingRank: 3, Name: "C, C", FIDERating: 2300}

	win := result.WhiteWins
	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2, p3},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &p2.ID, Result: &win},
					{White: p3.ID, Black: nil, Result: codePtr(result.WhiteHalf)},
				},
			},
		},
		Settings: &trf.RoundSettings[string]{
			WithdrawnPlayers: []string{p3.ID},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Round 2 (offset 101) should show Z bye for withdrawn p3.
	line3 := padTo(findPlayerLine(t, out, 3), 119)
	assertRound(t, line3, 101, "0000", '-', 'Z', "p3 withdrawn r2")

	// Non-withdrawn players should have blank round 2 data.
	line1 := padTo(findPlayerLine(t, out, 1), 119)
	r2data := line1[101:109]
	if strings.TrimSpace(r2data) != "" {
		t.Errorf("p1 round 2 should be blank, got %q", r2data)
	}
}

func TestGenerate_HalfPointByes(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}

	win := result.WhiteWins
	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &p2.ID, Result: &win},
				},
			},
		},
		Settings: &trf.RoundSettings[string]{
			HalfPointByes: []string{p2.ID},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Round 2 (offset 101) should show H bye for p2.
	line2 := padTo(findPlayerLine(t, out, 2), 119)
	assertRound(t, line2, 101, "0000", '-', 'H', "p2 half-bye r2")

	// Points: p2 has 0 from r1 + 0.5 from half-bye = 0.5
	assertPoints(t, line2, " 0.5", "p2 with half-bye")
}

func TestGenerate_FirstPlayerColor(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2},
		Settings: &trf.RoundSettings[string]{
			FirstPlayerColor: "white",
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !containsLine(out, "XXC white1") {
		t.Error("missing XXC white1 line")
	}

	// Test black
	input.Settings.FirstPlayerColor = "black"
	out, _, err = trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if !containsLine(out, "XXC black1") {
		t.Error("missing XXC black1 line")
	}
}

func TestGenerate_ForbiddenPairings(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	p3 := trf.PlayerData[string]{ID: newID(), StartingRank: 3, Name: "C, C", FIDERating: 2300}

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2, p3},
		Settings: &trf.RoundSettings[string]{
			ForbiddenPairings: []trf.ForbiddenPairTRF[string]{
				{Player1: p1.ID, Player2: p3.ID},
			},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !containsLine(out, "XXF 1 3") {
		t.Errorf("missing XXF line, output:\n%s", out)
	}
}

func TestGenerate_LateEntries(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	p3 := trf.PlayerData[string]{ID: newID(), StartingRank: 3, Name: "Late, Player", FIDERating: 2200}

	win := result.WhiteWins
	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2, p3},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &p2.ID, Result: &win},
					// p3 not paired in round 1 (joined late)
				},
			},
			{
				RoundNumber: 2,
				Pairings: []trf.PairingData[string]{
					{White: p3.ID, Black: &p1.ID, Result: &win},
					{White: p2.ID, Black: nil, Result: codePtr(result.WhiteHalf)},
				},
			},
		},
		Settings: &trf.RoundSettings[string]{
			LateEntries: []string{p3.ID},
		},
	}

	out, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// p3 should have Z bye in round 1 (missed) and normal data in round 2.
	line3 := padTo(findPlayerLine(t, out, 3), 119)
	assertRound(t, line3, 91, "0000", '-', 'Z', "p3 late entry r1")
	assertRound(t, line3, 101, "   1", 'w', '1', "p3 r2 normal")

	// Points: 0 (Z) + 1 (win) = 1.0
	assertPoints(t, line3, " 1.0", "p3 late entry")
}

func TestGenerate_MissingWhitePlayerInHistoricalPairing(t *testing.T) {
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	ghost := newID() // was white in round 1, no longer confirmed

	win := result.WhiteWins
	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p2},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: ghost, Black: &p2.ID, Result: &win},
				},
			},
		},
	}

	_, _, err := trf.Generate(input)
	if err == nil {
		t.Error("expected error when white player in historical pairing is not in input.Players")
	}
}

func TestGenerate_VoidPairingWithMissingPlayerNoError(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	ghost := newID() // missing player, but pairing is voided

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &ghost, Result: codePtr(result.Void)},
				},
			},
		},
	}

	_, _, err := trf.Generate(input)
	if err != nil {
		t.Errorf("expected no error for voided pairing with missing player, got: %v", err)
	}
}

func TestGenerate_MissingPlayerErrorContainsRoundNumber(t *testing.T) {
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	ghost := newID()

	win := result.WhiteWins
	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &p2.ID, Result: &win},
				},
			},
			{
				RoundNumber: 2,
				Pairings: []trf.PairingData[string]{
					{White: p1.ID, Black: &ghost, Result: &win},
				},
			},
		},
	}

	_, _, err := trf.Generate(input)
	if err == nil {
		t.Fatal("expected error for missing player in round 2")
	}
	if !strings.Contains(err.Error(), "round 2") {
		t.Errorf("error message should reference round 2, got: %q", err.Error())
	}
}

func TestGenerate_ByePairingWithMissingWhitePlayerNoError(t *testing.T) {
	ghost := newID() // missing player who received a bye

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2000}},
		Rounds: []trf.RoundData[string]{
			{
				RoundNumber: 1,
				Pairings: []trf.PairingData[string]{
					{White: ghost, Black: nil, Result: codePtr(result.WhiteWins)},
				},
			},
		},
	}

	_, _, err := trf.Generate(input)
	if err != nil {
		t.Errorf("expected no error for bye pairing with missing white player, got: %v", err)
	}
}

// --- helpers ---

func containsLine(output, want string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimRight(line, " \r") == want {
			return true
		}
	}
	return false
}

func findPlayerLine(t *testing.T, output string, rank int) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 8 || line[0:3] != "001" {
			continue
		}
		s := strings.TrimSpace(line[4:8])
		if n, err := strconv.Atoi(s); err == nil && n == rank {
			return line
		}
	}
	t.Fatalf("no 001 line for rank %d", rank)
	return ""
}

func padTo(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func assertRound(t *testing.T, line string, offset int, wantOpp string, wantColor, wantResult byte, label string) {
	t.Helper()
	got := line[offset : offset+4]
	if got != wantOpp {
		t.Errorf("%s opponent: got %q, want %q", label, got, wantOpp)
	}
	if line[offset+5] != wantColor {
		t.Errorf("%s color: got %q, want %q", label, string(line[offset+5]), string(wantColor))
	}
	if line[offset+7] != wantResult {
		t.Errorf("%s result: got %q, want %q", label, string(line[offset+7]), string(wantResult))
	}
}

func assertPoints(t *testing.T, line, want, label string) {
	t.Helper()
	got := line[80:84]
	if got != want {
		t.Errorf("%s points: got %q, want %q", label, got, want)
	}
}
