package pairing

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
)

// varmaTables holds the FIDE Handbook C.05 Annex 2 Varma tables indexed by
// even table size N in {10, 12, 14, 16, 18, 20, 22, 24}.
var varmaTables = map[int][4][]int{
	10: {
		{3, 4, 8},
		{5, 7, 9},
		{1, 6},
		{2, 10},
	},
	12: {
		{4, 5, 9, 10},
		{1, 2, 7},
		{6, 8, 12},
		{3, 11},
	},
	14: {
		{4, 5, 6, 11, 12},
		{1, 2, 8, 9},
		{7, 10, 13},
		{3, 14},
	},
	16: {
		{5, 6, 7, 12, 13, 14},
		{1, 2, 3, 9, 10},
		{8, 11, 15},
		{4, 16},
	},
	18: {
		{5, 6, 7, 8, 14, 15, 16},
		{1, 2, 3, 10, 11, 12},
		{9, 13, 17},
		{4, 18},
	},
	20: {
		{6, 7, 8, 9, 15, 16, 17, 18},
		{1, 2, 3, 11, 12, 13, 14},
		{5, 10, 19},
		{4, 20},
	},
	22: {
		{6, 7, 8, 9, 10, 17, 18, 19, 20},
		{1, 2, 3, 4, 12, 13, 14, 15},
		{11, 16, 21},
		{5, 22},
	},
	24: {
		{6, 7, 8, 9, 10, 11, 19, 20, 21, 22},
		{1, 2, 3, 4, 13, 14, 15, 16, 17},
		{12, 18, 23},
		{5, 24},
	},
}

// VarmaTable returns the four FIDE Varma groups (A, B, C, D) of 1-based
// Berger pairing numbers for playerCount players (9 <= playerCount <= 24),
// as defined in FIDE Handbook C.05 Annex 2.
//
// When playerCount is odd, the virtual bye slot (playerCount+1) is omitted so
// the returned groups partition 1..playerCount. Returned slices are fresh copies.
// Returns ErrVarmaUnsupportedPlayerCount if playerCount < 9 or playerCount > 24.
func VarmaTable(playerCount int) ([4][]int, error) {
	var out [4][]int
	if playerCount < 9 || playerCount > 24 {
		return out, fmt.Errorf("%w: got %d", ErrVarmaUnsupportedPlayerCount, playerCount)
	}

	evenN := playerCount + playerCount%2
	base := varmaTables[evenN]
	for i, group := range base {
		cp := make([]int, 0, len(group))
		for _, num := range group {
			if num <= playerCount {
				cp = append(cp, num)
			}
		}
		out[i] = cp
	}
	return out, nil
}

type varmaContingent[ID comparable] struct {
	fed     string
	players []Player[ID]
}

