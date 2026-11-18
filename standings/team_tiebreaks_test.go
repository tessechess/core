package standings

import (
	"testing"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

func twoBoardMatch(roundNum, table int, home, away string, b1Res, b2Res result.Code) round.Match[string, string] {
	return round.Match[string, string]{
		Round:             roundNum,
		Table:             table,
		Home:              home,
		Away:              strPtr(away),
		WhiteOnFirstBoard: true,
		Boards: []round.Game[string]{
			{Round: roundNum, Board: 1, White: home + "1", Black: strPtr(away + "1"), Result: codePtr(b1Res)},
			{Round: roundNum, Board: 2, White: away + "2", Black: strPtr(home + "2"), Result: codePtr(b2Res)},
		},
	}
}

func TestTeamTiebreaks_MatchPointsAndGamePoints(t *testing.T) {
	// R1: A beats C 1.5-0.5 (A: 2 MP, 1.5 GP; C: 0 MP, 0.5 GP)
	// R1: B beats D 2.0-0.0 (B: 2 MP, 2.0 GP; D: 0 MP, 0.0 GP)
	matches := []round.Match[string, string]{
		twoBoardMatch(1, 1, "A", "C", result.WhiteWins, result.Draw),
		twoBoardMatch(1, 2, "B", "D", result.WhiteWins, result.BlackWins),
	}

	// Both A and B have 2 MP, but B has 2.0 GP vs A's 1.5 GP.
	ranked := RankTeams(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 2,
		PrimaryScore:   TeamScoreMatchPoints,
		Tiebreaks:      []TiebreakConfig{{Key: GamePoints}, {Key: MatchPoints}},
	})
	if ranked[0].Score.ID != "B" || ranked[0].Score.Tiebreaks["game_points"] != 2.0 || ranked[0].Score.Tiebreaks["match_points"] != 2.0 {
		t.Errorf("ranked[0] = %+v, want B with GP=2.0, MP=2.0", ranked[0].Score)
	}
	if ranked[1].Score.ID != "A" || ranked[1].Score.Tiebreaks["game_points"] != 1.5 {
		t.Errorf("ranked[1] = %+v, want A with GP=1.5", ranked[1].Score)
	}
}

func TestTeamTiebreaks_DirectEncounter(t *testing.T) {
	// 3 teams all finish with 2 MP and 2.0 GP:
	// R1: A beats B 1.5-0.5 (A: 2 MP, 1.5 GP; B: 0 MP, 0.5 GP)
	// R2: B beats C 1.5-0.5 (B: 2 MP, 2.0 GP; C: 0 MP, 0.5 GP)
	// R2: A loses to D 0.5-1.5 (A: 2 MP, 2.0 GP; D: 2 MP, 1.5 GP)
	// Between A and B (both 2 MP, 2.0 GP), A beat B head-to-head!
	matches := []round.Match[string, string]{
		twoBoardMatch(1, 1, "A", "B", result.WhiteWins, result.Draw),
		twoBoardMatch(1, 2, "C", "D", result.Draw, result.Draw), // C: 1 MP, 1.0 GP; D: 1 MP, 1.0 GP
		twoBoardMatch(2, 1, "B", "C", result.WhiteWins, result.Draw),
		twoBoardMatch(2, 2, "D", "A", result.WhiteWins, result.Draw), // D ends with 3 MP; A and B both have 2 MP, 2.0 GP
	}

	ranked := RankTeams(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 2,
		Tiebreaks:      []TiebreakConfig{{Key: GamePoints}, {Key: DirectEncounter}},
	})

	// 1st: D (3 MP)
	// 2nd: A (2 MP, 2.0 GP, DE=2.0 against B)
	// 3rd: B (2 MP, 2.0 GP, DE=0.0 against A)
	// 4th: C (1 MP)
	if ranked[0].Score.ID != "D" {
		t.Errorf("rank 1 = %s, want D", ranked[0].Score.ID)
	}
	if ranked[1].Score.ID != "A" || ranked[1].Score.Tiebreaks["direct_encounter"] != 2.0 {
		t.Errorf("rank 2 = %+v, want A with DE=2.0", ranked[1].Score)
	}
	if ranked[2].Score.ID != "B" || ranked[2].Score.Tiebreaks["direct_encounter"] != 0.0 {
		t.Errorf("rank 3 = %+v, want B with DE=0.0", ranked[2].Score)
	}
}

