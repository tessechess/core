package standings

import (
	"math"
	"sort"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

// ============================================================================
// FIDE Type A Tiebreaks - Subset of Games by Tied Participants
// ============================================================================
//
// FIDE Article 4.3: "Type A - a subset of the games by the tied participants.
// Tie-Breaks of this type may appear multiple times in the tie-break list."
//
// These tiebreaks require knowing which players are tied and comparing their
// results against each other. They must be computed during the sorting process,
// not pre-computed for all players.
//
// FIDE tiebreaks in this category:
//   - Direct Encounter (DE): head-to-head results among tied players
//
// Implementation note: These need special handling in GetStandings() sorting logic.

// computeDirectEncounter computes the Direct Encounter (DE) tiebreak for a group
// of tied players. Returns each player's score in games against the other players
// in the tied group.
//
// FIDE Article 6.1: The sum of the scores from encounters between tied players
// is used to produce separate standings.
//
// FIDE Article 6.1.1: Forfeited games are excluded (for Swiss tournaments).
//
// FIDE Article 6.1.2: If two participants met more than once, the addend is the
// average score of those games, not the sum.
func computeDirectEncounter[ID comparable](
	tiedPlayerIDs []ID,
	allInfo map[ID]*playerPairingInfo[ID],
) map[ID]float64 {
	tiedSet := make(map[ID]bool, len(tiedPlayerIDs))
	for _, id := range tiedPlayerIDs {
		tiedSet[id] = true
	}

	scores := make(map[ID]float64, len(tiedPlayerIDs))
	for _, id := range tiedPlayerIDs {
		scores[id] = 0
	}

	for _, playerID := range tiedPlayerIDs {
		info := allInfo[playerID]
		if info == nil {
			continue
		}

		// Group pairings by opponent to handle Article 6.1.2 (multiple games).
		opponentGames := make(map[ID][]float64)
		for _, p := range info.pairings {
			var opponentID *ID
			if p.White == playerID {
				opponentID = p.Black
			} else {
				opponentID = &p.White
			}

			// Skip byes and opponents not in the tied group.
			if opponentID == nil || !tiedSet[*opponentID] {
				continue
			}

			// Article 6.1.1: exclude forfeited games in Swiss tournaments.
			if p.Result != nil {
				r := *p.Result
				if r == result.WhiteForfeit || r == result.BlackForfeit {
					continue
				}
			}

			opponentGames[*opponentID] = append(opponentGames[*opponentID], p.Points(playerID))
		}

		// Sum contributions, averaging when players met more than once (Article 6.1.2).
		var total float64
		for _, games := range opponentGames {
			if len(games) == 1 {
				total += games[0]
			} else {
				var sum float64
				for _, g := range games {
					sum += g
				}
				total += sum / float64(len(games))
			}
		}
		scores[playerID] = total
	}

	return scores
}

// ============================================================================
// FIDE Type B Tiebreaks - Participant's Own Results
// ============================================================================
//
// FIDE Article 4.3: "Type B - participants' own results, so their value can be
// calculated or predicted by the involved participants before or during their
// own games"
//
// These tiebreaks depend only on the player's own game results and can be
// computed independently for each player before sorting.
//
// FIDE tiebreaks in this category:
//   - Number of Wins (WIN): count of rounds where player scored 1.0
//   - Number of Games Won (WON): count of over-the-board wins
//   - Number of Games Played with Black (BPG)
//   - Number of Games Won with Black (BWG)
//   - Progressive Scores (PS): cumulative score sum
//   - Games one Elected to Play (GE): rounds minus byes/forfeits

// computeGamesOneElectedToPlay computes the Games Elected to Play (GE) tiebreak.
//
// FIDE Article 7.6: The number of rounds reduced by the number of half-point-byes,
// zero-point-byes or forfeit losses that a participant had in the tournament.
//
// Deducted rounds:
//   - Half-point bye  (Black nil, result "0.5-0")
//   - Zero-point bye  (Black nil, result "0-0")
//   - Forfeit loss as White (result "-/+", opponent wins by forfeit)
//   - Forfeit loss as Black (result "+/-", white wins by forfeit)
//
// NOT deducted: full-point byes (player elected to receive the point) and
// forfeit wins (opponent didn't show; the player themselves was present).
func computeGamesOneElectedToPlay[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	var deductions int
	for _, p := range info.pairings {
		if p.Result == nil {
			continue
		}
		r := *p.Result
		if p.Black == nil {
			// Bye round — only half-point and zero-point byes are deducted.
			if r == result.WhiteHalf || r == result.ZeroPointBye {
				deductions++
			}
			continue
		}
		// Real game — check for forfeit loss by this player.
		if p.White == playerID && r == result.BlackForfeit {
			deductions++ // White forfeited; black wins "-/+"
		} else if *p.Black == playerID && r == result.WhiteForfeit {
			deductions++ // Black forfeited; white wins "+/-"
		}
	}

	return float64(maxRoundNumber(allInfo) - deductions)
}

// computeNumberOfPlayedGamesBlack computes the Number of Games Played with Black
// (BPG) tiebreak for playerID.
//
// FIDE Article 7.3: The number of games played over the board with the black pieces.
// A game counts when the player appears as Black (non-nil Black equal to
// playerID) and a result has been recorded (not nil, not unplayed, not void).
func computeNumberOfPlayedGamesBlack[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}
	var bpg float64
	for _, p := range info.pairings {
		if p.Black == nil || *p.Black != playerID {
			continue // bye or player is white
		}
		if p.Result == nil {
			continue // not yet played
		}
		r := *p.Result
		if r == result.Unplayed || r == result.Void {
			continue
		}
		bpg++
	}
	return bpg
}

