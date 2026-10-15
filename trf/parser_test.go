package trf_test

import (
	"strings"
	"testing"

	"github.com/ValkoM28/tournamentessentials/core/result"
	"github.com/ValkoM28/tournamentessentials/core/round"
	"github.com/ValkoM28/tournamentessentials/core/trf"
)

func TestParseBBPPairingsOutput(t *testing.T) {
	// Build a minimal TRF with 4 players and 1 completed round,
	// then append a second round produced by the engine.
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	p3 := trf.PlayerData[string]{ID: newID(), StartingRank: 3, Name: "C, C", FIDERating: 2300}
	p4 := trf.PlayerData[string]{ID: newID(), StartingRank: 4, Name: "D, D", FIDERating: 2200}

	win := result.WhiteWins

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2, p3, p4},
		Games: []round.Game[string]{
			{Round: 1, White: p1.ID, Black: &p2.ID, Result: &win},
			{Round: 1, White: p3.ID, Black: &p4.ID, Result: &win},
		},
		CompletedRounds: 1,
	}

	generated, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Simulate bbpPairings output: append round 2 data to each player line.
	// Round 2 pairings: 1 vs 3 (white), 4 vs 2 (white).
	var engineOutput strings.Builder
	for _, line := range strings.Split(generated, "\n") {
		if len(line) < 4 || line[0:3] != "001" {
			engineOutput.WriteString(line + "\n")
			continue
		}

		buf := []byte(padTo(line, 109))
		rank := strings.TrimSpace(string(buf[4:8]))
		// Round 2 at offset 101.
		switch rank {
		case "1": // white vs 3
			copy(buf[101:109], []byte("   3 w  "))
		case "2": // black vs 4
			copy(buf[101:109], []byte("   4 b  "))
		case "3": // black vs 1
			copy(buf[101:109], []byte("   1 b  "))
		case "4": // white vs 2
			copy(buf[101:109], []byte("   2 w  "))
		}
		engineOutput.WriteString(string(buf) + "\n")
	}

	pairings, err := trf.ParseBBPPairingsOutput(engineOutput.String(), 1)
	if err != nil {
		t.Fatalf("ParseBBPPairingsOutput error: %v", err)
	}

	// Expect 2 pairings (from white perspective): 1 vs 3, 4 vs 2.
	if len(pairings) != 2 {
		t.Fatalf("got %d pairings, want 2", len(pairings))
	}

	// Board 1: rank 1 (white) vs rank 3.
	if pairings[0].WhiteRank != 1 || pairings[0].BlackRank != 3 {
		t.Errorf("pairing 1: got %d vs %d, want 1 vs 3", pairings[0].WhiteRank, pairings[0].BlackRank)
	}
	// Board 2: rank 4 (white) vs rank 2.
	if pairings[1].WhiteRank != 4 || pairings[1].BlackRank != 2 {
		t.Errorf("pairing 2: got %d vs %d, want 4 vs 2", pairings[1].WhiteRank, pairings[1].BlackRank)
	}
}

func TestParseBBPPairingsOutput_WithBye(t *testing.T) {
	// 3 players, round 1 completed. Engine output for round 2 has a bye.
	p1 := trf.PlayerData[string]{ID: newID(), StartingRank: 1, Name: "A, A", FIDERating: 2500}
	p2 := trf.PlayerData[string]{ID: newID(), StartingRank: 2, Name: "B, B", FIDERating: 2400}
	p3 := trf.PlayerData[string]{ID: newID(), StartingRank: 3, Name: "C, C", FIDERating: 2300}

	win := result.WhiteWins
	halfBye := result.WhiteHalf

	input := &trf.GeneratorInput[string]{
		TournamentName: "T",
		TotalRounds:    5,
		Players:        []trf.PlayerData[string]{p1, p2, p3},
		Games: []round.Game[string]{
			{Round: 1, White: p1.ID, Black: &p2.ID, Result: &win},
			{Round: 1, White: p3.ID, Black: nil, Result: &halfBye},
		},
		CompletedRounds: 1,
	}

	generated, _, err := trf.Generate(input)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Simulate engine output for round 2: 1 vs 3 (white), 2 gets bye.
	var engineOutput strings.Builder
	for _, line := range strings.Split(generated, "\n") {
		if len(line) < 4 || line[0:3] != "001" {
			engineOutput.WriteString(line + "\n")
			continue
		}

		buf := []byte(padTo(line, 109))
		rank := strings.TrimSpace(string(buf[4:8]))
		switch rank {
		case "1":
			copy(buf[101:109], []byte("   3 w  "))
		case "2":
			copy(buf[101:109], []byte("0000 - U"))
		case "3":
			copy(buf[101:109], []byte("   1 b  "))
		}
		engineOutput.WriteString(string(buf) + "\n")
	}

	pairings, err := trf.ParseBBPPairingsOutput(engineOutput.String(), 1)
	if err != nil {
		t.Fatalf("ParseBBPPairingsOutput error: %v", err)
	}

	if len(pairings) != 2 {
		t.Fatalf("got %d pairings, want 2", len(pairings))
	}

	// Board 1: 1 vs 3.
	if pairings[0].WhiteRank != 1 || pairings[0].BlackRank != 3 {
		t.Errorf("pairing 1: got %d vs %d, want 1 vs 3", pairings[0].WhiteRank, pairings[0].BlackRank)
	}
	// Board 2: bye for rank 2.
	if pairings[1].WhiteRank != 2 || pairings[1].BlackRank != 0 {
		t.Errorf("pairing 2: got %d vs %d, want 2 vs 0 (bye)", pairings[1].WhiteRank, pairings[1].BlackRank)
	}
}

func TestParseJaVaFoOutput(t *testing.T) {
	output := "3\n1 4\n2 3\n5 0\n"

	pairings, err := trf.ParseJaVaFoOutput(output)
	if err != nil {
		t.Fatalf("ParseJaVaFoOutput error: %v", err)
	}

	if len(pairings) != 3 {
		t.Fatalf("got %d pairings, want 3", len(pairings))
	}

	want := []struct{ w, b int }{{1, 4}, {2, 3}, {5, 0}}
	for i, w := range want {
		if pairings[i].WhiteRank != w.w || pairings[i].BlackRank != w.b {
			t.Errorf("pairing %d: got %d vs %d, want %d vs %d",
				i+1, pairings[i].WhiteRank, pairings[i].BlackRank, w.w, w.b)
		}
		if pairings[i].BoardNumber != i+1 {
			t.Errorf("pairing %d board: got %d, want %d", i+1, pairings[i].BoardNumber, i+1)
		}
	}
}

func TestParseJaVaFoOutput_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"bad count", "abc\n"},
		{"too few lines", "3\n1 2\n"},
		{"bad pairing", "1\nnot a number\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := trf.ParseJaVaFoOutput(tt.input); err == nil {
				t.Error("expected error")
			}
		})
	}
}