func TestTeamTiebreaks_ExtendedSonnebornBerger(t *testing.T) {
	// 4 teams in 2 rounds on 2 boards:
	// R1: A beats B 1.5-0.5 (A: 2 MP, 1.5 GP; B: 0 MP, 0.5 GP)
	// R1: C draws D 1.0-1.0 (C: 1 MP, 1.0 GP; D: 1 MP, 1.0 GP)
	// R2: A draws C 1.0-1.0 (A: +1 MP, +1.0 GP -> total 3 MP, 2.5 GP; C: +1 MP, +1.0 GP -> total 2 MP, 2.0 GP)
	// R2: B beats D 2.0-0.0 (B: +2 MP, +2.0 GP -> total 2 MP, 2.5 GP; D: +0 MP, +0.0 GP -> total 1 MP, 1.0 GP)
	matches := []round.Match[string, string]{
		twoBoardMatch(1, 1, "A", "B", result.WhiteWins, result.Draw),
		twoBoardMatch(1, 2, "C", "D", result.Draw, result.Draw),
		twoBoardMatch(2, 1, "A", "C", result.Draw, result.Draw),
		twoBoardMatch(2, 2, "B", "D", result.WhiteWins, result.BlackWins),
	}

	// Totals:
	// A: 3 MP, 2.5 GP (opponents: B [2 MP, 2.5 GP], C [2 MP, 2.0 GP])
	//   In R1 vs B: A scored 2 MP, 1.5 GP
	//   In R2 vs C: A scored 1 MP, 1.0 GP
	//   A's EMMSB = (B.MP * 2) + (C.MP * 1) = (2 * 2) + (2 * 1) = 6.0
	//   A's EMGSB = (B.MP * 1.5) + (C.MP * 1.0) = (2 * 1.5) + (2 * 1.0) = 5.0
	//   A's EGMSB = (B.GP * 2) + (C.GP * 1) = (2.5 * 2) + (2.0 * 1) = 7.0
	//   A's EGGSB = (B.GP * 1.5) + (C.GP * 1.0) = (2.5 * 1.5) + (2.0 * 1.0) = 5.75
	scores := TeamScores(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 2,
		Tiebreaks: []TiebreakConfig{
			{Key: ExtendedMatchMatchSB},
			{Key: ExtendedMatchGameSB},
			{Key: ExtendedGameMatchSB},
			{Key: ExtendedGameGameSB},
			{Key: SonnebornBerger},
		},
	})

	byID := make(map[string]*TeamScore[string], len(scores))
	for _, s := range scores {
		byID[s.ID] = s
	}
	tbA := byID["A"].Tiebreaks
	if tbA["emmsb"] != 6.0 {
		t.Errorf("A EMMSB = %v, want 6.0", tbA["emmsb"])
	}
	if tbA["emgsb"] != 5.0 {
		t.Errorf("A EMGSB = %v, want 5.0", tbA["emgsb"])
	}
	if tbA["egmsb"] != 7.0 {
		t.Errorf("A EGMSB = %v, want 7.0", tbA["egmsb"])
	}
	if tbA["eggsb"] != 5.75 {
		t.Errorf("A EGGSB = %v, want 5.75", tbA["eggsb"])
	}
	if tbA["sonneborn_berger"] != 6.0 {
		t.Errorf("A SB (MP primary) = %v, want 6.0", tbA["sonneborn_berger"])
	}
}

