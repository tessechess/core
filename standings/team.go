package standings

import (
	"sort"

	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
)

// TeamScoreType selects the primary score used to rank teams.
type TeamScoreType string

const (
	// TeamScoreMatchPoints ranks teams primarily by match points. It is the
	// default when PrimaryScore is empty.
	TeamScoreMatchPoints TeamScoreType = "match_points"
	// TeamScoreGamePoints ranks teams primarily by total board game points.
	TeamScoreGamePoints TeamScoreType = "game_points"
)

// MatchPointsConfig customizes match points and bye awards in a team
// tournament. A nil *MatchPointsConfig uses FIDE C.06 defaults:
//   - Win: 2.0, Draw: 1.0, Loss: 0.0
//   - Full-point bye: Win match points (2.0) and 50% of BoardsPerMatch game points
//   - Half-point bye: Draw match points (1.0) and 50% of BoardsPerMatch game points
//   - Zero-point bye: 0.0 match points and 0.0 game points
type MatchPointsConfig struct {
	Win             float64  // match points for a match win (default 2.0 when Win == 0 and Draw == 0)
	Draw            float64  // match points for a match draw (default 1.0 when Win == 0 and Draw == 0)
	Loss            float64  // match points for a match loss (default 0.0)
	FullByeMatchPts *float64 // nil = Win (default 2.0)
	FullByeGamePts  *float64 // nil = 0.5 * BoardsPerMatch
	HalfByeMatchPts *float64 // nil = Draw (default 1.0)
	HalfByeGamePts  *float64 // nil = 0.5 * BoardsPerMatch
	ZeroByeMatchPts float64  // default 0.0
	ZeroByeGamePts  float64  // default 0.0
}

// resolvedTeamScoring holds the effective match-point and bye values after
// applying defaults.
type resolvedTeamScoring struct {
	win            float64
	draw           float64
	loss           float64
	fullByeMP      float64
	fullByeGP      float64
	halfByeMP      float64
	halfByeGP      float64
	zeroByeMP      float64
	zeroByeGP      float64
	boardsPerMatch int
	primary        TeamScoreType
}

func resolveTeamScoring(cfg *MatchPointsConfig, boardsPerMatch int, primary TeamScoreType) resolvedTeamScoring {
	if primary != TeamScoreGamePoints {
		primary = TeamScoreMatchPoints
	}
	halfBoards := 0.5 * float64(boardsPerMatch)
	if cfg == nil {
		return resolvedTeamScoring{
			win:            2.0,
			draw:           1.0,
			loss:           0.0,
			fullByeMP:      2.0,
			fullByeGP:      halfBoards,
			halfByeMP:      1.0,
			halfByeGP:      halfBoards,
			zeroByeMP:      0.0,
			zeroByeGP:      0.0,
			boardsPerMatch: boardsPerMatch,
			primary:        primary,
		}
	}

	win, draw, loss := cfg.Win, cfg.Draw, cfg.Loss
	if win == 0 && draw == 0 {
		win = 2.0
		draw = 1.0
	}
	fullMP := win
	if cfg.FullByeMatchPts != nil {
		fullMP = *cfg.FullByeMatchPts
	}
	fullGP := halfBoards
	if cfg.FullByeGamePts != nil {
		fullGP = *cfg.FullByeGamePts
	}
	halfMP := draw
	if cfg.HalfByeMatchPts != nil {
		halfMP = *cfg.HalfByeMatchPts
	}
	halfGP := halfBoards
	if cfg.HalfByeGamePts != nil {
		halfGP = *cfg.HalfByeGamePts
	}

	return resolvedTeamScoring{
		win:            win,
		draw:           draw,
		loss:           loss,
		fullByeMP:      fullMP,
		fullByeGP:      fullGP,
		halfByeMP:      halfMP,
		halfByeGP:      halfGP,
		zeroByeMP:      cfg.ZeroByeMatchPts,
		zeroByeGP:      cfg.ZeroByeGamePts,
		boardsPerMatch: boardsPerMatch,
		primary:        primary,
	}
}

// TeamInput is everything needed to compute team standings for one section:
// the matches of the completed rounds, the number of boards per match, the
// scoring configuration, and the tiebreaks to apply.
//
// Only teams that appear in at least one match are ranked; a team with no
// match yet is not in the result.
type TeamInput[TeamID, PlayerID comparable] struct {
	Matches        []round.Match[TeamID, PlayerID] // matches of the completed rounds
	BoardsPerMatch int                             // boards per match (used for bye GP and BoardCount)
	PrimaryScore   TeamScoreType                   // empty defaults to TeamScoreMatchPoints
	Scoring        *MatchPointsConfig              // nil uses FIDE C.06 defaults
	Tiebreaks      []TiebreakConfig                // tiebreaks in order of significance; empty = ranking by primary score only
}

