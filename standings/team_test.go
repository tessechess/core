package standings

import (
	"testing"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

func floatPtr(v float64) *float64 { return &v }
func strPtr(s string) *string     { return &s }

func makeMatch(
	roundNum, table int,
	home string,
	away *string,
	whiteOnFirst bool,
	boards []round.Game[string],
	res *result.Code,
) round.Match[string, string] {
	return round.Match[string, string]{
		Round:             roundNum,
		Table:             table,
		Home:              home,
		Away:              away,
		WhiteOnFirstBoard: whiteOnFirst,
		Boards:            boards,
		Result:            res,
	}
}

func TestRankTeams_MatchPointsAndGamePointsPrimary(t *testing.T) {
	// 3 teams on 2 boards over 2 rounds:
	// Team A wins both matches narrowly (1.5-0.5, 1.5-0.5) -> 4 MP, 3.0 GP
	// Team B wins one match big (2.0-0.0) and loses to A (0.5-1.5) -> 2 MP, 2.5 GP
	// Team C loses to A (0.5-1.5) and loses to B (0.0-2.0), plus gets a full-point bye in R3 -> 2 MP, 1.5 GP
	// Wait: let's make B have MORE GamePoints than A so MP-primary and GP-primary rank differently!
	// On 4 boards:
	// R1: A beats B 2.5 - 1.5 (A: 2 MP, 2.5 GP; B: 0 MP, 1.5 GP)
	// R2: A beats C 2.5 - 1.5 (A: 4 MP, 5.0 GP; C: 0 MP, 1.5 GP)
	// R2: B beats D 4.0 - 0.0 (B: 2 MP, 5.5 GP; D: 0 MP, 0.0 GP)
	matches := []round.Match[string, string]{
		makeMatch(1, 1, "A", strPtr("B"), true, []round.Game[string]{
			{Round: 1, Board: 1, White: "A1", Black: strPtr("B1"), Result: codePtr(result.WhiteWins)},
			{Round: 1, Board: 2, White: "B2", Black: strPtr("A2"), Result: codePtr(result.BlackWins)},
			{Round: 1, Board: 3, White: "A3", Black: strPtr("B3"), Result: codePtr(result.Draw)},
			{Round: 1, Board: 4, White: "B4", Black: strPtr("A4"), Result: codePtr(result.WhiteWins)},
		}, nil),
		makeMatch(2, 1, "A", strPtr("C"), true, []round.Game[string]{
			{Round: 2, Board: 1, White: "A1", Black: strPtr("C1"), Result: codePtr(result.WhiteWins)},
			{Round: 2, Board: 2, White: "C2", Black: strPtr("A2"), Result: codePtr(result.BlackWins)},
			{Round: 2, Board: 3, White: "A3", Black: strPtr("C3"), Result: codePtr(result.Draw)},
			{Round: 2, Board: 4, White: "C4", Black: strPtr("A4"), Result: codePtr(result.WhiteWins)},
		}, nil),
		makeMatch(2, 2, "B", strPtr("D"), true, []round.Game[string]{
			{Round: 2, Board: 1, White: "B1", Black: strPtr("D1"), Result: codePtr(result.WhiteWins)},
			{Round: 2, Board: 2, White: "D2", Black: strPtr("B2"), Result: codePtr(result.BlackWins)},
			{Round: 2, Board: 3, White: "B3", Black: strPtr("D3"), Result: codePtr(result.WhiteWins)},
			{Round: 2, Board: 4, White: "D4", Black: strPtr("B4"), Result: codePtr(result.BlackWins)},
		}, nil),
	}

	// Default PrimaryScore (MatchPoints): A (4 MP, 5.0 GP) > B (2 MP, 5.5 GP)
	byMP := RankTeams(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 4,
	})
	if len(byMP) != 4 {
		t.Fatalf("len(byMP) = %d, want 4", len(byMP))
	}
	if byMP[0].Score.ID != "A" || byMP[0].Score.Points != 4.0 || byMP[0].Score.MatchPoints != 4.0 || byMP[0].Score.GamePoints != 5.0 {
		t.Errorf("byMP[0] = %+v, want A with Points=4, MP=4, GP=5.0", byMP[0].Score)
	}
	if byMP[1].Score.ID != "B" || byMP[1].Score.Points != 2.0 || byMP[1].Score.MatchPoints != 2.0 || byMP[1].Score.GamePoints != 5.5 {
		t.Errorf("byMP[1] = %+v, want B with Points=2, MP=2, GP=5.5", byMP[1].Score)
	}

	// GamePoints PrimaryScore: B (5.5 GP, 2 MP) > A (5.0 GP, 4 MP) > C (1.5 GP) > D (0.0 GP)
	byGP := RankTeams(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 4,
		PrimaryScore:   TeamScoreGamePoints,
	})
	if byGP[0].Score.ID != "B" || byGP[0].Score.Points != 5.5 || byGP[0].Score.MatchPoints != 2.0 {
		t.Errorf("byGP[0] = %+v, want B with Points=5.5, MP=2", byGP[0].Score)
	}
	if byGP[1].Score.ID != "A" || byGP[1].Score.Points != 5.0 || byGP[1].Score.MatchPoints != 4.0 {
		t.Errorf("byGP[1] = %+v, want A with Points=5.0, MP=4", byGP[1].Score)
	}
	if byGP[2].Score.ID != "C" || byGP[2].Score.Points != 1.5 {
		t.Errorf("byGP[2] = %+v, want C with Points=1.5", byGP[2].Score)
	}
	if byGP[3].Score.ID != "D" || byGP[3].Score.Points != 0.0 {
		t.Errorf("byGP[3] = %+v, want D with Points=0.0", byGP[3].Score)
	}
}

