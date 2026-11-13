package pairing

import (
	"fmt"

	"github.com/tessechess/core/round"
)

// BergerPair represents a single board pairing produced by the Berger function
// using 1-based pairing numbers (1..N). When the player count P is odd,
// N = P+1 represents the virtual bye slot.
type BergerPair struct {
	// White is the 1-based pairing number of the player with the white pieces.
	White int
	// Black is the 1-based pairing number of the player with the black pieces.
	Black int
}

// Berger computes the full single-cycle round-robin schedule for n players
// using the algorithmic Berger function (as used in Vega Chess and defined by
// FIDE Handbook C.05 Annex 1).
//
// Let N = n + (n % 2) be the even table size. The returned slice contains N-1
// rounds, each with N/2 board pairings using 1-based pairing numbers in 1..N.
// When n is odd, pairing number N = n+1 represents the virtual bye slot.
//
// If n < 2, Berger returns an error wrapping ErrNotEnoughPlayers.
func Berger(n int) ([][]BergerPair, error) {
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

	schedule := make([][]BergerPair, totalRounds)
	for r := 0; r < totalRounds; r++ {
		roundPairs := make([]BergerPair, pairs)

		firstPartner := shift(1, r)
		if (r+1)%2 == 1 {
			roundPairs[0] = BergerPair{White: firstPartner, Black: evenN}
		} else {
			roundPairs[0] = BergerPair{White: evenN, Black: firstPartner}
		}

		for b := 2; b <= pairs; b++ {
			roundPairs[b-1] = BergerPair{
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

// PairRoundRobin generates the full round-robin tournament schedule upfront
// across the specified number of cycles using the Berger function.
//
// Every player in players must have a unique 1-based StartingRank in
// 1..len(players) (see AssignStartingRanks). For cycles >= 2 (e.g. a double
// round-robin with cycles == 2), even-numbered cycles reverse colors on every
// played board relative to the first cycle.
//
// When len(players) is odd, the player paired against the virtual slot
// N = len(players)+1 receives a bye (White set to the player's ID, Black nil)
// placed on the last board (N/2), while played games are numbered sequentially
// starting at board 1. Player-level FixedBoard assignments are applied to the
// played boards of each round via ApplyFixedBoards.
//
// Errors can be tested with errors.Is: ErrNotEnoughPlayers (fewer than 2
// players), ErrInvalidCycles (cycles < 1), and ErrInvalidStartingRank (missing,
// out-of-range, or duplicate StartingRank).
func PairRoundRobin[ID comparable](players []Player[ID], cycles int) ([][]round.Game[ID], error) {
	p := len(players)
	if p < 2 {
		return nil, fmt.Errorf("%w: got %d", ErrNotEnoughPlayers, p)
	}
	if cycles < 1 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidCycles, cycles)
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

	baseCycle, err := Berger(p)
	if err != nil {
		return nil, err
	}

	roundsPerCycle := len(baseCycle)
	totalRounds := cycles * roundsPerCycle
	schedule := make([][]round.Game[ID], 0, totalRounds)

	for c := 0; c < cycles; c++ {
		reverseColors := c%2 == 1
		for rIdx, pairs := range baseCycle {
			roundNum := c*roundsPerCycle + rIdx + 1
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
						Round: roundNum,
						White: id,
						Black: nil,
					}
					continue
				}
				if bRank > p {
					// Virtual bye slot is Black; wRank is the real player receiving the bye.
					id := rankToID[wRank]
					byeGame = &round.Game[ID]{
						Round: roundNum,
						White: id,
						Black: nil,
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
