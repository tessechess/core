package pairing

import (
	"fmt"
	"math"
	"sort"

	"github.com/tessechess/core/round"
)

// Team is one confirmed team of a team section.
type Team[TeamID, PlayerID comparable] struct {
	ID           TeamID             // the caller's ID, used in History, Settings and the result
	StartingRank int                // pairing number; 0 = not assigned
	Name         string             // team name
	Federation   string             // 3-letter code, optional
	Players      []Player[PlayerID] // team roster (main boards and reserves)
	FixedTable   *int               // team-level fixed table; a round-level FixedBoard for the same team wins
}

// FixedTable pins a team to a table number. It is applied after a round of
// matches has been paired: the team's match moves to that table and the other
// tables are renumbered around it.
type FixedTable[TeamID comparable] struct {
	TableNumber int    // 1-based; must be unique within the round
	Team        TeamID // the team that is put on that table
}

// TeamRating returns the team's effective rating as the rounded average rating
// (0.5 rounds up) of the top boardsPerMatch highest-rated players in players.
// Unrated players (Rating <= 0) and missing board slots (when len(players) <
// boardsPerMatch) count as 0. If boardsPerMatch < 1 or players is empty,
// TeamRating returns 0. players is not modified.
func TeamRating[PlayerID comparable](players []Player[PlayerID], boardsPerMatch int) int {
	if boardsPerMatch < 1 || len(players) == 0 {
		return 0
	}
	ratings := make([]int, len(players))
	for i, p := range players {
		if p.Rating > 0 {
			ratings[i] = p.Rating
		}
	}
	sort.Slice(ratings, func(i, j int) bool {
		return ratings[i] > ratings[j]
	})

	limit := boardsPerMatch
	if len(ratings) < limit {
		limit = len(ratings)
	}
	var sum int
	for i := 0; i < limit; i++ {
		sum += ratings[i]
	}
	if sum == 0 {
		return 0
	}
	return int(math.Round(float64(sum) / float64(boardsPerMatch)))
}

// TeamStartingOrder returns the teams' IDs in starting-rank order (index + 1 is
// the starting rank), following FIDE initial ranking for team competitions:
//
//  1. team rating (TeamRating over boardsPerMatch), highest first; unrated
//     teams (rating 0) after all rated ones;
//  2. alphabetically by Name.
//
// Teams equal on both keep their input order. teams is not modified, and
// StartingRank is ignored (it is what this function decides).
func TeamStartingOrder[TeamID, PlayerID comparable](teams []Team[TeamID, PlayerID], boardsPerMatch int) []TeamID {
	type ratedTeam struct {
		team   Team[TeamID, PlayerID]
		rating int
	}
	sorted := make([]ratedTeam, len(teams))
	for i, t := range teams {
		sorted[i] = ratedTeam{
			team:   t,
			rating: TeamRating(t.Players, boardsPerMatch),
		}
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.rating != b.rating {
			return a.rating > b.rating
		}
		return a.team.Name < b.team.Name
	})

	ids := make([]TeamID, len(sorted))
	for i, rt := range sorted {
		ids[i] = rt.team.ID
	}
	return ids
}

