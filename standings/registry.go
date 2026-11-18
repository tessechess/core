package standings

import "sort"

// TiebreakKey identifies a tiebreak algorithm in TiebreakConfig.Key. The values
// are the keys stored in tournament settings and sent over the API.
type TiebreakKey string

const (
	// Buchholz is the sum of the opponents' scores. Supports cuts.
	Buchholz TiebreakKey = "buchholz"
	// ForeBuchholz is Buchholz calculated as if the unfinished games of the
	// final round had ended in draws (FIDE 8.3). Supports cuts.
	ForeBuchholz TiebreakKey = "fore_buchholz"
	// AverageOpponentBuchholz is the average of the opponents' Buchholz scores
	// (AOB). Supports cuts.
	AverageOpponentBuchholz TiebreakKey = "aob"
	// SonnebornBerger is the sum of the scores of the opponents beaten plus half
	// the scores of the opponents drawn.
	SonnebornBerger TiebreakKey = "sonneborn_berger"
	// KoyaSystem is the score made against opponents who reached a threshold
	// score (50% of the rounds by default). Supports a threshold.
	KoyaSystem TiebreakKey = "koya_system"
	// DirectEncounter is the result among the players tied on points; Rank
	// computes it per points group.
	DirectEncounter TiebreakKey = "direct_encounter"
	// AverageRatingOfOpponents is the average rating of the opponents (ARO).
	AverageRatingOfOpponents TiebreakKey = "aro"
	// TournamentPerformanceRating is the performance rating (TPR).
	TournamentPerformanceRating TiebreakKey = "tpr"
	// PerfectTournamentPerformance is the rating that gives an expected score
	// equal to the player's score against the same opponents (PTP, FIDE 10.3).
	PerfectTournamentPerformance TiebreakKey = "ptp"
	// AveragePerfectPerformanceRatingOfOpponents is the average of the
	// opponents' performance ratings (APRO, FIDE 10.4).
	AveragePerfectPerformanceRatingOfOpponents TiebreakKey = "apro"
	// AveragePerfectPerformanceOfOpponents is the average of the opponents'
	// perfect performances (APPO, FIDE 10.5).
	AveragePerfectPerformanceOfOpponents TiebreakKey = "appo"
	// Wins is the number of rounds won with or without playing: over-the-board
	// wins, forfeit wins and full-point byes (FIDE 7.1).
	Wins TiebreakKey = "wins"
	// GamesWon is the number of games won over the board; forfeit wins and byes
	// do not count (FIDE 7.2).
	GamesWon TiebreakKey = "games_won"
	// GamesPlayedBlack is the number of games played over the board with black
	// (FIDE 7.3).
	GamesPlayedBlack TiebreakKey = "games_played_black"
	// WinsWithBlack is the number of games won over the board with black
	// (FIDE 7.4).
	WinsWithBlack TiebreakKey = "wins_with_black"
	// ProgressiveScores is the sum of the cumulative scores after each round
	// (FIDE 7.5).
	ProgressiveScores TiebreakKey = "progressive_scores"
	// GamesElectedToPlay is the number of rounds minus the half-point byes,
	// zero-point byes and forfeit losses (FIDE 7.6).
	GamesElectedToPlay TiebreakKey = "games_elected_to_play"
	// MatchPoints is the total match points scored by a team (FIDE C.07). Team
	// competitions only.
	MatchPoints TiebreakKey = "match_points"
	// GamePoints is the total board game points scored by all members of a
	// team (FIDE C.07). Team competitions only.
	GamePoints TiebreakKey = "game_points"
	// ExtendedMatchMatchSB is the Extended Sonneborn-Berger for teams using
	// opponent match points × match points scored against them (EMMSB, FIDE
	// C.07 13.1). Team competitions only.
	ExtendedMatchMatchSB TiebreakKey = "emmsb"
	// ExtendedMatchGameSB is the Extended Sonneborn-Berger for teams using
	// opponent match points × game points scored against them (EMGSB, FIDE
	// C.07 13.2). Team competitions only.
	ExtendedMatchGameSB TiebreakKey = "emgsb"
	// ExtendedGameMatchSB is the Extended Sonneborn-Berger for teams using
	// opponent game points × match points scored against them (EGMSB, FIDE
	// C.07 13.3). Team competitions only.
	ExtendedGameMatchSB TiebreakKey = "egmsb"
	// ExtendedGameGameSB is the Extended Sonneborn-Berger for teams using
	// opponent game points × game points scored against them (EGGSB, FIDE
	// C.07 13.4). Team competitions only.
	ExtendedGameGameSB TiebreakKey = "eggsb"
	// BoardCount is the Berlin system board-weighted score: on a B-board match,
	// board b (1..B) has weight B-b+1 multiplied by the game points scored on
	// that board (BC, FIDE C.07 11.2). Team competitions only.
	BoardCount TiebreakKey = "board_count"
	// TopBoardResults breaks ties among tied teams by comparing total game
	// points on board 1, then board 2, and so on down to board B (TBR / BBE,
	// FIDE C.07 11.3). RankTeams computes it per tied group. Team competitions
	// only.
	TopBoardResults TiebreakKey = "top_board_results"
)

