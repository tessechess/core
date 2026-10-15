package pairing

import (
	"fmt"

	"github.com/ValkoM28/tournamentessentials/core/round"
)

// Validate checks the settings on their own: allowed values, fixed boards,
// duplicates and players placed in conflicting lists. Whether the referenced
// players belong to the section is the caller's check (see ReferencedPlayers).
// Errors wrap ErrInvalidSettings or ErrPlayerConflict.
func (s Settings[ID]) Validate() error {
	// Validate enum values
	if s.FirstPlayerColor != "" && s.FirstPlayerColor != round.ColorWhite && s.FirstPlayerColor != round.ColorBlack {
		return fmt.Errorf("%w: first_player_color must be 'white' or 'black'", ErrInvalidSettings)
	}
	if s.Acceleration != "" && s.Acceleration != AccelerationNone && s.Acceleration != AccelerationBaku && s.Acceleration != AccelerationDutch {
		return fmt.Errorf("%w: acceleration must be 'none', 'baku', or 'dutch'", ErrInvalidSettings)
	}

	// Validate forbidden pairings
	for _, fp := range s.ForbiddenPairings {
		if fp.Player1 == fp.Player2 {
			return fmt.Errorf("%w: forbidden pairing cannot have the same player on both sides", ErrInvalidSettings)
		}
	}

	// Validate fixed boards
	boardNumbers := make(map[int]bool)
	fixedPlayers := make(map[ID]bool)
	for _, fb := range s.FixedBoards {
		if fb.BoardNumber < 1 {
			return fmt.Errorf("%w: fixed board number must be >= 1", ErrInvalidSettings)
		}
		if boardNumbers[fb.BoardNumber] {
			return fmt.Errorf("%w: duplicate fixed board number %d", ErrInvalidSettings, fb.BoardNumber)
		}
		boardNumbers[fb.BoardNumber] = true
		if fixedPlayers[fb.Player] {
			return fmt.Errorf("%w: player appears in multiple fixed boards", ErrInvalidSettings)
		}
		fixedPlayers[fb.Player] = true
	}

	// Check no duplicates within each list
	if hasDuplicates(s.WithdrawnPlayers) {
		return fmt.Errorf("%w: duplicate player in withdrawn list", ErrInvalidSettings)
	}
	if hasDuplicates(s.HalfPointByes) {
		return fmt.Errorf("%w: duplicate player in half-point bye list", ErrInvalidSettings)
	}
	if hasDuplicates(s.LateEntries) {
		return fmt.Errorf("%w: duplicate player in late entries list", ErrInvalidSettings)
	}

	// Check no overlap between withdrawn, half-bye, and fixed-board players
	withdrawn := toSet(s.WithdrawnPlayers)
	halfByes := toSet(s.HalfPointByes)

	for id := range halfByes {
		if withdrawn[id] {
			return fmt.Errorf("%w: player cannot be both withdrawn and on half-point bye", ErrPlayerConflict)
		}
	}
	for id := range fixedPlayers {
		if withdrawn[id] {
			return fmt.Errorf("%w: player cannot be both withdrawn and on a fixed board", ErrPlayerConflict)
		}
		if halfByes[id] {
			return fmt.Errorf("%w: player cannot be both on half-point bye and on a fixed board", ErrPlayerConflict)
		}
	}

	return nil
}

// ReferencedPlayers returns every player the settings mention (withdrawn,
// half-point byes, late entries, forbidden pairs, fixed boards), each once, in
// first-seen order. Callers use it to check the players belong to the section.
func (s Settings[ID]) ReferencedPlayers() []ID {
	var ids []ID
	seen := make(map[ID]bool)
	add := func(id ID) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range s.WithdrawnPlayers {
		add(id)
	}
	for _, id := range s.HalfPointByes {
		add(id)
	}
	for _, id := range s.LateEntries {
		add(id)
	}
	for _, fp := range s.ForbiddenPairings {
		add(fp.Player1)
		add(fp.Player2)
	}
	for _, fb := range s.FixedBoards {
		add(fb.Player)
	}
	return ids
}

func hasDuplicates[ID comparable](ids []ID) bool {
	seen := make(map[ID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return true
		}
		seen[id] = true
	}
	return false
}

func toSet[ID comparable](ids []ID) map[ID]bool {
	s := make(map[ID]bool, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}
