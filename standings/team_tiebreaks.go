package standings

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

// computeTeamTiebreak returns one team's value for a per-team tiebreak and
// whether the key is handled. Group-dependent keys (DirectEncounter,
// TopBoardResults), individual-only keys, and unknown keys return (0, false);
// callers store 0 for them and RankTeams fills in group-dependent keys via
// fillTeamGroupTiebreaks.
func computeTeamTiebreak[TeamID, PlayerID comparable](
	cfg TiebreakConfig,
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	sc resolvedTeamScoring,
) (float64, bool) {
	switch cfg.Key {
	case MatchPoints:
		return totals[teamID].mp, true
	case GamePoints:
		return totals[teamID].gp, true
	case Buchholz:
		return computeTeamBuchholz(teamID, allInfo, totals, sc, cfg.CutLowest, cfg.CutHighest), true
	case ForeBuchholz:
		return computeTeamForeBuchholz(teamID, allInfo, totals, sc, cfg.CutLowest, cfg.CutHighest), true
	case AverageOpponentBuchholz:
		return computeTeamAverageOpponentBuchholz(teamID, allInfo, totals, sc, cfg.CutLowest, cfg.CutHighest), true
	case SonnebornBerger:
		oppGame := sc.primary == TeamScoreGamePoints
		return computeTeamExtendedSB(teamID, allInfo, totals, sc, oppGame, oppGame), true
	case ExtendedMatchMatchSB:
		return computeTeamExtendedSB(teamID, allInfo, totals, sc, false, false), true
	case ExtendedMatchGameSB:
		return computeTeamExtendedSB(teamID, allInfo, totals, sc, false, true), true
	case ExtendedGameMatchSB:
		return computeTeamExtendedSB(teamID, allInfo, totals, sc, true, false), true
	case ExtendedGameGameSB:
		return computeTeamExtendedSB(teamID, allInfo, totals, sc, true, true), true
	case KoyaSystem:
		return computeTeamKoyaSystem(teamID, allInfo, totals, sc, cfg.Threshold), true
	case Wins:
		return computeTeamWins(teamID, allInfo, sc), true
	case GamesWon:
		return computeTeamMatchesWon(teamID, allInfo, sc), true
	case ProgressiveScores:
		return computeTeamProgressiveScores(teamID, allInfo, sc), true
	case GamesElectedToPlay:
		return computeTeamMatchesElectedToPlay(teamID, allInfo), true
	case BoardCount:
		return computeTeamBoardCount(teamID, allInfo, sc), true
	}
	return 0, false
}

// fillTeamGroupTiebreaks evaluates group-dependent team tiebreaks
// (DirectEncounter, TopBoardResults) in tiebreak order, grouping teams that
// remain tied on Points and all preceding tiebreaks (FIDE C.07 Article 5.2).
func fillTeamGroupTiebreaks[TeamID, PlayerID comparable](
	scores []*TeamScore[TeamID],
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	sc resolvedTeamScoring,
	tiebreakCfgs []TiebreakConfig,
) {
	for idx, cfg := range tiebreakCfgs {
		if cfg.Key != DirectEncounter && cfg.Key != TopBoardResults {
			continue
		}
		mapKey := cfg.MapKey()
		groups := groupTiedTeams(scores, tiebreakCfgs[:idx])
		for _, group := range groups {
			if len(group) < 2 {
				if len(group) == 1 {
					group[0].Tiebreaks[mapKey] = 0
				}
				continue
			}
			ids := make([]TeamID, len(group))
			for i, ts := range group {
				ids[i] = ts.ID
			}
			var vals map[TeamID]float64
			if cfg.Key == DirectEncounter {
				vals = computeTeamDirectEncounter(ids, allInfo, sc)
			} else {
				vals = computeTeamTopBoardResults(ids, allInfo, sc)
			}
			for _, ts := range group {
				ts.Tiebreaks[mapKey] = vals[ts.ID]
			}
		}
	}
}

