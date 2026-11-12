//go:build unix

package engine_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/tessechess/core/engine"
	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
	"github.com/tessechess/core/trf"
)

// These tests run the real pairing programs. They are skipped unless the paths
// are given, e.g. via `make test-engines`:
//
//	TE_BBPPAIRINGS_PATH  path to the bbpPairings binary
//	TE_JAVAFO_JAR_PATH   path to javafo.jar (also needs `java` on PATH)

func TestRealBBPPairings(t *testing.T) {
	path := os.Getenv("TE_BBPPAIRINGS_PATH")
	if path == "" {
		t.Skip("TE_BBPPAIRINGS_PATH not set")
	}
	pairWithRealEngine(t, engine.NewBBPPairings(path, 30*time.Second))
}

func TestRealJaVaFo(t *testing.T) {
	jar := os.Getenv("TE_JAVAFO_JAR_PATH")
	if jar == "" {
		t.Skip("TE_JAVAFO_JAR_PATH not set")
	}
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("java not on PATH")
	}
	pairWithRealEngine(t, engine.NewJaVaFo("java", jar, 60*time.Second))
}

// pairWithRealEngine pairs round 1 and round 2 of a 4-player Swiss through the
// same path the backend uses: trf.Generate, the engine, then the parser chosen
// by OutputFormat.
func pairWithRealEngine(t *testing.T, eng engine.PairingEngine) {
	t.Helper()
	players := []trf.PlayerData[string]{
		{ID: "a", StartingRank: 1, Name: "Alpha, A", FIDERating: 2400},
		{ID: "b", StartingRank: 2, Name: "Bravo, B", FIDERating: 2300},
		{ID: "c", StartingRank: 3, Name: "Charlie, C", FIDERating: 2200},
		{ID: "d", StartingRank: 4, Name: "Delta, D", FIDERating: 2100},
	}
	win, draw := result.WhiteWins, result.Draw
	b, c := "b", "c"
	round1 := []round.Game[string]{
		{Round: 1, White: "a", Black: &c, Result: &win},
		{Round: 1, White: "d", Black: &b, Result: &draw},
	}

	cases := []struct {
		name      string
		games     []round.Game[string]
		completed int
	}{
		{"round 1", nil, 0},
		{"round 2", round1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := &trf.GeneratorInput[string]{
				TournamentName:  "Real engine test",
				TotalRounds:     3,
				Players:         players,
				Games:           tc.games,
				CompletedRounds: tc.completed,
				// bbpPairings refuses round 1 without an initial colour (XXC).
				Settings: &trf.RoundSettings[string]{FirstPlayerColor: "white"},
			}
			in, mapping, err := trf.Generate(input)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			out, err := eng.GeneratePairings(context.Background(), in)
			if err != nil {
				t.Fatalf("GeneratePairings: %v", err)
			}

			var parsed []trf.ParsedPairing
			switch eng.OutputFormat() {
			case engine.OutputTRF:
				parsed, err = trf.ParseTRFRound(out, tc.completed)
			case engine.OutputSimplePairs:
				parsed, err = trf.ParseSimplePairs(out)
			default:
				t.Fatalf("unknown output format %q", eng.OutputFormat())
			}
			if err != nil {
				t.Fatalf("parse %s output: %v\noutput:\n%s", eng.OutputFormat(), err, out)
			}

			if len(parsed) != 2 {
				t.Fatalf("got %d boards, want 2: %+v", len(parsed), parsed)
			}
			seen := map[string]bool{}
			for _, p := range parsed {
				for _, rank := range []int{p.WhiteRank, p.BlackRank} {
					id, ok := mapping.RankToID[rank]
					if !ok {
						t.Fatalf("engine returned unknown rank %d", rank)
					}
					if seen[id] {
						t.Fatalf("player %s paired twice", id)
					}
					seen[id] = true
				}
			}
		})
	}
}
