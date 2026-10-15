package standings

import (
	"sort"

	"github.com/ValkoM28/tournamentessentials/core/result"
	"github.com/ValkoM28/tournamentessentials/core/round"
)

// Input is everything needed to compute standings for one section.
type Input[ID comparable] struct {
	Games     []round.Game[ID] // games of the completed rounds
	Ratings   map[ID]int       // rating per player; 0 or missing = unrated
	KFactors  map[ID]*int      // K-factor per player; nil or missing = no rating change
	Tiebreaks []TiebreakConfig
}

// RatingInfo holds rating-change and performance-rating data for a player.
// These are informational only and do not affect standings ordering.
type RatingInfo struct {
	PerformanceRating float64
	RatingChange      *float64 // nil when the player has no K-factor or no rating
}

// PlayerScore is one player's points, tiebreak values and rating info.
type PlayerScore[ID comparable] struct {
	ID         ID
	Points     float64
	Tiebreaks  map[string]float64 // keyed by TiebreakConfig.MapKey()
	RatingInfo *RatingInfo
}

// Standing is a ranked PlayerScore.
type Standing[ID comparable] struct {
	Rank  int
	Score *PlayerScore[ID]
}

// RoundResult is one player's game in one round, for a crosstable.
type RoundResult struct {
	Round        int
	OpponentRank *int         // nil for bye or unplayed
	Color        string       // "w" or "b"; empty for bye/unplayed
	Result       *result.Code // nil if the player had no pairing this round
	Points       float64
	IsForfeit    bool
	IsBye        bool // paired, but no opponent
	Played       bool // false if the player had no pairing this round
}

// CrosstableRow is one ranked player with their per-round results.
type CrosstableRow[ID comparable] struct {
	Rank   int
	Score  *PlayerScore[ID]
	Rounds []RoundResult // one per round, ordered by round number
}

// Scores returns every player's points, tiebreaks and rating info, in no
// particular order. Direct Encounter is left at 0 here; Rank fills it in.
func Scores[ID comparable](in Input[ID]) []*PlayerScore[ID] {
	scores, _ := scoresWithInfo(in)
	return scores
}

// Rank returns the standings: scores with Direct Encounter filled in, sorted by
// points and then by the configured tiebreaks, in order.
func Rank[ID comparable](in Input[ID]) []*Standing[ID] {
	standings, _ := rankWithInfo(in)
	return standings
}

// Crosstable returns the number of rounds (the highest round number among the
// games) and one row per player in standings order.
func Crosstable[ID comparable](in Input[ID]) (int, []CrosstableRow[ID]) {
	standings, allInfo := rankWithInfo(in)

	rankMap := make(map[ID]int, len(standings))
	for _, st := range standings {
		rankMap[st.Score.ID] = st.Rank
	}

	totalRounds := 0
	for _, info := range allInfo {
		for _, p := range info.pairings {
			if p.Round > totalRounds {
				totalRounds = p.Round
			}
		}
	}

	rows := make([]CrosstableRow[ID], len(standings))
	for i, st := range standings {
		id := st.Score.ID
		rows[i] = CrosstableRow[ID]{
			Rank:   st.Rank,
			Score:  st.Score,
			Rounds: buildRoundResults(id, allInfo[id], rankMap, totalRounds),
		}
	}
	return totalRounds, rows
}

func scoresWithInfo[ID comparable](in Input[ID]) ([]*PlayerScore[ID], map[ID]*playerPairingInfo[ID]) {
	// The tiebreak code shares one *round.Game between both players' records;
	// point into the caller's slice (read-only).
	games := make([]*round.Game[ID], len(in.Games))
	for i := range in.Games {
		games[i] = &in.Games[i]
	}
	rawScores := buildRawScores(games)
	allInfo := buildPlayerPairingInfo(games)

	var scores []*PlayerScore[ID]
	for playerID, points := range rawScores {
		tb := make(map[string]float64)
		for _, cfg := range in.Tiebreaks {
			// direct_encounter and unknown keys stay 0 here.
			tb[cfg.MapKey()], _ = computeTiebreak(cfg, playerID, allInfo, rawScores, in.Ratings)
		}
		rp := computeTournamentPerformanceRating(playerID, allInfo, in.Ratings)
		ri := &RatingInfo{PerformanceRating: rp}
		if k := in.KFactors[playerID]; k != nil {
			if r := in.Ratings[playerID]; r > 0 {
				delta := computeRatingChange(playerID, allInfo, in.Ratings, r, *k)
				ri.RatingChange = &delta
			}
		}
		scores = append(scores, &PlayerScore[ID]{
			ID:         playerID,
			Points:     points,
			Tiebreaks:  tb,
			RatingInfo: ri,
		})
	}
	return scores, allInfo
}