func groupTiedTeams[TeamID comparable](scores []*TeamScore[TeamID], prior []TiebreakConfig) [][]*TeamScore[TeamID] {
	byKey := make(map[string][]*TeamScore[TeamID])
	var order []string
	for _, ts := range scores {
		var sb strings.Builder
		fmt.Fprintf(&sb, "%g", ts.Points)
		for _, cfg := range prior {
			fmt.Fprintf(&sb, "|%g", ts.Tiebreaks[cfg.MapKey()])
		}
		k := sb.String()
		if _, exists := byKey[k]; !exists {
			order = append(order, k)
		}
		byKey[k] = append(byKey[k], ts)
	}
	groups := make([][]*TeamScore[TeamID], len(order))
	for i, k := range order {
		groups[i] = byKey[k]
	}
	return groups
}

// computeTeamDirectEncounter computes Direct Encounter (DE) among a group of
// tied teams using the tournament's primary score (FIDE C.07 Article 6).
// Forfeited matches are excluded (Art. 6.1.1), and multiple encounters between
// the same two teams are averaged (Art. 6.1.2).
func computeTeamDirectEncounter[TeamID, PlayerID comparable](
	tiedTeamIDs []TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	sc resolvedTeamScoring,
) map[TeamID]float64 {
	tiedSet := make(map[TeamID]bool, len(tiedTeamIDs))
	for _, id := range tiedTeamIDs {
		tiedSet[id] = true
	}

	out := make(map[TeamID]float64, len(tiedTeamIDs))
	for _, teamID := range tiedTeamIDs {
		info := allInfo[teamID]
		if info == nil {
			continue
		}

		oppScores := make(map[TeamID][]float64)
		for _, m := range info.matches {
			if m.Away == nil {
				continue
			}
			var oppID TeamID
			if m.Home == teamID {
				oppID = *m.Away
			} else {
				oppID = m.Home
			}
			if !tiedSet[oppID] || isMatchForfeit(m) {
				continue
			}
			hMP, aMP, hGP, aGP, eff := matchTeamPoints(m, sc)
			if !eff.Counts() {
				continue
			}
			var pts float64
			if m.Home == teamID {
				pts = teamRawTotals{mp: hMP, gp: hGP}.primary(sc.primary)
			} else {
				pts = teamRawTotals{mp: aMP, gp: aGP}.primary(sc.primary)
			}
			oppScores[oppID] = append(oppScores[oppID], pts)
		}

		var total float64
		for _, games := range oppScores {
			var sum float64
			for _, g := range games {
				sum += g
			}
			total += sum / float64(len(games))
		}
		out[teamID] = total
	}
	return out
}

// computeTeamTopBoardResults computes Top Board Results / Board-by-Board
// Elimination (TBR / BBE, FIDE C.07 Article 11.3) for a group of tied teams.
//
// Tied teams are compared by total game points on board 1; teams still tied on
// board 1 are compared on board 2, and so on down to sc.boardsPerMatch.
// When a single board b resolves all differences in the tied group, each team's
// score on board b is returned directly; when multiple boards are needed to
// separate a 3+ team group, descending lexicographic scores are returned so
// that RankTeams orders the group strictly according to FIDE Article 11.3.
func computeTeamTopBoardResults[TeamID, PlayerID comparable](
	tiedTeamIDs []TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	sc resolvedTeamScoring,
) map[TeamID]float64 {
	boardsCount := sc.boardsPerMatch
	if boardsCount < 1 {
		boardsCount = maxBoardInMatches(allInfo)
	}
	if boardsCount < 1 {
		return make(map[TeamID]float64, len(tiedTeamIDs))
	}

	vectors := make(map[TeamID][]float64, len(tiedTeamIDs))
	for _, id := range tiedTeamIDs {
		vectors[id] = teamBoardTotals(id, allInfo[id], boardsCount, sc)
	}

	// Find the first board index where not all tied teams have the same total.
	firstDiff := -1
	for b := 0; b < boardsCount; b++ {
		first := vectors[tiedTeamIDs[0]][b]
		for _, id := range tiedTeamIDs[1:] {
			if vectors[id][b] != first {
				firstDiff = b
				break
			}
		}
		if firstDiff != -1 {
			break
		}
	}

	out := make(map[TeamID]float64, len(tiedTeamIDs))
	if firstDiff == -1 {
		// All teams in the group are identical on every board.
		for _, id := range tiedTeamIDs {
			out[id] = vectors[id][0]
		}
		return out
	}

	// Check whether any two teams that tie on firstDiff differ on a later board.
	needsMultiBoard := false
	for i := 0; i < len(tiedTeamIDs) && !needsMultiBoard; i++ {
		for j := i + 1; j < len(tiedTeamIDs); j++ {
			v1, v2 := vectors[tiedTeamIDs[i]], vectors[tiedTeamIDs[j]]
			if v1[firstDiff] == v2[firstDiff] && !equalFloatSlice(v1[firstDiff+1:], v2[firstDiff+1:]) {
				needsMultiBoard = true
				break
			}
		}
	}

	if !needsMultiBoard {
		for _, id := range tiedTeamIDs {
			out[id] = vectors[id][firstDiff]
		}
		return out
	}

	// Multi-board lexicographical ordering across the tied group.
	sorted := append([]TeamID(nil), tiedTeamIDs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareFloatSlicesDesc(vectors[sorted[i]], vectors[sorted[j]]) < 0
	})

	score := float64(len(sorted))
	for i, id := range sorted {
		if i > 0 && compareFloatSlicesDesc(vectors[sorted[i-1]], vectors[id]) != 0 {
			score = float64(len(sorted) - i)
		}
		out[id] = score
	}
	return out
}