// TeamScore is one team's primary points, match points, game points, and
// tiebreak values.
type TeamScore[TeamID comparable] struct {
	ID          TeamID             // the team's ID, as in TeamInput.Matches
	Points      float64            // primary score (MatchPoints or GamePoints per PrimaryScore)
	MatchPoints float64            // total match points from all matches and byes
	GamePoints  float64            // total board game points from all matches and byes
	Tiebreaks   map[string]float64 // keyed by TiebreakConfig.MapKey()
}

// TeamStanding is a ranked TeamScore.
type TeamStanding[TeamID comparable] struct {
	Rank  int                // 1 = first place; consecutive, see RankTeams
	Score *TeamScore[TeamID] // the team's points and tiebreaks
}

// TeamRoundResult is one team's match in one round, for a team crosstable.
type TeamRoundResult struct {
	Round             int          // round number, 1-based
	OpponentRank      *int         // nil for bye or unplayed
	Home              bool         // true if this team was Home in the match
	WhiteOnFirstBoard bool         // true if this team had white on board 1
	Result            *result.Code // settled or derived match result ("1-0", "0.5-0.5", "0-1", etc.); nil if no pairing
	MatchPoints       float64      // match points the team scored in the match
	GamePoints        float64      // board game points the team scored in the match
	IsForfeit         bool         // the match was decided by forfeit
	IsBye             bool         // paired, but no away team
	Played            bool         // false if the team had no pairing this round
}

// TeamCrosstableRow is one ranked team with its per-round results.
type TeamCrosstableRow[TeamID comparable] struct {
	Rank   int                // the team's rank, as in TeamStanding
	Score  *TeamScore[TeamID] // the team's points and tiebreaks
	Rounds []TeamRoundResult  // one per round, ordered by round number
}

// TeamCrosstableResult is a team ranking crosstable: one row per team and the
// number of rounds the rows' Rounds cover.
type TeamCrosstableResult[TeamID comparable] struct {
	// TotalRounds is the highest round number among the matches; every row has
	// that many round cells (rounds without matches are marked not played).
	TotalRounds int
	// Rows has one row per team, in standings order.
	Rows []TeamCrosstableRow[TeamID]
}

// teamPairingInfo holds all matches for one team and derived fields used by
// team standings and tiebreaks.
type teamPairingInfo[TeamID, PlayerID comparable] struct {
	matches       []*round.Match[TeamID, PlayerID]
	lastRealRound int
}

// teamRawTotals holds a team's accumulated MP and GP across all rounds.
type teamRawTotals struct {
	mp float64
	gp float64
}

func (t teamRawTotals) primary(primary TeamScoreType) float64 {
	if primary == TeamScoreGamePoints {
		return t.gp
	}
	return t.mp
}

// TeamScores returns the primary points, match points, game points and
// tiebreaks of every team that appears in a match, in no particular order.
// Group-dependent tiebreaks (DirectEncounter, TopBoardResults) are left at 0
// here; RankTeams fills them in.
func TeamScores[TeamID, PlayerID comparable](in TeamInput[TeamID, PlayerID]) []*TeamScore[TeamID] {
	scores, _, _, _ := teamScoresWithInfo(in)
	return scores
}

// RankTeams returns the team standings: scores with group-dependent tiebreaks
// (DirectEncounter, TopBoardResults) filled in, sorted by Points and then by
// the configured tiebreaks, in order, highest first. Ranks are consecutive
// (1, 2, 3, ...). Only teams that appear in a match are included.
func RankTeams[TeamID, PlayerID comparable](in TeamInput[TeamID, PlayerID]) []*TeamStanding[TeamID] {
	standings, _ := rankTeamsWithInfo(in)
	return standings
}

// TeamCrosstable returns the team ranking crosstable: one row per team in
// standings order (see RankTeams), each with one cell per round from 1 to
// TotalRounds, the highest round number among the matches.
func TeamCrosstable[TeamID, PlayerID comparable](in TeamInput[TeamID, PlayerID]) TeamCrosstableResult[TeamID] {
	sc := resolveTeamScoring(in.Scoring, in.BoardsPerMatch, in.PrimaryScore)
	standings, allInfo := rankTeamsWithInfo(in)

	rankMap := make(map[TeamID]int, len(standings))
	for _, st := range standings {
		rankMap[st.Score.ID] = st.Rank
	}

	totalRounds := 0
	for _, info := range allInfo {
		for _, m := range info.matches {
			if m.Round > totalRounds {
				totalRounds = m.Round
			}
		}
	}

	rows := make([]TeamCrosstableRow[TeamID], len(standings))
	for i, st := range standings {
		id := st.Score.ID
		rows[i] = TeamCrosstableRow[TeamID]{
			Rank:   st.Rank,
			Score:  st.Score,
			Rounds: buildTeamRoundResults(id, allInfo[id], rankMap, totalRounds, sc),
		}
	}
	return TeamCrosstableResult[TeamID]{TotalRounds: totalRounds, Rows: rows}
}

