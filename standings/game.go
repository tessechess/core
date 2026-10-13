// Package standings computes FIDE tiebreaks, standings and crosstable rows from
// the games of one section.
//
// Players are identified by a caller-defined ID type (e.g. a registration UUID
// in the backend). The package has no persistence: callers load the games,
// ratings, K-factors and tiebreak configuration and pass them in an Input.
package standings

import (
	"fmt"

	"github.com/ValkoM28/tournamentessentials/core/result"
)

// Game is one pairing of a round, as seen by the standings calculation.
type Game[ID comparable] struct {
	Round  int
	White  ID
	Black  *ID          // nil for a bye
	Result *result.Code // nil if not yet played
}

// playerPairingInfo holds all games for one player and a derived field
// used for the FIDE Article 16.3.2 withdrawal adjustment.
type playerPairingInfo[ID comparable] struct {
	// pairings is every round this player appears in (as white or black),
	// in arbitrary order.
	pairings []*Game[ID]

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
	Key        string
	CutLowest  int
	CutHighest int
	Threshold  float64
}

// MapKey returns the unique string key used in PlayerScore.Tiebreaks and for
// sorting. For cut-based algorithms the cut parameters are encoded into the key
// so that BH and BH-C1 can coexist in the same tiebreak list without collision.
func (c TiebreakConfig) MapKey() string {
	if !SupportsCut(c.Key) {
		return c.Key
	}
	if c.CutLowest == 0 && c.CutHighest == 0 {
		return c.Key
	}
	if c.CutHighest == 0 {
		return fmt.Sprintf("%s_cut%d", c.Key, c.CutLowest)
	}
	if c.CutLowest == c.CutHighest {
		return fmt.Sprintf("%s_median%d", c.Key, c.CutLowest)
	}
	return fmt.Sprintf("%s_c%d_c%d", c.Key, c.CutLowest, c.CutHighest)
}
