package standings_test

import (
	"testing"

	"github.com/ValkoM28/tournamentessentials/core/result"
	"github.com/ValkoM28/tournamentessentials/core/round"
	"github.com/ValkoM28/tournamentessentials/core/standings"
)

// intPtr is a helper to create an *int.
func intPtr(v int) *int {
	return &v
}

// TestBuildRoundResults_Basic tests a 4-player, 3-round tournament with all
// fields populated correctly (rank, color, points, played).
func TestBuildRoundResults_Basic(t *testing.T) {
	t.Log("=== Testing Basic Crosstable Round Results ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 (white) beats P2, P3 (white) beats P4
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)),
		// Round 2: P1 (white) beats P3, P2 (white) beats P4
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)),
		// Round 3: P1 (white) beats P4, P2 (white) draws P3
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.Draw)),
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)

	// Rankings: P1=1, P2=2, P3=3, P4=4
	rankMap := map[string]int{
		p1: 1, p2: 2, p3: 3, p4: 4,
	}

	totalRounds := 3

	// --- P1: 3 wins, all as white ---
	t.Log("\nP1 round results (3 wins as white):")
	results := standings.BuildRoundResults(p1, allInfo[p1], rankMap, totalRounds)
	if len(results) != totalRounds {
		t.Fatalf("P1: expected %d round results, got %d", totalRounds, len(results))
	}

	// Round 1: white vs P2 (rank 2), won 1.0
	assertRoundResult(t, results[0], 1, intPtr(2), "w", 1.0, false, false, true)
	// Round 2: white vs P3 (rank 3), won 1.0
	assertRoundResult(t, results[1], 2, intPtr(3), "w", 1.0, false, false, true)
	// Round 3: white vs P4 (rank 4), won 1.0
	assertRoundResult(t, results[2], 3, intPtr(4), "w", 1.0, false, false, true)

	// --- P2: 1 win, 1 loss, 1 draw ---
	t.Log("\nP2 round results:")
	results = standings.BuildRoundResults(p2, allInfo[p2], rankMap, totalRounds)
	if len(results) != totalRounds {
		t.Fatalf("P2: expected %d round results, got %d", totalRounds, len(results))
	}

	// Round 1: black vs P1 (rank 1), lost 0.0
	assertRoundResult(t, results[0], 1, intPtr(1), "b", 0.0, false, false, true)
	// Round 2: white vs P4 (rank 4), won 1.0
	assertRoundResult(t, results[1], 2, intPtr(4), "w", 1.0, false, false, true)
	// Round 3: white vs P3 (rank 3), drew 0.5
	assertRoundResult(t, results[2], 3, intPtr(3), "w", 0.5, false, false, true)

	// --- P4: 3 losses, all as black ---
	t.Log("\nP4 round results (3 losses as black):")
	results = standings.BuildRoundResults(p4, allInfo[p4], rankMap, totalRounds)
	if len(results) != totalRounds {
		t.Fatalf("P4: expected %d round results, got %d", totalRounds, len(results))
	}

	// Round 1: black vs P3 (rank 3), lost 0.0
	assertRoundResult(t, results[0], 1, intPtr(3), "b", 0.0, false, false, true)
	// Round 2: black vs P2 (rank 2), lost 0.0
	assertRoundResult(t, results[1], 2, intPtr(2), "b", 0.0, false, false, true)
	// Round 3: black vs P1 (rank 1), lost 0.0
	assertRoundResult(t, results[2], 3, intPtr(1), "b", 0.0, false, false, true)

	t.Log("\n✓ Basic crosstable round results passed!")
}

// TestBuildRoundResults_Bye tests that a bye round shows is_bye=true,
// no opponent rank, and correct points.
func TestBuildRoundResults_Bye(t *testing.T) {
	t.Log("=== Testing Bye Handling in Crosstable ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 gets a full-point bye
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, nil, codePtr(result.WhiteWins)), // bye: black=nil, result=1-0
		// Round 2: P1 beats P3, P2 gets a half-point bye
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, nil, codePtr(result.WhiteHalf)), // half-point bye
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)
	rankMap := map[string]int{p1: 1, p2: 2, p3: 3}
	totalRounds := 2

	// P3 Round 1: bye with 1.0 point
	t.Log("\nP3 round 1 should be a bye:")
	results := standings.BuildRoundResults(p3, allInfo[p3], rankMap, totalRounds)
	if len(results) != totalRounds {
		t.Fatalf("P3: expected %d results, got %d", totalRounds, len(results))
	}

	r1 := results[0]
	if !r1.IsBye {
		t.Error("P3 R1: expected is_bye=true")
	}
	if r1.OpponentRank != nil {
		t.Errorf("P3 R1: expected nil opponent_rk for bye, got %d", *r1.OpponentRank)
	}
	if r1.Color != "" {
		t.Errorf("P3 R1: expected empty color for bye, got %q", r1.Color)
	}
	if r1.Points != 1.0 {
		t.Errorf("P3 R1: expected 1.0 points for full-point bye, got %f", r1.Points)
	}
	if !r1.Played {
		t.Error("P3 R1: expected played=true (bye is a pairing)")
	}
	t.Logf("  ✓ Bye round: is_bye=%v, opponent_rk=%v, color=%q, points=%.1f, played=%v",
		r1.IsBye, r1.OpponentRank, r1.Color, r1.Points, r1.Played)

	// P2 Round 2: half-point bye
	t.Log("\nP2 round 2 should be a half-point bye:")
	results = standings.BuildRoundResults(p2, allInfo[p2], rankMap, totalRounds)
	r2 := results[1]
	if !r2.IsBye {
		t.Error("P2 R2: expected is_bye=true")
	}
	if r2.Points != 0.5 {
		t.Errorf("P2 R2: expected 0.5 points for half-point bye, got %f", r2.Points)
	}
	t.Logf("  ✓ Half-point bye: is_bye=%v, points=%.1f", r2.IsBye, r2.Points)

	t.Log("\n✓ Bye handling passed!")
}

