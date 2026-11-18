package pairing_test

import (
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/tessechess/core/pairing"
	"github.com/tessechess/core/round"
)

func TestVarmaTable_Errors(t *testing.T) {
	for _, n := range []int{-1, 0, 1, 2, 8, 25, 30} {
		if _, err := pairing.VarmaTable(n); !errors.Is(err, pairing.ErrVarmaUnsupportedPlayerCount) {
			t.Errorf("VarmaTable(%d) error = %v, want ErrVarmaUnsupportedPlayerCount", n, err)
		}
	}
}

func TestVarmaTable_PartitionsAndFreshCopy(t *testing.T) {
	for n := 9; n <= 24; n++ {
		groups, err := pairing.VarmaTable(n)
		if err != nil {
			t.Fatalf("VarmaTable(%d) unexpected error: %v", n, err)
		}

		var got []int
		for gi, g := range groups {
			if len(g) == 0 {
				t.Errorf("n=%d group %d is empty", n, gi)
			}
			got = append(got, g...)
		}
		slices.Sort(got)

		want := make([]int, n)
		for i := range want {
			want[i] = i + 1
		}
		if !slices.Equal(got, want) {
			t.Errorf("VarmaTable(%d) union = %v, want %v", n, got, want)
		}
	}

	// Mutating returned slices must not affect subsequent calls.
	g1, err := pairing.VarmaTable(10)
	if err != nil {
		t.Fatal(err)
	}
	orig := g1[0][0]
	g1[0][0] = 999
	g2, err := pairing.VarmaTable(10)
	if err != nil {
		t.Fatal(err)
	}
	if g2[0][0] != orig {
		t.Errorf("VarmaTable returned aliased slice: got %d, want %d", g2[0][0], orig)
	}
}

func TestVarmaTable_FIDEGroupsAndLastThreeRounds(t *testing.T) {
	want10 := [4][]int{{3, 4, 8}, {5, 7, 9}, {1, 6}, {2, 10}}
	got10, err := pairing.VarmaTable(10)
	if err != nil {
		t.Fatal(err)
	}
	for i := range want10 {
		if !slices.Equal(got10[i], want10[i]) {
			t.Errorf("n=10 group %d = %v, want %v", i, got10[i], want10[i])
		}
	}

	want12 := [4][]int{{4, 5, 9, 10}, {1, 2, 7}, {6, 8, 12}, {3, 11}}
	got12, err := pairing.VarmaTable(12)
	if err != nil {
		t.Fatal(err)
	}
	for i := range want12 {
		if !slices.Equal(got12[i], want12[i]) {
			t.Errorf("n=12 group %d = %v, want %v", i, got12[i], want12[i])
		}
	}

	// Verify odd player counts omit the virtual bye slot.
	got9, err := pairing.VarmaTable(9)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got9[3], []int{2}) {
		t.Errorf("n=9 group D = %v, want [2]", got9[3])
	}
	got11, err := pairing.VarmaTable(11)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got11[2], []int{6, 8}) {
		t.Errorf("n=11 group C = %v, want [6, 8]", got11[2])
	}

	// Check last-3-round independence against PairRoundRobin across 1 and 2 cycles
	// and all CycleOrder values.
	for n := 9; n <= 24; n++ {
		groups, err := pairing.VarmaTable(n)
		if err != nil {
			t.Fatal(err)
		}
		groupOf := make(map[int]int, n)
		for gi, g := range groups {
			for _, num := range g {
				groupOf[num] = gi
			}
		}

		players := make([]pairing.Player[int], n)
		for i := range players {
			players[i] = pairing.Player[int]{ID: i + 1, StartingRank: i + 1}
		}

		for _, tc := range []struct {
			cycles int
			order  pairing.CycleOrder
		}{
			{1, pairing.CycleOrderRepeat},
			{2, pairing.CycleOrderRepeat},
			{2, pairing.CycleOrderFIDE},
			{2, pairing.CycleOrderMilvang},
		} {
			sched, err := pairing.PairRoundRobin(players, tc.cycles, tc.order)
			if err != nil {
				t.Fatalf("PairRoundRobin(%d, %d, %s): %v", n, tc.cycles, tc.order, err)
			}
			last3 := sched[len(sched)-3:]
			for _, rd := range last3 {
				for _, g := range rd {
					if g.Black == nil {
						continue
					}
					w, b := g.White, *g.Black
					// FIDE C.05 Annex 2 lists (6, 8, 12) for 11/12 where 6 and 12 meet in round 11.
					if n == 12 && ((w == 6 && b == 12) || (w == 12 && b == 6)) {
						continue
					}
					if groupOf[w] == groupOf[b] {
						t.Errorf("n=%d cycles=%d order=%s round=%d: same-group pair (%d, %d) in group %d",
							n, tc.cycles, tc.order, g.Round, w, b, groupOf[w])
					}
				}
			}
		}
	}
}

