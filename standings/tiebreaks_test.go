package standings

import (
	"fmt"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

// idSeq backs newID, which gives each test player a unique 8-character ID.
var idSeq atomic.Int64

func newID() string { return fmt.Sprintf("p%07d", idSeq.Add(1)) }

// Helper to create a PairingRow
func makePairing(roundNum int, whiteID string, blackID *string, res *result.Code) *round.Game[string] {
	return &round.Game[string]{
		Round:  roundNum,
		White:  whiteID,
		Black:  blackID,
		Result: res,
	}
}

// Helper to convert a result code to pointer
func codePtr(c result.Code) *result.Code {
	return &c
}

// TestComputeBuchholz_BasicCalculation tests plain Buchholz (no cuts)
func TestComputeBuchholz_BasicCalculation(t *testing.T) {
	t.Log("=== Testing Basic Buchholz Calculation (no cuts) ===")

	// Setup: 4 players, 3 rounds
	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	t.Logf("Player IDs: P1=%s, P2=%s, P3=%s, P4=%s", p1[:8], p2[:8], p3[:8], p4[:8])

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 beats P4
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // P1: 1, P2: 0
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)), // P3: 1, P4: 0
		// Round 2: P1 beats P3, P2 beats P4
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)), // P1: 2, P3: 1
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)), // P2: 1, P4: 0
		// Round 3: P1 beats P4, P2 draws P3
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)), // P1: 3, P4: 0
		makePairing(3, p2, &p3, codePtr(result.Draw)),      // P2: 1.5, P3: 1.5
	}

	t.Log("Tournament scenario:")
	t.Log("  Round 1: P1 beats P2 (1-0), P3 beats P4 (1-0)")
	t.Log("  Round 2: P1 beats P3 (1-0), P2 beats P4 (1-0)")
	t.Log("  Round 3: P1 beats P4 (1-0), P2 draws P3 (0.5-0.5)")

	// Build info and scores
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0, // 3 wins
		p2: 1.5, // 1 win, 1 draw
		p3: 1.5, // 1 win, 1 draw
		p4: 0.0, // 0 wins
	}

	t.Log("\nFinal Scores:")
	t.Logf("  P1: %.1f points (3 wins)", rawScores[p1])
	t.Logf("  P2: %.1f points (1 win, 1 draw)", rawScores[p2])
	t.Logf("  P3: %.1f points (1 win, 1 draw)", rawScores[p3])
	t.Logf("  P4: %.1f points (0 wins)", rawScores[p4])

	// P1 played: P2 (1.5), P3 (1.5), P4 (0.0) → Buchholz = 3.0
	t.Log("\nCalculating P1 Buchholz:")
	t.Log("  P1 opponents: P2 (1.5), P3 (1.5), P4 (0.0)")
	t.Log("  Expected Buchholz: 1.5 + 1.5 + 0.0 = 3.0")
	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Actual Buchholz: %.1f", bh)
	if bh != 3.0 {
		t.Errorf("P1 Buchholz: expected 3.0, got %f", bh)
	}

	// P2 played: P1 (3.0), P4 (0.0), P3 (1.5) → Buchholz = 4.5
	t.Log("\nCalculating P2 Buchholz:")
	t.Log("  P2 opponents: P1 (3.0), P4 (0.0), P3 (1.5)")
	t.Log("  Expected Buchholz: 3.0 + 0.0 + 1.5 = 4.5")
	bh = computeBuchholz(p2, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Actual Buchholz: %.1f", bh)
	if bh != 4.5 {
		t.Errorf("P2 Buchholz: expected 4.5, got %f", bh)
	}

	// P3 played: P4 (0.0), P1 (3.0), P2 (1.5) → Buchholz = 4.5
	t.Log("\nCalculating P3 Buchholz:")
	t.Log("  P3 opponents: P4 (0.0), P1 (3.0), P2 (1.5)")
	t.Log("  Expected Buchholz: 0.0 + 3.0 + 1.5 = 4.5")
	bh = computeBuchholz(p3, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Actual Buchholz: %.1f", bh)
	if bh != 4.5 {
		t.Errorf("P3 Buchholz: expected 4.5, got %f", bh)
	}

	// P4 played: P3 (1.5), P2 (1.5), P1 (3.0) → Buchholz = 6.0
	t.Log("\nCalculating P4 Buchholz:")
	t.Log("  P4 opponents: P3 (1.5), P2 (1.5), P1 (3.0)")
	t.Log("  Expected Buchholz: 1.5 + 1.5 + 3.0 = 6.0")
	bh = computeBuchholz(p4, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Actual Buchholz: %.1f", bh)
	if bh != 6.0 {
		t.Errorf("P4 Buchholz: expected 6.0, got %f", bh)
	}

	t.Log("\n✓ All basic Buchholz calculations passed!")
}

// TestComputeBuchholz_CutLowest tests Buchholz Cut-1 (discard lowest)
func TestComputeBuchholz_CutLowest(t *testing.T) {
	t.Log("=== Testing Buchholz Cut-1 (discard lowest opponent score) ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0,
		p2: 1.5,
		p3: 1.5,
		p4: 0.0,
	}

	t.Log("Scores: P1=3.0, P2=1.5, P3=1.5, P4=0.0")

	// P1 opponents: P2 (1.5), P3 (1.5), P4 (0.0)
	// Cut lowest (0.0) → 1.5 + 1.5 = 3.0
	t.Log("\nP1 Buchholz-Cut1:")
	t.Log("  Opponents: P2 (1.5), P3 (1.5), P4 (0.0)")
	t.Log("  After cutting lowest (0.0): 1.5 + 1.5 = 3.0")
	bhCut1 := computeBuchholz(p1, allInfo, rawScores, 1, 0)
	t.Logf("  ✓ Result: %.1f", bhCut1)
	if bhCut1 != 3.0 {
		t.Errorf("P1 Buchholz-Cut1: expected 3.0, got %f", bhCut1)
	}

	// P2 opponents: P1 (3.0), P4 (0.0), P3 (1.5)
	// Cut lowest (0.0) → 3.0 + 1.5 = 4.5
	t.Log("\nP2 Buchholz-Cut1:")
	t.Log("  Opponents: P1 (3.0), P4 (0.0), P3 (1.5)")
	t.Log("  After cutting lowest (0.0): 3.0 + 1.5 = 4.5")
	bhCut1 = computeBuchholz(p2, allInfo, rawScores, 1, 0)
	t.Logf("  ✓ Result: %.1f", bhCut1)
	if bhCut1 != 4.5 {
		t.Errorf("P2 Buchholz-Cut1: expected 4.5, got %f", bhCut1)
	}

	t.Log("\n✓ Buchholz Cut-1 calculations passed!")
}

// TestComputeBuchholz_Median tests Buchholz Median (cut lowest and highest)
func TestComputeBuchholz_Median(t *testing.T) {
	t.Log("=== Testing Buchholz Median (cut lowest AND highest) ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0,
		p2: 1.5,
		p3: 1.5,
		p4: 0.0,
	}

	t.Log("Scores: P1=3.0, P2=1.5, P3=1.5, P4=0.0")

	// P2 opponents: P1 (3.0), P4 (0.0), P3 (1.5)
	// Cut lowest (0.0) and highest (3.0) → only 1.5 remains
	t.Log("\nP2 Buchholz-Median:")
	t.Log("  Opponents: P1 (3.0), P4 (0.0), P3 (1.5)")
	t.Log("  Sorted: [0.0, 1.5, 3.0]")
	t.Log("  After cutting lowest (0.0) AND highest (3.0): 1.5")
	bhMedian := computeBuchholz(p2, allInfo, rawScores, 1, 1)
	t.Logf("  ✓ Result: %.1f", bhMedian)
	if bhMedian != 1.5 {
		t.Errorf("P2 Buchholz-Median: expected 1.5, got %f", bhMedian)
	}

	t.Log("\n✓ Buchholz Median calculation passed!")
}

// TestComputeBuchholz_InsufficientRounds tests when not enough rounds for cuts
func TestComputeBuchholz_InsufficientRounds(t *testing.T) {
	t.Log("=== Testing Insufficient Rounds for Cuts ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 1.0,
		p2: 0.0,
	}

	t.Log("Scenario: Only 1 round played")
	t.Log("P1 score: 1.0, P2 score: 0.0")

	// Only 1 round, can't cut 1 lowest
	t.Log("\nAttempting Cut-1 with only 1 round:")
	t.Log("  Cannot cut 1 opponent when only 1 opponent exists")
	bhCut1 := computeBuchholz(p1, allInfo, rawScores, 1, 0)
	t.Logf("  ✓ Returns 0 as expected: %.1f", bhCut1)
	if bhCut1 != 0.0 {
		t.Errorf("Insufficient rounds should return 0, got %f", bhCut1)
	}

	// Only 1 round, can't cut 1 lowest and 1 highest
	t.Log("\nAttempting Median with only 1 round:")
	t.Log("  Cannot cut 2 opponents when only 1 opponent exists")
	bhMedian := computeBuchholz(p1, allInfo, rawScores, 1, 1)
	t.Logf("  ✓ Returns 0 as expected: %.1f", bhMedian)
	if bhMedian != 0.0 {
		t.Errorf("Insufficient rounds should return 0, got %f", bhMedian)
	}

	t.Log("\n✓ Insufficient rounds handling passed!")
}

// TestComputeBuchholz_WithBye tests virtual opponent for bye (FIDE 16.4)
func TestComputeBuchholz_WithBye(t *testing.T) {
	t.Log("=== Testing Bye Handling (FIDE 16.4 - Virtual Opponent) ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 gets bye (1 point)
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, nil, codePtr(result.WhiteWins)),
		// Round 2: P1 beats P3, P2 gets bye
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, nil, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 2.0, // 2 wins
		p2: 1.0, // 1 bye
		p3: 1.0, // 1 bye
	}

	t.Log("Scenario:")
	t.Log("  Round 1: P1 beats P2, P3 gets bye")
	t.Log("  Round 2: P1 beats P3, P2 gets bye")
	t.Log("Final scores: P1=2.0, P2=1.0, P3=1.0")

	// P1 played: P2 (1.0), P3 (1.0) → Buchholz = 2.0
	t.Log("\nP1 Buchholz (played 2 real games):")
	t.Log("  Opponents: P2 (1.0), P3 (1.0)")
	t.Log("  Buchholz: 1.0 + 1.0 = 2.0")
	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Result: %.1f", bh)
	if bh != 2.0 {
		t.Errorf("P1 Buchholz: expected 2.0, got %f", bh)
	}

	// P2 played: P1 (2.0), bye (virtual opponent = 1.0) → Buchholz = 3.0
	// Per FIDE 16.4: bye uses player's own score as virtual opponent
	t.Log("\nP2 Buchholz (1 game + 1 bye):")
	t.Log("  Real opponents: P1 (2.0)")
	t.Log("  Virtual opponent (bye): uses P2's own score = 1.0")
	t.Log("  Buchholz: 2.0 + 1.0 = 3.0")
	bh = computeBuchholz(p2, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Result: %.1f", bh)
	if bh != 3.0 {
		t.Errorf("P2 Buchholz with bye: expected 3.0, got %f", bh)
	}

	// P3 played: bye (1.0), P1 (2.0) → Buchholz = 3.0
	t.Log("\nP3 Buchholz (1 bye + 1 game):")
	t.Log("  Virtual opponent (bye): uses P3's own score = 1.0")
	t.Log("  Real opponents: P1 (2.0)")
	t.Log("  Buchholz: 1.0 + 2.0 = 3.0")
	bh = computeBuchholz(p3, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Result: %.1f", bh)
	if bh != 3.0 {
		t.Errorf("P3 Buchholz with bye: expected 3.0, got %f", bh)
	}

	t.Log("\n✓ Bye handling (virtual opponent) passed!")
}

// TestComputeBuchholz_WithWithdrawal tests FIDE 16.3.2 adjustment
func TestComputeBuchholz_WithWithdrawal(t *testing.T) {
	t.Log("=== Testing Withdrawal Adjustment (FIDE 16.3.2) ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 gets bye
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, nil, codePtr(result.WhiteWins)),
		// Round 2: P1 beats P3, P2 withdraws (gets 0-0 forfeit)
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, nil, codePtr(result.ZeroPointBye)), // withdrawal after last real game
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 2.0, // 2 wins
		p2: 0.0, // lost round 1, withdrew round 2
		p3: 1.0, // 1 bye
	}

	t.Log("Scenario:")
	t.Log("  Round 1: P1 beats P2, P3 gets bye")
	t.Log("  Round 2: P1 beats P3, P2 withdraws (0-0 forfeit)")
	t.Log("Final scores: P1=2.0, P2=0.0, P3=1.0")

	// P1 played: P2 (0.0), P3 (1.0)
	// P2's last real game was round 1, round 2 is withdrawal (0-0 forfeit)
	// Per FIDE 16.3.2: add 0.5 for each withdrawal round → P2 adjusted score = 0.0 + 0.5 = 0.5
	// P1's Buchholz = 0.5 (adjusted P2) + 1.0 (P3) = 1.5
	t.Log("\nP1's opponents:")
	t.Log("  P2 played in round 1 (lost), then withdrew in round 2")
	t.Log("  P2's raw score: 0.0")
	t.Log("  P2's adjusted score (FIDE 16.3.2): 0.0 + 0.5 (withdrawal) = 0.5")
	t.Log("  P3's score: 1.0")
	t.Log("  P1 Buchholz: 0.5 + 1.0 = 1.5")
	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Result: %.1f", bh)
	if bh != 1.5 {
		t.Errorf("P1 Buchholz with opponent withdrawal: expected 1.5, got %f", bh)
	}

	t.Log("\n✓ Withdrawal adjustment (FIDE 16.3.2) passed!")
}

// TestComputeBuchholz_NoPlayer tests when player doesn't exist
func TestComputeBuchholz_NoPlayer(t *testing.T) {
	t.Log("=== Testing Non-existent Player ===")

	p1 := newID()
	p2 := newID()
	nonExistent := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 1.0,
		p2: 0.0,
	}

	t.Log("Only P1 and P2 exist in the tournament")
	t.Logf("Querying Buchholz for non-existent player: %s", nonExistent[:8])

	bh := computeBuchholz(nonExistent, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Returns 0 as expected: %.1f", bh)
	if bh != 0.0 {
		t.Errorf("Non-existent player should return 0, got %f", bh)
	}

	t.Log("\n✓ Non-existent player handling passed!")
}

// TestComputeBuchholz_MultipleWithdrawals tests multiple withdrawal rounds
func TestComputeBuchholz_MultipleWithdrawals(t *testing.T) {
	t.Log("=== Testing Multiple Withdrawal Rounds ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		// Round 2: P2 withdraws
		makePairing(2, p2, nil, codePtr(result.ZeroPointBye)),
		// Round 3: P2 still withdrawn
		makePairing(3, p2, nil, codePtr(result.ZeroPointBye)),
		// P1 gets byes in rounds 2 and 3
		makePairing(2, p1, nil, codePtr(result.WhiteWins)),
		makePairing(3, p1, nil, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0, // 3 byes/wins
		p2: 0.0, // lost then withdrew twice
	}

	t.Log("Scenario:")
	t.Log("  Round 1: P1 beats P2")
	t.Log("  Round 2: P1 gets bye, P2 withdraws (0-0)")
	t.Log("  Round 3: P1 gets bye, P2 withdraws (0-0)")
	t.Log("Final scores: P1=3.0, P2=0.0")

	// P1 played: P2 (0.0), bye (3.0), bye (3.0)
	// P2's last real game was round 1, rounds 2 and 3 are withdrawals
	// P2 adjusted score = 0.0 + 0.5*2 = 1.0
	// P1's Buchholz = 1.0 (adjusted P2) + 3.0 (virtual) + 3.0 (virtual) = 7.0
	t.Log("\nP1's opponents:")
	t.Log("  Round 1: P2 (lost, then withdrew 2 rounds)")
	t.Log("    P2's raw score: 0.0")
	t.Log("    P2's adjusted score: 0.0 + 0.5×2 withdrawals = 1.0")
	t.Log("  Round 2: Bye (virtual opponent = 3.0)")
	t.Log("  Round 3: Bye (virtual opponent = 3.0)")
	t.Log("  P1 Buchholz: 1.0 + 3.0 + 3.0 = 7.0")
	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Result: %.1f", bh)
	if bh != 7.0 {
		t.Errorf("P1 Buchholz with multiple opponent withdrawals: expected 7.0, got %f", bh)
	}

	t.Log("\n✓ Multiple withdrawals handling passed!")
}

// TestComputeBuchholz_ComplexScenario tests a complex real-world scenario
func TestComputeBuchholz_ComplexScenario(t *testing.T) {
	t.Log("=== Testing Complex Real-World Scenario ===")

	// 5 players, 4 rounds with various results
	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()
	p5 := newID()

	pairings := []*round.Game[string]{
		// Round 1
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.Draw)),
		makePairing(1, p5, nil, codePtr(result.WhiteWins)), // bye
		// Round 2
		makePairing(2, p1, &p3, codePtr(result.Draw)),
		makePairing(2, p2, &p5, codePtr(result.BlackWins)),
		makePairing(2, p4, nil, codePtr(result.WhiteWins)), // bye
		// Round 3
		makePairing(3, p1, &p5, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p4, codePtr(result.Draw)),
		makePairing(3, p3, nil, codePtr(result.WhiteWins)), // bye
		// Round 4
		makePairing(4, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(4, p3, &p5, codePtr(result.WhiteWins)),
		makePairing(4, p2, nil, codePtr(result.ZeroPointBye)), // withdrawal
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.5, // 3 wins, 1 draw
		p2: 0.5, // 1 draw, 1 loss, 1 withdrawal
		p3: 2.5, // 1 win, 1 draw, 1 bye
		p4: 1.5, // 2 draws, 1 bye
		p5: 2.0, // 1 win, 1 bye, 1 loss
	}

	t.Log("Tournament with 5 players, 4 rounds:")
	t.Log("  Round 1: P1 beats P2, P3 draws P4, P5 bye")
	t.Log("  Round 2: P1 draws P3, P5 beats P2, P4 bye")
	t.Log("  Round 3: P1 beats P5, P2 draws P4, P3 bye")
	t.Log("  Round 4: P1 beats P4, P3 beats P5, P2 withdraws")
	t.Log("\nFinal Scores:")
	t.Logf("  P1: %.1f (3W, 1D)", rawScores[p1])
	t.Logf("  P2: %.1f (1D, withdrew)", rawScores[p2])
	t.Logf("  P3: %.1f (1W, 1D, 1 bye)", rawScores[p3])
	t.Logf("  P4: %.1f (2D, 1 bye)", rawScores[p4])
	t.Logf("  P5: %.1f (1W, 1L, 1 bye)", rawScores[p5])

	// P1 played: P2, P3, P5, P4
	// P2's last real game was round 3, round 4 is withdrawal
	// P2 adjusted = 0.5 + 0.5 = 1.0
	// P1's opponents: 1.0 (P2 adjusted), 2.5 (P3), 2.0 (P5), 1.5 (P4) = 7.0
	t.Log("\nP1 Basic Buchholz:")
	t.Log("  P1 played: P2, P3, P5, P4")
	t.Log("  P2 withdrew in round 4, adjusted score: 0.5 + 0.5 = 1.0")
	t.Log("  Opponent scores: 1.0 (P2), 2.5 (P3), 2.0 (P5), 1.5 (P4)")
	t.Log("  Buchholz: 1.0 + 2.5 + 2.0 + 1.5 = 7.0")
	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)
	t.Logf("  ✓ Result: %.1f", bh)
	if bh != 7.0 {
		t.Errorf("P1 Buchholz in complex scenario: expected 7.0, got %f", bh)
	}

	// Test with Cut-1
	// P1's opponents sorted: 1.0, 1.5, 2.0, 2.5
	// Cut lowest (1.0) → 1.5 + 2.0 + 2.5 = 6.0
	t.Log("\nP1 Buchholz-Cut1:")
	t.Log("  Opponent scores sorted: [1.0, 1.5, 2.0, 2.5]")
	t.Log("  After cutting lowest (1.0): 1.5 + 2.0 + 2.5 = 6.0")
	bhCut1 := computeBuchholz(p1, allInfo, rawScores, 1, 0)
	t.Logf("  ✓ Result: %.1f", bhCut1)
	if bhCut1 != 6.0 {
		t.Errorf("P1 Buchholz-Cut1 in complex scenario: expected 6.0, got %f", bhCut1)
	}

	// Test with Median
	// Cut lowest (1.0) and highest (2.5) → 1.5 + 2.0 = 3.5
	t.Log("\nP1 Buchholz-Median:")
	t.Log("  After cutting lowest (1.0) AND highest (2.5): 1.5 + 2.0 = 3.5")
	bhMedian := computeBuchholz(p1, allInfo, rawScores, 1, 1)
	t.Logf("  ✓ Result: %.1f", bhMedian)
	if bhMedian != 3.5 {
		t.Errorf("P1 Buchholz-Median in complex scenario: expected 3.5, got %f", bhMedian)
	}

	t.Log("\n✓ Complex scenario with all variants passed!")
}