func teamBoardTotals[TeamID, PlayerID comparable](
	teamID TeamID,
	info *teamPairingInfo[TeamID, PlayerID],
	boardsCount int,
	sc resolvedTeamScoring,
) []float64 {
	totals := make([]float64, boardsCount)
	if info == nil {
		return totals
	}
	for _, m := range info.matches {
		if m.Away == nil {
			continue
		}
		if m.Result != nil && *m.Result == result.Void {
			continue
		}
		isHome := m.Home == teamID
		if len(m.Boards) == 0 && m.Result != nil && m.Result.Counts() {
			// Match-level result without individual boards: distribute across boards.
			hMP, aMP, _, _, _ := matchTeamPoints(m, sc)
			var boardPts float64
			if (isHome && hMP > aMP) || (!isHome && aMP > hMP) {
				boardPts = 1.0
			} else if hMP == aMP && hMP > 0 {
				boardPts = 0.5
			}
			for b := 0; b < boardsCount; b++ {
				totals[b] += boardPts
			}
			continue
		}
		for i, g := range m.Boards {
			if g.IsBye() || g.Result == nil || !g.Result.Counts() {
				continue
			}
			bNum := g.Board
			if bNum < 1 {
				bNum = i + 1
			}
			if bNum > boardsCount {
				continue
			}
			wPts, bPts := g.Result.Points()
			homeIsWhite := (bNum%2 == 1) == m.WhiteOnFirstBoard
			if isHome == homeIsWhite {
				totals[bNum-1] += wPts
			} else {
				totals[bNum-1] += bPts
			}
		}
	}
	return totals
}

func maxBoardInMatches[TeamID, PlayerID comparable](allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID]) int {
	maxB := 0
	for _, info := range allInfo {
		for _, m := range info.matches {
			if len(m.Boards) > maxB {
				maxB = len(m.Boards)
			}
			for _, g := range m.Boards {
				if g.Board > maxB {
					maxB = g.Board
				}
			}
		}
	}
	return maxB
}

func equalFloatSlice(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compareFloatSlicesDesc(a, b []float64) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] > b[i] {
			return -1
		}
		if a[i] < b[i] {
			return 1
		}
	}
	return 0
}

// computeTeamBoardCount computes the Berlin system Board Count tiebreak (BC,
// FIDE C.07 Article 11.2): on a B-board match, board b (1..B) has weight
// B - b + 1 multiplied by the game points scored on that board.
func computeTeamBoardCount[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	sc resolvedTeamScoring,
) float64 {
	boardsCount := sc.boardsPerMatch
	if boardsCount < 1 {
		boardsCount = maxBoardInMatches(allInfo)
	}
	if boardsCount < 1 {
		return 0
	}
	perBoard := teamBoardTotals(teamID, allInfo[teamID], boardsCount, sc)
	var total float64
	for i, pts := range perBoard {
		weight := float64(boardsCount - i)
		total += weight * pts
	}
	return total
}