// computeNumberOfGamesWon computes the Number of Games Won (WON) tiebreak.
//
// FIDE Article 7.2: The number of games won over the board.
// Only ResultWhiteWins / ResultBlackWins against a real opponent count —
// forfeit wins and full-point byes are excluded.
func computeNumberOfGamesWon[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}
	var won float64
	for _, p := range info.pairings {
		if p.Black == nil || p.Result == nil {
			continue // bye or unplayed
		}
		r := *p.Result
		if p.White == playerID && r == result.WhiteWins {
			won++
		} else if *p.Black == playerID && r == result.BlackWins {
			won++
		}
	}
	return won
}

// computeNumberOfGamesWonBlack computes the Number of Games Won with Black
// (BWG) tiebreak for playerID.
//
// FIDE Article 7.4: The number of games won over the board with the black pieces.
// Only ResultBlackWins counts — forfeit wins as Black are excluded.
func computeNumberOfGamesWonBlack[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}
	var bwg float64
	for _, p := range info.pairings {
		if p.Black == nil || *p.Black != playerID {
			continue // bye or player is white
		}
		if p.Result != nil && *p.Result == result.BlackWins {
			bwg++
		}
	}
	return bwg
}

// computeNumberOfWins computes the Number of Wins (WIN) tiebreak for playerID.
//
// FIDE Article 7.1: The number of rounds where a participant obtains, with or
// without playing, as many points as awarded for a win (i.e. exactly 1.0).
// This includes over-the-board wins, forfeit wins, and full-point byes.
func computeNumberOfWins[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}
	var wins float64
	for _, p := range info.pairings {
		if p.Points(playerID) == 1.0 {
			wins++
		}
	}
	return wins
}

// computeSumOfProgressiveScores computes the (Sum of) Progressive Scores (PS)
// tiebreak for playerID.
//
// FIDE Article 7.5: After each round a participant has a certain tournament score.
// This tie-break is the sum of those cumulative scores — one snapshot per round.
//
// A player who scores points early accumulates a higher PS than one who scores
// the same total later in the tournament.
func computeSumOfProgressiveScores[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	// Collect the player's score earned in each round.
	scoreByRound := make(map[int]float64)
	for _, p := range info.pairings {
		scoreByRound[p.Round] += p.Points(playerID)
	}

	// Walk every round 1..maxRound, accumulating the running total.
	// Rounds where the player has no pairing contribute 0 to that round's score,
	// but the running total carries forward unchanged.
	maxRound := maxRoundNumber(allInfo)
	var running, ps float64
	for r := 1; r <= maxRound; r++ {
		running += scoreByRound[r]
		ps += running
	}
	return ps
}