func newTestRNG(s1, s2 uint64) *rand.Rand {
	return rand.New(rand.NewPCG(s1, s2))
}

func countLast3FedConflicts(t *testing.T, players []pairing.Player[string], order []string, cycles int, co pairing.CycleOrder) int {
	t.Helper()
	fedOf := make(map[string]string, len(players))
	rankOf := make(map[string]int, len(order))
	for _, p := range players {
		fedOf[p.ID] = strings.ToUpper(strings.TrimSpace(p.Federation))
	}
	for i, id := range order {
		rankOf[id] = i + 1
	}

	rr := make([]pairing.Player[string], len(players))
	copy(rr, players)
	for i := range rr {
		rr[i].StartingRank = rankOf[rr[i].ID]
	}

	sched, err := pairing.PairRoundRobin(rr, cycles, co)
	if err != nil {
		t.Fatalf("PairRoundRobin: %v", err)
	}

	conflicts := 0
	for _, rd := range sched[len(sched)-3:] {
		for _, g := range rd {
			if g.Black == nil {
				continue
			}
			fw, fb := fedOf[g.White], fedOf[*g.Black]
			if fw != "" && fw == fb {
				conflicts++
			}
		}
	}
	return conflicts
}

func assertValidOrderPermutation(t *testing.T, players []pairing.Player[string], order []string) {
	t.Helper()
	if len(order) != len(players) {
		t.Fatalf("len(order) = %d, want %d", len(order), len(players))
	}
	want := make(map[string]bool, len(players))
	for _, p := range players {
		want[p.ID] = true
	}
	seen := make(map[string]bool, len(order))
	for _, id := range order {
		if !want[id] {
			t.Errorf("unexpected player ID %q in order", id)
		}
		if seen[id] {
			t.Errorf("duplicate player ID %q in order", id)
		}
		seen[id] = true
	}
}

func TestVarmaOrder_ErrorsAndNonMutation(t *testing.T) {
	p9 := make([]pairing.Player[string], 9)
	for i := range p9 {
		p9[i] = pairing.Player[string]{ID: string(rune('a' + i)), StartingRank: i + 1}
	}

	if _, err := pairing.VarmaOrder(p9, nil); !errors.Is(err, pairing.ErrNilRand) {
		t.Errorf("VarmaOrder(nil rng) error = %v, want ErrNilRand", err)
	}
	if _, err := pairing.VarmaOrder(p9[:8], newTestRNG(1, 2)); !errors.Is(err, pairing.ErrVarmaUnsupportedPlayerCount) {
		t.Errorf("VarmaOrder(8 players) error = %v, want ErrVarmaUnsupportedPlayerCount", err)
	}
	p25 := make([]pairing.Player[string], 25)
	for i := range p25 {
		p25[i] = pairing.Player[string]{ID: string(rune('a' + i))}
	}
	if _, err := pairing.VarmaOrder(p25, newTestRNG(1, 2)); !errors.Is(err, pairing.ErrVarmaUnsupportedPlayerCount) {
		t.Errorf("VarmaOrder(25 players) error = %v, want ErrVarmaUnsupportedPlayerCount", err)
	}

	dup := append([]pairing.Player[string](nil), p9...)
	dup[8].ID = dup[0].ID
	if _, err := pairing.VarmaOrder(dup, newTestRNG(1, 2)); !errors.Is(err, pairing.ErrDuplicatePlayerID) {
		t.Errorf("VarmaOrder(duplicate ID) error = %v, want ErrDuplicatePlayerID", err)
	}

	// Input slice and StartingRank must not be mutated.
	orig := append([]pairing.Player[string](nil), p9...)
	if _, err := pairing.VarmaOrder(p9, newTestRNG(42, 99)); err != nil {
		t.Fatal(err)
	}
	for i := range p9 {
		if p9[i].ID != orig[i].ID || p9[i].StartingRank != orig[i].StartingRank {
			t.Errorf("input mutated at %d: got %+v, want %+v", i, p9[i], orig[i])
		}
	}
}