// PairTeamRoundRobin generates the full team round-robin match schedule upfront
// across the specified number of cycles using the Berger function.
//
// The caller assigns the pairing numbers: every team in teams must have a
// unique 1-based StartingRank in 1..len(teams). For cycles >= 2, even-numbered
// cycles reverse Home/Away (and thus Board 1 colors) on every played table
// relative to the first cycle. Every paired match has WhiteOnFirstBoard set to
// true so that Home plays white on odd boards (1, 3, ...) and black on even
// boards (2, 4, ...).
//
// When len(teams) is odd, the team paired against the virtual slot
// N = len(teams)+1 has a free round: a bye match (Home set to the team's ID,
// Away nil) with Result result.Void on the last table. Team-level FixedTable
// assignments are applied to the played matches of each round via
// ApplyFixedTables.
//
// order chooses the order of the rounds when cycles >= 2 (see CycleOrder); the
// empty value means CycleOrderRepeat.
//
// Errors can be tested with errors.Is: ErrNotEnoughPlayers (fewer than 2
// teams), ErrInvalidCycles (cycles < 1), ErrInvalidCycleOrder (unknown order),
// and ErrInvalidStartingRank (missing, out-of-range, or duplicate StartingRank).
func PairTeamRoundRobin[TeamID, PlayerID comparable](
	teams []Team[TeamID, PlayerID],
	cycles int,
	order CycleOrder,
) ([][]round.Match[TeamID, PlayerID], error) {
	tCount := len(teams)
	if tCount < 2 {
		return nil, fmt.Errorf("%w: got %d", ErrNotEnoughPlayers, tCount)
	}
	if cycles < 1 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidCycles, cycles)
	}
	if !order.valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidCycleOrder, string(order))
	}

	rankToID := make(map[int]TeamID, tCount)
	var fixedTables []FixedTable[TeamID]
	for _, tm := range teams {
		if tm.StartingRank < 1 || tm.StartingRank > tCount {
			return nil, fmt.Errorf("%w: starting rank %d out of range 1..%d", ErrInvalidStartingRank, tm.StartingRank, tCount)
		}
		if _, exists := rankToID[tm.StartingRank]; exists {
			return nil, fmt.Errorf("%w: duplicate starting rank %d", ErrInvalidStartingRank, tm.StartingRank)
		}
		rankToID[tm.StartingRank] = tm.ID
		if tm.FixedTable != nil {
			fixedTables = append(fixedTables, FixedTable[TeamID]{
				TableNumber: *tm.FixedTable,
				Team:        tm.ID,
			})
		}
	}

	baseCycle, err := berger(tCount)
	if err != nil {
		return nil, err
	}

	roundsPerCycle := len(baseCycle)
	totalRounds := cycles * roundsPerCycle
	schedule := make([][]round.Match[TeamID, PlayerID], 0, totalRounds)

	for c := 0; c < cycles; c++ {
		reverseColors := c%2 == 1
		for pos, rIdx := range order.roundOrder(c, cycles, roundsPerCycle) {
			pairs := baseCycle[rIdx]
			roundNum := c*roundsPerCycle + pos + 1
			played := make([]round.Match[TeamID, PlayerID], 0, len(pairs))
			var byeMatch *round.Match[TeamID, PlayerID]

			for _, bp := range pairs {
				wRank, bRank := bp.White, bp.Black
				if reverseColors {
					wRank, bRank = bRank, wRank
				}

				if wRank > tCount {
					id := rankToID[bRank]
					byeMatch = &round.Match[TeamID, PlayerID]{
						Round:  roundNum,
						Home:   id,
						Away:   nil,
						Result: freeRoundResult(),
					}
					continue
				}
				if bRank > tCount {
					id := rankToID[wRank]
					byeMatch = &round.Match[TeamID, PlayerID]{
						Round:  roundNum,
						Home:   id,
						Away:   nil,
						Result: freeRoundResult(),
					}
					continue
				}

				homeID := rankToID[wRank]
				awayID := rankToID[bRank]
				played = append(played, round.Match[TeamID, PlayerID]{
					Round:             roundNum,
					Table:             len(played) + 1,
					Home:              homeID,
					Away:              &awayID,
					WhiteOnFirstBoard: true,
				})
			}

			if len(fixedTables) > 0 && len(played) > 0 {
				played = ApplyFixedTables(played, fixedTables)
			}

			if byeMatch != nil {
				byeTable := len(played) + 1
				if len(played) > 0 && played[len(played)-1].Table >= byeTable {
					byeTable = played[len(played)-1].Table + 1
				}
				byeMatch.Table = byeTable
				played = append(played, *byeMatch)
			}

			schedule = append(schedule, played)
		}
	}

	return schedule, nil
}