// ============================================================================
// FIDE Type C Tiebreaks - Opponents' Final Results
// ============================================================================
//
// FIDE Article 4.3: "Type C - opponents' (final) results, so they can be
// calculated only at the end of the round or tournament."
//
// These tiebreaks sum or average the scores of opponents. They can be computed
// independently for each player once all games are finished.
//
// FIDE tiebreaks in this category:
//   - Buchholz (BH): sum of opponents' scores
//   - Koya System (KS): number of points scored against opponents who finished with 50% or more
//	 - Sonneborn-Berger (SB): sum of (opponent's score × points scored against them)
//   - Average Opponent Buchholz (AOB): average of opponents' BH scores
//
// Note: Sonneborn-Berger (SB) is classified as Type BC (combined B+C)

// computeBuchholz computes the Buchholz tiebreak for playerID.
//
//	cutLowest  = number of lowest opponent scores to discard
//	cutHighest = number of highest opponent scores to discard
//
// Common variants:
//
//	BH        (plain):    cutLowest=0, cutHighest=0
//	BH-C1     (cut-1):   cutLowest=1, cutHighest=0
//	BH-Median (median-1): cutLowest=1, cutHighest=1
//
// Returns 0 if there are not enough rounds to apply the requested cuts.
func computeBuchholz[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
	cutLowest, cutHighest int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	scores := buchholzOpponentScores(playerID, info, rawScores, allInfo)

	if len(scores) <= cutLowest+cutHighest {
		return 0 // not enough rounds to apply cuts
	}

	sort.Float64s(scores)
	scores = scores[cutLowest : len(scores)-cutHighest]

	var total float64
	for _, s := range scores {
		total += s
	}
	return total
}

// computeKoyaSystem computes the Koya System (KS) tiebreak for playerID.
//
// FIDE Article 9.2: The number of points achieved against all participants
// who have scored at least 50% of the maximum possible tournament score.
//
// scoreThreshold is the minimum score an opponent must have reached to be
// counted. If scoreThreshold <= 0, it is automatically set to 50% of the
// highest round number found across all pairings in allInfo.
//
// The threshold can be raised or lowered in half-point steps per FIDE Article
// 14.5 (Limit modifier). Only games played over the board (non-nil opponent)
// are considered.
func computeKoyaSystem[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
	scoreThreshold float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	if scoreThreshold <= 0 {
		scoreThreshold = koyaDefaultThreshold(allInfo)
	}

	var total float64
	for _, p := range info.pairings {
		var opponentID *ID
		if p.White == playerID {
			opponentID = p.Black
		} else {
			opponentID = &p.White
		}

		if opponentID == nil {
			continue // byes do not count for Koya
		}

		if rawScores[*opponentID] >= scoreThreshold {
			total += p.Points(playerID)
		}
	}
	return total
}

// computeAverageOpponentBuchholz computes the Average Opponent Buchholz (AOB)
// tiebreak for playerID.
//
// AOB = average of each over-the-board opponent's own Buchholz score,
// rounded to the nearest whole number (0.5 rounds up).
//
// cutLowest / cutHighest are forwarded to each opponent's Buchholz calculation,
// so AOB, AOB-C1, AOB-Median, etc. are all expressible.
//
// Returns 0 if the player has no over-the-board opponents.
func computeAverageOpponentBuchholz[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
	cutLowest, cutHighest int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	opponents := overTheBoardOpponents(playerID, info)
	if len(opponents) == 0 {
		return 0
	}

	var sum float64
	for _, oppID := range opponents {
		sum += computeBuchholz(oppID, allInfo, rawScores, cutLowest, cutHighest)
	}

	return math.Round(sum / float64(len(opponents)))
}