func TestTeamScores_CustomMatchPointsAndByes(t *testing.T) {
	// Test default FIDE byes vs custom MatchPointsConfig on 4 boards:
	matches := []round.Match[string, string]{
		// Full-point bye for F
		makeMatch(1, 1, "F", nil, false, nil, codePtr(result.WhiteWins)),
		// Half-point bye for H
		makeMatch(1, 2, "H", nil, false, nil, codePtr(result.WhiteHalf)),
		// Zero-point bye for Z
		makeMatch(1, 3, "Z", nil, false, nil, codePtr(result.ZeroPointBye)),
		// Void free round for V (Paired against virtual slot in odd RR)
		makeMatch(1, 4, "V", nil, false, nil, codePtr(result.Void)),
	}

	// Default config: Full bye = 2 MP, 2.0 GP (50% of 4); Half bye = 1 MP, 2.0 GP; Zero/Void = 0 MP, 0 GP.
	scores := TeamScores(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 4,
	})
	byID := make(map[string]*TeamScore[string], len(scores))
	for _, s := range scores {
		byID[s.ID] = s
	}

	if byID["F"].MatchPoints != 2.0 || byID["F"].GamePoints != 2.0 {
		t.Errorf("default full bye: got MP=%v GP=%v, want MP=2.0 GP=2.0", byID["F"].MatchPoints, byID["F"].GamePoints)
	}
	if byID["H"].MatchPoints != 1.0 || byID["H"].GamePoints != 2.0 {
		t.Errorf("default half bye: got MP=%v GP=%v, want MP=1.0 GP=2.0", byID["H"].MatchPoints, byID["H"].GamePoints)
	}
	if byID["Z"].MatchPoints != 0.0 || byID["Z"].GamePoints != 0.0 {
		t.Errorf("default zero bye: got MP=%v GP=%v, want MP=0 GP=0", byID["Z"].MatchPoints, byID["Z"].GamePoints)
	}
	if byID["V"].MatchPoints != 0.0 || byID["V"].GamePoints != 0.0 {
		t.Errorf("void bye: got MP=%v GP=%v, want MP=0 GP=0", byID["V"].MatchPoints, byID["V"].GamePoints)
	}

	// Custom config: Win=3, Draw=1, Loss=0; Full bye = 3 MP, 4.0 GP (100% of boards); Half bye = 1 MP, 1.5 GP
	customScores := TeamScores(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 4,
		Scoring: &MatchPointsConfig{
			Win:             3.0,
			Draw:            1.0,
			Loss:            0.0,
			FullByeGamePts:  floatPtr(4.0),
			HalfByeMatchPts: floatPtr(1.0),
			HalfByeGamePts:  floatPtr(1.5),
		},
	})
	customByID := make(map[string]*TeamScore[string], len(customScores))
	for _, s := range customScores {
		customByID[s.ID] = s
	}
	if customByID["F"].MatchPoints != 3.0 || customByID["F"].GamePoints != 4.0 {
		t.Errorf("custom full bye: got MP=%v GP=%v, want MP=3.0 GP=4.0", customByID["F"].MatchPoints, customByID["F"].GamePoints)
	}
	if customByID["H"].MatchPoints != 1.0 || customByID["H"].GamePoints != 1.5 {
		t.Errorf("custom half bye: got MP=%v GP=%v, want MP=1.0 GP=1.5", customByID["H"].MatchPoints, customByID["H"].GamePoints)
	}
}