// ============================================================================
// Sonneborn-Berger Tests
// ============================================================================

// pointsFor returns the points a player scored in a pairing
func pointsFor(playerID string, p *round.Game[string]) float64 {
	if p.Result == nil {
		return 0
	}
	isWhite := p.White == playerID
	switch *p.Result {
	case result.WhiteWins:
		if isWhite {
			return 1.0
		}
		return 0.0
	case result.BlackWins:
		if isWhite {
			return 0.0
		}
		return 1.0
	case result.Draw:
		return 0.5
	default:
		return 0
	}
}

// TestComputeSonnebornBerger_BasicCalculation tests plain SB calculation
func TestComputeSonnebornBerger_BasicCalculation(t *testing.T) {
	t.Log("=== Testing Basic Sonneborn-Berger Calculation ===")

	// Setup: 4 players, 3 rounds
	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	t.Logf("Player IDs: P1=%s, P2=%s, P3=%s, P4=%s", p1[:8], p2[:8], p3[:8], p4[:8])

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 beats P4
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // P1: 1, P2: 0
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)), // P3: 1, P4: 0
		// Round 2: P1 beats P3, P2 beats P4
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)), // P1: 2, P3: 1
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)), // P2: 1, P4: 0
		// Round 3: P1 beats P4, P2 draws P3
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)), // P1: 3, P4: 0
		makePairing(3, p2, &p3, codePtr(result.Draw)),      // P2: 1.5, P3: 1.5
	}

	t.Log("Tournament scenario:")
	t.Log("  Round 1: P1 beats P2 (1-0), P3 beats P4 (1-0)")
	t.Log("  Round 2: P1 beats P3 (1-0), P2 beats P4 (1-0)")
	t.Log("  Round 3: P1 beats P4 (1-0), P2 draws P3 (0.5-0.5)")

	// Build info and scores
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0, // 3 wins
		p2: 1.5, // 1 win, 1 draw
		p3: 1.5, // 1 win, 1 draw
		p4: 0.0, // 0 wins
	}

	t.Log("\nFinal Scores:")
	t.Logf("  P1: %.1f points (3 wins)", rawScores[p1])
	t.Logf("  P2: %.1f points (1 win, 1 draw)", rawScores[p2])
	t.Logf("  P3: %.1f points (1 win, 1 draw)", rawScores[p3])
	t.Logf("  P4: %.1f points (0 wins)", rawScores[p4])

	// P1 played: P2 (1.5)×1 + P3 (1.5)×1 + P4 (0.0)×1 = 1.5 + 1.5 + 0.0 = 3.0
	t.Log("\nCalculating P1 Sonneborn-Berger:")
	t.Log("  P1 vs P2 (1.5): scored 1.0 → 1.5 × 1.0 = 1.5")
	t.Log("  P1 vs P3 (1.5): scored 1.0 → 1.5 × 1.0 = 1.5")
	t.Log("  P1 vs P4 (0.0): scored 1.0 → 0.0 × 1.0 = 0.0")
	t.Log("  Expected SB: 1.5 + 1.5 + 0.0 = 3.0")
	sb := computeSonnebornBerger(p1, allInfo, rawScores)
	t.Logf("  ✓ Actual SB: %.1f", sb)
	if sb != 3.0 {
		t.Errorf("P1 SB: expected 3.0, got %f", sb)
	}

	// P2 played: P1 (3.0)×0 + P4 (0.0)×1 + P3 (1.5)×0.5 = 0.0 + 0.0 + 0.75 = 0.75
	t.Log("\nCalculating P2 Sonneborn-Berger:")
	t.Log("  P2 vs P1 (3.0): scored 0.0 → 3.0 × 0.0 = 0.0")
	t.Log("  P2 vs P4 (0.0): scored 1.0 → 0.0 × 1.0 = 0.0")
	t.Log("  P2 vs P3 (1.5): scored 0.5 → 1.5 × 0.5 = 0.75")
	t.Log("  Expected SB: 0.0 + 0.0 + 0.75 = 0.75")
	sb = computeSonnebornBerger(p2, allInfo, rawScores)
	t.Logf("  ✓ Actual SB: %.2f", sb)
	if sb != 0.75 {
		t.Errorf("P2 SB: expected 0.75, got %f", sb)
	}

	// P3 played: P4 (0.0)×1 + P1 (3.0)×0 + P2 (1.5)×0.5 = 0.0 + 0.0 + 0.75 = 0.75
	t.Log("\nCalculating P3 Sonneborn-Berger:")
	t.Log("  P3 vs P4 (0.0): scored 1.0 → 0.0 × 1.0 = 0.0")
	t.Log("  P3 vs P1 (3.0): scored 0.0 → 3.0 × 0.0 = 0.0")
	t.Log("  P3 vs P2 (1.5): scored 0.5 → 1.5 × 0.5 = 0.75")
	t.Log("  Expected SB: 0.0 + 0.0 + 0.75 = 0.75")
	sb = computeSonnebornBerger(p3, allInfo, rawScores)
	t.Logf("  ✓ Actual SB: %.2f", sb)
	if sb != 0.75 {
		t.Errorf("P3 SB: expected 0.75, got %f", sb)
	}

	// P4 played: P3 (1.5)×0 + P2 (1.5)×0 + P1 (3.0)×0 = 0.0 + 0.0 + 0.0 = 0.0
	t.Log("\nCalculating P4 Sonneborn-Berger:")
	t.Log("  P4 vs P3 (1.5): scored 0.0 → 1.5 × 0.0 = 0.0")
	t.Log("  P4 vs P2 (1.5): scored 0.0 → 1.5 × 0.0 = 0.0")
	t.Log("  P4 vs P1 (3.0): scored 0.0 → 3.0 × 0.0 = 0.0")
	t.Log("  Expected SB: 0.0 + 0.0 + 0.0 = 0.0")
	sb = computeSonnebornBerger(p4, allInfo, rawScores)
	t.Logf("  ✓ Actual SB: %.1f", sb)
	if sb != 0.0 {
		t.Errorf("P4 SB: expected 0.0, got %f", sb)
	}

	t.Log("\n✓ All basic Sonneborn-Berger calculations passed!")
}

// TestComputeSonnebornBerger_WithBye tests virtual opponent for bye (FIDE 16.4)
func TestComputeSonnebornBerger_WithBye(t *testing.T) {
	t.Log("=== Testing SB with Bye (FIDE 16.4 - Virtual Opponent) ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 gets bye (1 point)
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, nil, codePtr(result.WhiteWins)),
		// Round 2: P1 beats P3, P2 gets bye
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, nil, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 2.0, // 2 wins
		p2: 1.0, // 1 loss, 1 bye
		p3: 1.0, // 1 bye, 1 loss
	}

	t.Log("Scenario:")
	t.Log("  Round 1: P1 beats P2, P3 gets bye")
	t.Log("  Round 2: P1 beats P3, P2 gets bye")
	t.Log("Final scores: P1=2.0, P2=1.0, P3=1.0")

	// P1 played: P2 (1.0)×1 + P3 (1.0)×1 = 1.0 + 1.0 = 2.0
	t.Log("\nP1 Sonneborn-Berger (played 2 real games):")
	t.Log("  P1 vs P2 (1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  P1 vs P3 (1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  SB: 1.0 + 1.0 = 2.0")
	sb := computeSonnebornBerger(p1, allInfo, rawScores)
	t.Logf("  ✓ Result: %.1f", sb)
	if sb != 2.0 {
		t.Errorf("P1 SB: expected 2.0, got %f", sb)
	}

	// P2 played: P1 (2.0)×0 + bye (1.0 virtual)×1 = 0.0 + 1.0 = 1.0
	// Per FIDE 16.4: bye uses player's own score as virtual opponent
	t.Log("\nP2 Sonneborn-Berger (1 game + 1 bye):")
	t.Log("  P2 vs P1 (2.0): scored 0.0 → 2.0 × 0.0 = 0.0")
	t.Log("  P2 vs Bye (virtual 1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  SB: 0.0 + 1.0 = 1.0")
	sb = computeSonnebornBerger(p2, allInfo, rawScores)
	t.Logf("  ✓ Result: %.1f", sb)
	if sb != 1.0 {
		t.Errorf("P2 SB with bye: expected 1.0, got %f", sb)
	}

	// P3 played: bye (1.0 virtual)×1 + P1 (2.0)×0 = 1.0 + 0.0 = 1.0
	t.Log("\nP3 Sonneborn-Berger (1 bye + 1 game):")
	t.Log("  P3 vs Bye (virtual 1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  P3 vs P1 (2.0): scored 0.0 → 2.0 × 0.0 = 0.0")
	t.Log("  SB: 1.0 + 0.0 = 1.0")
	sb = computeSonnebornBerger(p3, allInfo, rawScores)
	t.Logf("  ✓ Result: %.1f", sb)
	if sb != 1.0 {
		t.Errorf("P3 SB with bye: expected 1.0, got %f", sb)
	}

	t.Log("\n✓ Bye handling (virtual opponent) passed!")
}