// computeSonnebornBerger computes the Sonneborn-Berger tiebreak for playerID.
//
// Note: This is classified as Type BC (combined B+C) in FIDE regulations,
// but placed here with Type C tiebreaks for code organization since it
// primarily depends on opponents' scores.
//
// SB = sum of (opponent's score × points scored against them)
func computeSonnebornBerger[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}
	ownScore := rawScores[playerID]

	var total float64
	for _, p := range info.pairings {
		isWhite := p.White == playerID
		pointsAgainstOpponent := p.Points(playerID)

		var opponentID *ID

		if isWhite {
			opponentID = p.Black
		} else {
			opponentID = &p.White
		}

		if opponentID == nil {
			total += ownScore * pointsAgainstOpponent
		} else {
			oppRaw := rawScores[*opponentID]
			oppInfo := allInfo[*opponentID]
			adjustedOpp := adjustedOpponentScore(oppInfo, oppRaw)
			total += adjustedOpp * pointsAgainstOpponent
		}

	}
	return total
}

// ============================================================================
// FIDE Type D Tiebreaks - Opponents' Prior Known Data
// ============================================================================
//
// FIDE Article 4.3: "Type D - opponents' prior known data (e.g. ratings, but
// also results of previous rounds), so their values can be calculated after
// the pairings are published (i.e. before the games are played)"
//
// These tiebreaks use opponent ratings or other pre-known information.
// They can be computed before games are played.
//
// FIDE tiebreaks in this category:
//   - Average Rating of Opponents (ARO)         Type D   Article 10.1
//   - Fore Buchholz (FB)                         Type D   Article 8.3
//   - Tournament Performance Rating (TPR)        Type DB  Article 10.2
//   - Perfect Tournament Performance (PTP)       Type DB  Article 10.3
//   - Avg Performance Rating of Opponents (APRO) Type DC  Article 10.4
//   - Avg Perfect Performance of Opponents (APPO)Type DC  Article 10.5

// computeAverageRatingOfOpponents computes the ARO tiebreak for playerID.
//
// FIDE Article 10.1: The average of the ratings of the opponents played over
// the board, rounded to the nearest whole number (0.5 rounded up).
//
// "Over the board" means: the pairing had a real opponent (Black ≠ nil)
// and the game reached a played result (not a forfeit, void, or unplayed).
//
// Returns 0 if the player has no qualifying games.
func computeAverageRatingOfOpponents[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	ratings map[ID]int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	opponents := overTheBoardOpponents(playerID, info)
	if len(opponents) == 0 {
		return 0
	}

	var sum, count int
	for _, oppID := range opponents {
		if r := ratings[oppID]; r > 0 {
			sum += r
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return math.Round(float64(sum) / float64(count))
}

// fractionalScoreToRD converts a fractional score (points / games, in [0,1])
// to an Elo rating difference using the FIDE formula:
//
//	RD = 400 × log10(p / (1 − p))
//
// Capped to [−800, +800]. Special-cased at the boundaries (p=0 → −800,
// p=1 → +800) where the formula is undefined.
func fractionalScoreToRD(p float64) float64 {
	if p >= 1.0 {
		return 800
	}
	if p <= 0.0 {
		return -800
	}
	rd := 400 * math.Log10(p/(1-p))
	if rd > 800 {
		return 800
	}
	if rd < -800 {
		return -800
	}
	return rd
}

// expectedScore returns the Elo expected score for a player with rating R
// facing an opponent with the given opponent rating:
//
//	E = 1 / (1 + 10^((opponentRating − R) / 400))
//
// The full rating scale is used — no ±400 cap (required by PTP, Article 10.3).
func expectedScore(playerRating, opponentRating float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, (opponentRating-playerRating)/400))
}

