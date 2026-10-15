package pairing

import "sort"

// StartingOrder returns the players' IDs in starting-rank order (index + 1 is
// the starting rank), following FIDE's initial ranking:
//
//  1. rating, highest first; unrated players (rating 0) after all rated ones;
//  2. title, in player.Title.Order;
//  3. alphabetically by "Lastname, Firstname".
//
// Players equal on all three keep their input order. players is not modified.
//
// Only ID, Rating, Title, LastName and FirstName are read. StartingRank is
// ignored (it is what this function decides), and Rating is simply "the rating
// to rank by", so a caller may pass an effective rating such as a display
// rating.
func StartingOrder[ID comparable](players []Player[ID]) []ID {
	sorted := append([]Player[ID](nil), players...)
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