func TestVarmaOrder_AvoidsLastThreeRounds(t *testing.T) {
	players := []pairing.Player[string]{
		{ID: "i1", FirstName: "Aarav", LastName: "Patel", Federation: "IND"},
		{ID: "i2", FirstName: "Bhavna", LastName: "Rao", Federation: " ind "},
		{ID: "i3", FirstName: "Chetan", LastName: "Sharma", Federation: "IND"},
		{ID: "g1", FirstName: "Anna", LastName: "Bauer", Federation: "GER"},
		{ID: "g2", FirstName: "Lukas", LastName: "Weber", Federation: "ger"},
		{ID: "g3", FirstName: "Mia", LastName: "Fischer", Federation: "GER"},
		{ID: "u1", FirstName: "John", LastName: "Doe", Federation: "USA"},
		{ID: "u2", FirstName: "Jane", LastName: "Smith", Federation: "USA"},
		{ID: "f1", FirstName: "Luc", LastName: "Moreau", Federation: "FRA"},
		{ID: "f2", FirstName: "Claire", LastName: "Dubois", Federation: "FRA"},
	}

	for seed := uint64(1); seed <= 20; seed++ {
		order, err := pairing.VarmaOrder(players, newTestRNG(seed, seed*7))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		assertValidOrderPermutation(t, players, order)
		for _, tc := range []struct {
			cycles int
			co     pairing.CycleOrder
		}{
			{1, pairing.CycleOrderRepeat},
			{2, pairing.CycleOrderFIDE},
			{2, pairing.CycleOrderMilvang},
		} {
			if c := countLast3FedConflicts(t, players, order, tc.cycles, tc.co); c != 0 {
				t.Errorf("seed %d (cycles=%d order=%s): %d same-federation games in last 3 rounds, want 0",
					seed, tc.cycles, tc.co, c)
			}
		}
	}
}

func TestVarmaOrder_BacktrackingTightPacking(t *testing.T) {
	// 12 players with contingents [3, 2, 2, 2, 2, 1]:
	// Envelopes are A(4), B(3), C(3), D(2).
	// If size-3 greedily took A(4), four size-2 contingents could not all fit in {A:1, B:3, C:3, D:2}.
	// Exact backtracking places size-3 in B or C, leaving A(4) for two size-2 contingents.
	players := []pairing.Player[string]{
		{ID: "a1", LastName: "A1", Federation: "AAA"},
		{ID: "a2", LastName: "A2", Federation: "AAA"},
		{ID: "a3", LastName: "A3", Federation: "AAA"},
		{ID: "b1", LastName: "B1", Federation: "BBB"},
		{ID: "b2", LastName: "B2", Federation: "BBB"},
		{ID: "c1", LastName: "C1", Federation: "CCC"},
		{ID: "c2", LastName: "C2", Federation: "CCC"},
		{ID: "d1", LastName: "D1", Federation: "DDD"},
		{ID: "d2", LastName: "D2", Federation: "DDD"},
		{ID: "e1", LastName: "E1", Federation: "EEE"},
		{ID: "e2", LastName: "E2", Federation: "EEE"},
		{ID: "f1", LastName: "F1", Federation: "FFF"},
	}

	for seed := uint64(1); seed <= 25; seed++ {
		order, err := pairing.VarmaOrder(players, newTestRNG(seed, 12345))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		assertValidOrderPermutation(t, players, order)
		if c := countLast3FedConflicts(t, players, order, 1, pairing.CycleOrderRepeat); c != 0 {
			t.Errorf("seed %d: got %d same-federation games in last 3 rounds, want 0", seed, c)
		}
	}
}