// TestComputeSonnebornBerger_WithWithdrawal tests FIDE 16.3.2 adjustment
func TestComputeSonnebornBerger_WithWithdrawal(t *testing.T) {
	t.Log("=== Testing SB with Withdrawal Adjustment (FIDE 16.3.2) ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 gets bye
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, nil, codePtr(result.WhiteWins)),
		// Round 2: P1 beats P3, P2 withdraws (gets 0-0 forfeit)
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, nil, codePtr(result.ZeroPointBye)), // withdrawal after last real game
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 2.0, // 2 wins
		p2: 0.0, // lost round 1, withdrew round 2
		p3: 1.0, // 1 bye, 1 loss
	}

	t.Log("Scenario:")
	t.Log("  Round 1: P1 beats P2, P3 gets bye")
	t.Log("  Round 2: P1 beats P3, P2 withdraws (0-0 forfeit)")
	t.Log("Final scores: P1=2.0, P2=0.0, P3=1.0")

	// P1 played: P2 (adjusted 0.5)×1 + P3 (1.0)×1
	// P2's last real game was round 1, round 2 is withdrawal (0-0 forfeit)
	// Per FIDE 16.3.2: add 0.5 for each withdrawal round → P2 adjusted score = 0.0 + 0.5 = 0.5
	// P1's SB = 0.5 × 1.0 + 1.0 × 1.0 = 0.5 + 1.0 = 1.5
	t.Log("\nP1's opponents:")
	t.Log("  P2 played in round 1 (lost), then withdrew in round 2")
	t.Log("  P2's raw score: 0.0")
	t.Log("  P2's adjusted score (FIDE 16.3.2): 0.0 + 0.5 (withdrawal) = 0.5")
	t.Log("  P1 vs P2 (adjusted 0.5): scored 1.0 → 0.5 × 1.0 = 0.5")
	t.Log("  P1 vs P3 (1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  P1 SB: 0.5 + 1.0 = 1.5")
	sb := computeSonnebornBerger(p1, allInfo, rawScores)
	t.Logf("  ✓ Result: %.1f", sb)
	if sb != 1.5 {
		t.Errorf("P1 SB with opponent withdrawal: expected 1.5, got %f", sb)
	}

	t.Log("\n✓ Withdrawal adjustment (FIDE 16.3.2) passed!")
}

// TestComputeSonnebornBerger_MultipleWithdrawals tests multiple withdrawal rounds
func TestComputeSonnebornBerger_MultipleWithdrawals(t *testing.T) {
	t.Log("=== Testing SB with Multiple Withdrawal Rounds ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		// Round 2: P2 withdraws
		makePairing(2, p2, nil, codePtr(result.ZeroPointBye)),
		// Round 3: P2 still withdrawn
		makePairing(3, p2, nil, codePtr(result.ZeroPointBye)),
		// P1 gets byes in rounds 2 and 3
		makePairing(2, p1, nil, codePtr(result.WhiteWins)),
		makePairing(3, p1, nil, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0, // 3 byes/wins
		p2: 0.0, // lost then withdrew twice
	}

	t.Log("Scenario:")
	t.Log("  Round 1: P1 beats P2")
	t.Log("  Round 2: P1 gets bye, P2 withdraws (0-0)")
	t.Log("  Round 3: P1 gets bye, P2 withdraws (0-0)")
	t.Log("Final scores: P1=3.0, P2=0.0")

	// P1 played: P2 (adjusted 1.0)×1 + bye (3.0)×1 + bye (3.0)×1
	// P2's last real game was round 1, rounds 2 and 3 are withdrawals
	// P2 adjusted score = 0.0 + 0.5×2 = 1.0
	// P1's SB = 1.0×1 + 3.0×1 + 3.0×1 = 1.0 + 3.0 + 3.0 = 7.0
	t.Log("\nP1's opponents:")
	t.Log("  Round 1: P2 (lost, then withdrew 2 rounds)")
	t.Log("    P2's raw score: 0.0")
	t.Log("    P2's adjusted score: 0.0 + 0.5×2 withdrawals = 1.0")
	t.Log("    P1 vs P2 (adjusted 1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  Round 2: Bye (virtual opponent = 3.0): scored 1.0 → 3.0 × 1.0 = 3.0")
	t.Log("  Round 3: Bye (virtual opponent = 3.0): scored 1.0 → 3.0 × 1.0 = 3.0")
	t.Log("  P1 SB: 1.0 + 3.0 + 3.0 = 7.0")
	sb := computeSonnebornBerger(p1, allInfo, rawScores)
	t.Logf("  ✓ Result: %.1f", sb)
	if sb != 7.0 {
		t.Errorf("P1 SB with multiple opponent withdrawals: expected 7.0, got %f", sb)
	}

	t.Log("\n✓ Multiple withdrawals handling passed!")
}

// TestComputeSonnebornBerger_WithDraws tests SB calculation with draws
func TestComputeSonnebornBerger_WithDraws(t *testing.T) {
	t.Log("=== Testing SB with Draws ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 draws P2, P3 draws P4
		makePairing(1, p1, &p2, codePtr(result.Draw)),
		makePairing(1, p3, &p4, codePtr(result.Draw)),
		// Round 2: P1 draws P3, P2 draws P4
		makePairing(2, p1, &p3, codePtr(result.Draw)),
		makePairing(2, p2, &p4, codePtr(result.Draw)),
		// Round 3: P1 draws P4, P2 draws P3
		makePairing(3, p1, &p4, codePtr(result.Draw)),
		makePairing(3, p2, &p3, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 1.5, // 3 draws
		p2: 1.5, // 3 draws
		p3: 1.5, // 3 draws
		p4: 1.5, // 3 draws
	}

	t.Log("Scenario: All games drawn, everyone scores 1.5")
	t.Log("  Round 1: P1 draws P2, P3 draws P4")
	t.Log("  Round 2: P1 draws P3, P2 draws P4")
	t.Log("  Round 3: P1 draws P4, P2 draws P3")

	// P1 played: P2 (1.5)×0.5 + P3 (1.5)×0.5 + P4 (1.5)×0.5 = 0.75 + 0.75 + 0.75 = 2.25
	t.Log("\nP1 Sonneborn-Berger:")
	t.Log("  P1 vs P2 (1.5): scored 0.5 → 1.5 × 0.5 = 0.75")
	t.Log("  P1 vs P3 (1.5): scored 0.5 → 1.5 × 0.5 = 0.75")
	t.Log("  P1 vs P4 (1.5): scored 0.5 → 1.5 × 0.5 = 0.75")
	t.Log("  SB: 0.75 + 0.75 + 0.75 = 2.25")
	sb := computeSonnebornBerger(p1, allInfo, rawScores)
	t.Logf("  ✓ Result: %.2f", sb)
	if sb != 2.25 {
		t.Errorf("P1 SB: expected 2.25, got %f", sb)
	}

	// All players should have the same SB
	t.Log("\nAll players should have identical SB (2.25):")
	for i, pid := range []string{p2, p3, p4} {
		sb = computeSonnebornBerger(pid, allInfo, rawScores)
		t.Logf("  P%d SB: %.2f", i+2, sb)
		if sb != 2.25 {
			t.Errorf("P%d SB: expected 2.25, got %f", i+2, sb)
		}
	}

	t.Log("\n✓ Draws handling passed!")
}

// TestComputeSonnebornBerger_NoPlayer tests when player doesn't exist
func TestComputeSonnebornBerger_NoPlayer(t *testing.T) {
	t.Log("=== Testing Non-existent Player ===")

	p1 := newID()
	p2 := newID()
	nonExistent := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 1.0,
		p2: 0.0,
	}

	t.Log("Only P1 and P2 exist in the tournament")
	t.Logf("Querying SB for non-existent player: %s", nonExistent[:8])

	sb := computeSonnebornBerger(nonExistent, allInfo, rawScores)
	t.Logf("  ✓ Returns 0 as expected: %.1f", sb)
	if sb != 0.0 {
		t.Errorf("Non-existent player should return 0, got %f", sb)
	}

	t.Log("\n✓ Non-existent player handling passed!")
}

// TestComputeSonnebornBerger_ComplexScenario tests a complex real-world scenario
func TestComputeSonnebornBerger_ComplexScenario(t *testing.T) {
	t.Log("=== Testing Complex Real-World Scenario for SB ===")

	// 5 players, 4 rounds with various results
	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()
	p5 := newID()

	pairings := []*round.Game[string]{
		// Round 1
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.Draw)),
		makePairing(1, p5, nil, codePtr(result.WhiteWins)), // bye
		// Round 2
		makePairing(2, p1, &p3, codePtr(result.Draw)),
		makePairing(2, p2, &p5, codePtr(result.BlackWins)),
		makePairing(2, p4, nil, codePtr(result.WhiteWins)), // bye
		// Round 3
		makePairing(3, p1, &p5, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p4, codePtr(result.Draw)),
		makePairing(3, p3, nil, codePtr(result.WhiteWins)), // bye
		// Round 4
		makePairing(4, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(4, p3, &p5, codePtr(result.WhiteWins)),
		makePairing(4, p2, nil, codePtr(result.ZeroPointBye)), // withdrawal
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.5, // 3 wins, 1 draw
		p2: 0.5, // 1 draw, 1 loss, 1 withdrawal
		p3: 2.5, // 1 win, 1 draw, 1 bye
		p4: 1.5, // 2 draws, 1 bye
		p5: 2.0, // 1 win, 1 bye, 1 loss
	}

	t.Log("Tournament with 5 players, 4 rounds:")
	t.Log("  Round 1: P1 beats P2, P3 draws P4, P5 bye")
	t.Log("  Round 2: P1 draws P3, P5 beats P2, P4 bye")
	t.Log("  Round 3: P1 beats P5, P2 draws P4, P3 bye")
	t.Log("  Round 4: P1 beats P4, P3 beats P5, P2 withdraws")
	t.Log("\nFinal Scores:")
	t.Logf("  P1: %.1f (3W, 1D)", rawScores[p1])
	t.Logf("  P2: %.1f (1D, withdrew)", rawScores[p2])
	t.Logf("  P3: %.1f (1W, 1D, 1 bye)", rawScores[p3])
	t.Logf("  P4: %.1f (2D, 1 bye)", rawScores[p4])
	t.Logf("  P5: %.1f (1W, 1L, 1 bye)", rawScores[p5])

	// P1 played: P2, P3, P5, P4
	// P2's last real game was round 3, round 4 is withdrawal
	// P2 adjusted = 0.5 + 0.5 = 1.0
	// P1's SB:
	//   vs P2 (adjusted 1.0): scored 1.0 → 1.0 × 1.0 = 1.0
	//   vs P3 (2.5): scored 0.5 → 2.5 × 0.5 = 1.25
	//   vs P5 (2.0): scored 1.0 → 2.0 × 1.0 = 2.0
	//   vs P4 (1.5): scored 1.0 → 1.5 × 1.0 = 1.5
	//   Total: 1.0 + 1.25 + 2.0 + 1.5 = 5.75
	t.Log("\nP1 Sonneborn-Berger:")
	t.Log("  P1 played: P2, P3, P5, P4")
	t.Log("  P2 withdrew in round 4, adjusted score: 0.5 + 0.5 = 1.0")
	t.Log("  P1 vs P2 (adjusted 1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  P1 vs P3 (2.5): scored 0.5 → 2.5 × 0.5 = 1.25")
	t.Log("  P1 vs P5 (2.0): scored 1.0 → 2.0 × 1.0 = 2.0")
	t.Log("  P1 vs P4 (1.5): scored 1.0 → 1.5 × 1.0 = 1.5")
	t.Log("  SB: 1.0 + 1.25 + 2.0 + 1.5 = 5.75")
	sb := computeSonnebornBerger(p1, allInfo, rawScores)
	t.Logf("  ✓ Result: %.2f", sb)
	if sb != 5.75 {
		t.Errorf("P1 SB in complex scenario: expected 5.75, got %f", sb)
	}

	// P3 played: P4 (1.5)×0.5, P1 (3.5)×0.5, bye (2.5)×1, P5 (2.0)×1
	// P3's SB = 1.5×0.5 + 3.5×0.5 + 2.5×1 + 2.0×1 = 0.75 + 1.75 + 2.5 + 2.0 = 7.0
	t.Log("\nP3 Sonneborn-Berger:")
	t.Log("  P3 vs P4 (1.5): scored 0.5 → 1.5 × 0.5 = 0.75")
	t.Log("  P3 vs P1 (3.5): scored 0.5 → 3.5 × 0.5 = 1.75")
	t.Log("  P3 vs Bye (virtual 2.5): scored 1.0 → 2.5 × 1.0 = 2.5")
	t.Log("  P3 vs P5 (2.0): scored 1.0 → 2.0 × 1.0 = 2.0")
	t.Log("  SB: 0.75 + 1.75 + 2.5 + 2.0 = 7.0")
	sb = computeSonnebornBerger(p3, allInfo, rawScores)
	t.Logf("  ✓ Result: %.2f", sb)
	if sb != 7.0 {
		t.Errorf("P3 SB in complex scenario: expected 7.0, got %f", sb)
	}

	// P5 played: bye (2.0)×1, P2 (adjusted 1.0)×1, P1 (3.5)×0, P3 (2.5)×0
	// P5's SB = 2.0×1 + 1.0×1 + 3.5×0 + 2.5×0 = 2.0 + 1.0 + 0 + 0 = 3.0
	t.Log("\nP5 Sonneborn-Berger:")
	t.Log("  P5 vs Bye (virtual 2.0): scored 1.0 → 2.0 × 1.0 = 2.0")
	t.Log("  P5 vs P2 (adjusted 1.0): scored 1.0 → 1.0 × 1.0 = 1.0")
	t.Log("  P5 vs P1 (3.5): scored 0.0 → 3.5 × 0.0 = 0.0")
	t.Log("  P5 vs P3 (2.5): scored 0.0 → 2.5 × 0.0 = 0.0")
	t.Log("  SB: 2.0 + 1.0 + 0.0 + 0.0 = 3.0")
	sb = computeSonnebornBerger(p5, allInfo, rawScores)
	t.Logf("  ✓ Result: %.2f", sb)
	if sb != 3.0 {
		t.Errorf("P5 SB in complex scenario: expected 3.0, got %f", sb)
	}

	t.Log("\n✓ Complex scenario passed!")
}

// TestComputeSonnebornBerger_HalfPointBye tests half-point bye handling
func TestComputeSonnebornBerger_HalfPointBye(t *testing.T) {
	t.Log("=== Testing SB with Half-Point Bye ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 gets half-point bye
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, nil, codePtr(result.Draw)), // half-point bye
		// Round 2: P1 beats P3
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 2.0, // 2 wins
		p2: 0.0, // 1 loss
		p3: 0.5, // 1 half-point bye, 1 loss
	}

	t.Log("Scenario:")
	t.Log("  Round 1: P1 beats P2, P3 gets half-point bye (0.5)")
	t.Log("  Round 2: P1 beats P3")
	t.Log("Final scores: P1=2.0, P2=0.0, P3=0.5")

	// P3 played: half-point bye (virtual 0.5)×0.5, P1 (2.0)×0
	// P3's SB = 0.5×0.5 + 2.0×0 = 0.25 + 0 = 0.25
	t.Log("\nP3 Sonneborn-Berger (half-point bye):")
	t.Log("  P3 vs Half-Bye (virtual 0.5): scored 0.5 → 0.5 × 0.5 = 0.25")
	t.Log("  P3 vs P1 (2.0): scored 0.0 → 2.0 × 0.0 = 0.0")
	t.Log("  SB: 0.25 + 0.0 = 0.25")
	sb := computeSonnebornBerger(p3, allInfo, rawScores)
	t.Logf("  ✓ Result: %.2f", sb)
	if sb != 0.25 {
		t.Errorf("P3 SB with half-point bye: expected 0.25, got %f", sb)
	}

	t.Log("\n✓ Half-point bye handling passed!")
}

// ============================================================================
// Koya System Tests
// ============================================================================