// computeForeBuchholz computes the Fore Buchholz (FB) tiebreak for playerID.
//
// FIDE Article 8.3: Buchholz score calculated as if all paired games for the
// final round had ended in draws.
//
// "Final round" is the highest round number present in allInfo. Any real
// pairing (Black ≠ nil) in that round whose result is nil or "*" is
// simulated as a draw — 0.5 added to each player's score — before running the
// standard Buchholz calculation.
//
// Byes in the final round are NOT simulated as draws; they are handled by the
// normal Article 16 virtual-opponent logic inside buchholzOpponentScores.
//
// cutLowest and cutHighest follow the same convention as computeBuchholz.
func computeForeBuchholz[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
	cutLowest, cutHighest int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	// 1. Find the maximum round number (the "final" round).
	var maxRound int
	for _, pInfo := range allInfo {
		for _, p := range pInfo.pairings {
			if p.Round > maxRound {
				maxRound = p.Round
			}
		}
	}

	// 2. Build simulated scores: copy rawScores, then add 0.5 to each player
	//    involved in an unplayed real pairing in the final round.
	//    Each *round.Game[ID] is shared between white and black in allInfo, so track
	//    visited pointers to avoid double-counting.
	simScores := make(map[ID]float64, len(rawScores))
	for id, s := range rawScores {
		simScores[id] = s
	}
	visited := make(map[*round.Game[ID]]bool)
	for _, pInfo := range allInfo {
		for _, p := range pInfo.pairings {
			if visited[p] {
				continue
			}
			visited[p] = true
			if p.Round != maxRound {
				continue
			}
			if p.Black == nil {
				continue // bye — not a paired game; handled by Art. 16
			}
			if p.Result != nil && *p.Result != result.Unplayed {
				continue // game already has a settled result
			}
			// Simulate a draw for both players.
			simScores[p.White] += 0.5
			simScores[*p.Black] += 0.5
		}
	}

	// 3. Compute Buchholz using the simulated scores.
	//    buchholzOpponentScores accepts any score map, so passing simScores
	//    propagates the draw simulation into opponent contributions.
	scores := buchholzOpponentScores(playerID, info, simScores, allInfo)
	if len(scores) <= cutLowest+cutHighest {
		return 0
	}
	sort.Float64s(scores)
	scores = scores[cutLowest : len(scores)-cutHighest]
	var total float64
	for _, s := range scores {
		total += s
	}
	return total
}

// computeTournamentPerformanceRating computes the TPR tiebreak for playerID.
//
// FIDE Article 10.2: Calculated by adding to ARO a rating difference (RD)
// resulting from the conversion of the fractional score (points scored in
// over-the-board games divided by the number of such games) into RD using
// the FIDE Rating Regulations conversion table.
//
// Implementation:
//
//	ARO_raw = unrounded average of opponent ratings (over-the-board games only)
//	p       = total OTB points / number of OTB games
//	RD      = fractionalScoreToRD(p)          [capped ±800]
//	TPR     = math.Round(ARO_raw + RD)
//
// Returns 0 if the player has no qualifying over-the-board games.
func computeTournamentPerformanceRating[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	ratings map[ID]int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	ps := overTheBoardPairings(playerID, info)
	if len(ps) == 0 {
		return 0
	}

	// Sum opponent ratings and points scored in OTB games against rated opponents.
	// Games against unrated opponents (rating == 0) are excluded from both components.
	var ratingSum int
	var points float64
	var ratedCount int
	for _, p := range ps {
		var oppID ID
		if p.White == playerID {
			oppID = *p.Black
		} else {
			oppID = p.White
		}
		if r := ratings[oppID]; r > 0 {
			ratingSum += r
			points += p.Points(playerID)
			ratedCount++
		}
	}
	if ratedCount == 0 {
		return 0
	}

	aroRaw := float64(ratingSum) / float64(ratedCount)
	frac := points / float64(ratedCount)
	return math.Round(aroRaw + fractionalScoreToRD(frac))
}

