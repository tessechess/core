package pairing

import (
	"fmt"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

// bergerPair represents a single board pairing produced by the Berger function
// using 1-based pairing numbers (1..N). When the player count P is odd,
// N = P+1 represents the virtual bye slot.
type bergerPair struct {
	// White is the 1-based pairing number of the player with the white pieces.
	White int
	// Black is the 1-based pairing number of the player with the black pieces.
	Black int
}

// berger computes the full single-cycle round-robin schedule for n players
// using the algorithmic Berger function (as used in Vega Chess and defined by
// FIDE Handbook C.05 Annex 1).
//
// Let N = n + (n % 2) be the even table size. The returned slice contains N-1
// rounds, each with N/2 board pairings using 1-based pairing numbers in 1..N.
// When n is odd, pairing number N = n+1 represents the virtual bye slot.
//
// If n < 2, berger returns an error wrapping ErrNotEnoughPlayers.
func berger(n int) ([][]bergerPair, error) {
	if n < 2 {
		return nil, fmt.Errorf("%w: got %d", ErrNotEnoughPlayers, n)
	}

	evenN := n + n%2
	pairs := evenN / 2
	totalRounds := evenN - 1
	mod := evenN - 1

	shift := func(x, roundIdx int) int {
		return (x-1+roundIdx*pairs)%mod + 1
	}

	schedule := make([][]bergerPair, totalRounds)
	for r := 0; r < totalRounds; r++ {
		roundPairs := make([]bergerPair, pairs)

		firstPartner := shift(1, r)
		if (r+1)%2 == 1 {
			roundPairs[0] = bergerPair{White: firstPartner, Black: evenN}
		} else {
			roundPairs[0] = bergerPair{White: evenN, Black: firstPartner}
		}

		for b := 2; b <= pairs; b++ {
			roundPairs[b-1] = bergerPair{
				White: shift(b, r),
				Black: shift(evenN-b+1, r),
			}
		}

		schedule[r] = roundPairs
	}

	return schedule, nil
}

// RoundRobinTotalRounds returns the total number of rounds for a round-robin
// tournament with playerCount players over the given number of cycles.
// For playerCount >= 2 and cycles >= 1, this is cycles * (N - 1), where
// N = playerCount + (playerCount % 2). Otherwise it returns 0.
func RoundRobinTotalRounds(playerCount, cycles int) int {
	if playerCount < 2 || cycles < 1 {
		return 0
	}
	evenN := playerCount + playerCount%2
	return cycles * (evenN - 1)
}

// CycleOrder selects in which order the rounds of the Berger table are played
// when a round-robin has more than one cycle (for example a double round-robin).
// Played as it is, a double round-robin can give a player the same colour in
// three consecutive games where the first cycle ends and the second begins; the
// orders below differ in how they deal with that. The order only changes which
// round is played when: every cycle still contains each pairing once, and the
// colours of a round do not depend on the order.
//
// The first cycle is the only one reordered, and only when cycles >= 2; a
// single round-robin always follows the table. Cycles after the first are
// played in table order, with the colours reversed in every even-numbered cycle
// (the second, the fourth, ...).
type CycleOrder string

const (
	// CycleOrderRepeat plays every cycle in table order (round 1, 2, ... ), with
	// the colours reversed in the second cycle. It is the default for the empty
	// value. Some players get three consecutive games with the same colour
	// where the cycles meet.
	CycleOrderRepeat CycleOrder = "repeat"

	// CycleOrderFIDE follows the recommendation of FIDE Handbook C.05 Annex 1:
	// the last two rounds of the first cycle are played in reverse order, "to
	// avoid three consecutive games with the same colour". The second cycle is
	// in table order with reversed colours.
	CycleOrderFIDE CycleOrder = "fide"

	// CycleOrderMilvang is the arrangement proposed by Otto Milvang in "Berger
	// explained": the first cycle starts with the last round of the table and
	// then plays rounds 1, 2, ...; the second cycle is in table order with
	// reversed colours. It keeps the regular structure of the table.
	CycleOrderMilvang CycleOrder = "milvang"
)

// valid reports whether o is one of the defined orders or empty.
func (o CycleOrder) valid() bool {
	switch o {
	case "", CycleOrderRepeat, CycleOrderFIDE, CycleOrderMilvang:
		return true
	}
	return false
}

// roundOrder returns the table rounds (0-based indexes into the single-cycle
// schedule) in the order they are played in the given cycle (0-based).
func (o CycleOrder) roundOrder(cycle, cycles, rounds int) []int {
	order := make([]int, rounds)
	for i := range order {
		order[i] = i
	}
	if cycle != 0 || cycles < 2 {
		return order
	}
	switch o {
	case CycleOrderFIDE:
		if rounds >= 2 {
			order[rounds-2], order[rounds-1] = order[rounds-1], order[rounds-2]
		}
	case CycleOrderMilvang:
		for i := 1; i < rounds; i++ {
			order[i] = i - 1
		}
		order[0] = rounds - 1
	}
	return order
}

// freeRoundResult returns a fresh result.Void for the free round of an odd-sized round-robin.
func freeRoundResult() *result.Code {
	code := result.Void
	return &code
}

// PairRoundRobin generates the full round-robin tournament schedule upfront
// across the specified number of cycles using the Berger function.
//
// The caller assigns the pairing numbers: every player in players must have a
// unique 1-based StartingRank in 1..len(players) (round-robin numbers are
// usually drawn by lot, so nothing is assigned here). For cycles >= 2 (e.g. a
// double round-robin with cycles == 2), even-numbered cycles reverse colors on
// every played board relative to the first cycle.
//
// When len(players) is odd, the player paired against the virtual slot
// N = len(players)+1 has a free round: a bye game (White set to the player's
// ID, Black nil) with Result result.Void. A free round in a round-robin scores
// nothing (unlike the full-point bye of a Swiss round), and Void keeps it out of
// all calculations The bye is placed on the last board (N/2), while played games
// are numbered sequentially starting at board 1. Player-level FixedBoard assignments
// are applied to the played boards of each round via ApplyFixedBoards.
//
// order chooses the order of the rounds when cycles >= 2 (see CycleOrder); the
// empty value means CycleOrderRepeat.
//
// Errors can be tested with errors.Is: ErrNotEnoughPlayers (fewer than 2
// players), ErrInvalidCycles (cycles < 1), ErrInvalidCycleOrder (unknown
// order), and ErrInvalidStartingRank (missing, out-of-range, or duplicate
// StartingRank).
func PairRoundRobin[ID comparable](players []Player[ID], cycles int, order CycleOrder) ([][]round.Game[ID], error) {
	p := len(players)
	if p < 2 {
		return nil, fmt.Errorf("%w: got %d", ErrNotEnoughPlayers, p)
	}
	if cycles < 1 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidCycles, cycles)
	}
	if !order.valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidCycleOrder, string(order))
	}

	rankToID := make(map[int]ID, p)
	var fixedBoards []FixedBoard[ID]
	for _, pl := range players {
		if pl.StartingRank < 1 || pl.StartingRank > p {
			return nil, fmt.Errorf("%w: starting rank %d out of range 1..%d", ErrInvalidStartingRank, pl.StartingRank, p)
		}
		if _, exists := rankToID[pl.StartingRank]; exists {
			return nil, fmt.Errorf("%w: duplicate starting rank %d", ErrInvalidStartingRank, pl.StartingRank)
		}
		rankToID[pl.StartingRank] = pl.ID
		if pl.FixedBoard != nil {
			fixedBoards = append(fixedBoards, FixedBoard[ID]{
				BoardNumber: *pl.FixedBoard,
				Player:      pl.ID,
			})
		}
	}

	baseCycle, err := berger(p)
	if err != nil {
		return nil, err
	}

	roundsPerCycle := len(baseCycle)
	totalRounds := cycles * roundsPerCycle
	schedule := make([][]round.Game[ID], 0, totalRounds)

	for c := 0; c < cycles; c++ {
		reverseColors := c%2 == 1
		for pos, rIdx := range order.roundOrder(c, cycles, roundsPerCycle) {
			pairs := baseCycle[rIdx]
			roundNum := c*roundsPerCycle + pos + 1
			played := make([]round.Game[ID], 0, len(pairs))
			var byeGame *round.Game[ID]

			for _, bp := range pairs {
				wRank, bRank := bp.White, bp.Black
				if reverseColors {
					wRank, bRank = bRank, wRank
				}

				if wRank > p {
					// Virtual bye slot is White; bRank is the real player receiving the bye.
					id := rankToID[bRank]
					byeGame = &round.Game[ID]{
						Round:  roundNum,
						White:  id,
						Black:  nil,
						Result: freeRoundResult(),
					}
					continue
				}
				if bRank > p {
					// Virtual bye slot is Black; wRank is the real player receiving the bye.
					id := rankToID[wRank]
					byeGame = &round.Game[ID]{
						Round:  roundNum,
						White:  id,
						Black:  nil,
						Result: freeRoundResult(),
					}
					continue
				}

				whiteID := rankToID[wRank]
				blackID := rankToID[bRank]
				played = append(played, round.Game[ID]{
					Round: roundNum,
					Board: len(played) + 1,
					White: whiteID,
					Black: &blackID,
				})
			}

			if len(fixedBoards) > 0 && len(played) > 0 {
				played = ApplyFixedBoards(played, fixedBoards)
			}

			if byeGame != nil {
				byeBoard := len(played) + 1
				if len(played) > 0 && played[len(played)-1].Board >= byeBoard {
					byeBoard = played[len(played)-1].Board + 1
				}
				byeGame.Board = byeBoard
				played = append(played, *byeGame)
			}

			schedule = append(schedule, played)
		}
	}

	return schedule, nil
}