// TestComputeKoyaSystem_Basic tests the standard Koya System calculation.
// FIDE Art. 9.2: points scored against opponents with >= 50% of max score.
func TestComputeKoyaSystem_Basic(t *testing.T) {
	t.Log("=== Testing Basic Koya System Calculation ===")

	// 4-player Round Robin, 3 rounds.
	// Each player plays every other player once → max score = 3.
	// Default threshold = 3 * 0.5 = 1.5.
	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 beats P4
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)),
		// Round 2: P1 beats P3, P2 beats P4
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)),
		// Round 3: P1 beats P4, P2 draws P3
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0, // beats everyone
		p2: 1.5, // beats P4, draws P3, loses to P1
		p3: 1.5, // beats P4, draws P2, loses to P1
		p4: 0.0, // loses to everyone
	}

	t.Log("Scores: P1=3.0, P2=1.5, P3=1.5, P4=0.0")
	t.Log("Max rounds: 3, default threshold: 1.5")
	t.Log("Opponents with score >= 1.5: P1(3.0), P2(1.5), P3(1.5) — NOT P4(0.0)")

	// P1 played: P2(1.5✓), P3(1.5✓), P4(0.0✗)
	// Points scored: vs P2=1.0, vs P3=1.0 → KS = 2.0
	t.Log("\nP1 Koya (threshold=1.5):")
	t.Log("  vs P2 (1.5 >= 1.5): scored 1.0 → counts")
	t.Log("  vs P3 (1.5 >= 1.5): scored 1.0 → counts")
	t.Log("  vs P4 (0.0 < 1.5): scored 1.0 → excluded")
	t.Log("  KS = 1.0 + 1.0 = 2.0")
	ks := computeKoyaSystem(p1, allInfo, rawScores, 0)
	if ks != 2.0 {
		t.Errorf("P1 KS: expected 2.0, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	// P2 played: P1(3.0✓), P4(0.0✗), P3(1.5✓)
	// Points scored: vs P1=0.0, vs P3=0.5 → KS = 0.5
	t.Log("\nP2 Koya (threshold=1.5):")
	t.Log("  vs P1 (3.0 >= 1.5): scored 0.0 → counts (0)")
	t.Log("  vs P4 (0.0 < 1.5): scored 1.0 → excluded")
	t.Log("  vs P3 (1.5 >= 1.5): scored 0.5 → counts")
	t.Log("  KS = 0.0 + 0.5 = 0.5")
	ks = computeKoyaSystem(p2, allInfo, rawScores, 0)
	if ks != 0.5 {
		t.Errorf("P2 KS: expected 0.5, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	// P3 played: P4(0.0✗), P1(3.0✓), P2(1.5✓)
	// Points scored: vs P1=0.0, vs P2=0.5 → KS = 0.5
	t.Log("\nP3 Koya (threshold=1.5):")
	t.Log("  vs P4 (0.0 < 1.5): excluded")
	t.Log("  vs P1 (3.0 >= 1.5): scored 0.0 → counts (0)")
	t.Log("  vs P2 (1.5 >= 1.5): scored 0.5 → counts")
	t.Log("  KS = 0.0 + 0.5 = 0.5")
	ks = computeKoyaSystem(p3, allInfo, rawScores, 0)
	if ks != 0.5 {
		t.Errorf("P3 KS: expected 0.5, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	// P4 played: P3(1.5✓), P2(1.5✓), P1(3.0✓)
	// Points scored: vs all = 0.0 → KS = 0.0
	t.Log("\nP4 Koya (threshold=1.5):")
	t.Log("  vs P3 (1.5 >= 1.5): scored 0.0 → counts (0)")
	t.Log("  vs P2 (1.5 >= 1.5): scored 0.0 → counts (0)")
	t.Log("  vs P1 (3.0 >= 1.5): scored 0.0 → counts (0)")
	t.Log("  KS = 0.0")
	ks = computeKoyaSystem(p4, allInfo, rawScores, 0)
	if ks != 0.0 {
		t.Errorf("P4 KS: expected 0.0, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	t.Log("\n✓ Basic Koya System calculation passed!")
}

// TestComputeKoyaSystem_ExplicitThreshold tests passing an explicit score threshold.
// FIDE Art. 14.5: threshold can be raised/lowered in half-point steps.
func TestComputeKoyaSystem_ExplicitThreshold(t *testing.T) {
	t.Log("=== Testing Koya System with Explicit Threshold ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 3.0,
		p2: 1.5,
		p3: 1.5,
		p4: 0.0,
	}

	// Raise threshold to 2.0 — only P1 (3.0) qualifies.
	// P2 played: P1(3.0✓), P4(0.0✗), P3(1.5✗)
	// Points vs P1 = 0.0 → KS = 0.0
	t.Log("Threshold raised to 2.0 — only P1 (3.0) qualifies")
	t.Log("\nP2 Koya (threshold=2.0):")
	t.Log("  vs P1 (3.0 >= 2.0): scored 0.0 → counts (0)")
	t.Log("  vs P4 (0.0 < 2.0): excluded")
	t.Log("  vs P3 (1.5 < 2.0): excluded")
	t.Log("  KS = 0.0")
	ks := computeKoyaSystem(p2, allInfo, rawScores, 2.0)
	if ks != 0.0 {
		t.Errorf("P2 KS (threshold=2.0): expected 0.0, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	// Lower threshold to 1.0 — P1(3.0), P2(1.5), P3(1.5) all qualify (P4=0 still out).
	// Same result as default threshold=1.5 for this dataset.
	t.Log("\nP1 Koya (threshold=1.0 — same qualifying set as 1.5 here):")
	ks = computeKoyaSystem(p1, allInfo, rawScores, 1.0)
	if ks != 2.0 {
		t.Errorf("P1 KS (threshold=1.0): expected 2.0, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	t.Log("\n✓ Explicit threshold handling passed!")
}

// TestComputeKoyaSystem_ByesIgnored tests that byes are not counted.
func TestComputeKoyaSystem_ByesIgnored(t *testing.T) {
	t.Log("=== Testing Koya System — Byes Are Ignored ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		// Round 1: P1 beats P2, P3 gets full-point bye
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, nil, codePtr(result.WhiteWins)),
		// Round 2: P1 beats P3
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{
		p1: 2.0,
		p2: 0.0,
		p3: 1.0,
	}

	// max rounds = 2, default threshold = 1.0
	// P1 played: P2(0.0✗), P3(1.0✓)
	// Points vs P3 = 1.0 → KS = 1.0
	t.Log("Max rounds: 2, threshold: 1.0")
	t.Log("P1 Koya:")
	t.Log("  vs P2 (0.0 < 1.0): excluded")
	t.Log("  vs P3 (1.0 >= 1.0): scored 1.0 → counts")
	t.Log("  KS = 1.0")
	ks := computeKoyaSystem(p1, allInfo, rawScores, 0)
	if ks != 1.0 {
		t.Errorf("P1 KS: expected 1.0, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	// P3 had a bye in round 1 — byes are never counted in Koya.
	// P3 played: bye(✗, ignored), P1(2.0✓)
	// Points vs P1 = 0.0 → KS = 0.0
	t.Log("\nP3 Koya (has a bye in round 1):")
	t.Log("  Round 1 bye: ignored (not a real opponent)")
	t.Log("  vs P1 (2.0 >= 1.0): scored 0.0 → counts (0)")
	t.Log("  KS = 0.0")
	ks = computeKoyaSystem(p3, allInfo, rawScores, 0)
	if ks != 0.0 {
		t.Errorf("P3 KS: expected 0.0, got %f", ks)
	}
	t.Logf("  ✓ Result: %.1f", ks)

	t.Log("\n✓ Bye handling passed!")
}

// TestComputeKoyaSystem_NoPlayer tests a non-existent player returns 0.
func TestComputeKoyaSystem_NoPlayer(t *testing.T) {
	p1 := newID()
	p2 := newID()
	ghost := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1.0, p2: 0.0}

	ks := computeKoyaSystem(ghost, allInfo, rawScores, 0)
	if ks != 0.0 {
		t.Errorf("Non-existent player KS: expected 0.0, got %f", ks)
	}
	t.Log("✓ Non-existent player returns 0")
}

// ============================================================================
// Fore Buchholz (FB) Tests
// ============================================================================

// TestForeBuchholz_AllPlayed verifies that FB equals plain BH when all games
// in the final round have already been played.
func TestForeBuchholz_AllPlayed(t *testing.T) {
	t.Log("=== Testing FB: all games played → equals BH ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		// Round 1
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)),
		// Round 2 — all played
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p4, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 0.5, p3: 1, p4: 0.5}

	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)
	fb := computeForeBuchholz(p1, allInfo, rawScores, 0, 0)
	if bh != fb {
		t.Errorf("FB should equal BH when all games played: BH=%f, FB=%f", bh, fb)
	}
	t.Logf("✓ FB = BH = %f when no unplayed games in final round", fb)
}

// TestForeBuchholz_OneUnplayed verifies that an unplayed game in the final
// round is simulated as a draw, raising each player's Buchholz contribution.
func TestForeBuchholz_OneUnplayed(t *testing.T) {
	t.Log("=== Testing FB: one unplayed game in final round ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	// Round 1: P1 beats P2, P3 has no game (bye)
	// Round 2: P1 vs P3 — UNPLAYED (nil result)
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, nil), // unplayed
	}

	allInfo := buildPlayerPairingInfo(pairings)
	// Scores counting only round 1 (round 2 is unplayed → 0 pts each)
	rawScores := map[string]float64{p1: 1, p2: 0, p3: 0}

	// BH for P1: round 1 opponent P2 (score 0) + round 2 opponent P3 (score 0) = 0
	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)

	// FB simulates P1 vs P3 as a draw → simScores: P1=1.5, P3=0.5
	// FB for P1: P2's sim score (0) + P3's sim score (0.5) = 0.5
	fb := computeForeBuchholz(p1, allInfo, rawScores, 0, 0)
	if fb <= bh {
		t.Errorf("FB should exceed BH when unplayed game exists: BH=%f, FB=%f", bh, fb)
	}
	if fb != 0.5 {
		t.Errorf("FB for P1: expected 0.5, got %f", fb)
	}
	t.Logf("✓ BH=%f, FB=%f (unplayed game simulated as draw adds 0.5 to P3's contribution)", bh, fb)
}

// TestForeBuchholz_ByeNotSimulated verifies that byes in the final round are
// NOT treated as draws by FB — they follow the normal Article 16 logic.
func TestForeBuchholz_ByeNotSimulated(t *testing.T) {
	t.Log("=== Testing FB: bye in final round is NOT simulated ===")

	p1 := newID()
	p2 := newID()

	// Round 1: P1 beats P2
	// Round 2: P1 gets a bye (nil black)
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, nil, codePtr(result.WhiteWins)), // full-point bye
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 0}

	bh := computeBuchholz(p1, allInfo, rawScores, 0, 0)
	fb := computeForeBuchholz(p1, allInfo, rawScores, 0, 0)

	// Bye is not a paired game — FB must not simulate it as a draw.
	// Both BH and FB use the Art. 16.4 virtual opponent (own score) for the bye.
	if bh != fb {
		t.Errorf("FB should equal BH for bye round: BH=%f, FB=%f", bh, fb)
	}
	t.Logf("✓ FB = BH = %f (bye not treated as simulated draw)", fb)
}

// TestForeBuchholz_ResultUnplayed verifies that ResultUnplayed ("*") is
// treated the same as a nil result (game not yet played).
func TestForeBuchholz_ResultUnplayed(t *testing.T) {
	t.Log("=== Testing FB: '*' result is also simulated as draw ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.Unplayed)), // "*"
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1, p2: 0, p3: 0}

	fbNil := computeForeBuchholz(p1, allInfo, rawScores, 0, 0)

	// Replace nil with "*" for the same pairing and verify same FB value
	pairings2 := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, nil),
	}
	allInfo2 := buildPlayerPairingInfo(pairings2)
	fbStar := computeForeBuchholz(p1, allInfo2, rawScores, 0, 0)

	if fbNil != fbStar {
		t.Errorf("nil and '*' result should produce same FB: nil=%f, star=%f", fbNil, fbStar)
	}
	t.Logf("✓ nil result and '*' both simulate to draw, FB=%f", fbNil)
}

// TestForeBuchholz_CutVariants verifies that Cut-1 works the same way as in BH
// when all games in the final round are already played.
func TestForeBuchholz_CutVariants(t *testing.T) {
	t.Log("=== Testing FB: cut variants match BH when no unplayed games ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 3, p2: 0, p3: 1, p4: 2}

	bhC1 := computeBuchholz(p1, allInfo, rawScores, 1, 0)
	fbC1 := computeForeBuchholz(p1, allInfo, rawScores, 1, 0)
	if bhC1 != fbC1 {
		t.Errorf("FB-C1 should equal BH-C1 when all played: BH-C1=%f, FB-C1=%f", bhC1, fbC1)
	}
	t.Logf("✓ FB-C1 = BH-C1 = %f", fbC1)
}

// ============================================================================
// APPO (Average Perfect Tournament Performance of Opponents) Tests
// ============================================================================

// TestComputeAPPO_SingleOpponent verifies that with one opponent APPO = PTP of
// that opponent.
// P1 beats P2; PTP(P2): faced P1(2000), scored 0 → lo = 2000-800 = 1200.
func TestComputeAPPO_SingleOpponent(t *testing.T) {
	t.Log("=== Testing APPO: single opponent → APPO = PTP(opp) ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800}

	// PTP(P2): faced P1(2000), scored 0 → lower bound = 2000-800 = 1200
	appo := computeAveragePerfectPerformanceOfOpponents(p1, allInfo, ratings)
	if appo != 1200 {
		t.Errorf("APPO single opp: expected 1200, got %f", appo)
	}
	t.Log("✓ APPO = PTP(P2) = 2000 - 800 = 1200")
}

// TestComputeAPPO_RoundRobin verifies a 3-player round-robin where each
// opponent's PTP is known.
// P1 beats P2 and P3; P2 beats P3. Ratings: P1=2000, P2=1800, P3=1600.
//   - PTP(P2): faced P1(2000)+P3(1600), scored 1/2 → symmetric around 1800 → PTP=1800.
//   - PTP(P3): faced P1(2000)+P2(1800), scored 0/2 → lower bound = 1800-800 = 1000.
//   - APPO(P1) = round((1800+1000)/2) = 1400.
func TestComputeAPPO_RoundRobin(t *testing.T) {
	t.Log("=== Testing APPO: 3-player round-robin → 1400 ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1600}

	appo := computeAveragePerfectPerformanceOfOpponents(p1, allInfo, ratings)
	if appo != 1400 {
		t.Errorf("APPO round-robin: expected 1400, got %f", appo)
	}
	t.Log("✓ APPO(P1) = round((PTP(P2)=1800 + PTP(P3)=1000) / 2) = 1400")
}

// TestComputeAPPO_HigherThanAPRO verifies that for opponents with a zero score,
// PTP (lower bound) is lower than TPR (ARO-800), so APPO can differ from APRO.
// With strong opponents (P1=2000) and a zero-score P3, APPO <= APRO.
func TestComputeAPPO_CompareWithAPRO(t *testing.T) {
	t.Log("=== Testing APPO: relationship with APRO ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1600}

	apro := computeAveragePerfectPerformanceRatingOfOpponents(p1, allInfo, ratings)
	appo := computeAveragePerfectPerformanceOfOpponents(p1, allInfo, ratings)

	// APRO=1450, APPO=1400 — both measure opposition quality but via different methods
	if apro != 1450 || appo != 1400 {
		t.Errorf("APRO=%f (expected 1450), APPO=%f (expected 1400)", apro, appo)
	}
	t.Logf("✓ APRO=%f, APPO=%f (different methods, same opponents)", apro, appo)
}

// TestComputeAPPO_ByeExcluded verifies that bye rounds are not counted.
func TestComputeAPPO_ByeExcluded(t *testing.T) {
	t.Log("=== Testing APPO: bye excluded from opponent list ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // bye
		makePairing(2, p1, &p2, codePtr(result.WhiteWins)), // real game
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800}

	// Only P2 counts; PTP(P2) = 1200 (same as single-opponent test).
	appo := computeAveragePerfectPerformanceOfOpponents(p1, allInfo, ratings)
	if appo != 1200 {
		t.Errorf("APPO with bye: expected 1200 (bye excluded), got %f", appo)
	}
	t.Log("✓ Bye excluded → APPO = PTP of the single real opponent")
}

// TestComputeAPPO_NoGames returns 0 when the player has no OTB games.
func TestComputeAPPO_NoGames(t *testing.T) {
	t.Log("=== Testing APPO: no OTB games → 0 ===")

	p1 := newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000}

	appo := computeAveragePerfectPerformanceOfOpponents(p1, allInfo, ratings)
	if appo != 0 {
		t.Errorf("APPO no games: expected 0, got %f", appo)
	}
	t.Log("✓ No OTB games → APPO = 0")
}

// TestComputeAPPO_UnratedOpponent is a regression test for #379: when an
// opponent's PTP reaches the negative range (because that opponent lost to an
// unrated player), APPO must terminate and average the negative PTP rather than
// hanging.
func TestComputeAPPO_UnratedOpponent(t *testing.T) {
	p1 := "p1"
	p2 := "p2"
	unrated := "unrated"

	// P1 beats P2; P2 also loses to unrated (rating 0).
	// P2's opponents are P1 (2000) and unrated (0), with score 0/2 →
	// minOpp(P2) = 0 → PTP(P2) = 0 - 800 = -800 → APPO(P1) = -800.
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p2, &unrated, codePtr(result.BlackWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800}

	done := make(chan float64, 1)
	go func() {
		done <- computeAveragePerfectPerformanceOfOpponents(p1, allInfo, ratings)
	}()

	select {
	case got := <-done:
		if got != -800 {
			t.Errorf("computeAveragePerfectPerformanceOfOpponents() = %v, want -800", got)
		}
	case <-time.After(time.Second):
		t.Fatal("computeAveragePerfectPerformanceOfOpponents() hung (timed out after 1s)")
	}
}

// ============================================================================
// APRO (Average Tournament Performance Rating of Opponents) Tests
// ============================================================================

// TestComputeAPRO_SingleOpponent verifies that with one opponent APRO = TPR of
// that opponent.
// P1 beats P2; P2 faced only P1(2000), scored 0 → TPR(P2) = 2000-800 = 1200.
func TestComputeAPRO_SingleOpponent(t *testing.T) {
	t.Log("=== Testing APRO: single opponent → APRO = TPR(opp) ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800}

	// TPR(P2): faced P1(2000), scored 0 → ARO=2000, p=0 → RD=-800 → TPR=1200
	apro := computeAveragePerfectPerformanceRatingOfOpponents(p1, allInfo, ratings)
	if apro != 1200 {
		t.Errorf("APRO single opp: expected 1200, got %f", apro)
	}
	t.Log("✓ APRO = TPR(P2) = 2000 - 800 = 1200")
}

// TestComputeAPRO_RoundRobin verifies a 3-player round-robin where each
// opponent's TPR is known.
// P1 beats P2 and P3; P2 beats P3. Ratings: P1=2000, P2=1800, P3=1600.
//   - TPR(P2): faced P1(2000)+P3(1600), scored 1/2 → ARO=1800, p=0.5 → TPR=1800.
//   - TPR(P3): faced P1(2000)+P2(1800), scored 0/2 → ARO=1900, p=0 → TPR=1100.
//   - APRO(P1) = round((1800+1100)/2) = 1450.
func TestComputeAPRO_RoundRobin(t *testing.T) {
	t.Log("=== Testing APRO: 3-player round-robin → 1450 ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // P1 beats P2
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)), // P1 beats P3
		makePairing(3, p2, &p3, codePtr(result.WhiteWins)), // P2 beats P3
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1600}

	apro := computeAveragePerfectPerformanceRatingOfOpponents(p1, allInfo, ratings)
	if apro != 1450 {
		t.Errorf("APRO round-robin: expected 1450, got %f", apro)
	}
	t.Log("✓ APRO(P1) = round((TPR(P2)=1800 + TPR(P3)=1100) / 2) = 1450")
}

// TestComputeAPRO_ByeExcluded verifies that bye rounds are not counted as
// opponents in the APRO average.
func TestComputeAPRO_ByeExcluded(t *testing.T) {
	t.Log("=== Testing APRO: bye excluded from opponent list ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // bye
		makePairing(2, p1, &p2, codePtr(result.WhiteWins)), // real game
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800}

	// Only P2 counts; TPR(P2) = 1200 (same as single-opponent test).
	apro := computeAveragePerfectPerformanceRatingOfOpponents(p1, allInfo, ratings)
	if apro != 1200 {
		t.Errorf("APRO with bye: expected 1200 (bye excluded), got %f", apro)
	}
	t.Log("✓ Bye excluded → APRO = TPR of the single real opponent")
}

// TestComputeAPRO_Rounding verifies that an average of opponent TPRs that falls
// exactly on X.5 rounds up (FIDE Article 10.4: "0.5 rounded up").
//
// Setup: P1 beats P2 and P3. P2 also drew Pa(1600); P3 also drew Pb(1601).
//   - TPR(P2): faced P1(2000)+Pa(1600), scored 0.5/2 → p=0.25 → TPR=1609
//   - TPR(P3): faced P1(2000)+Pb(1601), scored 0.5/2 → p=0.25 → TPR=1610
//   - APRO(P1) = round((1609+1610)/2) = round(1609.5) = 1610  ← rounds up
func TestComputeAPRO_Rounding(t *testing.T) {
	t.Log("=== Testing APRO: 0.5 rounds up ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	pa := newID()
	pb := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // P1 beats P2
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)), // P1 beats P3
		makePairing(3, p2, &pa, codePtr(result.Draw)),      // P2 draws Pa
		makePairing(4, p3, &pb, codePtr(result.Draw)),      // P3 draws Pb
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1900, p3: 1900, pa: 1600, pb: 1601}

	// TPR(P2)=1609, TPR(P3)=1610 → avg=1609.5 → round up to 1610
	apro := computeAveragePerfectPerformanceRatingOfOpponents(p1, allInfo, ratings)
	if apro != 1610 {
		t.Errorf("APRO rounding: expected 1610 (1609.5 rounds up), got %f", apro)
	}
	t.Log("✓ avg(TPR(P2)=1609, TPR(P3)=1610) = 1609.5 rounds up to 1610")
}

// TestComputeAPRO_ForfeitInOpponentRecordExcluded verifies that forfeits in an
// opponent's own record are excluded when computing that opponent's TPR, which
// in turn affects the APRO value.
//
// Setup: P1 beats P2 and P3 (real games). P3 beats P2 by forfeit (+/-).
// The forfeit is excluded from P2's OTB games, so TPR(P2) is computed only
// from P2's game vs P1 — not the forfeit vs P3.
//
//   - TPR(P2) correct: faced only P1(2000), scored 0 → TPR=1200
//   - TPR(P2) wrong (if forfeit included): faced P1(2000)+P3(1600), scored 0 → TPR=1000
//   - APRO(P1) correct: round((1200+1200)/2) = 1200
func TestComputeAPRO_ForfeitInOpponentRecordExcluded(t *testing.T) {
	t.Log("=== Testing APRO: forfeit in opponent's record excluded from their TPR ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),    // P1 beats P2 (real)
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),    // P1 beats P3 (real)
		makePairing(3, p3, &p2, codePtr(result.WhiteForfeit)), // P3 beats P2 by forfeit
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1600}

	// Correct: forfeit excluded from P2 and P3's OTB records.
	// TPR(P2) = TPR from {vs P1(2000), score=0} = 2000-800 = 1200
	// TPR(P3) = TPR from {vs P1(2000), score=0} = 2000-800 = 1200
	// APRO(P1) = round((1200+1200)/2) = 1200
	apro := computeAveragePerfectPerformanceRatingOfOpponents(p1, allInfo, ratings)
	if apro != 1200 {
		// If forfeit was wrongly included, TPR(P2) would be 1000 and APRO would be 1100.
		t.Errorf("APRO forfeit exclusion: expected 1200, got %f (1100 would mean forfeit was included)", apro)
	}
	t.Log("✓ Forfeit excluded from opponent's OTB record → APRO = 1200 (not 1100)")
}

// TestComputeAPRO_NoGames returns 0 when the player has no OTB games.
func TestComputeAPRO_NoGames(t *testing.T) {
	t.Log("=== Testing APRO: no OTB games → 0 ===")

	p1 := newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000}

	apro := computeAveragePerfectPerformanceRatingOfOpponents(p1, allInfo, ratings)
	if apro != 0 {
		t.Errorf("APRO no games: expected 0, got %f", apro)
	}
	t.Log("✓ No OTB games → APRO = 0")
}

// ============================================================================
// PTP (Perfect Tournament Performance) Tests
// ============================================================================

// TestComputePTP_Symmetric verifies the analytically known result: 1W1L against
// two opponents equidistant from a midpoint gives PTP = that midpoint.
// At R=1700: E(1700,1800) + E(1700,1600) = 0.360 + 0.640 = 1.0 = actual.
func TestComputePTP_Symmetric(t *testing.T) {
	t.Log("=== Testing PTP: 1W1L against 1800 and 1600 → PTP = 1700 ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p3, &p1, codePtr(result.WhiteWins)), // P1 loses
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1600}

	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)
	if ptp != 1700 {
		t.Errorf("PTP symmetric: expected 1700, got %f", ptp)
	}
	t.Log("✓ 1W1L vs 1800/1600 → PTP = 1700")
}