// computeTeamExtendedSB computes Extended Team Sonneborn-Berger (FIDE C.07
// Article 13):
//   - oppUseGP=false, ownUseGP=false: EMMSB (opponent MP × match MP)
//   - oppUseGP=false, ownUseGP=true:  EMGSB (opponent MP × match GP)
//   - oppUseGP=true,  ownUseGP=false: EGMSB (opponent GP × match MP)
//   - oppUseGP=true,  ownUseGP=true:  EGGSB (opponent GP × match GP)
func computeTeamExtendedSB[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	sc resolvedTeamScoring,
	oppUseGP, ownUseGP bool,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}

	ownTot := totals[teamID]
	ownVirtualOpp := ownTot.mp
	if oppUseGP {
		ownVirtualOpp = ownTot.gp
	}

	var total float64
	for _, m := range info.matches {
		hMP, aMP, hGP, aGP, eff := matchTeamPoints(m, sc)
		if !eff.Counts() {
			continue
		}

		isHome := m.Home == teamID
		var matchScore float64
		if isHome {
			if ownUseGP {
				matchScore = hGP
			} else {
				matchScore = hMP
			}
		} else {
			if ownUseGP {
				matchScore = aGP
			} else {
				matchScore = aMP
			}
		}

		if m.Away == nil {
			total += ownVirtualOpp * matchScore
			continue
		}

		var oppID TeamID
		if isHome {
			oppID = *m.Away
		} else {
			oppID = m.Home
		}
		adjOpp := adjustedTeamTotals(allInfo[oppID], totals[oppID], sc)
		oppScore := adjOpp.mp
		if oppUseGP {
			oppScore = adjOpp.gp
		}
		total += oppScore * matchScore
	}
	return total
}

// computeTeamBuchholz computes the team Buchholz tiebreak over the tournament's
// primary score with optional CutLowest and CutHighest cuts.
func computeTeamBuchholz[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	sc resolvedTeamScoring,
	cutLowest, cutHighest int,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}
	scores := teamBuchholzOpponentScores(teamID, info, totals, allInfo, sc)
	if len(scores) <= cutLowest+cutHighest {
		return 0
	}
	sort.Float64s(scores)
	scores = scores[cutLowest : len(scores)-cutHighest]
	var sum float64
	for _, s := range scores {
		sum += s
	}
	return sum
}

// computeTeamForeBuchholz computes Fore Buchholz (FB, FIDE Article 8.3) for
// teams by simulating unplayed real matches in the final round as drawn
// matches.
func computeTeamForeBuchholz[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	sc resolvedTeamScoring,
	cutLowest, cutHighest int,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}

	maxRound := maxTeamRoundNumber(allInfo)
	simTotals := make(map[TeamID]teamRawTotals, len(totals))
	for id, t := range totals {
		simTotals[id] = t
	}

	visited := make(map[*round.Match[TeamID, PlayerID]]bool)
	drawGP := 0.5 * float64(sc.boardsPerMatch)
	for _, tInfo := range allInfo {
		for _, m := range tInfo.matches {
			if visited[m] {
				continue
			}
			visited[m] = true
			if m.Round != maxRound || m.Away == nil {
				continue
			}
			if m.Settled() || (m.Result != nil && *m.Result == result.Void) {
				continue
			}
			// Subtract any partial board GP already counted in totals before adding simulated draw.
			_, _, hPartialGP, aPartialGP, _ := matchTeamPoints(m, sc)
			ht := simTotals[m.Home]
			ht.mp += sc.draw
			ht.gp += drawGP - hPartialGP
			simTotals[m.Home] = ht

			at := simTotals[*m.Away]
			at.mp += sc.draw
			at.gp += drawGP - aPartialGP
			simTotals[*m.Away] = at
		}
	}

	scores := teamBuchholzOpponentScores(teamID, info, simTotals, allInfo, sc)
	if len(scores) <= cutLowest+cutHighest {
		return 0
	}
	sort.Float64s(scores)
	scores = scores[cutLowest : len(scores)-cutHighest]
	var sum float64
	for _, s := range scores {
		sum += s
	}
	return sum
}