// TestBuildRoundResults_Forfeit tests that forfeit results are flagged correctly.
func TestBuildRoundResults_Forfeit(t *testing.T) {
	t.Log("=== Testing Forfeit Handling in Crosstable ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		// Round 1: normal game
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.Draw)),
		// Round 2: P1 wins by forfeit (white forfeit = "+/-" means white gets the point)
		makePairing(2, p1, &p3, codePtr(result.WhiteForfeit)),
		// Round 2: P4 wins by forfeit (black forfeit = "-/+" means black gets the point)
		makePairing(2, p2, &p4, codePtr(result.BlackForfeit)),
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)
	rankMap := map[string]int{p1: 1, p2: 2, p3: 3, p4: 4}
	totalRounds := 2

	// P1 Round 2: white forfeit win (+/-)
	t.Log("\nP1 round 2 should be a forfeit win:")
	results := standings.BuildRoundResults(p1, allInfo[p1], rankMap, totalRounds)
	r2 := results[1]
	if !r2.IsForfeit {
		t.Error("P1 R2: expected is_forfeit=true for +/-")
	}
	if r2.Points != 1.0 {
		t.Errorf("P1 R2: expected 1.0 points for forfeit win, got %f", r2.Points)
	}
	if r2.Color != "w" {
		t.Errorf("P1 R2: expected color 'w', got %q", r2.Color)
	}
	t.Logf("  ✓ White forfeit win: is_forfeit=%v, points=%.1f, color=%s", r2.IsForfeit, r2.Points, r2.Color)

	// P3 Round 2: lost by forfeit (opponent got +/-)
	t.Log("\nP3 round 2 should be a forfeit loss:")
	results = standings.BuildRoundResults(p3, allInfo[p3], rankMap, totalRounds)
	r2 = results[1]
	if !r2.IsForfeit {
		t.Error("P3 R2: expected is_forfeit=true for +/-")
	}
	if r2.Points != 0.0 {
		t.Errorf("P3 R2: expected 0.0 points for forfeit loss, got %f", r2.Points)
	}
	t.Logf("  ✓ Black forfeit loss: is_forfeit=%v, points=%.1f", r2.IsForfeit, r2.Points)

	// P4 Round 2: black forfeit win (-/+)
	t.Log("\nP4 round 2 should be a forfeit win as black:")
	results = standings.BuildRoundResults(p4, allInfo[p4], rankMap, totalRounds)
	r2 = results[1]
	if !r2.IsForfeit {
		t.Error("P4 R2: expected is_forfeit=true for -/+")
	}
	if r2.Points != 1.0 {
		t.Errorf("P4 R2: expected 1.0 points for forfeit win, got %f", r2.Points)
	}
	if r2.Color != "b" {
		t.Errorf("P4 R2: expected color 'b', got %q", r2.Color)
	}
	t.Logf("  ✓ Black forfeit win: is_forfeit=%v, points=%.1f, color=%s", r2.IsForfeit, r2.Points, r2.Color)

	// P2 Round 2: lost by forfeit (opponent got -/+)
	t.Log("\nP2 round 2 should be a forfeit loss:")
	results = standings.BuildRoundResults(p2, allInfo[p2], rankMap, totalRounds)
	r2 = results[1]
	if !r2.IsForfeit {
		t.Error("P2 R2: expected is_forfeit=true for -/+")
	}
	if r2.Points != 0.0 {
		t.Errorf("P2 R2: expected 0.0 points for forfeit loss, got %f", r2.Points)
	}
	t.Logf("  ✓ White forfeit loss: is_forfeit=%v, points=%.1f", r2.IsForfeit, r2.Points)

	t.Log("\n✓ Forfeit handling passed!")
}