func TestTeamCrosstable(t *testing.T) {
	// 3 teams over 2 rounds on 2 boards:
	// Round 1: A (Home, WhiteOnFirstBoard=true) beats B 1.5-0.5; C gets full-point bye
	// Round 2: B (Home, WhiteOnFirstBoard=false) forfeits to C ("-/+" BlackForfeit); A has no pairing
	matches := []round.Match[string, string]{
		makeMatch(1, 1, "A", strPtr("B"), true, []round.Game[string]{
			{Round: 1, Board: 1, White: "A1", Black: strPtr("B1"), Result: codePtr(result.WhiteWins)},
			{Round: 1, Board: 2, White: "B2", Black: strPtr("A2"), Result: codePtr(result.Draw)},
		}, nil),
		makeMatch(1, 2, "C", nil, false, nil, codePtr(result.WhiteWins)),
		makeMatch(2, 1, "B", strPtr("C"), false, nil, codePtr(result.BlackForfeit)),
	}

	ct := TeamCrosstable(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 2,
	})

	if ct.TotalRounds != 2 {
		t.Fatalf("TotalRounds = %d, want 2", ct.TotalRounds)
	}
	if len(ct.Rows) != 3 {
		t.Fatalf("len(Rows) = %d, want 3", len(ct.Rows))
	}

	// Standings order:
	// 1. C: R1 full bye (2 MP, 1.0 GP) + R2 forfeit win (2 MP, 2.0 GP) = 4 MP, 3.0 GP
	// 2. A: R1 win (2 MP, 1.5 GP) + R2 unplayed = 2 MP, 1.5 GP
	// 3. B: R1 loss (0 MP, 0.5 GP) + R2 forfeit loss (0 MP, 0.0 GP) = 0 MP, 0.5 GP
	rowC := ct.Rows[0]
	if rowC.Score.ID != "C" || rowC.Rank != 1 || rowC.Score.MatchPoints != 4.0 || rowC.Score.GamePoints != 3.0 {
		t.Fatalf("row 0 = %+v (score %+v), want C rank 1 with 4 MP, 3.0 GP", rowC, rowC.Score)
	}
	// C Round 1: bye
	if !rowC.Rounds[0].IsBye || !rowC.Rounds[0].Played || rowC.Rounds[0].MatchPoints != 2.0 || rowC.Rounds[0].GamePoints != 1.0 {
		t.Errorf("C R1 = %+v, want bye with 2 MP, 1 GP", rowC.Rounds[0])
	}
	// C Round 2: away vs B (rank 3), WhiteOnFirstBoard=true (since match had WhiteOnFirstBoard=false for Home=B), forfeit win
	if rowC.Rounds[1].Home || !rowC.Rounds[1].WhiteOnFirstBoard || !rowC.Rounds[1].IsForfeit || rowC.Rounds[1].OpponentRank == nil || *rowC.Rounds[1].OpponentRank != 3 {
		t.Errorf("C R2 = %+v, want Away, WhiteOnFirstBoard=true, IsForfeit=true, OpponentRank=3", rowC.Rounds[1])
	}

	// A Round 2: not paired (Played == false)
	rowA := ct.Rows[1]
	if rowA.Score.ID != "A" || rowA.Rounds[1].Played {
		t.Errorf("A R2 = %+v, want Played=false", rowA.Rounds[1])
	}
}