func teamScoresWithInfo[TeamID, PlayerID comparable](
	in TeamInput[TeamID, PlayerID],
) (
	[]*TeamScore[TeamID],
	map[TeamID]*teamPairingInfo[TeamID, PlayerID],
	map[TeamID]teamRawTotals,
	resolvedTeamScoring,
) {
	sc := resolveTeamScoring(in.Scoring, in.BoardsPerMatch, in.PrimaryScore)
	matches := make([]*round.Match[TeamID, PlayerID], len(in.Matches))
	for i := range in.Matches {
		matches[i] = &in.Matches[i]
	}

	totals := buildTeamRawTotals(matches, sc)
	allInfo := buildTeamPairingInfo(matches)

	scores := make([]*TeamScore[TeamID], 0, len(totals))
	for teamID, tot := range totals {
		tb := make(map[string]float64, len(in.Tiebreaks))
		for _, cfg := range in.Tiebreaks {
			tb[cfg.MapKey()], _ = computeTeamTiebreak(cfg, teamID, allInfo, totals, sc)
		}
		scores = append(scores, &TeamScore[TeamID]{
			ID:          teamID,
			Points:      tot.primary(sc.primary),
			MatchPoints: tot.mp,
			GamePoints:  tot.gp,
			Tiebreaks:   tb,
		})
	}
	return scores, allInfo, totals, sc
}

func rankTeamsWithInfo[TeamID, PlayerID comparable](
	in TeamInput[TeamID, PlayerID],
) ([]*TeamStanding[TeamID], map[TeamID]*teamPairingInfo[TeamID, PlayerID]) {
	scores, allInfo, totals, sc := teamScoresWithInfo(in)
	fillTeamGroupTiebreaks(scores, allInfo, totals, sc, in.Tiebreaks)
	return sortAndRankTeams(scores, in.Tiebreaks), allInfo
}

func sortAndRankTeams[TeamID comparable](scores []*TeamScore[TeamID], tiebreakCfgs []TiebreakConfig) []*TeamStanding[TeamID] {
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

	standings := make([]*TeamStanding[TeamID], len(scores))
	for i, ts := range scores {
		standings[i] = &TeamStanding[TeamID]{Rank: i + 1, Score: ts}
	}
	return standings
}

// matchTeamPoints returns the (MatchPoints, GamePoints) awarded to Home and
// Away for a single match under sc, along with the effective match result code
// (from Home vs Away perspective).
func matchTeamPoints[TeamID, PlayerID comparable](
	m *round.Match[TeamID, PlayerID],
	sc resolvedTeamScoring,
) (homeMP, awayMP, homeGP, awayGP float64, effResult result.Code) {
	if m.Result != nil && *m.Result == result.Void {
		return 0, 0, 0, 0, result.Void
	}

	// Team bye (Away is nil).
	if m.Away == nil {
		if m.Result == nil || !m.Result.Counts() {
			return 0, 0, 0, 0, result.Unplayed
		}
		switch *m.Result {
		case result.WhiteWins, result.WhiteForfeit:
			return sc.fullByeMP, 0, sc.fullByeGP, 0, *m.Result
		case result.WhiteHalf, result.Draw:
			return sc.halfByeMP, 0, sc.halfByeGP, 0, *m.Result
		default:
			return sc.zeroByeMP, 0, sc.zeroByeGP, 0, *m.Result
		}
	}

	// Played or forfeited match against an opponent (Away != nil).
	if m.Result != nil {
		if !m.Result.Counts() {
			return 0, 0, 0, 0, *m.Result
		}
		r := *m.Result
		hGP, aGP := m.GamePoints()
		hasPlayedBoards := len(m.Boards) > 0 && (hGP > 0 || aGP > 0 || m.Settled())
		fullBoards := float64(sc.boardsPerMatch)
		switch r {
		case result.WhiteWins, result.WhiteForfeit:
			if !hasPlayedBoards {
				hGP, aGP = fullBoards, 0
			}
			return sc.win, sc.loss, hGP, aGP, r
		case result.BlackWins, result.BlackForfeit:
			if !hasPlayedBoards {
				hGP, aGP = 0, fullBoards
			}
			return sc.loss, sc.win, hGP, aGP, r
		case result.Draw:
			if !hasPlayedBoards {
				hGP, aGP = 0.5*fullBoards, 0.5*fullBoards
			}
			return sc.draw, sc.draw, hGP, aGP, r
		case result.WhiteHalf:
			if !hasPlayedBoards {
				hGP, aGP = 0.5*fullBoards, 0
			}
			return sc.draw, sc.loss, hGP, aGP, r
		case result.BlackHalf:
			if !hasPlayedBoards {
				hGP, aGP = 0, 0.5*fullBoards
			}
			return sc.loss, sc.draw, hGP, aGP, r
		default:
			return sc.loss, sc.loss, hGP, aGP, r
		}
	}

	// m.Result == nil: outcome is determined from m.Boards.
	hGP, aGP := m.GamePoints()
	if !m.Settled() {
		return 0, 0, hGP, aGP, result.Unplayed
	}
	switch {
	case hGP > aGP:
		return sc.win, sc.loss, hGP, aGP, result.WhiteWins
	case hGP < aGP:
		return sc.loss, sc.win, hGP, aGP, result.BlackWins
	default:
		return sc.draw, sc.draw, hGP, aGP, result.Draw
	}
}