// computeTeamAverageOpponentBuchholz computes AOB for teams, rounded to the
// nearest whole number (0.5 rounds up).
func computeTeamAverageOpponentBuchholz[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	sc resolvedTeamScoring,
	cutLowest, cutHighest int,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}
	opponents := overTheBoardTeamOpponents(teamID, info)
	if len(opponents) == 0 {
		return 0
	}
	var sum float64
	for _, oppID := range opponents {
		sum += computeTeamBuchholz(oppID, allInfo, totals, sc, cutLowest, cutHighest)
	}
	return math.Round(sum / float64(len(opponents)))
}

// computeTeamKoyaSystem computes the Koya System (KS) tiebreak for teams using
// the tournament's primary score. When threshold <= 0, the default threshold is
// 50% of the maximum possible primary score across maxRound rounds.
func computeTeamKoyaSystem[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	sc resolvedTeamScoring,
	threshold float64,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}
	if threshold <= 0 {
		maxRound := float64(maxTeamRoundNumber(allInfo))
		if sc.primary == TeamScoreGamePoints {
			threshold = maxRound * 0.5 * float64(sc.boardsPerMatch)
		} else {
			threshold = maxRound * 0.5 * sc.win
		}
	}

	var total float64
	for _, m := range info.matches {
		if m.Away == nil {
			continue
		}
		var oppID TeamID
		isHome := m.Home == teamID
		if isHome {
			oppID = *m.Away
		} else {
			oppID = m.Home
		}
		if totals[oppID].primary(sc.primary) >= threshold {
			hMP, aMP, hGP, aGP, eff := matchTeamPoints(m, sc)
			if !eff.Counts() {
				continue
			}
			if isHome {
				total += teamRawTotals{mp: hMP, gp: hGP}.primary(sc.primary)
			} else {
				total += teamRawTotals{mp: aMP, gp: aGP}.primary(sc.primary)
			}
		}
	}
	return total
}

// computeTeamWins computes Number of Wins (WIN, FIDE Article 7.1) for a team:
// rounds where the team won the match (over-the-board match wins, match forfeit
// wins, and full-point byes).
func computeTeamWins[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	sc resolvedTeamScoring,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}
	var wins float64
	for _, m := range info.matches {
		hMP, aMP, _, _, eff := matchTeamPoints(m, sc)
		if !eff.Counts() {
			continue
		}
		if m.Away == nil {
			if eff == result.WhiteWins || eff == result.WhiteForfeit {
				wins++
			}
			continue
		}
		if (m.Home == teamID && hMP > aMP) || (*m.Away == teamID && aMP > hMP) {
			wins++
		}
	}
	return wins
}

// computeTeamMatchesWon computes Number of Matches Won over the board (WON,
// FIDE Article 7.2), excluding match forfeits and byes.
func computeTeamMatchesWon[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	sc resolvedTeamScoring,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}
	var won float64
	for _, m := range info.matches {
		if m.Away == nil || isMatchForfeit(m) {
			continue
		}
		hMP, aMP, _, _, eff := matchTeamPoints(m, sc)
		if !eff.Counts() {
			continue
		}
		if (m.Home == teamID && hMP > aMP) || (*m.Away == teamID && aMP > hMP) {
			won++
		}
	}
	return won
}

// computeTeamProgressiveScores computes Progressive Scores (PS, FIDE Article
// 7.5) for a team over the tournament's primary score.
func computeTeamProgressiveScores[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	sc resolvedTeamScoring,
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}
	scoreByRound := make(map[int]float64)
	for _, m := range info.matches {
		hMP, aMP, hGP, aGP, eff := matchTeamPoints(m, sc)
		if !eff.Counts() {
			continue
		}
		if m.Home == teamID {
			scoreByRound[m.Round] += teamRawTotals{mp: hMP, gp: hGP}.primary(sc.primary)
		} else {
			scoreByRound[m.Round] += teamRawTotals{mp: aMP, gp: aGP}.primary(sc.primary)
		}
	}

	maxRound := maxTeamRoundNumber(allInfo)
	var running, ps float64
	for r := 1; r <= maxRound; r++ {
		running += scoreByRound[r]
		ps += running
	}
	return ps
}