// fidePDTableMaxD holds the upper bound of each rating-difference bucket in
// FIDE Rating Regulations (B.02) Table 8.1.2. Index i (0..49) corresponds to a
// higher-rated player scoring probability of (50 + i) / 100. Any difference
// above 735 gives 1.00 for the higher-rated player and 0.00 for the lower-rated
// player.
var fidePDTableMaxD = [50]int{
	3, 10, 17, 25, 32, 39, 46, 53, 61, 68,
	76, 83, 91, 98, 106, 113, 121, 129, 137, 145,
	153, 162, 170, 179, 188, 197, 206, 215, 225, 235,
	245, 256, 267, 278, 290, 302, 315, 328, 344, 357,
	374, 391, 411, 432, 456, 484, 517, 559, 619, 735,
}

// fideScoringProbability returns the player's expected score probability PD
// for a single game under FIDE Rating Regulations (B.02) Articles 8.1.2 and 8.3.1.
//
// Per Article 8.3.1 (effective 1 October 2025), a rating difference of more
// than 400 points is counted as 400 points for players rated below 2650; for
// players rated 2650 and above, the full rating difference is used.
func fideScoringProbability(playerRating, opponentRating int) float64 {
	diff := playerRating - opponentRating
	if playerRating < 2650 {
		if diff > 400 {
			diff = 400
		} else if diff < -400 {
			diff = -400
		}
	}
	absDiff := diff
	if absDiff < 0 {
		absDiff = -absDiff
	}
	for i, maxD := range fidePDTableMaxD {
		if absDiff <= maxD {
			if diff >= 0 {
				return float64(50+i) / 100.0
			}
			return float64(50-i) / 100.0
		}
	}
	if diff >= 0 {
		return 1.0
	}
	return 0.0
}

// computeRatingChange returns the FIDE rating change for a player: K × (W − We).
// playerRating is the player's pre-tournament rating; kFactor is their FIDE coefficient.
// Only over-the-board games against rated opponents are counted (byes, forfeits,
// and games against unrated opponents are excluded per FIDE B.02 Articles 5.1 and 8.3.1).
func computeRatingChange[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	ratings map[ID]int,
	playerRating int,
	kFactor int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}
	ps := overTheBoardPairings(playerID, info)
	if len(ps) == 0 {
		return 0
	}

	var we, w float64
	for _, p := range ps {
		var oppID ID
		if p.White == playerID {
			oppID = *p.Black
		} else {
			oppID = p.White
		}
		oppRating := ratings[oppID]
		if oppRating <= 0 {
			continue
		}
		we += fideScoringProbability(playerRating, oppRating)
		w += p.Points(playerID)
	}
	// Round to one decimal place per FIDE tournament rating change display.
	return math.Round(float64(kFactor)*(w-we)*10) / 10
}

// computePerfectTournamentPerformance computes the PTP tiebreak for playerID.
//
// FIDE Article 10.3: The lowest integer rating R such that the player's
// expected score against all over-the-board opponents (using the Elo formula)
// is ≥ their actual tournament score.
//
// The full rating scale is used — no ±400 cap (explicitly stated in Art. 10.3).
//
// Algorithm: binary search on the strictly monotone expectedScore sum.
//
//	lo = min(opponent ratings) − 800   (guarantees expected ≈ 0)
//	hi = max(opponent ratings) + 3000  (guarantees expected ≈ n for any score)
//
// Returns 0 if the player has no qualifying over-the-board games.
func computePerfectTournamentPerformance[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	ratings map[ID]int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	ps := overTheBoardPairings(playerID, info)
	if len(ps) == 0 {
		return 0
	}

	// Collect opponent ratings and actual score from OTB games.
	oppRatings := make([]int, 0, len(ps))
	var actual float64
	for _, p := range ps {
		var oppID ID
		if p.White == playerID {
			oppID = *p.Black
		} else {
			oppID = p.White
		}
		oppRatings = append(oppRatings, ratings[oppID])
		actual += p.Points(playerID)
	}

	// Binary search bounds.
	minOpp, maxOpp := oppRatings[0], oppRatings[0]
	for _, r := range oppRatings[1:] {
		if r < minOpp {
			minOpp = r
		}
		if r > maxOpp {
			maxOpp = r
		}
	}
	lo := minOpp - 800
	hi := maxOpp + 3000 // generous upper bound; covers perfect scores

	// expTotal returns the sum of Elo expected scores for hypothetical rating R.
	expTotal := func(R int) float64 {
		var sum float64
		for _, opp := range oppRatings {
			sum += expectedScore(float64(R), float64(opp))
		}
		return sum
	}

	// Find the lowest integer R where expTotal(R) >= actual.
	for lo < hi {
		mid := lo + (hi-lo)/2
		if expTotal(mid) >= actual {
			hi = mid
		} else {
			lo = mid + 1
		}
	}

	return float64(lo)
}