// TestComputePTP_DrawEqualsOpponent verifies that drawing against one opponent
// gives PTP equal to that opponent's rating (E(R,opp)=0.5 ↔ R=opp).
func TestComputePTP_DrawEqualsOpponent(t *testing.T) {
	t.Log("=== Testing PTP: 1 draw vs 1800 → PTP = 1800 ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800}

	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)
	if ptp != 1800 {
		t.Errorf("PTP 1 draw vs 1800: expected 1800, got %f", ptp)
	}
	t.Log("✓ 1 draw vs 1800 → PTP = 1800")
}

// TestComputePTP_KnownValue verifies PTP against a precomputed exact value
// for an asymmetric case: 2W1L vs opponents 1800/1700/1600.
func TestComputePTP_KnownValue(t *testing.T) {
	t.Log("=== Testing PTP: 2W1L vs 1800/1700/1600 → PTP = 1827 ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p4, &p1, codePtr(result.WhiteWins)), // P1 loses
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1700, p4: 1600}

	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)
	if ptp != 1827 {
		t.Errorf("PTP known value: expected 1827, got %f", ptp)
	}
	t.Log("✓ 2W1L vs 1800/1700/1600 → PTP = 1827")
}

// TestComputePTP_GreaterThanTPRAbove50Pct verifies that when a player scores
// above 50%, PTP ≥ TPR. This holds because the Elo expected-score function is
// nonlinear: achieving a high score against varied opponents requires a higher
// hypothetical rating than the linear TPR estimate implies.
//
// Uses the same 2W1L scenario as TestComputePTP_KnownValue: TPR=1820, PTP=1827.
func TestComputePTP_GreaterThanTPRAbove50Pct(t *testing.T) {
	t.Log("=== Testing PTP: PTP >= TPR when score > 50% ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p4, &p1, codePtr(result.WhiteWins)), // P1 loses
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1700, p4: 1600}

	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)

	if ptp < tpr {
		t.Errorf("PTP should be >= TPR when score > 50%%: PTP=%f, TPR=%f", ptp, tpr)
	}
	if tpr != 1820 || ptp != 1827 {
		t.Errorf("Expected TPR=1820, PTP=1827; got TPR=%f, PTP=%f", tpr, ptp)
	}
	t.Logf("✓ PTP=%f >= TPR=%f when p=2/3 > 0.5", ptp, tpr)
}

// TestComputePTP_PerfectScore verifies that a perfect score (all wins) returns
// the search upper bound (maxOppRating + 3000), since E < 1 per game always.
func TestComputePTP_PerfectScore(t *testing.T) {
	t.Log("=== Testing PTP: all wins → returns upper bound maxOpp+3000 ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	// max opp rating is 1800; upper bound = 1800 + 3000 = 4800
	ratings := map[string]int{p1: 2000, p2: 1600, p3: 1800}

	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)
	if ptp != 4800 {
		t.Errorf("PTP perfect score: expected 4800 (upper bound), got %f", ptp)
	}
	t.Log("✓ Perfect score → PTP = maxOpp + 3000 = 4800")
}

// TestComputePTP_ZeroScore verifies that a zero score (all losses) returns
// the search lower bound (minOppRating - 800).
func TestComputePTP_ZeroScore(t *testing.T) {
	t.Log("=== Testing PTP: all losses → returns lower bound minOpp-800 ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.BlackWins)), // P1 loses
		makePairing(2, p1, &p3, codePtr(result.BlackWins)), // P1 loses
	}

	allInfo := buildPlayerPairingInfo(pairings)
	// min opp rating is 1600; lower bound = 1600 - 800 = 800
	ratings := map[string]int{p1: 2000, p2: 1600, p3: 1800}

	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)
	if ptp != 800 {
		t.Errorf("PTP zero score: expected 800 (lower bound), got %f", ptp)
	}
	t.Log("✓ Zero score → PTP = minOpp - 800 = 800")
}

// TestComputePTP_ByeExcluded verifies that bye rounds don't affect PTP.
func TestComputePTP_ByeExcluded(t *testing.T) {
	t.Log("=== Testing PTP: bye excluded ===")

	p1 := newID()
	p2 := newID()

	// Round 1: bye; Round 2: draw vs 1800
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // bye
		makePairing(2, p1, &p2, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800}

	// Same as TestComputePTP_DrawEqualsOpponent: only the real game counts
	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)
	if ptp != 1800 {
		t.Errorf("PTP with bye: expected 1800 (bye excluded), got %f", ptp)
	}
	t.Log("✓ Bye excluded → PTP based on real game only")
}

// TestComputePTP_NoGames returns 0 when the player has no OTB games.
func TestComputePTP_NoGames(t *testing.T) {
	t.Log("=== Testing PTP: no OTB games → 0 ===")

	p1 := newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000}

	ptp := computePerfectTournamentPerformance(p1, allInfo, ratings)
	if ptp != 0 {
		t.Errorf("PTP no games: expected 0, got %f", ptp)
	}
	t.Log("✓ No OTB games → PTP = 0")
}