// computeTeamMatchesElectedToPlay computes Games/Matches Elected to Play (GE,
// FIDE Article 7.6) for a team: maxRound minus half-point byes, zero-point byes
// and match forfeit losses.
func computeTeamMatchesElectedToPlay[TeamID, PlayerID comparable](
	teamID TeamID,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
) float64 {
	info := allInfo[teamID]
	if info == nil {
		return 0
	}
	var deductions int
	for _, m := range info.matches {
		if m.Away == nil {
			if m.Result == nil {
				continue
			}
			r := *m.Result
			if r == result.WhiteHalf || r == result.ZeroPointBye {
				deductions++
			}
			continue
		}
		if !isMatchForfeit(m) {
			continue
		}
		if m.Result != nil {
			r := *m.Result
			if (m.Home == teamID && r == result.BlackForfeit) || (*m.Away == teamID && r == result.WhiteForfeit) {
				deductions++
			}
			continue
		}
		// Board-level forfeit of all boards: check which side scored 0.
		hGP, aGP := m.GamePoints()
		if (m.Home == teamID && hGP == 0 && aGP > 0) || (*m.Away == teamID && aGP == 0 && hGP > 0) {
			deductions++
		}
	}
	return float64(maxTeamRoundNumber(allInfo) - deductions)
}

func adjustedTeamTotals[TeamID, PlayerID comparable](
	info *teamPairingInfo[TeamID, PlayerID],
	raw teamRawTotals,
	sc resolvedTeamScoring,
) teamRawTotals {
	if info == nil || info.lastRealRound == -1 {
		return raw
	}
	var withdrawalRounds int
	for _, m := range info.matches {
		if m.Away != nil || m.Round <= info.lastRealRound {
			continue
		}
		if m.Result != nil && *m.Result == result.ZeroPointBye {
			withdrawalRounds++
		}
	}
	if withdrawalRounds == 0 {
		return raw
	}
	return teamRawTotals{
		mp: raw.mp + float64(withdrawalRounds)*sc.halfByeMP,
		gp: raw.gp + float64(withdrawalRounds)*sc.halfByeGP,
	}
}

func teamBuchholzOpponentScores[TeamID, PlayerID comparable](
	teamID TeamID,
	info *teamPairingInfo[TeamID, PlayerID],
	totals map[TeamID]teamRawTotals,
	allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	sc resolvedTeamScoring,
) []float64 {
	ownScore := totals[teamID].primary(sc.primary)
	var contributions []float64
	for _, m := range info.matches {
		if m.Result != nil && *m.Result == result.Void {
			continue
		}
		if m.Away == nil {
			contributions = append(contributions, ownScore)
			continue
		}
		var oppID TeamID
		if m.Home == teamID {
			oppID = *m.Away
		} else {
			oppID = m.Home
		}
		adj := adjustedTeamTotals(allInfo[oppID], totals[oppID], sc)
		contributions = append(contributions, adj.primary(sc.primary))
	}
	return contributions
}

func overTheBoardTeamOpponents[TeamID, PlayerID comparable](
	teamID TeamID,
	info *teamPairingInfo[TeamID, PlayerID],
) []TeamID {
	var opps []TeamID
	for _, m := range info.matches {
		if m.Away == nil || isMatchForfeit(m) || !m.Settled() {
			continue
		}
		if m.Home == teamID {
			opps = append(opps, *m.Away)
		} else {
			opps = append(opps, m.Home)
		}
	}
	return opps
}

func maxTeamRoundNumber[TeamID, PlayerID comparable](allInfo map[TeamID]*teamPairingInfo[TeamID, PlayerID]) int {
	var maxR int
	for _, info := range allInfo {
		for _, m := range info.matches {
			if m.Round > maxR {
				maxR = m.Round
			}
		}
	}
	return maxR
}
