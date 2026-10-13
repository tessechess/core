package standings

import "sort"

// tiebreakInfo describes which optional parameters a tiebreak key accepts.
type tiebreakInfo struct {
	cut       bool // accepts CutLowest / CutHighest
	threshold bool // accepts Threshold
}

// tiebreaks is the single list of supported tiebreak keys. computeTiebreak must
// handle every key except "direct_encounter", which Rank computes per points
// group.
var tiebreaks = map[string]tiebreakInfo{
	"buchholz":              {cut: true},
	"fore_buchholz":         {cut: true},
	"aob":                   {cut: true},
	"sonneborn_berger":      {},
	"koya_system":           {threshold: true},
	"direct_encounter":      {},
	"aro":                   {},
	"tpr":                   {},
	"ptp":                   {},
	"apro":                  {},
	"appo":                  {},
	"wins":                  {},
	"games_won":             {},
	"games_played_black":    {},
	"wins_with_black":       {},
	"progressive_scores":    {},
	"games_elected_to_play": {},
}

// IsTiebreakKey reports whether key is a supported tiebreak.
func IsTiebreakKey(key string) bool {
	_, ok := tiebreaks[key]
	return ok
}

// SupportsCut reports whether the tiebreak accepts CutLowest / CutHighest.
func SupportsCut(key string) bool { return tiebreaks[key].cut }

// SupportsThreshold reports whether the tiebreak accepts Threshold.
func SupportsThreshold(key string) bool { return tiebreaks[key].threshold }

// TiebreakKeys returns all supported tiebreak keys in alphabetical order.
func TiebreakKeys() []string {
	keys := make([]string, 0, len(tiebreaks))
	for k := range tiebreaks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// computeTiebreak returns one player's value for a per-player tiebreak and
// whether the key is handled. Unhandled keys ("direct_encounter", unknown keys)
// return (0, false); callers store 0 for them.
func computeTiebreak[ID comparable](
	cfg TiebreakConfig,
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
	ratings map[ID]int,
) (float64, bool) {
	switch cfg.Key {
	case "buchholz":
		return ComputeBuchholz(playerID, allInfo, rawScores, cfg.CutLowest, cfg.CutHighest), true
	case "fore_buchholz":
		return ComputeForeBuchholz(playerID, allInfo, rawScores, cfg.CutLowest, cfg.CutHighest), true
	case "aob":
		return ComputeAverageOpponentBuchholz(playerID, allInfo, rawScores, cfg.CutLowest, cfg.CutHighest), true
	case "sonneborn_berger":
		return ComputeSonnebornBerger(playerID, allInfo, rawScores), true
	case "koya_system":
		return ComputeKoyaSystem(playerID, allInfo, rawScores, cfg.Threshold), true
	case "aro":
		return ComputeAverageRatingOfOpponents(playerID, allInfo, ratings), true
	case "tpr":
		return ComputeTournamentPerformanceRating(playerID, allInfo, ratings), true
	case "ptp":
		return ComputePerfectTournamentPerformance(playerID, allInfo, ratings), true
	case "apro":
		return ComputeAveragePerfectPerformanceRatingOfOpponents(playerID, allInfo, ratings), true
	case "appo":
		return ComputeAveragePerfectPerformanceOfOpponents(playerID, allInfo, ratings), true
	case "wins":
		return ComputeNumberOfWins(playerID, allInfo, rawScores), true
	case "games_won":
		return ComputeNumberOfGamesWon(playerID, allInfo, rawScores), true
	case "games_played_black":
		return ComputeNumberOfPlayedGamesBlack(playerID, allInfo, rawScores), true
	case "wins_with_black":
		return ComputeNumberOfGamesWonBlack(playerID, allInfo, rawScores), true
	case "progressive_scores":
		return ComputeSumOfProgressiveScores(playerID, allInfo, rawScores), true
	case "games_elected_to_play":
		return ComputeGamesOneElectedToPlay(playerID, allInfo, rawScores), true
	}
	return 0, false
}