// TestComputePTP_UnratedOpponent is a regression test for #379: when a player
// loses to an unrated opponent (rating 0), the binary search enters the
// negative range [−800, 3000] and must terminate at −800 instead of hanging.
func TestComputePTP_UnratedOpponent(t *testing.T) {
	cases := []struct {
		name      string
		oppRating int
		res       result.Code
		want      float64
	}{
		{"loss vs unrated (0)", 0, result.BlackWins, -800},
		{"draw vs unrated (0)", 0, result.Draw, 0},
		{"win vs unrated (0)", 0, result.WhiteWins, 3000},
		{"loss vs 1500", 1500, result.BlackWins, 700},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := "opp"
			res := tc.res
			games := []*round.Game[string]{{Round: 1, White: "player", Black: &b, Result: &res}}
			allInfo := buildPlayerPairingInfo(games)
			ratings := map[string]int{"player": 1500}
			if tc.oppRating > 0 {
				ratings["opp"] = tc.oppRating
			}

			done := make(chan float64, 1)
			go func() {
				done <- computePerfectTournamentPerformance("player", allInfo, ratings)
			}()

			select {
			case got := <-done:
				if got != tc.want {
					t.Errorf("computePerfectTournamentPerformance() = %v, want %v", got, tc.want)
				}
			case <-time.After(time.Second):
				t.Fatal("computePerfectTournamentPerformance() hung (timed out after 1s)")
			}
		})
	}
}

// ============================================================================
// TPR (Tournament Performance Rating) Tests
// ============================================================================

// TestComputeTPR_FiftyPercent verifies that a 50% score produces TPR = ARO
// (RD = 0 at p = 0.5).
func TestComputeTPR_FiftyPercent(t *testing.T) {
	t.Log("=== Testing TPR: 50% score → TPR = ARO ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	// P1: wins vs P2, loses to P3 → 1 point from 2 games = 50%
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p3, &p1, codePtr(result.WhiteWins)), // P3 wins as white
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1600, p3: 1800}

	// ARO for P1: (1600 + 1800) / 2 = 1700; p = 0.5 → RD = 0 → TPR = 1700
	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	if tpr != 1700 {
		t.Errorf("TPR at 50%%: expected 1700, got %f", tpr)
	}
	t.Log("✓ 50% score → TPR = ARO = 1700")
}

// TestComputeTPR_PerfectScore verifies that a 100% score produces TPR = ARO + 800.
func TestComputeTPR_PerfectScore(t *testing.T) {
	t.Log("=== Testing TPR: 100% score → TPR = ARO + 800 ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1600}

	// ARO = 1600, p = 1.0 → RD = +800 (capped) → TPR = 1600 + 800 = 2400
	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	if tpr != 2400 {
		t.Errorf("TPR at 100%%: expected 2400, got %f", tpr)
	}
	t.Log("✓ 100% score → TPR = ARO + 800 = 2400")
}

// TestComputeTPR_ZeroScore verifies that a 0% score produces TPR = ARO - 800.
func TestComputeTPR_ZeroScore(t *testing.T) {
	t.Log("=== Testing TPR: 0% score → TPR = ARO - 800 ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.BlackWins)), // P1 loses
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1600}

	// ARO = 1600, p = 0.0 → RD = -800 (capped) → TPR = 1600 - 800 = 800
	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	if tpr != 800 {
		t.Errorf("TPR at 0%%: expected 800, got %f", tpr)
	}
	t.Log("✓ 0% score → TPR = ARO - 800 = 800")
}

// TestComputeTPR_KnownValue verifies a concrete calculation: 2 wins from 3 games
// against opponents rated 1800/1700/1600 (ARO=1700, p=2/3 → RD≈120 → TPR=1820).
func TestComputeTPR_KnownValue(t *testing.T) {
	t.Log("=== Testing TPR: 2/3 score, ARO 1700 → TPR 1820 ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()
	p4 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // +1
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)), // +1
		makePairing(3, p4, &p1, codePtr(result.WhiteWins)), // 0 (P1 loses)
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1800, p3: 1700, p4: 1600}

	// ARO = (1800+1700+1600)/3 = 1700.0, p = 2/3
	// RD = 400 * log10((2/3)/(1/3)) = 400 * log10(2) ≈ 120.41
	// TPR = round(1700 + 120.41) = 1820
	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	if tpr != 1820 {
		t.Errorf("TPR known value: expected 1820, got %f", tpr)
	}
	t.Log("✓ 2/3 score, ARO=1700 → TPR=1820")
}

// TestComputeTPR_ByeExcluded verifies that bye rounds are excluded from both
// the ARO and the score percentage.
func TestComputeTPR_ByeExcluded(t *testing.T) {
	t.Log("=== Testing TPR: bye excluded from ARO and score% ===")

	p1 := newID()
	p2 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // full-point bye
		makePairing(2, p1, &p2, codePtr(result.WhiteWins)), // real game
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000, p2: 1600}

	// Only round 2 counts: ARO=1600, p=1.0 → TPR=1600+800=2400
	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	if tpr != 2400 {
		t.Errorf("TPR with bye: expected 2400 (bye excluded), got %f", tpr)
	}
	t.Log("✓ Bye excluded → TPR based on real game only")
}

// TestComputeTPR_NoGames returns 0 when the player has no OTB games.
func TestComputeTPR_NoGames(t *testing.T) {
	t.Log("=== Testing TPR: no OTB games → 0 ===")

	p1 := newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // bye only
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000}

	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	if tpr != 0 {
		t.Errorf("TPR no games: expected 0, got %f", tpr)
	}
	t.Log("✓ No OTB games → TPR = 0")
}

// TestComputeTPR_UnratedOpponentExcluded verifies that games against unrated opponents
// are excluded from both the ARO component and the score percentage.
func TestComputeTPR_UnratedOpponentExcluded(t *testing.T) {
	p1 := newID()
	p2 := newID() // rated 1500
	p3 := newID() // unrated

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // win vs rated
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)), // win vs unrated
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{
		p1: 1600,
		p2: 1500,
		// p3 absent = unrated
	}

	// Only the rated game counts: ARO=1500, score%=100% → TPR = 1500+800 = 2300
	tpr := computeTournamentPerformanceRating(p1, allInfo, ratings)
	if tpr != 2300 {
		t.Errorf("TPR with one unrated opponent: expected 2300 (unrated game excluded), got %f", tpr)
	}
}

// ============================================================================
// ARO (Average Rating of Opponents) Tests
// ============================================================================

// TestComputeARO_Basic tests a simple 3-player round-robin with known ratings.
func TestComputeARO_Basic(t *testing.T) {
	t.Log("=== Testing ARO: basic 3-player round-robin ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	// Round 1: P1 beats P2
	// Round 2: P1 beats P3
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p3, codePtr(result.BlackWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{
		p1: 2000,
		p2: 1800,
		p3: 1600,
	}

	// P1 faced P2 (1800) and P3 (1600) → ARO = (1800+1600)/2 = 1700
	aro := computeAverageRatingOfOpponents(p1, allInfo, ratings)
	if aro != 1700 {
		t.Errorf("P1 ARO: expected 1700, got %f", aro)
	}
	t.Log("✓ P1 ARO = 1700")

	// P2 faced P1 (2000) and P3 (1600) → ARO = (2000+1600)/2 = 1800
	aro = computeAverageRatingOfOpponents(p2, allInfo, ratings)
	if aro != 1800 {
		t.Errorf("P2 ARO: expected 1800, got %f", aro)
	}
	t.Log("✓ P2 ARO = 1800")

	// P3 faced P1 (2000) and P2 (1800) → ARO = (2000+1800)/2 = 1900
	aro = computeAverageRatingOfOpponents(p3, allInfo, ratings)
	if aro != 1900 {
		t.Errorf("P3 ARO: expected 1900, got %f", aro)
	}
	t.Log("✓ P3 ARO = 1900")
}

// TestComputeARO_Rounding verifies that 0.5 rounds up (standard FIDE rounding).
func TestComputeARO_Rounding(t *testing.T) {
	t.Log("=== Testing ARO: 0.5 rounds up ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	// Ratings 1800 and 1801 → average = 1800.5 → should round to 1801
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{
		p1: 2000,
		p2: 1800,
		p3: 1801,
	}

	aro := computeAverageRatingOfOpponents(p1, allInfo, ratings)
	if aro != 1801 {
		t.Errorf("ARO rounding: expected 1801 (0.5 rounds up), got %f", aro)
	}
	t.Log("✓ Average 1800.5 rounds up to 1801")
}

// TestComputeARO_ByeExcluded verifies that bye rounds are not counted in ARO.
func TestComputeARO_ByeExcluded(t *testing.T) {
	t.Log("=== Testing ARO: bye rounds excluded ===")

	p1 := newID()
	p2 := newID()

	// Round 1: P1 gets a full-point bye (nil black)
	// Round 2: P1 beats P2
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // bye
		makePairing(2, p1, &p2, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{
		p1: 2000,
		p2: 1800,
	}

	// ARO should only count P2 (the bye has no opponent)
	aro := computeAverageRatingOfOpponents(p1, allInfo, ratings)
	if aro != 1800 {
		t.Errorf("ARO with bye: expected 1800 (bye excluded), got %f", aro)
	}
	t.Log("✓ Bye round excluded from ARO")
}

// TestComputeARO_ForfeitExcluded verifies that forfeited games are not counted.
func TestComputeARO_ForfeitExcluded(t *testing.T) {
	t.Log("=== Testing ARO: forfeit games excluded ===")

	p1 := newID()
	p2 := newID()
	p3 := newID()

	// Round 1: P1 wins by forfeit over P2 (P2 didn't show up)
	// Round 2: P1 beats P3 over the board
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteForfeit)), // forfeit win
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),    // real game
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{
		p1: 2000,
		p2: 1000, // if included, would skew ARO heavily
		p3: 1800,
	}

	// ARO should only count P3 (the forfeit vs P2 was not played over the board)
	aro := computeAverageRatingOfOpponents(p1, allInfo, ratings)
	if aro != 1800 {
		t.Errorf("ARO with forfeit: expected 1800 (forfeit excluded), got %f", aro)
	}
	t.Log("✓ Forfeit game excluded from ARO")
}

// TestComputeARO_AllByes returns 0 when a player has no real games.
func TestComputeARO_AllByes(t *testing.T) {
	t.Log("=== Testing ARO: player with only byes returns 0 ===")

	p1 := newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // full-point bye
		makePairing(2, p1, nil, codePtr(result.WhiteHalf)), // half-point bye
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{p1: 2000}

	aro := computeAverageRatingOfOpponents(p1, allInfo, ratings)
	if aro != 0 {
		t.Errorf("ARO all byes: expected 0, got %f", aro)
	}
	t.Log("✓ Player with only byes returns ARO = 0")
}

// TestComputeARO_UnknownPlayer returns 0 for a player not in allInfo.
func TestComputeARO_UnknownPlayer(t *testing.T) {
	t.Log("=== Testing ARO: unknown player returns 0 ===")

	ghost := newID()
	allInfo := buildPlayerPairingInfo([]*round.Game[string]{})
	ratings := map[string]int{}

	aro := computeAverageRatingOfOpponents(ghost, allInfo, ratings)
	if aro != 0 {
		t.Errorf("ARO unknown player: expected 0, got %f", aro)
	}
	t.Log("✓ Unknown player returns ARO = 0")
}

// TestComputeARO_UnratedOpponentExcluded verifies that opponents with a zero/absent
// rating are excluded from the ARO sum and denominator.
func TestComputeARO_UnratedOpponentExcluded(t *testing.T) {
	p1 := newID()
	p2 := newID() // rated
	p3 := newID() // unrated

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{
		p1: 1600,
		p2: 1500,
		// p3 is absent (unrated) — zero-value map lookup
	}

	aro := computeAverageRatingOfOpponents(p1, allInfo, ratings)
	if aro != 1500 {
		t.Errorf("ARO with one unrated opponent: expected 1500 (unrated excluded), got %f", aro)
	}
}

// ============================================================================
// Type B Tiebreaks
// ============================================================================

// TestComputeNumberOfWins_AllWins — player wins every round over the board.
func TestComputeNumberOfWins_AllWins(t *testing.T) {
	p1, p2, p3, p4 := newID(), newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 3, p2: 0, p3: 0, p4: 0}

	got := computeNumberOfWins(p1, allInfo, rawScores)
	if got != 3 {
		t.Errorf("AllWins: expected 3, got %f", got)
	}
	t.Log("✓ All OTB wins counted")
}

// TestComputeNumberOfWins_Mixed — wins, draw, loss, full-point bye, half-point bye.
func TestComputeNumberOfWins_Mixed(t *testing.T) {
	p1, p2, p3 := newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // win → counts
		makePairing(2, p1, &p3, codePtr(result.Draw)),      // draw → no
		makePairing(3, p1, &p2, codePtr(result.BlackWins)), // loss → no
		makePairing(4, p1, nil, codePtr(result.WhiteWins)), // full-point bye → counts
		makePairing(5, p1, nil, codePtr(result.WhiteHalf)), // half-point bye → no
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2.5, p2: 1.5, p3: 0.5}

	got := computeNumberOfWins(p1, allInfo, rawScores)
	if got != 2 {
		t.Errorf("Mixed: expected 2, got %f", got)
	}
	t.Log("✓ Only full-point rounds counted")
}

// TestComputeNumberOfWins_ForfeitWin — forfeit win counts (player is awarded 1.0).
func TestComputeNumberOfWins_ForfeitWin(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		// p2 forfeited; white (p1) wins by forfeit
		makePairing(1, p1, &p2, codePtr(result.WhiteForfeit)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1, p2: 0}

	got := computeNumberOfWins(p1, allInfo, rawScores)
	if got != 1 {
		t.Errorf("ForfeitWin: expected 1, got %f", got)
	}
	t.Log("✓ Forfeit win counts as WIN")
}

// TestComputeNumberOfWins_ZeroPointBye — zero-point bye gives 0.0, not counted.
func TestComputeNumberOfWins_ZeroPointBye(t *testing.T) {
	p1 := newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.ZeroPointBye)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 0}

	got := computeNumberOfWins(p1, allInfo, rawScores)
	if got != 0 {
		t.Errorf("ZeroPointBye: expected 0, got %f", got)
	}
	t.Log("✓ Zero-point bye not counted")
}

// TestComputeNumberOfWins_NonExistentPlayer — returns 0 for unknown player.
func TestComputeNumberOfWins_NonExistentPlayer(t *testing.T) {
	allInfo := buildPlayerPairingInfo[string](nil)
	rawScores := map[string]float64{}
	ghost := newID()

	got := computeNumberOfWins(ghost, allInfo, rawScores)
	if got != 0 {
		t.Errorf("NonExistent: expected 0, got %f", got)
	}
	t.Log("✓ Non-existent player returns 0")
}

// TestComputeNumberOfGamesWon_OTBOnly — only real board wins count.
func TestComputeNumberOfGamesWon_OTBOnly(t *testing.T) {
	p1, p2, p3, p4 := newID(), newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)), // OTB win as white → counts
		makePairing(2, p3, &p1, codePtr(result.BlackWins)), // OTB win as black → counts
		makePairing(3, p1, &p4, codePtr(result.Draw)),      // draw → no
		makePairing(4, p1, &p2, codePtr(result.BlackWins)), // loss as white → no
		makePairing(5, p1, nil, codePtr(result.WhiteWins)), // full-point bye → no
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2.5, p2: 1, p3: 0, p4: 0.5}

	got := computeNumberOfGamesWon(p1, allInfo, rawScores)
	if got != 2 {
		t.Errorf("OTBOnly: expected 2, got %f", got)
	}
	t.Log("✓ Only over-the-board wins counted")
}

// TestComputeNumberOfGamesWon_ForfeitWinExcluded — forfeit wins do not count.
func TestComputeNumberOfGamesWon_ForfeitWinExcluded(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		// p2 didn't show; p1 wins by forfeit
		makePairing(1, p1, &p2, codePtr(result.WhiteForfeit)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1, p2: 0}

	got := computeNumberOfGamesWon(p1, allInfo, rawScores)
	if got != 0 {
		t.Errorf("ForfeitWin: expected 0, got %f", got)
	}
	t.Log("✓ Forfeit win excluded from WON")
}