func TestTeamTiebreaks_BoardCountAndTopBoardResults(t *testing.T) {
	// 3 teams on 3 boards (weights: Board 1 = 3, Board 2 = 2, Board 3 = 1):
	// Team A: Board 1 wins (1.0), Board 2 draws (0.5), Board 3 loses (0.0) -> 1.5 GP, BC = 3*1 + 2*0.5 + 1*0 = 4.0
	// Team B: Board 1 loses (0.0), Board 2 draws (0.5), Board 3 wins (1.0) -> 1.5 GP, BC = 3*0 + 2*0.5 + 1*1 = 2.0
	matches := []round.Match[string, string]{
		{
			Round:             1,
			Table:             1,
			Home:              "A",
			Away:              strPtr("B"),
			WhiteOnFirstBoard: true,
			Boards: []round.Game[string]{
				{Round: 1, Board: 1, White: "A1", Black: strPtr("B1"), Result: codePtr(result.WhiteWins)},
				{Round: 1, Board: 2, White: "B2", Black: strPtr("A2"), Result: codePtr(result.Draw)},
				{Round: 1, Board: 3, White: "A3", Black: strPtr("B3"), Result: codePtr(result.BlackWins)},
			},
		},
	}

	rankedBC := RankTeams(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 3,
		Tiebreaks:      []TiebreakConfig{{Key: BoardCount}},
	})
	if rankedBC[0].Score.ID != "A" || rankedBC[0].Score.Tiebreaks["board_count"] != 4.0 {
		t.Errorf("rankedBC[0] = %+v, want A with BC=4.0", rankedBC[0].Score)
	}
	if rankedBC[1].Score.ID != "B" || rankedBC[1].Score.Tiebreaks["board_count"] != 2.0 {
		t.Errorf("rankedBC[1] = %+v, want B with BC=2.0", rankedBC[1].Score)
	}

	// TopBoardResults: A scored 1.0 on Board 1, B scored 0.0 on Board 1.
	rankedTBR := RankTeams(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 3,
		Tiebreaks:      []TiebreakConfig{{Key: TopBoardResults}},
	})
	if rankedTBR[0].Score.ID != "A" || rankedTBR[0].Score.Tiebreaks["top_board_results"] != 1.0 {
		t.Errorf("rankedTBR[0] = %+v, want A with TBR=1.0", rankedTBR[0].Score)
	}
	if rankedTBR[1].Score.ID != "B" || rankedTBR[1].Score.Tiebreaks["top_board_results"] != 0.0 {
		t.Errorf("rankedTBR[1] = %+v, want B with TBR=0.0", rankedTBR[1].Score)
	}

	// Multi-board TBR: 3 tied teams where A and B tie on Board 1 (both 1.0) and differ on Board 2 (A=1.0, B=0.5),
	// while C has 0.5 on Board 1 and 1.0 on Board 2. Order must be A > B > C.
	matchesMulti := []round.Match[string, string]{
		{
			Round:             1,
			Table:             1,
			Home:              "A",
			Away:              strPtr("X"),
			WhiteOnFirstBoard: true,
			Boards: []round.Game[string]{
				{Round: 1, Board: 1, White: "A1", Black: strPtr("X1"), Result: codePtr(result.WhiteWins)}, // A B1 = 1.0
				{Round: 1, Board: 2, White: "X2", Black: strPtr("A2"), Result: codePtr(result.BlackWins)}, // A B2 = 1.0
				{Round: 1, Board: 3, White: "A3", Black: strPtr("X3"), Result: codePtr(result.BlackWins)}, // A B3 = 0.0 (total 2.0 GP, 2 MP)
			},
		},
		{
			Round:             1,
			Table:             2,
			Home:              "B",
			Away:              strPtr("Y"),
			WhiteOnFirstBoard: true,
			Boards: []round.Game[string]{
				{Round: 1, Board: 1, White: "B1", Black: strPtr("Y1"), Result: codePtr(result.WhiteWins)}, // B B1 = 1.0
				{Round: 1, Board: 2, White: "Y2", Black: strPtr("B2"), Result: codePtr(result.Draw)},      // B B2 = 0.5
				{Round: 1, Board: 3, White: "B3", Black: strPtr("Y3"), Result: codePtr(result.Draw)},      // B B3 = 0.5 (total 2.0 GP, 2 MP)
			},
		},
		{
			Round:             1,
			Table:             3,
			Home:              "C",
			Away:              strPtr("Z"),
			WhiteOnFirstBoard: true,
			Boards: []round.Game[string]{
				{Round: 1, Board: 1, White: "C1", Black: strPtr("Z1"), Result: codePtr(result.Draw)},      // C B1 = 0.5
				{Round: 1, Board: 2, White: "Z2", Black: strPtr("C2"), Result: codePtr(result.BlackWins)}, // C B2 = 1.0
				{Round: 1, Board: 3, White: "C3", Black: strPtr("Z3"), Result: codePtr(result.Draw)},      // C B3 = 0.5 (total 2.0 GP, 2 MP)
			},
		},
	}

	rankedMulti := RankTeams(TeamInput[string, string]{
		Matches:        matchesMulti,
		BoardsPerMatch: 3,
		Tiebreaks:      []TiebreakConfig{{Key: GamePoints}, {Key: TopBoardResults}},
	})
	if rankedMulti[0].Score.ID != "A" || rankedMulti[1].Score.ID != "B" || rankedMulti[2].Score.ID != "C" {
		t.Errorf("multi-board TBR top 3 = [%s, %s, %s], want [A, B, C]",
			rankedMulti[0].Score.ID, rankedMulti[1].Score.ID, rankedMulti[2].Score.ID)
	}
}

