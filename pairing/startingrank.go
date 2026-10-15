package pairing

import (
	"sort"

	"github.com/ValkoM28/tournamentessentials/core/player"
)

// Entrant is a player to be given a starting rank (pairing number).
type Entrant[ID comparable] struct {
	ID        ID
	Rating    int // 0 = unrated
	Title     player.Title
	LastName  string
	FirstName string
}

// StartingOrder returns the players' IDs in starting-rank order (index + 1 is
// the starting rank), following FIDE's initial ranking:
//
//  1. rating, highest first; unrated players (rating 0) after all rated ones;
//  2. title, in player.Title.Order;
//  3. alphabetically by "Lastname, Firstname".
//
// Players equal on all three keep their input order. players is not modified.
func StartingOrder[ID comparable](players []Entrant[ID]) []ID {
	sorted := append([]Entrant[ID](nil), players...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Rating != b.Rating {
			if a.Rating == 0 {
				return false
			}
			if b.Rating == 0 {
				return true
			}
			return a.Rating > b.Rating
		}
		if ta, tb := a.Title.Order(), b.Title.Order(); ta != tb {
			return ta < tb
		}
		return a.LastName+", "+a.FirstName < b.LastName+", "+b.FirstName
	})

	ids := make([]ID, len(sorted))
	for i, p := range sorted {
		ids[i] = p.ID
	}
	return ids
}