// TestComputeNumberOfGamesWon_NonExistentPlayer — returns 0 for unknown player.
func TestComputeNumberOfGamesWon_NonExistentPlayer(t *testing.T) {
	allInfo := buildPlayerPairingInfo[string](nil)
	rawScores := map[string]float64{}

	got := computeNumberOfGamesWon(newID(), allInfo, rawScores)
	if got != 0 {
		t.Errorf("NonExistent: expected 0, got %f", got)
	}
	t.Log("✓ Non-existent player returns 0")
}

// TestComputeNumberOfPlayedGamesBlack_CountsBlackGames — counts only games as Black.
func TestComputeNumberOfPlayedGamesBlack_CountsBlackGames(t *testing.T) {
	p1, p2, p3 := newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p2, &p1, codePtr(result.BlackWins)), // p1 is Black, win → counts
		makePairing(2, p3, &p1, codePtr(result.WhiteWins)), // p1 is Black, loss → counts
		makePairing(3, p3, &p1, codePtr(result.Draw)),      // p1 is Black, draw → counts
		makePairing(4, p1, &p2, codePtr(result.WhiteWins)), // p1 is White → no
		makePairing(5, p1, nil, codePtr(result.WhiteWins)), // bye → no
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 1.5, p3: 1.5}

	got := computeNumberOfPlayedGamesBlack(p1, allInfo, rawScores)
	if got != 3 {
		t.Errorf("CountsBlackGames: expected 3, got %f", got)
	}
	t.Log("✓ All three Black games counted regardless of result")
}

// TestComputeNumberOfPlayedGamesBlack_ForfeitCounts — forfeit games still count (player was assigned Black).
func TestComputeNumberOfPlayedGamesBlack_ForfeitCounts(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		// p1 is Black, p2 (White) forfeited → result is ResultBlackForfeit
		makePairing(1, p2, &p1, codePtr(result.BlackForfeit)),
		// p1 is Black, p1 forfeited → result is ResultWhiteForfeit
		makePairing(2, p2, &p1, codePtr(result.WhiteForfeit)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1, p2: 1}

	got := computeNumberOfPlayedGamesBlack(p1, allInfo, rawScores)
	if got != 2 {
		t.Errorf("ForfeitCounts: expected 2, got %f", got)
	}
	t.Log("✓ Forfeit games count for BPG (player was assigned the Black seat)")
}

// TestComputeNumberOfPlayedGamesBlack_UnplayedExcluded — nil result and void excluded.
func TestComputeNumberOfPlayedGamesBlack_UnplayedExcluded(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p2, &p1, nil),                      // nil result → no
		makePairing(2, p2, &p1, codePtr(result.Unplayed)), // "*" → no
		makePairing(3, p2, &p1, codePtr(result.Void)),     // "void" → no
		makePairing(4, p2, &p1, codePtr(result.Draw)),     // played → counts
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 0.5, p2: 0.5}

	got := computeNumberOfPlayedGamesBlack(p1, allInfo, rawScores)
	if got != 1 {
		t.Errorf("UnplayedExcluded: expected 1, got %f", got)
	}
	t.Log("✓ Unplayed and void results excluded")
}

// TestComputeNumberOfPlayedGamesBlack_NonExistentPlayer — returns 0 for unknown player.
func TestComputeNumberOfPlayedGamesBlack_NonExistentPlayer(t *testing.T) {
	allInfo := buildPlayerPairingInfo[string](nil)
	rawScores := map[string]float64{}

	got := computeNumberOfPlayedGamesBlack(newID(), allInfo, rawScores)
	if got != 0 {
		t.Errorf("NonExistent: expected 0, got %f", got)
	}
	t.Log("✓ Non-existent player returns 0")
}

// TestComputeNumberOfGamesWonBlack_OTBOnly — only ResultBlackWins counts.
func TestComputeNumberOfGamesWonBlack_OTBOnly(t *testing.T) {
	p1, p2, p3 := newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p2, &p1, codePtr(result.BlackWins)), // OTB win as Black → counts
		makePairing(2, p3, &p1, codePtr(result.Draw)),      // draw as Black → no
		makePairing(3, p2, &p1, codePtr(result.WhiteWins)), // loss as Black → no
		makePairing(4, p1, &p2, codePtr(result.WhiteWins)), // win as White → no
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1.5, p2: 1.5, p3: 0.5}

	got := computeNumberOfGamesWonBlack(p1, allInfo, rawScores)
	if got != 1 {
		t.Errorf("OTBOnly: expected 1, got %f", got)
	}
	t.Log("✓ Only OTB wins as Black counted")
}

// TestComputeNumberOfGamesWonBlack_ForfeitWinExcluded — forfeit win as Black does not count.
func TestComputeNumberOfGamesWonBlack_ForfeitWinExcluded(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		// p2 (White) forfeited; p1 (Black) wins by forfeit → ResultBlackForfeit
		makePairing(1, p2, &p1, codePtr(result.BlackForfeit)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1, p2: 0}

	got := computeNumberOfGamesWonBlack(p1, allInfo, rawScores)
	if got != 0 {
		t.Errorf("ForfeitWin: expected 0, got %f", got)
	}
	t.Log("✓ Forfeit win as Black excluded from BWG")
}

// TestComputeNumberOfGamesWonBlack_WhiteWinIgnored — wins as White are not counted.
func TestComputeNumberOfGamesWonBlack_WhiteWinIgnored(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p2, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 0}

	got := computeNumberOfGamesWonBlack(p1, allInfo, rawScores)
	if got != 0 {
		t.Errorf("WhiteWinIgnored: expected 0, got %f", got)
	}
	t.Log("✓ Wins as White not counted for BWG")
}

// TestComputeNumberOfGamesWonBlack_NonExistentPlayer — returns 0 for unknown player.
func TestComputeNumberOfGamesWonBlack_NonExistentPlayer(t *testing.T) {
	allInfo := buildPlayerPairingInfo[string](nil)
	rawScores := map[string]float64{}

	got := computeNumberOfGamesWonBlack(newID(), allInfo, rawScores)
	if got != 0 {
		t.Errorf("NonExistent: expected 0, got %f", got)
	}
	t.Log("✓ Non-existent player returns 0")
}

// TestComputeSumOfProgressiveScores_EarlyWinsScoreHigher — same total, different timing.
func TestComputeSumOfProgressiveScores_EarlyWinsScoreHigher(t *testing.T) {
	// 4 players, 3 rounds. p1 and p2 both finish with 1 point.
	// p1 scores in round 1 → running: 1,1,1 → PS = 3
	// p2 scores in round 2 → running: 0,1,1 → PS = 2
	p1, p2, p3, p4 := newID(), newID(), newID(), newID()
	pairings := []*round.Game[string]{
		// Round 1: p1 beats p2; p3 beats p4
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p3, &p4, codePtr(result.WhiteWins)),
		// Round 2: p3 beats p1; p2 beats p4
		makePairing(2, p3, &p1, codePtr(result.WhiteWins)),
		makePairing(2, p2, &p4, codePtr(result.WhiteWins)),
		// Round 3: p4 beats p1; p3 beats p2
		makePairing(3, p4, &p1, codePtr(result.WhiteWins)),
		makePairing(3, p3, &p2, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1, p2: 1, p3: 3, p4: 1}

	ps1 := computeSumOfProgressiveScores(p1, allInfo, rawScores)
	ps2 := computeSumOfProgressiveScores(p2, allInfo, rawScores)

	// p1: rounds 1(+1), 2(0), 3(0) → running 1,1,1 → PS = 3
	// p2: rounds 1(0), 2(+1), 3(0) → running 0,1,1 → PS = 2
	if ps1 != 3 {
		t.Errorf("p1 PS: expected 3, got %f", ps1)
	}
	if ps2 != 2 {
		t.Errorf("p2 PS: expected 2, got %f", ps2)
	}
	if ps1 <= ps2 {
		t.Errorf("Early scorer should have higher PS: p1=%f p2=%f", ps1, ps2)
	}
	t.Log("✓ Early wins produce higher PS than same total scored late")
}

// TestComputeSumOfProgressiveScores_WithDraw — fractional scores accumulate correctly.
func TestComputeSumOfProgressiveScores_WithDraw(t *testing.T) {
	// Scores per round: 1, 0.5, 0  → running: 1, 1.5, 1.5 → PS = 4.0
	p1, p2, p3 := newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.Draw)),
		makePairing(3, p1, &p2, codePtr(result.BlackWins)), // loss
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1.5, p2: 1, p3: 0.5}

	got := computeSumOfProgressiveScores(p1, allInfo, rawScores)
	if got != 4.0 {
		t.Errorf("WithDraw: expected 4.0, got %f", got)
	}
	t.Log("✓ Fractional scores accumulate correctly (PS = 1 + 1.5 + 1.5 = 4.0)")
}

// TestComputeSumOfProgressiveScores_HalfPointBye — bye score carries into running total.
func TestComputeSumOfProgressiveScores_HalfPointBye(t *testing.T) {
	// Scores: 0.5 (half-point bye), 1 (win) → running: 0.5, 1.5 → PS = 2.0
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteHalf)),
		makePairing(2, p1, &p2, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1.5, p2: 0}

	got := computeSumOfProgressiveScores(p1, allInfo, rawScores)
	if got != 2.0 {
		t.Errorf("HalfPointBye: expected 2.0, got %f", got)
	}
	t.Log("✓ Bye score carries into running total")
}

// TestComputeSumOfProgressiveScores_NonExistentPlayer — returns 0 for unknown player.
func TestComputeSumOfProgressiveScores_NonExistentPlayer(t *testing.T) {
	allInfo := buildPlayerPairingInfo[string](nil)
	rawScores := map[string]float64{}

	got := computeSumOfProgressiveScores(newID(), allInfo, rawScores)
	if got != 0 {
		t.Errorf("NonExistent: expected 0, got %f", got)
	}
	t.Log("✓ Non-existent player returns 0")
}

// TestComputeGamesOneElectedToPlay_AllRealGames — no deductions when all rounds are real games.
func TestComputeGamesOneElectedToPlay_AllRealGames(t *testing.T) {
	p1, p2, p3 := newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.Draw)),
		makePairing(3, p2, &p1, codePtr(result.BlackWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 0, p3: 0.5}

	got := computeGamesOneElectedToPlay(p1, allInfo, rawScores)
	if got != 3 {
		t.Errorf("AllRealGames: expected 3, got %f", got)
	}
	t.Log("✓ No deductions for real games")
}

// TestComputeGamesOneElectedToPlay_HalfPointByeDeducted — half-point bye is deducted.
func TestComputeGamesOneElectedToPlay_HalfPointByeDeducted(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, nil, codePtr(result.WhiteHalf)), // half-point bye → deducted
		makePairing(3, p1, &p2, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 0.5}

	got := computeGamesOneElectedToPlay(p1, allInfo, rawScores)
	if got != 2 {
		t.Errorf("HalfPointBye: expected 2, got %f", got)
	}
	t.Log("✓ Half-point bye deducted")
}

// TestComputeGamesOneElectedToPlay_ZeroPointByeDeducted — zero-point bye is deducted.
func TestComputeGamesOneElectedToPlay_ZeroPointByeDeducted(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, nil, codePtr(result.ZeroPointBye)), // withdrawal → deducted
		makePairing(3, p1, &p2, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1.5, p2: 0.5}

	got := computeGamesOneElectedToPlay(p1, allInfo, rawScores)
	if got != 2 {
		t.Errorf("ZeroPointBye: expected 2, got %f", got)
	}
	t.Log("✓ Zero-point bye deducted")
}

// TestComputeGamesOneElectedToPlay_ForfeitLossDeducted — forfeit loss by the player is deducted.
func TestComputeGamesOneElectedToPlay_ForfeitLossDeducted(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		// p1 is White and forfeits: black wins "-/+"
		makePairing(2, p1, &p2, codePtr(result.BlackForfeit)),
		// p1 is Black and forfeits: white wins "+/-"
		makePairing(3, p2, &p1, codePtr(result.WhiteForfeit)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1, p2: 2}

	got := computeGamesOneElectedToPlay(p1, allInfo, rawScores)
	if got != 1 {
		t.Errorf("ForfeitLoss: expected 1, got %f", got)
	}
	t.Log("✓ Both forfeit-loss cases deducted (as white and as black)")
}

// TestComputeGamesOneElectedToPlay_FullPointByeNotDeducted — full-point bye is NOT deducted.
func TestComputeGamesOneElectedToPlay_FullPointByeNotDeducted(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, nil, codePtr(result.WhiteWins)), // full-point bye → NOT deducted
		makePairing(3, p1, &p2, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2.5, p2: 0.5}

	got := computeGamesOneElectedToPlay(p1, allInfo, rawScores)
	if got != 3 {
		t.Errorf("FullPointBye: expected 3, got %f", got)
	}
	t.Log("✓ Full-point bye not deducted")
}

// TestComputeGamesOneElectedToPlay_ForfeitWinNotDeducted — opponent's forfeit (player showed up) is NOT deducted.
func TestComputeGamesOneElectedToPlay_ForfeitWinNotDeducted(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		// p2 didn't show; p1 (White) wins by forfeit "+/-"
		makePairing(1, p1, &p2, codePtr(result.WhiteForfeit)),
		makePairing(2, p1, &p2, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1.5, p2: 0.5}

	got := computeGamesOneElectedToPlay(p1, allInfo, rawScores)
	if got != 2 {
		t.Errorf("ForfeitWin: expected 2, got %f", got)
	}
	t.Log("✓ Forfeit win (opponent didn't show) not deducted")
}

// TestComputeGamesOneElectedToPlay_NonExistentPlayer — returns 0 for unknown player.
func TestComputeGamesOneElectedToPlay_NonExistentPlayer(t *testing.T) {
	allInfo := buildPlayerPairingInfo[string](nil)
	rawScores := map[string]float64{}

	got := computeGamesOneElectedToPlay(newID(), allInfo, rawScores)
	if got != 0 {
		t.Errorf("NonExistent: expected 0, got %f", got)
	}
	t.Log("✓ Non-existent player returns 0")
}

// TestComputeSumOfProgressiveScores_AllWins — wins every round, PS = sum 1+2+3+4 = 10.
func TestComputeSumOfProgressiveScores_AllWins(t *testing.T) {
	p1, p2, p3, p4, p5 := newID(), newID(), newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p1, &p4, codePtr(result.WhiteWins)),
		makePairing(4, p1, &p5, codePtr(result.WhiteWins)),
		// filler so other players exist in allInfo
		makePairing(1, p3, &p4, codePtr(result.Draw)),
		makePairing(2, p4, &p5, codePtr(result.Draw)),
		makePairing(3, p2, &p5, codePtr(result.Draw)),
		makePairing(4, p2, &p3, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 4, p2: 1, p3: 1, p4: 1, p5: 1}

	// running after each round: 1, 2, 3, 4 → PS = 10
	got := computeSumOfProgressiveScores(p1, allInfo, rawScores)
	if got != 10 {
		t.Errorf("AllWins: expected 10, got %f", got)
	}
	t.Log("✓ All-wins PS = 1+2+3+4 = 10")
}