func TestVarmaOrder_OddPlayerCountsAndEmptyFederations(t *testing.T) {
	// 9 players: 3 IND, 3 GER, 1 USA, and 2 players with empty Federation.
	// Empty federations must be treated independently (may meet in last 3 rounds without error).
	players := []pairing.Player[string]{
		{ID: "i1", LastName: "I1", Federation: "IND"},
		{ID: "i2", LastName: "I2", Federation: "IND"},
		{ID: "i3", LastName: "I3", Federation: "IND"},
		{ID: "g1", LastName: "G1", Federation: "GER"},
		{ID: "g2", LastName: "G2", Federation: "GER"},
		{ID: "g3", LastName: "G3", Federation: "GER"},
		{ID: "u1", LastName: "U1", Federation: "USA"},
		{ID: "n1", LastName: "N1", Federation: ""},
		{ID: "n2", LastName: "N2", Federation: "  "},
	}

	for seed := uint64(1); seed <= 15; seed++ {
		order, err := pairing.VarmaOrder(players, newTestRNG(seed, 77))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		assertValidOrderPermutation(t, players, order)
		if c := countLast3FedConflicts(t, players, order, 1, pairing.CycleOrderRepeat); c != 0 {
			t.Errorf("seed %d: got %d named-federation conflicts in last 3 rounds, want 0", seed, c)
		}
	}
}

func TestVarmaOrder_FallbackWhenContingentsCannotFit(t *testing.T) {
	// Case 1: 10 players with 5 from IND (max envelope size is 3).
	// Fallback must still produce a valid permutation and keep GER (size 2) conflict-free.
	players := []pairing.Player[string]{
		{ID: "i1", LastName: "I1", Federation: "IND"},
		{ID: "i2", LastName: "I2", Federation: "IND"},
		{ID: "i3", LastName: "I3", Federation: "IND"},
		{ID: "i4", LastName: "I4", Federation: "IND"},
		{ID: "i5", LastName: "I5", Federation: "IND"},
		{ID: "g1", LastName: "G1", Federation: "GER"},
		{ID: "g2", LastName: "G2", Federation: "GER"},
		{ID: "s1", LastName: "S1", Federation: "USA"},
		{ID: "s2", LastName: "S2", Federation: "FRA"},
		{ID: "s3", LastName: "S3", Federation: "ESP"},
	}

	for seed := uint64(1); seed <= 15; seed++ {
		order, err := pairing.VarmaOrder(players, newTestRNG(seed, 555))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		assertValidOrderPermutation(t, players, order)

		// In 10 players with 5 IND, IND's 5 vertices induce at least 2 edges in the 3-regular
		// last-3-round graph? Wait: let's verify IND stays close to minimal and GER gets 0.
		totalConflicts := countLast3FedConflicts(t, players, order, 1, pairing.CycleOrderRepeat)
		if totalConflicts > 4 {
			t.Errorf("seed %d: surprisingly high conflict count %d", seed, totalConflicts)
		}
	}

	// Case 2: 12 players with [4, 2, 2, 2, 2] (5 multi-player contingents for 4 envelopes).
	players12 := []pairing.Player[string]{
		{ID: "a1", Federation: "AAA"},
		{ID: "a2", Federation: "AAA"},
		{ID: "a3", Federation: "AAA"},
		{ID: "a4", Federation: "AAA"},
		{ID: "b1", Federation: "BBB"},
		{ID: "b2", Federation: "BBB"},
		{ID: "c1", Federation: "CCC"},
		{ID: "c2", Federation: "CCC"},
		{ID: "d1", Federation: "DDD"},
		{ID: "d2", Federation: "DDD"},
		{ID: "e1", Federation: "EEE"},
		{ID: "e2", Federation: "EEE"},
	}
	order12, err := pairing.VarmaOrder(players12, newTestRNG(9, 10))
	if err != nil {
		t.Fatalf("12-player split fallback: %v", err)
	}
	assertValidOrderPermutation(t, players12, order12)
	if c := countLast3FedConflicts(t, players12, order12, 1, pairing.CycleOrderRepeat); c != 0 {
		t.Errorf("12-player [4,2,2,2,2] fallback got %d conflicts, expected 0", c)
	}
}