// VarmaOrder assigns 1-based Berger pairing numbers (index + 1 is the pairing
// number) using FIDE's restricted drawing of lots (C.05 Annex 2 / C.06) so
// players of the same federation do not meet in the last three rounds, if
// possible.
//
// Input players must have 9..24 unique IDs; StartingRank on input is ignored.
// Players with an empty Federation are treated as unaffiliated singletons.
// Contingent and player draw order follows FIDE C.06: largest contingents
// first, ties broken alphabetically by normalized FIDE country code, and players
// within a contingent alphabetically by "LastName, FirstName".
//
// rng must be non-nil and is used for choosing among valid envelopes and drawing
// numbers within an envelope.
func VarmaOrder[ID comparable](players []Player[ID], rng *rand.Rand) ([]ID, error) {
	if rng == nil {
		return nil, ErrNilRand
	}
	n := len(players)
	if n < 9 || n > 24 {
		return nil, fmt.Errorf("%w: got %d", ErrVarmaUnsupportedPlayerCount, n)
	}

	seen := make(map[ID]bool, n)
	byFed := make(map[string][]Player[ID])
	var singles []varmaContingent[ID]

	for _, p := range players {
		if seen[p.ID] {
			return nil, fmt.Errorf("%w: %v", ErrDuplicatePlayerID, p.ID)
		}
		seen[p.ID] = true

		fed := strings.ToUpper(strings.TrimSpace(p.Federation))
		if fed == "" {
			singles = append(singles, varmaContingent[ID]{
				fed:     "",
				players: []Player[ID]{p},
			})
			continue
		}
		byFed[fed] = append(byFed[fed], p)
	}

	contingents := make([]varmaContingent[ID], 0, len(byFed)+len(singles))
	for fed, ps := range byFed {
		sort.SliceStable(ps, func(i, j int) bool {
			return ps[i].LastName+", "+ps[i].FirstName < ps[j].LastName+", "+ps[j].FirstName
		})
		contingents = append(contingents, varmaContingent[ID]{
			fed:     fed,
			players: ps,
		})
	}
	contingents = append(contingents, singles...)

	sort.SliceStable(contingents, func(i, j int) bool {
		a, b := contingents[i], contingents[j]
		if len(a.players) != len(b.players) {
			return len(a.players) > len(b.players)
		}
		if (a.fed == "") != (b.fed == "") {
			return a.fed != ""
		}
		return a.fed < b.fed
	})

	pools, err := VarmaTable(n)
	if err != nil {
		return nil, err
	}
	conflicts, err := lastThreeConflicts(n)
	if err != nil {
		return nil, err
	}

	var multi []varmaContingent[ID]
	var singleContingents []varmaContingent[ID]
	for _, c := range contingents {
		if len(c.players) >= 2 {
			multi = append(multi, c)
		} else {
			singleContingents = append(singleContingents, c)
		}
	}

	result := make([]ID, n)

	var initCaps [4]int
	var indepCaps [4]int
	for e := range pools {
		initCaps[e] = len(pools[e])
		indepCaps[e] = maxConflictFreeInPool(pools[e], conflicts)
	}

	envAssign, ok := solveEnvelopes(multi, initCaps, indepCaps, true, rng)
	if !ok {
		envAssign, ok = solveEnvelopes(multi, initCaps, indepCaps, false, rng)
	}
	if ok {
		for i, c := range multi {
			e := envAssign[i]
			var placed []int
			for _, p := range c.players {
				num := drawMinConflictFromPool(&pools[e], placed, conflicts, rng)
				placed = append(placed, num)
				result[num-1] = p.ID
			}
		}
	} else {
		// Fallback ("if possible" per FIDE C.06): contingents cannot all fit
		// unsplit into the four envelopes. Place as much of each contingent
		// into an envelope as fits, then assign remaining players to numbers
		// minimizing last-3-round conflicts with same-federation players.
		for _, c := range multi {
			var placed []int
			rem := len(c.players)
			idx := 0

			// If an envelope fits the entire contingent, pick among tightest fits.
			fitEnv := chooseBestFitEnvelope(pools, rem, rng)
			if fitEnv >= 0 {
				for idx < len(c.players) {
					num := drawMinConflictFromPool(&pools[fitEnv], placed, conflicts, rng)
					placed = append(placed, num)
					result[num-1] = c.players[idx].ID
					idx++
				}
				continue
			}

			// Contingent must be split: first fill the largest available envelope.
			largestEnv := chooseLargestEnvelope(pools, rng)
			if largestEnv >= 0 {
				take := min(len(pools[largestEnv]), len(c.players))
				for k := 0; k < take; k++ {
					num := drawMinConflictFromPool(&pools[largestEnv], placed, conflicts, rng)
					placed = append(placed, num)
					result[num-1] = c.players[idx].ID
					idx++
				}
			}

			// Assign remaining players of this contingent minimizing last-3-round conflicts.
			for idx < len(c.players) {
				num := drawMinConflictAcrossPools(&pools, placed, conflicts, rng)
				placed = append(placed, num)
				result[num-1] = c.players[idx].ID
				idx++
			}
		}
	}

	// Remaining slots are drawn at random by size-1 contingents.
	var remaining []int
	for e := range pools {
		remaining = append(remaining, pools[e]...)
	}
	rng.Shuffle(len(remaining), func(i, j int) {
		remaining[i], remaining[j] = remaining[j], remaining[i]
	})
	for i, c := range singleContingents {
		result[remaining[i]-1] = c.players[0].ID
	}

	return result, nil
}

// lastThreeConflicts returns a symmetric 1-based adjacency matrix where
// conflicts[u][v] is true iff pairing numbers u and v meet in the last three
// rounds of the Berger table for n players.
func lastThreeConflicts(n int) ([][]bool, error) {
	sched, err := berger(n)
	if err != nil {
		return nil, err
	}
	conflicts := make([][]bool, n+1)
	for i := range conflicts {
		conflicts[i] = make([]bool, n+1)
	}
	start := max(0, len(sched)-3)
	for _, rd := range sched[start:] {
		for _, bp := range rd {
			if bp.White <= n && bp.Black <= n {
				conflicts[bp.White][bp.Black] = true
				conflicts[bp.Black][bp.White] = true
			}
		}
	}
	return conflicts, nil
}

type memoKey struct {
	idx  int
	caps [4]int
}