// TestComputeSumOfProgressiveScores_AbsentRound — a round with no pairing contributes 0;
// the running total carries forward unchanged.
func TestComputeSumOfProgressiveScores_AbsentRound(t *testing.T) {
	// p1 plays rounds 1 and 3 only (no pairing in round 2).
	// Round 1: win (+1) → running = 1
	// Round 2: absent (0)  → running = 1
	// Round 3: win (+1) → running = 2
	// PS = 1 + 1 + 2 = 4
	p1, p2, p3 := newID(), newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(3, p1, &p3, codePtr(result.WhiteWins)),
		// give p2 and p3 a round-2 pairing so maxRound reaches 3
		makePairing(2, p2, &p3, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 0.5, p3: 0.5}

	got := computeSumOfProgressiveScores(p1, allInfo, rawScores)
	if got != 4 {
		t.Errorf("AbsentRound: expected 4, got %f", got)
	}
	t.Log("✓ Absent round carries forward the running total unchanged")
}

// TestComputeSumOfProgressiveScores_SingleRound — PS equals the score in that one round.
func TestComputeSumOfProgressiveScores_SingleRound(t *testing.T) {
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 0.5, p2: 0.5}

	got := computeSumOfProgressiveScores(p1, allInfo, rawScores)
	if got != 0.5 {
		t.Errorf("SingleRound: expected 0.5, got %f", got)
	}
	t.Log("✓ Single-round PS equals the score in that round")
}

// TestComputeGamesOneElectedToPlay_MixedDeductions — different deduction types combine correctly.
func TestComputeGamesOneElectedToPlay_MixedDeductions(t *testing.T) {
	// 5 rounds: win, half-point bye, zero-point bye, forfeit loss as white, real game → GE = 5-3 = 2
	p1, p2 := newID(), newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),    // real win → no deduction
		makePairing(2, p1, nil, codePtr(result.WhiteHalf)),    // half-point bye → deducted
		makePairing(3, p1, nil, codePtr(result.ZeroPointBye)), // zero-point bye → deducted
		makePairing(4, p1, &p2, codePtr(result.BlackForfeit)), // p1 forfeited as white → deducted
		makePairing(5, p1, &p2, codePtr(result.Draw)),         // real draw → no deduction
	}
	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 1}

	got := computeGamesOneElectedToPlay(p1, allInfo, rawScores)
	if got != 2 {
		t.Errorf("MixedDeductions: expected 2, got %f", got)
	}
	t.Log("✓ Half-point bye + zero-point bye + forfeit loss all deducted (5-3=2)")
}

// ============================================================================
// AOB — Average Opponent Buchholz
// ============================================================================

// TestComputeAOB_Basic verifies AOB in a simple 3-player round-robin where
// each opponent's Buchholz is known.
//
// Setup (3 rounds, round-robin):
//
//	R1: P1 beats P2  (P1=white)
//	R2: P1 beats P3  (P1=white)
//	R3: P2 beats P3  (P2=white)
//
// Scores: P1=2, P2=1, P3=0
//
// P1's opponents: P2 and P3.
//
//	BH(P2) = P1's score + P3's score = 2 + 0 = 2
//	BH(P3) = P1's score + P2's score = 2 + 1 = 3
//
// AOB(P1) = round((2 + 3) / 2) = round(2.5) = 3  (0.5 rounds up)
func TestComputeAOB_Basic(t *testing.T) {
	t.Log("=== Testing AOB: basic 3-player round-robin ===")

	p1, p2, p3 := newID(), newID(), newID()

	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.WhiteWins)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 1, p3: 0}

	got := computeAverageOpponentBuchholz(p1, allInfo, rawScores, 0, 0)
	if got != 3 {
		t.Errorf("AOB basic: expected 3, got %f", got)
	}
	t.Log("✓ AOB(P1) = round((BH(P2)+BH(P3))/2) = round(2.5) = 3")
}

// TestComputeAOB_ByeExcluded verifies that bye rounds are not counted as
// OTB opponents for AOB — consistent with ARO and other Type D tiebreaks.
func TestComputeAOB_ByeExcluded(t *testing.T) {
	t.Log("=== Testing AOB: bye opponent excluded ===")

	p1, p2 := newID(), newID()

	// R1: P1 gets a full-point bye
	// R2: P1 beats P2
	// R3: (P2 has a bye)
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)), // bye
		makePairing(2, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(3, p2, nil, codePtr(result.WhiteWins)), // P2 bye
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 2}

	// P1's only OTB opponent is P2. BH(P2) = own score for bye + p1's score.
	// Virtual opponent for P2's bye = P2's raw score = 2. Plus P1's score = 2. BH(P2) = 4.
	// AOB(P1) = round(4 / 1) = 4
	got := computeAverageOpponentBuchholz(p1, allInfo, rawScores, 0, 0)
	if got != 4 {
		t.Errorf("AOB bye excluded: expected 4, got %f", got)
	}
	t.Log("✓ Bye round not counted as OTB opponent for AOB")
}

// TestComputeAOB_ForfeitExcluded verifies that forfeited games are excluded
// from the OTB opponent list used in AOB.
func TestComputeAOB_ForfeitExcluded(t *testing.T) {
	t.Log("=== Testing AOB: forfeit opponent excluded ===")

	p1, p2, p3 := newID(), newID(), newID()

	// R1: P1 wins by forfeit over P2 (not OTB)
	// R2: P1 beats P3 OTB
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteForfeit)), // forfeit
		makePairing(2, p1, &p3, codePtr(result.WhiteWins)),
		makePairing(3, p2, &p3, codePtr(result.Draw)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 2, p2: 0.5, p3: 0.5}

	// P1's OTB opponents: only P3 (P2 was a forfeit).
	// BH(P3) = P1's score + P2's score = 2 + 0.5 = 2.5
	// AOB(P1) = round(2.5 / 1) = 3  (0.5 rounds up)
	got := computeAverageOpponentBuchholz(p1, allInfo, rawScores, 0, 0)
	if got != 3 {
		t.Errorf("AOB forfeit excluded: expected 3, got %f", got)
	}
	t.Log("✓ Forfeit opponent not counted as OTB for AOB")
}

// TestComputeAOB_NoGames returns 0 when the player has no OTB games.
func TestComputeAOB_NoGames(t *testing.T) {
	t.Log("=== Testing AOB: player with only byes returns 0 ===")

	p1 := newID()
	pairings := []*round.Game[string]{
		makePairing(1, p1, nil, codePtr(result.WhiteWins)),
		makePairing(2, p1, nil, codePtr(result.WhiteHalf)),
	}

	allInfo := buildPlayerPairingInfo(pairings)
	rawScores := map[string]float64{p1: 1.5}

	got := computeAverageOpponentBuchholz(p1, allInfo, rawScores, 0, 0)
	if got != 0 {
		t.Errorf("AOB no games: expected 0, got %f", got)
	}
	t.Log("✓ Player with only byes returns AOB = 0")
}

// TestComputeAOB_UnknownPlayer returns 0 for a player not in allInfo.
func TestComputeAOB_UnknownPlayer(t *testing.T) {
	t.Log("=== Testing AOB: unknown player returns 0 ===")

	ghost := newID()
	allInfo := buildPlayerPairingInfo([]*round.Game[string]{})
	rawScores := map[string]float64{}

	got := computeAverageOpponentBuchholz(ghost, allInfo, rawScores, 0, 0)
	if got != 0 {
		t.Errorf("AOB unknown player: expected 0, got %f", got)
	}
	t.Log("✓ Unknown player returns AOB = 0")
}

// TestComputeAOB_WithdrawalAdjustment verifies that the FIDE Art. 16.3.2
// withdrawal adjustment propagates correctly through the Buchholz of each
// opponent used in the AOB average.
func TestComputeAOB_WithdrawalAdjustment(t *testing.T) {
	t.Log("=== Testing AOB: withdrawal adjustment propagates through opponent BH ===")

	p1, p2, p3 := newID(), newID(), newID()

	// P3 withdraws after round 1.
	// R1: P1 beats P2  /  P3 beats someone (represented as bye below for simplicity)
	// R2: P1 beats P3  /  P3 withdraws (zero-point bye)
	// R3: P2 has a bye
	pairings := []*round.Game[string]{
		makePairing(1, p1, &p2, codePtr(result.WhiteWins)),
		makePairing(1, p1, &p3, codePtr(result.WhiteWins)),    // duplicate round ok for test
		makePairing(2, p3, nil, codePtr(result.ZeroPointBye)), // withdrawal
	}

	allInfo := buildPlayerPairingInfo(pairings)
	// P1=2, P2=0, P3=0 (withdrew)
	rawScores := map[string]float64{p1: 2, p2: 0, p3: 0}

	// AOB(P1): OTB opponents are P2 and P3.
	// BH(P2) = opponent P1's adj score = 2 (no withdrawal rounds for P1)
	// BH(P3): P3 faced P1 in round 1 (last real round=1), withdrew round 2 → adj score for P3 = 0 + 0.5 = 0.5
	//   P3's BH = adj(P1) = 2
	// BH(P2) uses virtual opponent for P2's rounds not played; P2 only played R1.
	//   P2's only pairing is vs P1. BH(P2) = adj(P1) = 2.
	// AOB(P1) = round((BH(P2) + BH(P3)) / 2) = round((2 + 2) / 2) = 2
	got := computeAverageOpponentBuchholz(p1, allInfo, rawScores, 0, 0)
	if got != 2 {
		t.Errorf("AOB withdrawal: expected 2, got %f", got)
	}
	t.Log("✓ Withdrawal adjustment propagates through opponent BH into AOB")
}

// ===============================
// computeRatingChange
// ===============================

// TestComputeRatingChange_WinVsEqualRating verifies the basic case:
// winning against an equal-rated opponent gives K/2 rating points.
func TestComputeRatingChange_WinVsEqualRating(t *testing.T) {
	player := newID()
	opp := newID()

	pairings := []*round.Game[string]{
		makePairing(1, player, &opp, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1500, opp: 1500}

	got := computeRatingChange(player, allInfo, ratings, 1500, 20)

	// We = 0.5, W = 1.0 → ΔR = 20 × 0.5 = 10.0
	if got != 10.0 {
		t.Errorf("expected 10.0, got %g", got)
	}
}

// TestComputeRatingChange_DrawVsEqualRating verifies that a draw against
// an equal-rated opponent produces zero rating change.
func TestComputeRatingChange_DrawVsEqualRating(t *testing.T) {
	player := newID()
	opp := newID()

	pairings := []*round.Game[string]{
		makePairing(1, player, &opp, codePtr(result.Draw)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1500, opp: 1500}

	got := computeRatingChange(player, allInfo, ratings, 1500, 20)

	// We = 0.5, W = 0.5 → ΔR = 0
	if got != 0.0 {
		t.Errorf("expected 0.0, got %g", got)
	}
}

// TestComputeRatingChange_LossVsEqualRating verifies a loss against an
// equal-rated opponent costs K/2 rating points.
func TestComputeRatingChange_LossVsEqualRating(t *testing.T) {
	player := newID()
	opp := newID()

	pairings := []*round.Game[string]{
		makePairing(1, opp, &player, codePtr(result.WhiteWins)), // player is black, loses
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1500, opp: 1500}

	got := computeRatingChange(player, allInfo, ratings, 1500, 20)

	// We = 0.5, W = 0.0 → ΔR = 20 × (-0.5) = -10.0
	if got != -10.0 {
		t.Errorf("expected -10.0, got %g", got)
	}
}

// TestComputeRatingChange_WinVsStrongerOpponent verifies a win against an
// opponent rated 400 points higher yields a larger-than-average gain.
func TestComputeRatingChange_WinVsStrongerOpponent(t *testing.T) {
	player := newID()
	opp := newID()

	pairings := []*round.Game[string]{
		makePairing(1, player, &opp, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1200, opp: 1600}

	got := computeRatingChange(player, allInfo, ratings, 1200, 20)

	// We = 1/(1+10^1) ≈ 0.0909, W = 1.0 → ΔR ≈ 18.2
	if got != 18.2 {
		t.Errorf("expected 18.2, got %g", got)
	}
}

// TestComputeRatingChange_LossVsWeakerOpponent verifies that losing to an
// opponent rated 400 points lower produces a large negative rating change.
func TestComputeRatingChange_LossVsWeakerOpponent(t *testing.T) {
	player := newID()
	opp := newID()

	pairings := []*round.Game[string]{
		makePairing(1, opp, &player, codePtr(result.WhiteWins)), // player is black, loses
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1600, opp: 1200}

	got := computeRatingChange(player, allInfo, ratings, 1600, 20)

	// We = 10/11 ≈ 0.909, W = 0.0 → ΔR ≈ -18.2
	if got != -18.2 {
		t.Errorf("expected -18.2, got %g", got)
	}
}

// TestComputeRatingChange_MultipleGames verifies that rating change accumulates
// correctly across multiple OTB games.
func TestComputeRatingChange_MultipleGames(t *testing.T) {
	player := newID()
	opp1, opp2 := newID(), newID()

	pairings := []*round.Game[string]{
		makePairing(1, player, &opp1, codePtr(result.WhiteWins)), // win
		makePairing(2, opp2, &player, codePtr(result.Draw)),      // draw as black
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1500, opp1: 1500, opp2: 1500}

	got := computeRatingChange(player, allInfo, ratings, 1500, 20)

	// Game 1: We=0.5, W=1.0; Game 2: We=0.5, W=0.5 → ΔR = 20×(1.5-1.0) = 10.0
	if got != 10.0 {
		t.Errorf("expected 10.0, got %g", got)
	}
}

// TestComputeRatingChange_ByeDoesNotCount verifies that a full-point bye is
// excluded from rating-change calculation (only OTB games count).
func TestComputeRatingChange_ByeDoesNotCount(t *testing.T) {
	player := newID()
	opp := newID()

	pairings := []*round.Game[string]{
		makePairing(1, player, &opp, codePtr(result.WhiteWins)), // OTB win
		makePairing(2, player, nil, codePtr(result.WhiteWins)),  // full-point bye
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1500, opp: 1500}

	got := computeRatingChange(player, allInfo, ratings, 1500, 20)

	// Only the OTB game counts: We=0.5, W=1.0 → ΔR = 10.0 (not 5.0 which would come from 2 games)
	if got != 10.0 {
		t.Errorf("expected 10.0 (bye excluded), got %g", got)
	}
}

// TestComputeRatingChange_NoGames verifies that a player with no OTB games
// returns 0 regardless of K-factor.
func TestComputeRatingChange_NoGames(t *testing.T) {
	player := newID()

	allInfo := buildPlayerPairingInfo([]*round.Game[string]{})
	ratings := map[string]int{player: 1500}

	got := computeRatingChange(player, allInfo, ratings, 1500, 40)

	if got != 0.0 {
		t.Errorf("expected 0.0 for player with no games, got %g", got)
	}
}

// TestComputeRatingChange_KFactorScalesResult verifies that doubling K doubles
// the rating change.
func TestComputeRatingChange_KFactorScalesResult(t *testing.T) {
	player := newID()
	opp := newID()

	pairings := []*round.Game[string]{
		makePairing(1, player, &opp, codePtr(result.WhiteWins)),
	}
	allInfo := buildPlayerPairingInfo(pairings)
	ratings := map[string]int{player: 1500, opp: 1500}

	deltaK10 := computeRatingChange(player, allInfo, ratings, 1500, 10)
	deltaK40 := computeRatingChange(player, allInfo, ratings, 1500, 40)

	// K=10 → 5.0; K=40 → 20.0; ratio must be 1:4
	ratio := deltaK40 / deltaK10
	if math.Abs(ratio-4.0) > 0.01 {
		t.Errorf("K40/K10 ratio: expected 4.0, got %g", ratio)
	}
}
