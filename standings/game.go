// Package standings computes FIDE tiebreaks, standings and crosstable rows from
// the games of one section.
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend). The package has no persistence: callers load the games,
// ratings, K-factors and tiebreak configuration and pass them in an Input.
//
// # Known limitations
//
// PTP and APPO can hang (#379). PerfectTournamentPerformance and
// AveragePerfectPerformanceOfOpponents search for a rating with a binary search
// that never ends for some inputs, for example when a player lost to an unrated
// opponent (rating 0). Scores, Rank and Crosstable then never return, and there
// is no timeout. Do not put these two keys in a tiebreak list until the issue
// is fixed.
//
// Individual competitions only. Team tiebreaks (game points, match points,
// #78) are not implemented; no TiebreakKey stands for them.
//
// Unrated players (rating 0 or missing from Input.Ratings) are treated
// differently by the rating-based tiebreaks:
//
//   - AverageRatingOfOpponents leaves unrated opponents out of the average (0
//     if none of the opponents is rated).
//   - TournamentPerformanceRating leaves games against unrated opponents out
//     of both the rating average and the score (0 if no opponent is rated);
//     AveragePerfectPerformanceRatingOfOpponents builds on it.
//   - PerfectTournamentPerformance and AveragePerfectPerformanceOfOpponents use
//     rating 0 as it is (see the hang above).
//   - RatingInfo.RatingChange is computed only for a rated player with a
//     K-factor. Known bug (#383): an unrated opponent counts as rating 0 in
//     the expected score, so a game against one changes the rating (a
//     1500-rated player with K=20 gets +0 for a win, -10 for a draw and -20
//     for a loss). A game against an unrated player should not change the
//     rating at all; until this is fixed, RatingChange is wrong for players
//     who met unrated opponents.
//
// The other tiebreaks (Buchholz and its variants, Sonneborn-Berger, Koya,
// direct encounter, wins, games won/played, progressive scores) do not use
// ratings.
package standings

import (
	"fmt"

	"github.com/tessechess/core/round"
)

// playerPairingInfo holds all games for one player and a derived field
// used for the FIDE Article 16.3.2 withdrawal adjustment.
type playerPairingInfo[ID comparable] struct {
	// pairings is every round this player appears in (as white or black),
	// in arbitrary order.
	pairings []*round.Game[ID]

	// lastRealRound is the highest round number where this player faced a
	// non-nil opponent. -1 if the player has never played a real game.
	lastRealRound int
}

// TiebreakConfig defines a single tiebreak entry with its parameters.
// Key identifies the algorithm (see TiebreakKeys); the optional fields tune its
// behaviour.
//
//   - CutLowest / CutHighest: number of lowest/highest opponent scores to
//     discard before summing (keys where SupportsCut is true).
//   - Threshold: minimum opponent score for inclusion (keys where
//     SupportsThreshold is true).
type TiebreakConfig struct {
	Key        TiebreakKey // the algorithm
	CutLowest  int         // opponent scores to drop from the bottom; 0 = none
	CutHighest int         // opponent scores to drop from the top; 0 = none
	Threshold  float64     // minimum opponent score; 0 = the tiebreak's default
}

// MapKey returns the unique string key used in PlayerScore.Tiebreaks and for
// sorting. For cut-based algorithms the cut parameters are encoded into the key
// so that BH and BH-C1 can coexist in the same tiebreak list without collision.
func (c TiebreakConfig) MapKey() string {
	if !SupportsCut(c.Key) {
		return string(c.Key)
	}
	if c.CutLowest == 0 && c.CutHighest == 0 {
		return string(c.Key)
	}
	if c.CutHighest == 0 {
		return fmt.Sprintf("%s_cut%d", c.Key, c.CutLowest)
	}
	if c.CutLowest == c.CutHighest {
		return fmt.Sprintf("%s_median%d", c.Key, c.CutLowest)
	}
	return fmt.Sprintf("%s_c%d_c%d", c.Key, c.CutLowest, c.CutHighest)
}