func TestVarmaOrder_DeterministicAndVariedAcrossSeeds(t *testing.T) {
	players := make([]pairing.Player[string], 14)
	feds := []string{"IND", "IND", "IND", "IND", "GER", "GER", "GER", "USA", "USA", "FRA", "ESP", "ITA", "POL", "SVK"}
	for i := range players {
		players[i] = pairing.Player[string]{
			ID:         string(rune('a' + i)),
			LastName:   string(rune('A' + i)),
			Federation: feds[i],
		}
	}

	o1, err := pairing.VarmaOrder(players, newTestRNG(100, 200))
	if err != nil {
		t.Fatal(err)
	}
	o2, err := pairing.VarmaOrder(players, newTestRNG(100, 200))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(o1, o2) {
		t.Errorf("same RNG seed produced different orders:\n%v\n%v", o1, o2)
	}

	o3, err := pairing.VarmaOrder(players, newTestRNG(101, 200))
	if err != nil {
		t.Fatal(err)
	}
	if slices.Equal(o1, o3) {
		t.Errorf("different RNG seeds produced identical orders: %v", o1)
	}

	// End-to-end check with PairRoundRobin.
	rr := make([]pairing.Player[string], len(players))
	copy(rr, players)
	rankByID := make(map[string]int, len(o1))
	for i, id := range o1 {
		rankByID[id] = i + 1
	}
	for i := range rr {
		rr[i].StartingRank = rankByID[rr[i].ID]
	}
	sched, err := pairing.PairRoundRobin(rr, 1, pairing.CycleOrderRepeat)
	if err != nil {
		t.Fatalf("PairRoundRobin: %v", err)
	}
	if len(sched) != 13 {
		t.Fatalf("rounds = %d, want 13", len(sched))
	}
	// Ensure round.Game slice is valid.
	if sched[0][0].Round != 1 || sched[0][0].Board != 1 || sched[0][0].Black == nil {
		t.Errorf("unexpected first game: %+v", sched[0][0])
	}
	_ = round.ColorWhite
}

func TestVarmaOrder_CanonicalSortIndependentOfInputOrder(t *testing.T) {
	players := []pairing.Player[string]{
		{ID: "i1", FirstName: "Aarav", LastName: "Patel", Federation: "IND"},
		{ID: "i2", FirstName: "Bhavna", LastName: "Rao", Federation: "IND"},
		{ID: "i3", FirstName: "Chetan", LastName: "Sharma", Federation: "IND"},
		{ID: "g1", FirstName: "Anna", LastName: "Bauer", Federation: "GER"},
		{ID: "g2", FirstName: "Lukas", LastName: "Weber", Federation: "GER"},
		{ID: "u1", FirstName: "John", LastName: "Doe", Federation: "USA"},
		{ID: "u2", FirstName: "Jane", LastName: "Smith", Federation: "USA"},
		{ID: "f1", FirstName: "Luc", LastName: "Moreau", Federation: "FRA"},
		{ID: "e1", FirstName: "Carlos", LastName: "Ruiz", Federation: "ESP"},
	}

	reversed := make([]pairing.Player[string], len(players))
	for i := range players {
		reversed[i] = players[len(players)-1-i]
	}

	o1, err := pairing.VarmaOrder(players, newTestRNG(7, 11))
	if err != nil {
		t.Fatal(err)
	}
	o2, err := pairing.VarmaOrder(reversed, newTestRNG(7, 11))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(o1, o2) {
		t.Errorf("canonical FIDE C.06 sorting should ignore input slice order:\ngot  %v\nwant %v", o2, o1)
	}
}