func rankWithInfo[ID comparable](in Input[ID]) ([]*Standing[ID], map[ID]*playerPairingInfo[ID]) {
	scores, allInfo := scoresWithInfo(in)
	fillDE(scores, allInfo, in.Tiebreaks)
	return sortAndRank(scores, in.Tiebreaks), allInfo
}

// fillDE computes Direct Encounter tiebreaks by points group and fills them into scores.
func fillDE[ID comparable](scores []*PlayerScore[ID], allInfo map[ID]*playerPairingInfo[ID], tiebreakCfgs []TiebreakConfig) {
	deMapKey, ok := findDE(tiebreakCfgs)
	if !ok {
		return
	}
	deScores := computeDEByPointsGroup(scores, allInfo)
	for _, ps := range scores {
		if _, exists := ps.Tiebreaks[deMapKey]; exists {
			ps.Tiebreaks[deMapKey] = deScores[ps.ID]
		}
	}
}

// sortAndRank sorts player scores by points and tiebreaks descending, returns ranked standings.
func sortAndRank[ID comparable](scores []*PlayerScore[ID], tiebreakCfgs []TiebreakConfig) []*Standing[ID] {
	sort.SliceStable(scores, func(i, j int) bool {
		a, b := scores[i], scores[j]
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		for _, cfg := range tiebreakCfgs {
			k := cfg.MapKey()
			av, bv := a.Tiebreaks[k], b.Tiebreaks[k]
			if av != bv {
				return av > bv
			}
		}
		return false
	})

	standings := make([]*Standing[ID], len(scores))
	for i, ps := range scores {
		standings[i] = &Standing[ID]{Rank: i + 1, Score: ps}
	}
	return standings
}

// buildRoundResults produces a RoundResult for each round 1..totalRounds for a player.
func buildRoundResults[ID comparable](playerID ID, info *playerPairingInfo[ID], rankMap map[ID]int, totalRounds int) []RoundResult {
	// Index pairings by round number for quick lookup.
	byRound := make(map[int]*round.Game[ID])
	if info != nil {
		for _, p := range info.pairings {
			byRound[p.Round] = p
		}
	}

	results := make([]RoundResult, totalRounds)
	for r := 1; r <= totalRounds; r++ {
		p, ok := byRound[r]
		if !ok {
			results[r-1] = RoundResult{Round: r, Played: false}
			continue
		}

		res := p.Result
		if res == nil {
			unplayed := result.Unplayed
			res = &unplayed
		}

		rr := RoundResult{
			Round:  r,
			Result: res,
			Points: p.Points(playerID),
			Played: true,
		}

		isWhite := p.White == playerID
		isBye := p.Black == nil

		if isBye {
			rr.IsBye = true
		} else {
			if isWhite {
				rr.Color = "w"
				oppID := *p.Black
				if rank, exists := rankMap[oppID]; exists {
					rr.OpponentRank = &rank
				}
			} else {
				rr.Color = "b"
				if rank, exists := rankMap[p.White]; exists {
					rr.OpponentRank = &rank
				}
			}
		}

		// Detect forfeit from result.
		if p.Result != nil && p.Result.IsForfeit() {
			rr.IsForfeit = true
		}

		results[r-1] = rr
	}
	return results
}

// computeDEByPointsGroup groups players by their point score and computes
// Direct Encounter within each group (FIDE Article 6.1).
func computeDEByPointsGroup[ID comparable](scores []*PlayerScore[ID], allInfo map[ID]*playerPairingInfo[ID]) map[ID]float64 {
	byPoints := make(map[float64][]ID)
	for _, ps := range scores {
		byPoints[ps.Points] = append(byPoints[ps.Points], ps.ID)
	}

	de := make(map[ID]float64, len(scores))
	for _, group := range byPoints {
		if len(group) < 2 {
			// Only one player at this score — DE contributes nothing useful.
			if len(group) == 1 {
				de[group[0]] = 0
			}
			continue
		}
		for id, deScore := range computeDirectEncounter(group, allInfo) {
			de[id] = deScore
		}
	}
	return de
}

// findDE returns the map key for the direct_encounter config entry and true if
// one is present, or ("", false) if it is absent.
func findDE(cfgs []TiebreakConfig) (string, bool) {
	for _, cfg := range cfgs {
		if cfg.Key == DirectEncounter {
			return cfg.MapKey(), true
		}
	}
	return "", false
}

func buildRawScores[ID comparable](games []*round.Game[ID]) map[ID]float64 {
	scores := make(map[ID]float64)
	for _, p := range games {
		scores[p.White] += p.Points(p.White)
		if p.Black != nil {
			scores[*p.Black] += p.Points(*p.Black)
		}
	}
	return scores
}