func buildTeamRawTotals[TeamID, PlayerID comparable](
	matches []*round.Match[TeamID, PlayerID],
	sc resolvedTeamScoring,
) map[TeamID]teamRawTotals {
	totals := make(map[TeamID]teamRawTotals)
	for _, m := range matches {
		hMP, aMP, hGP, aGP, _ := matchTeamPoints(m, sc)
		ht := totals[m.Home]
		ht.mp += hMP
		ht.gp += hGP
		totals[m.Home] = ht

		if m.Away != nil {
			at := totals[*m.Away]
			at.mp += aMP
			at.gp += aGP
			totals[*m.Away] = at
		}
	}
	return totals
}

func buildTeamPairingInfo[TeamID, PlayerID comparable](
	matches []*round.Match[TeamID, PlayerID],
) map[TeamID]*teamPairingInfo[TeamID, PlayerID] {
	allInfo := make(map[TeamID]*teamPairingInfo[TeamID, PlayerID])

	ensure := func(id TeamID) *teamPairingInfo[TeamID, PlayerID] {
		if allInfo[id] == nil {
			allInfo[id] = &teamPairingInfo[TeamID, PlayerID]{lastRealRound: -1}
		}
		return allInfo[id]
	}

	for _, m := range matches {
		hi := ensure(m.Home)
		hi.matches = append(hi.matches, m)

		if m.Away != nil {
			ai := ensure(*m.Away)
			ai.matches = append(ai.matches, m)

			if m.Round > hi.lastRealRound {
				hi.lastRealRound = m.Round
			}
			if m.Round > ai.lastRealRound {
				ai.lastRealRound = m.Round
			}
		}
	}
	return allInfo
}

func buildTeamRoundResults[TeamID, PlayerID comparable](
	teamID TeamID,
	info *teamPairingInfo[TeamID, PlayerID],
	rankMap map[TeamID]int,
	totalRounds int,
	sc resolvedTeamScoring,
) []TeamRoundResult {
	byRound := make(map[int]*round.Match[TeamID, PlayerID])
	if info != nil {
		for _, m := range info.matches {
			byRound[m.Round] = m
		}
	}

	results := make([]TeamRoundResult, totalRounds)
	for r := 1; r <= totalRounds; r++ {
		m, ok := byRound[r]
		if !ok {
			results[r-1] = TeamRoundResult{Round: r, Played: false}
			continue
		}

		hMP, aMP, hGP, aGP, effCode := matchTeamPoints(m, sc)
		resCode := effCode
		isHome := m.Home == teamID

		trr := TeamRoundResult{
			Round:  r,
			Home:   isHome,
			Result: &resCode,
			Played: true,
		}

		if isHome {
			trr.MatchPoints = hMP
			trr.GamePoints = hGP
		} else {
			trr.MatchPoints = aMP
			trr.GamePoints = aGP
		}

		if m.Away == nil {
			trr.IsBye = true
		} else if isHome {
			trr.WhiteOnFirstBoard = m.WhiteOnFirstBoard
			if rank, exists := rankMap[*m.Away]; exists {
				trr.OpponentRank = &rank
			}
		} else {
			trr.WhiteOnFirstBoard = !m.WhiteOnFirstBoard
			if rank, exists := rankMap[m.Home]; exists {
				trr.OpponentRank = &rank
			}
		}

		if isMatchForfeit(m) {
			trr.IsForfeit = true
		}

		results[r-1] = trr
	}
	return results
}

func isMatchForfeit[TeamID, PlayerID comparable](m *round.Match[TeamID, PlayerID]) bool {
	if m.Result != nil && m.Result.IsForfeit() {
		return true
	}
	if len(m.Boards) == 0 {
		return false
	}
	counted := 0
	for _, g := range m.Boards {
		if g.Result == nil || *g.Result == result.Void {
			continue
		}
		if !g.Result.IsForfeit() {
			return false
		}
		counted++
	}
	return counted > 0
}