// maxConflictFreeInPool returns len(pool) if no two numbers in pool conflict in
// the last three rounds, or the largest conflict-free subset size otherwise.
func maxConflictFreeInPool(pool []int, conflicts [][]bool) int {
	hasEdge := false
	for i := 0; i < len(pool) && !hasEdge; i++ {
		for j := i + 1; j < len(pool); j++ {
			if conflicts[pool[i]][pool[j]] {
				hasEdge = true
				break
			}
		}
	}
	if !hasEdge {
		return len(pool)
	}
	// Backtrack maximum independent set (pool sizes are <= 10).
	best := 0
	var cur []int
	var search func(idx int)
	search = func(idx int) {
		if len(cur)+(len(pool)-idx) <= best {
			return
		}
		if idx == len(pool) {
			best = len(cur)
			return
		}
		cand := pool[idx]
		ok := true
		for _, u := range cur {
			if conflicts[cand][u] {
				ok = false
				break
			}
		}
		if ok {
			cur = append(cur, cand)
			search(idx + 1)
			cur = cur[:len(cur)-1]
		}
		search(idx + 1)
	}
	search(0)
	return best
}

// solveEnvelopes searches for an assignment of each multi-player contingent to
// one of the four envelopes without exceeding capacities. When requireIndep is
// true, a contingent can only enter an envelope if it fits within that
// envelope's conflict-free capacity as a single contingent (and total envelope
// capacity is not exceeded). Candidate envelopes are explored in randomized
// order via rng.
func solveEnvelopes[ID comparable](multi []varmaContingent[ID], initCaps, indepCaps [4]int, requireIndep bool, rng *rand.Rand) ([]int, bool) {
	assign := make([]int, len(multi))
	failed := make(map[memoKey]bool)

	var dfs func(idx int, caps [4]int) bool
	dfs = func(idx int, caps [4]int) bool {
		if idx == len(multi) {
			return true
		}
		key := memoKey{idx: idx, caps: caps}
		if failed[key] {
			return false
		}

		need := len(multi[idx].players)
		order := [4]int{0, 1, 2, 3}
		rng.Shuffle(4, func(i, j int) {
			order[i], order[j] = order[j], order[i]
		})

		for _, e := range order {
			if caps[e] < need {
				continue
			}
			if requireIndep && indepCaps[e] < need {
				continue
			}
			next := caps
			next[e] -= need
			assign[idx] = e
			if dfs(idx+1, next) {
				return true
			}
		}

		failed[key] = true
		return false
	}

	if !dfs(0, initCaps) {
		return nil, false
	}
	return assign, true
}

func conflictCount(candidate int, placed []int, conflicts [][]bool) int {
	cnt := 0
	for _, u := range placed {
		if conflicts[candidate][u] {
			cnt++
		}
	}
	return cnt
}

func drawMinConflictFromPool(pool *[]int, placed []int, conflicts [][]bool, rng *rand.Rand) int {
	p := *pool
	minC := len(placed) + 1
	var bestIdxs []int
	for i, cand := range p {
		c := conflictCount(cand, placed, conflicts)
		if c < minC {
			minC = c
			bestIdxs = bestIdxs[:0]
			bestIdxs = append(bestIdxs, i)
		} else if c == minC {
			bestIdxs = append(bestIdxs, i)
		}
	}
	pickIdx := bestIdxs[rng.IntN(len(bestIdxs))]
	chosen := p[pickIdx]
	*pool = append(p[:pickIdx], p[pickIdx+1:]...)
	return chosen
}

func drawMinConflictAcrossPools(pools *[4][]int, placed []int, conflicts [][]bool, rng *rand.Rand) int {
	type loc struct {
		env int
		idx int
	}
	minC := len(placed) + 1
	var best []loc
	for e := range pools {
		for i, cand := range pools[e] {
			c := conflictCount(cand, placed, conflicts)
			if c < minC {
				minC = c
				best = best[:0]
				best = append(best, loc{env: e, idx: i})
			} else if c == minC {
				best = append(best, loc{env: e, idx: i})
			}
		}
	}
	chosenLoc := best[rng.IntN(len(best))]
	p := pools[chosenLoc.env]
	chosen := p[chosenLoc.idx]
	pools[chosenLoc.env] = append(p[:chosenLoc.idx], p[chosenLoc.idx+1:]...)
	return chosen
}

func chooseBestFitEnvelope(pools [4][]int, need int, rng *rand.Rand) int {
	bestSize := -1
	var cands []int
	for e, p := range pools {
		if len(p) < need {
			continue
		}
		if bestSize == -1 || len(p) < bestSize {
			bestSize = len(p)
			cands = cands[:0]
			cands = append(cands, e)
		} else if len(p) == bestSize {
			cands = append(cands, e)
		}
	}
	if len(cands) == 0 {
		return -1
	}
	return cands[rng.IntN(len(cands))]
}

func chooseLargestEnvelope(pools [4][]int, rng *rand.Rand) int {
	bestSize := 0
	var cands []int
	for e, p := range pools {
		if len(p) == 0 {
			continue
		}
		if len(p) > bestSize {
			bestSize = len(p)
			cands = cands[:0]
			cands = append(cands, e)
		} else if len(p) == bestSize {
			cands = append(cands, e)
		}
	}
	if len(cands) == 0 {
		return -1
	}
	return cands[rng.IntN(len(cands))]
}