// computeAveragePerfectPerformanceRatingOfOpponents computes the APRO tiebreak
// for playerID.
//
// FIDE Article 10.4: The average of the performances (TPR) of the opponents
// played over the board, rounded to the nearest whole number (0.5 rounded up).
//
// Returns 0 if the player has no qualifying over-the-board games.
func computeAveragePerfectPerformanceRatingOfOpponents[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	ratings map[ID]int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	opponents := overTheBoardOpponents(playerID, info)
	if len(opponents) == 0 {
		return 0
	}

	var sum float64
	for _, oppID := range opponents {
		sum += computeTournamentPerformanceRating(oppID, allInfo, ratings)
	}

	return math.Round(sum / float64(len(opponents)))
}

// computeAveragePerfectPerformanceOfOpponents computes the APPO tiebreak for
// playerID.
//
// FIDE Article 10.5: The average of the perfect performances (PTP) of the
// opponents played over the board, rounded to the nearest whole number
// (0.5 rounded up).
//
// Returns 0 if the player has no qualifying over-the-board games.
func computeAveragePerfectPerformanceOfOpponents[ID comparable](
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	ratings map[ID]int,
) float64 {
	info := allInfo[playerID]
	if info == nil {
		return 0
	}

	opponents := overTheBoardOpponents(playerID, info)
	if len(opponents) == 0 {
		return 0
	}

	var sum float64
	for _, oppID := range opponents {
		sum += computePerfectTournamentPerformance(oppID, allInfo, ratings)
	}

	return math.Round(sum / float64(len(opponents)))
}

// ============================================================================
// Team Tournament Tiebreaks
// ============================================================================
//
// FIDE Article 4.3: "Type E - team tournaments: tiebreaks based on the
// results of the teams' players, so they can be calculated only at the end of
// the round or tournament."
//
// These tiebreaks apply to team tournaments and depend on the results of
// individual team members.
//
// FIDE tiebreaks in this category:
//   - Game Points (GP): total points scored by all team members
//   - Match Points (MP): total match points scored by the team (2 for win, 1 for draw, 0 for loss)

func computeGamePoints() { panic("not implemented") } // unexported until the implementation is done

func computeMatchPoints() { panic("not implemented") } // unexported until the implementation is done

// ============================================================================
// Helper functions
// ============================================================================

// buildPlayerPairingInfo groups pairings by player and computes lastRealRound.
// A "real" game is any pairing where Black is non-nil.
func buildPlayerPairingInfo[ID comparable](pairings []*round.Game[ID]) map[ID]*playerPairingInfo[ID] {
	allInfo := make(map[ID]*playerPairingInfo[ID])

	ensure := func(id ID) *playerPairingInfo[ID] {
		if allInfo[id] == nil {
			allInfo[id] = &playerPairingInfo[ID]{lastRealRound: -1}
		}
		return allInfo[id]
	}

	for _, p := range pairings {
		wi := ensure(p.White)
		wi.pairings = append(wi.pairings, p)

		if p.Black != nil {
			bi := ensure(*p.Black)
			bi.pairings = append(bi.pairings, p)

			if p.Round > wi.lastRealRound {
				wi.lastRealRound = p.Round
			}
			if p.Round > bi.lastRealRound {
				bi.lastRealRound = p.Round
			}
		}
	}
	return allInfo
}