// ApplyFixedTables moves designated teams' matches to their fixed table
// numbers and renumbers all other tables sequentially around them, sorted by
// Table ascending.
func ApplyFixedTables[TeamID, PlayerID comparable](
	matches []round.Match[TeamID, PlayerID],
	fixedTables []FixedTable[TeamID],
) []round.Match[TeamID, PlayerID] {
	teamToTable := make(map[TeamID]int, len(fixedTables))
	for _, ft := range fixedTables {
		teamToTable[ft.Team] = ft.TableNumber
	}

	var fixed []round.Match[TeamID, PlayerID]
	var rest []round.Match[TeamID, PlayerID]
	for _, m := range matches {
		if _, ok := teamToTable[m.Home]; ok {
			fixed = append(fixed, m)
			continue
		}
		if m.Away != nil {
			if _, ok := teamToTable[*m.Away]; ok {
				fixed = append(fixed, m)
				continue
			}
		}
		rest = append(rest, m)
	}

	for i := range fixed {
		if table, ok := teamToTable[fixed[i].Home]; ok {
			fixed[i].Table = table
		} else if fixed[i].Away != nil {
			if table, ok := teamToTable[*fixed[i].Away]; ok {
				fixed[i].Table = table
			}
		}
	}

	taken := make(map[int]bool, len(fixed))
	for _, m := range fixed {
		taken[m.Table] = true
	}

	table := 1
	for i := range rest {
		for taken[table] {
			table++
		}
		rest[i].Table = table
		table++
	}

	all := append(fixed, rest...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Table < all[j].Table
	})
	return all
}

// PairMatchBoards populates Boards (1..boardsPerMatch) for every non-bye match
// in matches using the ordered player lineups in lineups.
//
// On board b (1-based), Home's player has white when (b%2 == 1) ==
// m.WhiteOnFirstBoard and black otherwise, matching Match.BoardColor. Bye
// matches (Away is nil) are copied unchanged without requiring a lineup.
// Neither matches nor lineups is modified.
//
// Errors can be tested with errors.Is: ErrInvalidBoardsPerMatch (boardsPerMatch
// < 1) and ErrInvalidLineup (a paired team is missing from lineups, its lineup
// does not have exactly boardsPerMatch players, or a player appears more than
// once in a match).
func PairMatchBoards[TeamID, PlayerID comparable](
	matches []round.Match[TeamID, PlayerID],
	lineups map[TeamID][]PlayerID,
	boardsPerMatch int,
) ([]round.Match[TeamID, PlayerID], error) {
	if boardsPerMatch < 1 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidBoardsPerMatch, boardsPerMatch)
	}

	out := make([]round.Match[TeamID, PlayerID], len(matches))
	for i, m := range matches {
		out[i] = m
		if m.IsBye() {
			continue
		}

		homeLineup, ok := lineups[m.Home]
		if !ok {
			return nil, fmt.Errorf("%w: missing lineup for home team %v", ErrInvalidLineup, m.Home)
		}
		if len(homeLineup) != boardsPerMatch {
			return nil, fmt.Errorf("%w: home team %v lineup has %d players, want %d", ErrInvalidLineup, m.Home, len(homeLineup), boardsPerMatch)
		}
		if hasDuplicates(homeLineup) {
			return nil, fmt.Errorf("%w: duplicate player in home team %v lineup", ErrInvalidLineup, m.Home)
		}

		awayTeam := *m.Away
		awayLineup, ok := lineups[awayTeam]
		if !ok {
			return nil, fmt.Errorf("%w: missing lineup for away team %v", ErrInvalidLineup, awayTeam)
		}
		if len(awayLineup) != boardsPerMatch {
			return nil, fmt.Errorf("%w: away team %v lineup has %d players, want %d", ErrInvalidLineup, awayTeam, len(awayLineup), boardsPerMatch)
		}
		if hasDuplicates(awayLineup) {
			return nil, fmt.Errorf("%w: duplicate player in away team %v lineup", ErrInvalidLineup, awayTeam)
		}

		homeSet := toSet(homeLineup)
		for _, p := range awayLineup {
			if homeSet[p] {
				return nil, fmt.Errorf("%w: player %v appears in both home and away lineups", ErrInvalidLineup, p)
			}
		}

		boards := make([]round.Game[PlayerID], boardsPerMatch)
		for b := 1; b <= boardsPerMatch; b++ {
			homePlayer := homeLineup[b-1]
			awayPlayer := awayLineup[b-1]
			homeIsWhite := (b%2 == 1) == m.WhiteOnFirstBoard
			if homeIsWhite {
				black := awayPlayer
				boards[b-1] = round.Game[PlayerID]{
					Round: m.Round,
					Board: b,
					White: homePlayer,
					Black: &black,
				}
			} else {
				black := homePlayer
				boards[b-1] = round.Game[PlayerID]{
					Round: m.Round,
					Board: b,
					White: awayPlayer,
					Black: &black,
				}
			}
		}
		out[i].Boards = boards
	}

	return out, nil
}