// tiebreakInfo describes which optional parameters a tiebreak key accepts.
type tiebreakInfo struct {
	cut       bool // accepts CutLowest / CutHighest
	threshold bool // accepts Threshold
}

// tiebreaks is the single list of supported tiebreak keys.
var tiebreaks = map[TiebreakKey]tiebreakInfo{
	Buchholz:                     {cut: true},
	ForeBuchholz:                 {cut: true},
	AverageOpponentBuchholz:      {cut: true},
	SonnebornBerger:              {},
	KoyaSystem:                   {threshold: true},
	DirectEncounter:              {},
	AverageRatingOfOpponents:     {},
	TournamentPerformanceRating:  {},
	PerfectTournamentPerformance: {},
	AveragePerfectPerformanceRatingOfOpponents: {},
	AveragePerfectPerformanceOfOpponents:       {},
	Wins:                                       {},
	GamesWon:                                   {},
	GamesPlayedBlack:                           {},
	WinsWithBlack:                              {},
	ProgressiveScores:                          {},
	GamesElectedToPlay:                         {},
	MatchPoints:                                {},
	GamePoints:                                 {},
	ExtendedMatchMatchSB:                       {},
	ExtendedMatchGameSB:                        {},
	ExtendedGameMatchSB:                        {},
	ExtendedGameGameSB:                         {},
	BoardCount:                                 {},
	TopBoardResults:                            {},
}

// IsTiebreakKey reports whether key is a supported tiebreak.
func IsTiebreakKey(key TiebreakKey) bool {
	_, ok := tiebreaks[key]
	return ok
}

// SupportsCut reports whether the tiebreak accepts CutLowest / CutHighest.
func SupportsCut(key TiebreakKey) bool { return tiebreaks[key].cut }

// SupportsThreshold reports whether the tiebreak accepts Threshold.
func SupportsThreshold(key TiebreakKey) bool { return tiebreaks[key].threshold }

// TiebreakKeys returns all supported tiebreak keys in alphabetical order.
func TiebreakKeys() []TiebreakKey {
	keys := make([]TiebreakKey, 0, len(tiebreaks))
	for k := range tiebreaks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// computeTiebreak returns one player's value for a per-player tiebreak and
// whether the key is handled. Unhandled keys (DirectEncounter, unknown keys)
// return (0, false); callers store 0 for them.
func computeTiebreak[ID comparable](
	cfg TiebreakConfig,
	playerID ID,
	allInfo map[ID]*playerPairingInfo[ID],
	rawScores map[ID]float64,
	ratings map[ID]int,
) (float64, bool) {
	switch cfg.Key {
	case Buchholz:
		return computeBuchholz(playerID, allInfo, rawScores, cfg.CutLowest, cfg.CutHighest), true
	case ForeBuchholz:
		return computeForeBuchholz(playerID, allInfo, rawScores, cfg.CutLowest, cfg.CutHighest), true
	case AverageOpponentBuchholz:
		return computeAverageOpponentBuchholz(playerID, allInfo, rawScores, cfg.CutLowest, cfg.CutHighest), true
	case SonnebornBerger:
		return computeSonnebornBerger(playerID, allInfo, rawScores), true
	case KoyaSystem:
		return computeKoyaSystem(playerID, allInfo, rawScores, cfg.Threshold), true
	case AverageRatingOfOpponents:
		return computeAverageRatingOfOpponents(playerID, allInfo, ratings), true
	case TournamentPerformanceRating:
		return computeTournamentPerformanceRating(playerID, allInfo, ratings), true
	case PerfectTournamentPerformance:
		return computePerfectTournamentPerformance(playerID, allInfo, ratings), true
	case AveragePerfectPerformanceRatingOfOpponents:
		return computeAveragePerfectPerformanceRatingOfOpponents(playerID, allInfo, ratings), true
	case AveragePerfectPerformanceOfOpponents:
		return computeAveragePerfectPerformanceOfOpponents(playerID, allInfo, ratings), true
	case Wins:
		return computeNumberOfWins(playerID, allInfo, rawScores), true
	case GamesWon:
		return computeNumberOfGamesWon(playerID, allInfo, rawScores), true
	case GamesPlayedBlack:
		return computeNumberOfPlayedGamesBlack(playerID, allInfo, rawScores), true
	case WinsWithBlack:
		return computeNumberOfGamesWonBlack(playerID, allInfo, rawScores), true
	case ProgressiveScores:
		return computeSumOfProgressiveScores(playerID, allInfo, rawScores), true
	case GamesElectedToPlay:
		return computeGamesOneElectedToPlay(playerID, allInfo, rawScores), true
	}
	return 0, false
}