// adjustedOpponentScore returns the opponent's score adjusted for FIDE Article 16.3.2.
//
// For every round after the opponent's last real game where they received a
// nil-opponent forfeit pairing ("0-0"), we add 0.5 to their score.
// This protects the Buchholz of players who faced someone who later withdrew.
func adjustedOpponentScore[ID comparable](info *playerPairingInfo[ID], rawScore float64) float64 {
	if info.lastRealRound == -1 {
		// Opponent never played a real game — no adjustment needed.
		return rawScore
	}

	var withdrawalRounds int
	for _, p := range info.pairings {
		if p.Black != nil {
			continue // real game, not a withdrawal
		}
		if p.Round <= info.lastRealRound {
			continue // forfeit/bye before or during active play, not a withdrawal
		}
		if p.Result != nil && *p.Result == result.ZeroPointBye {
			withdrawalRounds++
		}
	}

	return rawScore + float64(withdrawalRounds)*0.5
}

// buchholzOpponentScores returns the list of opponent score contributions
// (one per round) for playerID's Buchholz calculation.
//
// Unplayed rounds (bye or forfeit, i.e. nil opponent) use a "virtual opponent"
// whose score equals the player's own score — per FIDE 16.4.
func buchholzOpponentScores[ID comparable](
	playerID ID,
	info *playerPairingInfo[ID],
	rawScores map[ID]float64,
	allInfo map[ID]*playerPairingInfo[ID],
) []float64 {
	ownScore := rawScores[playerID]
	var contributions []float64

	for _, p := range info.pairings {
		var opponentID *ID
		if p.White == playerID {
			opponentID = p.Black // nil for a bye/forfeit
		} else {
			opponentID = &p.White
		}

		if opponentID == nil {
			// Unplayed round: virtual opponent with player's own score (FIDE 16.4).
			contributions = append(contributions, ownScore)
		} else {
			oppRaw := rawScores[*opponentID]
			oppInfo := allInfo[*opponentID]
			contributions = append(contributions, adjustedOpponentScore(oppInfo, oppRaw))
		}
	}
	return contributions
}

// overTheBoardPairings returns pairings in which playerID actually played over
// the board — real pairings (Black ≠ nil) with a played result (not a
// forfeit, void, or unplayed round). This is the shared filter for all Type D
// rating-based tiebreaks.
func overTheBoardPairings[ID comparable](playerID ID, info *playerPairingInfo[ID]) []*round.Game[ID] {
	var games []*round.Game[ID]
	for _, p := range info.pairings {
		var opponentID *ID
		if p.White == playerID {
			opponentID = p.Black
		} else {
			opponentID = &p.White
		}
		if opponentID == nil {
			continue // bye
		}
		if p.Result == nil {
			continue // game not yet decided
		}
		switch *p.Result {
		case result.WhiteForfeit, result.BlackForfeit,
			result.Unplayed, result.Void:
			continue // not played over the board
		}
		games = append(games, p)
	}
	return games
}

// overTheBoardOpponents returns the registration IDs of opponents that playerID
// actually played over the board. It is a convenience wrapper over
// overTheBoardPairings.
func overTheBoardOpponents[ID comparable](playerID ID, info *playerPairingInfo[ID]) []ID {
	ps := overTheBoardPairings(playerID, info)
	opponents := make([]ID, len(ps))
	for i, p := range ps {
		if p.White == playerID {
			opponents[i] = *p.Black
		} else {
			opponents[i] = p.White
		}
	}
	return opponents
}

// maxRoundNumber returns the highest round number seen across all players' pairings.
func maxRoundNumber[ID comparable](allInfo map[ID]*playerPairingInfo[ID]) int {
	var max int
	for _, info := range allInfo {
		for _, p := range info.pairings {
			if p.Round > max {
				max = p.Round
			}
		}
	}
	return max
}

// koyaDefaultThreshold returns 50% of the maximum round number seen across
// all players' pairings, which equals half the maximum possible score in
// a round-robin tournament.
func koyaDefaultThreshold[ID comparable](allInfo map[ID]*playerPairingInfo[ID]) float64 {
	return float64(maxRoundNumber(allInfo)) * 0.5
}