// TestBuildRoundResults_PartialRounds tests that when we only have pairings
// through round 2, the result array has exactly 2 entries.
func TestBuildRoundResults_PartialRounds(t *testing.T) {
	t.Log("=== Testing Partial Rounds (crosstable after round 2 of 3) ===")

	p1 := newID()
	p2 := newID()

	// Only rounds 1 and 2 (simulating upToRound = round 2)
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p1, codePtr(result.Draw)),
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)
	rankMap := map[string]int{p1: 1, p2: 2}
	totalRounds := 2 // only 2 rounds included

	results := standings.BuildRoundResults(p1, allInfo[p1], rankMap, totalRounds)
	if len(results) != 2 {
		t.Fatalf("expected 2 round results, got %d", len(results))
	}

	// Round 1: white, won
	assertRoundResult(t, results[0], 1, intPtr(2), "w", 1.0, false, false, true)
	// Round 2: black, drew
	assertRoundResult(t, results[1], 2, intPtr(2), "b", 0.5, false, false, true)

	t.Log("  ✓ Partial round results only include rounds 1-2")
	t.Log("\n✓ Partial rounds passed!")
}

// TestBuildRoundResults_NoPairingInRound tests that a player with no pairing
// in a given round gets a {round: N, played: false} entry with no gaps.
func TestBuildRoundResults_NoPairingInRound(t *testing.T) {
	t.Log("=== Testing Player With No Pairing in a Round ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	// 3 rounds, but P3 only plays in rounds 1 and 3 (skips round 2)
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p2, codePtr(result.Draw)),
		makePairing(3, p3, &p2, codePtr(result.BlackWins)),
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)
	rankMap := map[string]int{p1: 1, p2: 2, p3: 3}
	totalRounds := 3

	results := standings.BuildRoundResults(p3, allInfo[p3], rankMap, totalRounds)
	if len(results) != totalRounds {
		t.Fatalf("expected %d round results (no gaps), got %d", totalRounds, len(results))
	}

	// Round 1: black vs P1 (rank 1), lost
	t.Log("\nP3 Round 1: played as black, lost")
	assertRoundResult(t, results[0], 1, intPtr(1), "b", 0.0, false, false, true)

	// Round 2: no pairing
	t.Log("P3 Round 2: no pairing (should be played=false)")
	r2 := results[1]
	if r2.Round != 2 {
		t.Errorf("expected round 2, got %d", r2.Round)
	}
	if r2.Played {
		t.Error("expected played=false for round with no pairing")
	}
	if r2.OpponentRank != nil {
		t.Errorf("expected nil opponent_rk, got %d", *r2.OpponentRank)
	}
	if r2.Color != "" {
		t.Errorf("expected empty color, got %q", r2.Color)
	}
	if r2.Points != 0 {
		t.Errorf("expected 0 points, got %f", r2.Points)
	}
	if r2.IsBye || r2.IsForfeit {
		t.Error("expected is_bye=false and is_forfeit=false for unplayed round")
	}
	t.Logf("  ✓ Round 2: played=%v, opponent_rk=%v, points=%.1f", r2.Played, r2.OpponentRank, r2.Points)

	// Round 3: white vs P2 (rank 2), black won (P3 is white, result=0-1 means P3 lost)
	// Wait, P3 is white and result is BlackWins, so P3 gets 0 points... no, let me re-check.
	// makePairing(3, p3, &p2, codePtr(result.BlackWins)) → P3 is white, P2 is black, black wins
	// So P3 loses, P2 wins.
	// Actually I want P3 to win. Let me check... ResultBlackWins = "0-1" means black wins.
	// P3 is white here, so P3 loses. Let me fix: P2 is black and wins. So P3 (white) gets 0.
	// That's fine for the test - the point is the gap handling.
	t.Log("P3 Round 3: played as white, lost")
	assertRoundResult(t, results[2], 3, intPtr(2), "w", 0.0, false, false, true)

	t.Log("\n✓ No-pairing round correctly fills gap with played=false!")
}

// TestBuildRoundResults_NilInfo tests that a player with nil info
// (no pairings at all) gets all unplayed rounds.
func TestBuildRoundResults_NilInfo(t *testing.T) {
	t.Log("=== Testing Player With No Pairings At All ===")

	p1 := newID()
	rankMap := map[string]int{p1: 1}
	totalRounds := 3

	results := standings.BuildRoundResults(p1, nil, rankMap, totalRounds)
	if len(results) != totalRounds {
		t.Fatalf("expected %d round results, got %d", totalRounds, len(results))
	}

	for i, r := range results {
		if r.Round != i+1 {
			t.Errorf("result %d: expected round %d, got %d", i, i+1, r.Round)
		}
		if r.Played {
			t.Errorf("round %d: expected played=false", r.Round)
		}
		if r.Points != 0 {
			t.Errorf("round %d: expected 0 points, got %f", r.Round, r.Points)
		}
	}

	t.Log("  ✓ All 3 rounds are unplayed")
	t.Log("\n✓ Nil info handling passed!")
}