func TestTeamTiebreaks_BuchholzForeBuchholzAOBKoyaWinsPSGE(t *testing.T) {
	// 4 teams over 2 rounds on 2 boards:
	// R1: A beats B 1.5-0.5 (A: 2 MP, B: 0 MP); C beats D 1.5-0.5 (C: 2 MP, D: 0 MP)
	// R2: A beats C 1.5-0.5 (A: 4 MP, C: 2 MP); B vs D unplayed (nil results on boards)
	matches := []round.Match[string, string]{
		twoBoardMatch(1, 1, "A", "B", result.WhiteWins, result.Draw),
		twoBoardMatch(1, 2, "C", "D", result.WhiteWins, result.Draw),
		twoBoardMatch(2, 1, "A", "C", result.WhiteWins, result.Draw),
		{
			Round:             2,
			Table:             2,
			Home:              "B",
			Away:              strPtr("D"),
			WhiteOnFirstBoard: true,
			Boards: []round.Game[string]{
				{Round: 2, Board: 1, White: "B1", Black: strPtr("D1")},
				{Round: 2, Board: 2, White: "D2", Black: strPtr("B2")},
			},
		},
	}

	scores := TeamScores(TeamInput[string, string]{
		Matches:        matches,
		BoardsPerMatch: 2,
		Tiebreaks: []TiebreakConfig{
			{Key: Buchholz},
			{Key: Buchholz, CutLowest: 1},
			{Key: ForeBuchholz},
			{Key: AverageOpponentBuchholz},
			{Key: KoyaSystem},
			{Key: Wins},
			{Key: GamesWon},
			{Key: ProgressiveScores},
			{Key: GamesElectedToPlay},
		},
	})

	byID := make(map[string]*TeamScore[string], len(scores))
	for _, s := range scores {
		byID[s.ID] = s
	}

	// A faced B (0 MP) and C (2 MP):
	// - Buchholz = 0 + 2 = 2.0
	// - Buchholz Cut-1 = 2.0
	// - ForeBuchholz (B vs D in R2 simulated as draw: B becomes 1 MP, D becomes 1 MP) -> A's FB = 1 + 2 = 3.0
	// - KoyaSystem (maxRound=2 -> default threshold = 2 * 0.5 * 2 = 2.0 MP): C has 2.0 MP >= 2.0, A scored 2 MP against C -> KS = 2.0
	// - Wins = 2, GamesWon = 2
	// - ProgressiveScores = R1(2) + R2(4) = 6.0
	// - GamesElectedToPlay = 2
	tbA := byID["A"].Tiebreaks
	if tbA["buchholz"] != 2.0 {
		t.Errorf("A BH = %v, want 2.0", tbA["buchholz"])
	}
	if tbA["buchholz_cut1"] != 2.0 {
		t.Errorf("A BH-C1 = %v, want 2.0", tbA["buchholz_cut1"])
	}
	if tbA["fore_buchholz"] != 3.0 {
		t.Errorf("A FB = %v, want 3.0", tbA["fore_buchholz"])
	}
	if tbA["koya_system"] != 2.0 {
		t.Errorf("A KS = %v, want 2.0", tbA["koya_system"])
	}
	if tbA["wins"] != 2.0 || tbA["games_won"] != 2.0 {
		t.Errorf("A Wins=%v GamesWon=%v, want 2, 2", tbA["wins"], tbA["games_won"])
	}
	if tbA["progressive_scores"] != 6.0 {
		t.Errorf("A PS = %v, want 6.0", tbA["progressive_scores"])
	}
	if tbA["games_elected_to_play"] != 2.0 {
		t.Errorf("A GE = %v, want 2.0", tbA["games_elected_to_play"])
	}
}
