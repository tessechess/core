package pairing

import (
	"errors"
	"testing"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

func TestBerger_Errors(t *testing.T) {
	for _, n := range []int{-5, 0, 1} {
		if _, err := Berger(n); !errors.Is(err, ErrNotEnoughPlayers) {
			t.Errorf("Berger(%d) error = %v, want ErrNotEnoughPlayers", n, err)
		}
	}
}

func TestBerger_FIDEHandbookTables(t *testing.T) {
	// Official FIDE Handbook C.05 Annex 1 Berger Tables.
	tests := []struct {
		name string
		n    int
		want [][]BergerPair
	}{
		{
			name: "2 players",
			n:    2,
			want: [][]BergerPair{
				{{1, 2}},
			},
		},
		{
			name: "3-4 players (odd n=3)",
			n:    3,
			want: [][]BergerPair{
				{{1, 4}, {2, 3}},
				{{4, 3}, {1, 2}},
				{{2, 4}, {3, 1}},
			},
		},
		{
			name: "3-4 players (even n=4)",
			n:    4,
			want: [][]BergerPair{
				{{1, 4}, {2, 3}},
				{{4, 3}, {1, 2}},
				{{2, 4}, {3, 1}},
			},
		},
		{
			name: "5-6 players",
			n:    6,
			want: [][]BergerPair{
				{{1, 6}, {2, 5}, {3, 4}},
				{{6, 4}, {5, 3}, {1, 2}},
				{{2, 6}, {3, 1}, {4, 5}},
				{{6, 5}, {1, 4}, {2, 3}},
				{{3, 6}, {4, 2}, {5, 1}},
			},
		},
		{
			name: "7-8 players",
			n:    8,
			want: [][]BergerPair{
				{{1, 8}, {2, 7}, {3, 6}, {4, 5}},
				{{8, 5}, {6, 4}, {7, 3}, {1, 2}},
				{{2, 8}, {3, 1}, {4, 7}, {5, 6}},
				{{8, 6}, {7, 5}, {1, 4}, {2, 3}},
				{{3, 8}, {4, 2}, {5, 1}, {6, 7}},
				{{8, 7}, {1, 6}, {2, 5}, {3, 4}},
				{{4, 8}, {5, 3}, {6, 2}, {7, 1}},
			},
		},
		{
			name: "9-10 players",
			n:    10,
			want: [][]BergerPair{
				{{1, 10}, {2, 9}, {3, 8}, {4, 7}, {5, 6}},
				{{10, 6}, {7, 5}, {8, 4}, {9, 3}, {1, 2}},
				{{2, 10}, {3, 1}, {4, 9}, {5, 8}, {6, 7}},
				{{10, 7}, {8, 6}, {9, 5}, {1, 4}, {2, 3}},
				{{3, 10}, {4, 2}, {5, 1}, {6, 9}, {7, 8}},
				{{10, 8}, {9, 7}, {1, 6}, {2, 5}, {3, 4}},
				{{4, 10}, {5, 3}, {6, 2}, {7, 1}, {8, 9}},
				{{10, 9}, {1, 8}, {2, 7}, {3, 6}, {4, 5}},
				{{5, 10}, {6, 4}, {7, 3}, {8, 2}, {9, 1}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Berger(tc.n)
			if err != nil {
				t.Fatalf("Berger(%d) unexpected error: %v", tc.n, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("Berger(%d) rounds = %d, want %d", tc.n, len(got), len(tc.want))
			}
			for r := range tc.want {
				if len(got[r]) != len(tc.want[r]) {
					t.Fatalf("round %d boards = %d, want %d", r+1, len(got[r]), len(tc.want[r]))
				}
				for b := range tc.want[r] {
					if got[r][b] != tc.want[r][b] {
						t.Errorf("round %d board %d = %+v, want %+v", r+1, b+1, got[r][b], tc.want[r][b])
					}
				}
			}
		})
	}
}

// vegaBergerReference implements Vega / Gacrux's iterative list-pop Berger
// construction to verify equivalence with our closed-form Berger function.
func vegaBergerReference(n int) [][]BergerPair {
	evenN := n + n%2
	pairs := evenN / 2
	totalRounds := evenN - 1

	schedule := make([][]BergerPair, totalRounds)
	round1 := make([]BergerPair, pairs)
	for b := 1; b <= pairs; b++ {
		round1[b-1] = BergerPair{White: b, Black: evenN - b + 1}
	}
	schedule[0] = round1

	prev := round1
	for r := 2; r <= totalRounds; r++ {
		playerList := make([]int, 0, evenN-1)
		for _, p := range prev {
			if p.White < evenN {
				playerList = append(playerList, p.White)
			}
			if p.Black < evenN {
				playerList = append(playerList, p.Black)
			}
		}

		pop := func() int {
			v := playerList[len(playerList)-1]
			playerList = playerList[:len(playerList)-1]
			return v
		}

		curr := make([]BergerPair, pairs)
		firstBoard := pop()
		if r%2 == 0 {
			curr[0] = BergerPair{White: evenN, Black: firstBoard}
		} else {
			curr[0] = BergerPair{White: firstBoard, Black: evenN}
		}
		for b := 2; b <= pairs; b++ {
			black := pop()
			white := pop()
			curr[b-1] = BergerPair{White: white, Black: black}
		}
		schedule[r-1] = curr
		prev = curr
	}
	return schedule
}

func TestBerger_MatchesVegaAndInvariants(t *testing.T) {
	for n := 2; n <= 24; n++ {
		got, err := Berger(n)
		if err != nil {
			t.Fatalf("Berger(%d) unexpected error: %v", n, err)
		}
		want := vegaBergerReference(n)

		evenN := n + n%2
		pairs := evenN / 2
		totalRounds := evenN - 1

		if len(got) != totalRounds {
			t.Fatalf("n=%d: len(got)=%d, want %d", n, len(got), totalRounds)
		}

		// Verify exact match with Vega reference generator.
		for r := 0; r < totalRounds; r++ {
			for b := 0; b < pairs; b++ {
				if got[r][b] != want[r][b] {
					t.Fatalf("n=%d round %d board %d: got %+v, want %+v", n, r+1, b+1, got[r][b], want[r][b])
				}
			}
		}

		// Verify all-play-all and color balance invariants.
		met := make(map[[2]int]int)
		whites := make(map[int]int)
		blacks := make(map[int]int)

		for r, roundPairs := range got {
			seenInRound := make(map[int]bool, evenN)
			for _, bp := range roundPairs {
				if seenInRound[bp.White] {
					t.Errorf("n=%d round %d: player %d appears twice", n, r+1, bp.White)
				}
				if seenInRound[bp.Black] {
					t.Errorf("n=%d round %d: player %d appears twice", n, r+1, bp.Black)
				}
				seenInRound[bp.White] = true
				seenInRound[bp.Black] = true

				whites[bp.White]++
				blacks[bp.Black]++

				a, b := bp.White, bp.Black
				if a > b {
					a, b = b, a
				}
				met[[2]int{a, b}]++
			}
			if len(seenInRound) != evenN {
				t.Errorf("n=%d round %d: expected %d distinct slots, got %d", n, r+1, evenN, len(seenInRound))
			}
		}

		for i := 1; i <= evenN; i++ {
			for j := i + 1; j <= evenN; j++ {
				if met[[2]int{i, j}] != 1 {
					t.Errorf("n=%d: pair (%d,%d) met %d times, want 1", n, i, j, met[[2]int{i, j}])
				}
			}
			diff := whites[i] - blacks[i]
			if diff != 1 && diff != -1 {
				t.Errorf("n=%d player %d: whites=%d blacks=%d (diff=%d, want ±1)", n, i, whites[i], blacks[i], diff)
			}
		}
	}
}

func TestRoundRobinTotalRounds(t *testing.T) {
	if got := RoundRobinTotalRounds(1, 1); got != 0 {
		t.Errorf("RoundRobinTotalRounds(1, 1) = %d, want 0", got)
	}
	if got := RoundRobinTotalRounds(6, 0); got != 0 {
		t.Errorf("RoundRobinTotalRounds(6, 0) = %d, want 0", got)
	}
	if got := RoundRobinTotalRounds(6, 1); got != 5 {
		t.Errorf("RoundRobinTotalRounds(6, 1) = %d, want 5", got)
	}
	if got := RoundRobinTotalRounds(5, 1); got != 5 {
		t.Errorf("RoundRobinTotalRounds(5, 1) = %d, want 5", got)
	}
	if got := RoundRobinTotalRounds(6, 2); got != 10 {
		t.Errorf("RoundRobinTotalRounds(6, 2) = %d, want 10", got)
	}
}

func TestPairRoundRobin_EvenSingleAndDouble(t *testing.T) {
	players := []Player[string]{
		{ID: "p1", StartingRank: 1},
		{ID: "p2", StartingRank: 2},
		{ID: "p3", StartingRank: 3},
		{ID: "p4", StartingRank: 4},
	}

	schedule, err := PairRoundRobin(players, 2, CycleOrderRepeat)
	if err != nil {
		t.Fatalf("PairRoundRobin unexpected error: %v", err)
	}
	if len(schedule) != 6 {
		t.Fatalf("expected 6 rounds for 4-player double RR, got %d", len(schedule))
	}

	// Verify Round 1: (p1 vs p4), (p2 vs p3)
	r1 := schedule[0]
	if len(r1) != 2 {
		t.Fatalf("round 1 boards = %d, want 2", len(r1))
	}
	if r1[0].Round != 1 || r1[0].Board != 1 || r1[0].White != "p1" || r1[0].Black == nil || *r1[0].Black != "p4" {
		t.Errorf("round 1 board 1 = %+v", r1[0])
	}
	if r1[1].Round != 1 || r1[1].Board != 2 || r1[1].White != "p2" || r1[1].Black == nil || *r1[1].Black != "p3" {
		t.Errorf("round 1 board 2 = %+v", r1[1])
	}

	// Verify Cycle 2 (rounds 4..6) has reversed colors compared to Cycle 1 (rounds 1..3).
	for r := 0; r < 3; r++ {
		c1 := schedule[r]
		c2 := schedule[r+3]
		if len(c1) != len(c2) {
			t.Fatalf("round %d and %d board count mismatch", r+1, r+4)
		}
		for b := range c1 {
			if c2[b].Round != r+4 || c2[b].Board != b+1 {
				t.Errorf("round %d board %d metadata = (Round:%d, Board:%d)", r+4, b+1, c2[b].Round, c2[b].Board)
			}
			if c2[b].White != *c1[b].Black || *c2[b].Black != c1[b].White {
				t.Errorf("round %d board %d colors not reversed: c1=(%s vs %s), c2=(%s vs %s)",
					r+4, b+1, c1[b].White, *c1[b].Black, c2[b].White, *c2[b].Black)
			}
		}
	}
}

func TestPairRoundRobin_OddPlayers_ByeOnLastBoard(t *testing.T) {
	players := []Player[string]{
		{ID: "p1", StartingRank: 1},
		{ID: "p2", StartingRank: 2},
		{ID: "p3", StartingRank: 3},
		{ID: "p4", StartingRank: 4},
		{ID: "p5", StartingRank: 5},
	}

	schedule, err := PairRoundRobin(players, 2, CycleOrderRepeat)
	if err != nil {
		t.Fatalf("PairRoundRobin unexpected error: %v", err)
	}
	if len(schedule) != 10 {
		t.Fatalf("expected 10 rounds for 5-player double RR, got %d", len(schedule))
	}

	// Expected bye recipients in rounds 1..5 for 5-6 Berger table:
	// R1: 1-6 -> p1, R2: 6-4 -> p4, R3: 2-6 -> p2, R4: 6-5 -> p5, R5: 3-6 -> p3
	wantBye := []string{"p1", "p4", "p2", "p5", "p3", "p1", "p4", "p2", "p5", "p3"}

	for r, games := range schedule {
		if len(games) != 3 {
			t.Fatalf("round %d: expected 3 games (2 played + 1 bye), got %d", r+1, len(games))
		}
		for b := 0; b < 2; b++ {
			if games[b].Board != b+1 {
				t.Errorf("round %d played game %d: Board = %d, want %d", r+1, b, games[b].Board, b+1)
			}
			if games[b].IsBye() {
				t.Errorf("round %d board %d should not be a bye", r+1, b+1)
			}
		}
		bye := games[2]
		if bye.Round != r+1 || bye.Board != 3 || !bye.IsBye() || bye.White != wantBye[r] {
			t.Errorf("round %d bye = %+v, want Round=%d Board=3 White=%s Black=nil", r+1, bye, r+1, wantBye[r])
		}
		// A free round scores nothing and takes no part in the tiebreaks: Void,
		// not ZeroPointBye (standings treat that like a Swiss bye) and not the
		// full-point bye.
		if bye.Result == nil || *bye.Result != result.Void {
			t.Errorf("round %d bye Result = %v, want %q", r+1, bye.Result, result.Void)
		} else if bye.Result.Counts() || bye.Points(bye.White) != 0 {
			t.Errorf("round %d bye should not count and award 0 points", r+1)
		}
		for b := 0; b < 2; b++ {
			if games[b].Result != nil {
				t.Errorf("round %d board %d: played game has Result %v, want nil", r+1, b+1, *games[b].Result)
			}
		}
	}
}

func TestPairRoundRobin_FixedBoards(t *testing.T) {
	fb1 := 1
	players := []Player[string]{
		{ID: "p1", StartingRank: 1, FixedBoard: &fb1},
		{ID: "p2", StartingRank: 2},
		{ID: "p3", StartingRank: 3},
		{ID: "p4", StartingRank: 4},
		{ID: "p5", StartingRank: 5},
	}

	schedule, err := PairRoundRobin(players, 1, "")
	if err != nil {
		t.Fatalf("PairRoundRobin unexpected error: %v", err)
	}

	// Round 1: p1 has a bye, so p1's bye must remain on board 3 while played games are on boards 1..2.
	if !schedule[0][2].IsBye() || schedule[0][2].White != "p1" || schedule[0][2].Board != 3 {
		t.Errorf("round 1: p1 bye should stay on board 3, got %+v", schedule[0][2])
	}

	// Rounds 2..5: p1 plays an opponent and must be placed on FixedBoard 1.
	for r := 1; r < 5; r++ {
		b1 := schedule[r][0]
		if b1.Board != 1 {
			t.Fatalf("round %d first game board = %d, want 1", r+1, b1.Board)
		}
		if b1.White != "p1" && (b1.Black == nil || *b1.Black != "p1") {
			t.Errorf("round %d board 1 = (%s vs %v), want p1 on board 1", r+1, b1.White, b1.Black)
		}
	}
}

func TestPairRoundRobin_ValidationErrors(t *testing.T) {
	validPlayers := []Player[string]{
		{ID: "p1", StartingRank: 1},
		{ID: "p2", StartingRank: 2},
	}

	if _, err := PairRoundRobin(validPlayers[:1], 1, ""); !errors.Is(err, ErrNotEnoughPlayers) {
		t.Errorf("expected ErrNotEnoughPlayers, got %v", err)
	}
	if _, err := PairRoundRobin(validPlayers, 0, ""); !errors.Is(err, ErrInvalidCycles) {
		t.Errorf("expected ErrInvalidCycles, got %v", err)
	}

	missingRank := []Player[string]{
		{ID: "p1", StartingRank: 1},
		{ID: "p2", StartingRank: 0},
	}
	if _, err := PairRoundRobin(missingRank, 1, ""); !errors.Is(err, ErrInvalidStartingRank) {
		t.Errorf("expected ErrInvalidStartingRank for rank 0, got %v", err)
	}

	outOfRangeRank := []Player[string]{
		{ID: "p1", StartingRank: 1},
		{ID: "p2", StartingRank: 3},
	}
	if _, err := PairRoundRobin(outOfRangeRank, 1, ""); !errors.Is(err, ErrInvalidStartingRank) {
		t.Errorf("expected ErrInvalidStartingRank for rank 3, got %v", err)
	}

	duplicateRank := []Player[string]{
		{ID: "p1", StartingRank: 1},
		{ID: "p2", StartingRank: 1},
	}
	if _, err := PairRoundRobin(duplicateRank, 1, ""); !errors.Is(err, ErrInvalidStartingRank) {
		t.Errorf("expected ErrInvalidStartingRank for duplicate rank, got %v", err)
	}
}

// rrPlayers returns n players whose ID equals their starting rank.
func rrPlayers(n int) []Player[int] {
	ps := make([]Player[int], n)
	for i := range ps {
		ps[i] = Player[int]{ID: i + 1, StartingRank: i + 1}
	}
	return ps
}

// tableIndex returns the 0-based index of the Berger round that games play,
// and whether the colours are those of the table (false = reversed).
func tableIndex(t *testing.T, base [][]BergerPair, p int, games []round.Game[int]) (idx int, sameColours bool) {
	t.Helper()
	for i, pairs := range base {
		same, reversed, played := true, true, 0
		for _, bp := range pairs {
			if bp.White > p || bp.Black > p {
				continue // free round
			}
			played++
			found := false
			for _, g := range games {
				if g.Black == nil {
					continue
				}
				if g.White == bp.White && *g.Black == bp.Black {
					found, reversed = true, false
				}
				if g.White == bp.Black && *g.Black == bp.White {
					found, same = true, false
				}
			}
			if !found {
				same, reversed = false, false
			}
		}
		if played > 0 && (same || reversed) && !(same && reversed) {
			return i, same
		}
	}
	t.Fatalf("round %+v matches no Berger round", games)
	return 0, false
}

func TestPairRoundRobin_CycleOrders(t *testing.T) {
	// Expected order of the table rounds (0-based) in the first cycle; the second
	// cycle is always in table order with reversed colours.
	tests := []struct {
		name  string
		n     int
		order CycleOrder
		want  []int
	}{
		{"repeat n=4", 4, CycleOrderRepeat, []int{0, 1, 2}},
		{"empty is repeat n=4", 4, "", []int{0, 1, 2}},
		{"fide n=4", 4, CycleOrderFIDE, []int{0, 2, 1}},
		{"milvang n=4", 4, CycleOrderMilvang, []int{2, 0, 1}},
		{"fide n=6", 6, CycleOrderFIDE, []int{0, 1, 2, 4, 3}},
		{"milvang n=6", 6, CycleOrderMilvang, []int{4, 0, 1, 2, 3}},
		{"fide odd n=5", 5, CycleOrderFIDE, []int{0, 1, 2, 4, 3}},
		{"milvang odd n=5", 5, CycleOrderMilvang, []int{4, 0, 1, 2, 3}},
		{"fide n=2 (one round, nothing to swap)", 2, CycleOrderFIDE, []int{0}},
		{"milvang n=2", 2, CycleOrderMilvang, []int{0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base, err := Berger(tc.n)
			if err != nil {
				t.Fatal(err)
			}
			sch, err := PairRoundRobin(rrPlayers(tc.n), 2, tc.order)
			if err != nil {
				t.Fatal(err)
			}
			r := len(base)
			if len(sch) != 2*r {
				t.Fatalf("got %d rounds, want %d", len(sch), 2*r)
			}
			for pos := 0; pos < r; pos++ {
				idx, same := tableIndex(t, base, tc.n, sch[pos])
				if idx != tc.want[pos] || !same {
					t.Errorf("cycle 1 round %d: table round %d (table colours %v), want %d with table colours", pos+1, idx, same, tc.want[pos])
				}
				idx, same = tableIndex(t, base, tc.n, sch[r+pos])
				if idx != pos || same {
					t.Errorf("cycle 2 round %d: table round %d (table colours %v), want %d with reversed colours", r+pos+1, idx, same, pos)
				}
				for _, g := range sch[pos] {
					if g.Round != pos+1 {
						t.Errorf("round numbering: got %d want %d", g.Round, pos+1)
					}
				}
			}
		})
	}
}

func TestPairRoundRobin_SingleCycleIgnoresOrder(t *testing.T) {
	for _, order := range []CycleOrder{"", CycleOrderRepeat, CycleOrderFIDE, CycleOrderMilvang} {
		base, _ := Berger(6)
		sch, err := PairRoundRobin(rrPlayers(6), 1, order)
		if err != nil {
			t.Fatal(err)
		}
		for pos := range sch {
			if idx, same := tableIndex(t, base, 6, sch[pos]); idx != pos || !same {
				t.Errorf("order %q: round %d is table round %d (table colours %v), want table order", order, pos+1, idx, same)
			}
		}
	}
}

func TestPairRoundRobin_ThreeCyclesReorderFirstOnly(t *testing.T) {
	base, _ := Berger(4)
	sch, err := PairRoundRobin(rrPlayers(4), 3, CycleOrderMilvang)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]int{{2, 0, 1}, {0, 1, 2}, {0, 1, 2}}
	wantSame := []bool{true, false, true}
	for c := 0; c < 3; c++ {
		for pos := 0; pos < 3; pos++ {
			idx, same := tableIndex(t, base, 4, sch[c*3+pos])
			if idx != want[c][pos] || same != wantSame[c] {
				t.Errorf("cycle %d round %d: table round %d (table colours %v), want %d (%v)", c+1, pos+1, idx, same, want[c][pos], wantSame[c])
			}
		}
	}
}

func TestPairRoundRobin_DoubleCycleColourRuns(t *testing.T) {
	// Counts only the games a player plays, so a free round does not break a run.
	stats := func(sch [][]round.Game[int]) (maxRun, maxRepeats int) {
		seq := map[int][]int{}
		for _, rd := range sch {
			for _, g := range rd {
				if g.Black == nil {
					continue
				}
				seq[g.White] = append(seq[g.White], 1)
				seq[*g.Black] = append(seq[*g.Black], -1)
			}
		}
		for _, cs := range seq {
			run, repeats := 1, 0
			for i := 1; i < len(cs); i++ {
				if cs[i] == cs[i-1] {
					run++
					repeats++
				} else {
					run = 1
				}
				maxRun = max(maxRun, run)
			}
			maxRepeats = max(maxRepeats, repeats)
		}
		return maxRun, maxRepeats
	}

	for n := 3; n <= 24; n++ {
		repeat, _ := PairRoundRobin(rrPlayers(n), 2, CycleOrderRepeat)
		fide, _ := PairRoundRobin(rrPlayers(n), 2, CycleOrderFIDE)
		milvang, _ := PairRoundRobin(rrPlayers(n), 2, CycleOrderMilvang)

		runRepeat, _ := stats(repeat)
		runFIDE, _ := stats(fide)
		runMilvang, repeatsMilvang := stats(milvang)

		// The problem the orders solve: played as it is, an even field has players
		// with the same colour three games in a row.
		if n%2 == 0 && runRepeat != 3 {
			t.Errorf("n=%d repeat: longest run = %d, want 3", n, runRepeat)
		}
		if runFIDE > 2 {
			t.Errorf("n=%d fide: longest run = %d, want <= 2", n, runFIDE)
		}
		if runMilvang > 2 {
			t.Errorf("n=%d milvang: longest run = %d, want <= 2", n, runMilvang)
		}
		if repeatsMilvang > 2 {
			t.Errorf("n=%d milvang: %d same-colour repeats for one player, want <= 2", n, repeatsMilvang)
		}

		// Every order is still a complete double round-robin.
		for name, sch := range map[string][][]round.Game[int]{"repeat": repeat, "fide": fide, "milvang": milvang} {
			meet := map[[2]int]int{}
			for _, rd := range sch {
				seen := map[int]bool{}
				for _, g := range rd {
					if seen[g.White] {
						t.Errorf("n=%d %s: player %d twice in a round", n, name, g.White)
					}
					seen[g.White] = true
					if g.Black == nil {
						continue
					}
					if seen[*g.Black] {
						t.Errorf("n=%d %s: player %d twice in a round", n, name, *g.Black)
					}
					seen[*g.Black] = true
					a, b := g.White, *g.Black
					if a > b {
						a, b = b, a
					}
					meet[[2]int{a, b}]++
				}
			}
			if len(meet) != n*(n-1)/2 {
				t.Errorf("n=%d %s: %d distinct pairs, want %d", n, name, len(meet), n*(n-1)/2)
			}
			for pair, c := range meet {
				if c != 2 {
					t.Errorf("n=%d %s: pair %v met %d times, want 2", n, name, pair, c)
				}
			}
		}
	}
}

func TestPairRoundRobin_InvalidCycleOrder(t *testing.T) {
	if _, err := PairRoundRobin(rrPlayers(4), 2, "bogus"); !errors.Is(err, ErrInvalidCycleOrder) {
		t.Errorf("expected ErrInvalidCycleOrder, got %v", err)
	}
	// An unknown order is rejected even when it would have no effect (one cycle).
	if _, err := PairRoundRobin(rrPlayers(4), 1, "bogus"); !errors.Is(err, ErrInvalidCycleOrder) {
		t.Errorf("expected ErrInvalidCycleOrder for one cycle, got %v", err)
	}
}