// TestBuildRoundResults_DrawResult verifies draw points and colors.
func TestBuildRoundResults_DrawResult(t *testing.T) {
	t.Log("=== Testing Draw Result ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.Draw)),
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)
	rankMap := map[string]int{p1: 1, p2: 2}

	// White side
	results := standings.BuildRoundResults(p1, allInfo[p1], rankMap, 1)
	assertRoundResult(t, results[0], 1, intPtr(2), "w", 0.5, false, false, true)
	t.Log("  ✓ White draw: 0.5 points")

	// Black side
	results = standings.BuildRoundResults(p2, allInfo[p2], rankMap, 1)
	assertRoundResult(t, results[0], 1, intPtr(1), "b", 0.5, false, false, true)
	t.Log("  ✓ Black draw: 0.5 points")

	t.Log("\n✓ Draw handling passed!")
}

// TestBuildRoundResults_ZeroPointBye tests zero-point bye (withdrawal).
func TestBuildRoundResults_ZeroPointBye(t *testing.T) {
	t.Log("=== Testing Zero-Point Bye ===")

	p1 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.ZeroPointBye)),
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)
	rankMap := map[string]int{p1: 1}

	results := standings.BuildRoundResults(p1, allInfo[p1], rankMap, 1)
	r := results[0]

	if !r.IsBye {
		t.Error("expected is_bye=true for zero-point bye")
	}
	if r.Points != 0.0 {
		t.Errorf("expected 0.0 points for zero-point bye, got %f", r.Points)
	}
	if r.OpponentRank != nil {
		t.Error("expected nil opponent_rk for bye")
	}
	if !r.Played {
		t.Error("expected played=true (bye is still a pairing)")
	}

	t.Logf("  ✓ Zero-point bye: is_bye=%v, points=%.1f, played=%v", r.IsBye, r.Points, r.Played)
	t.Log("\n✓ Zero-point bye passed!")
}

// TestBuildRoundResults_UnplayedResult tests a pairing with nil result.
func TestBuildRoundResults_UnplayedResult(t *testing.T) {
	t.Log("=== Testing Unplayed (nil result) ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, nil), // paired but no result yet
	}

	allInfo := standings.BuildPlayerPairingInfo(pairings)
	rankMap := map[string]int{p1: 1, p2: 2}

	results := standings.BuildRoundResults(p1, allInfo[p1], rankMap, 1)
	r := results[0]

	if !r.Played {
		t.Error("expected played=true (pairing exists)")
	}
	if r.Points != 0.0 {
		t.Errorf("expected 0.0 points for nil result, got %f", r.Points)
	}
	if r.Color != "w" {
		t.Errorf("expected color 'w', got %q", r.Color)
	}
	if r.IsForfeit || r.IsBye {
		t.Error("expected no forfeit or bye flags")
	}

	t.Logf("  ✓ Unplayed pairing: played=%v, points=%.1f, color=%s", r.Played, r.Points, r.Color)
	t.Log("\n✓ Unplayed result passed!")
}

// assertRoundResult is a helper to validate a RoundResult.
func assertRoundResult(t *testing.T, r standings.RoundResult, round int, oppRk *int, color string, points float64, isForfeit, isBye, played bool) {
	t.Helper()

	if r.Round != round {
		t.Errorf("expected round %d, got %d", round, r.Round)
	}
	if oppRk == nil {
		if r.OpponentRank != nil {
			t.Errorf("round %d: expected nil opponent_rk, got %d", round, *r.OpponentRank)
		}
	} else {
		if r.OpponentRank == nil {
			t.Errorf("round %d: expected opponent_rk=%d, got nil", round, *oppRk)
		} else if *r.OpponentRank != *oppRk {
			t.Errorf("round %d: expected opponent_rk=%d, got %d", round, *oppRk, *r.OpponentRank)
		}
	}
	if r.Color != color {
		t.Errorf("round %d: expected color=%q, got %q", round, color, r.Color)
	}
	if r.Points != points {
		t.Errorf("round %d: expected points=%.1f, got %.1f", round, points, r.Points)
	}
	if r.IsForfeit != isForfeit {
		t.Errorf("round %d: expected is_forfeit=%v, got %v", round, isForfeit, r.IsForfeit)
	}
	if r.IsBye != isBye {
		t.Errorf("round %d: expected is_bye=%v, got %v", round, isBye, r.IsBye)
	}
	if r.Played != played {
		t.Errorf("round %d: expected played=%v, got %v", round, played, r.Played)
	}
}
