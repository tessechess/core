package pairing

import "sort"

// Entrant is a player to be given a starting rank (pairing number).
type Entrant[ID comparable] struct {
	ID        ID
	Rating    int // 0 = unrated
	Title     string
	LastName  string
	FirstName string
}

// untitled is TitleOrder's value for players without a FIDE title.
const untitled = 8

// titleOrder lists the FIDE titles in the order used for the initial ranking.
var titleOrder = map[string]int{
	"GM": 0, "IM": 1, "WGM": 2, "FM": 3, "WIM": 4, "CM": 5, "WFM": 6, "WCM": 7,
}

// TitleOrder returns the position of a title in FIDE's initial-ranking order:
// GM, IM, WGM, FM, WIM, CM, WFM, WCM, then untitled. Arena titles (AGM, AIM,
// AFM, ACM) are online titles and rank as untitled, as does anything else.
func TitleOrder(title string) int {
	if o, ok := titleOrder[title]; ok {
		return o
	}
	return untitled
}

// StartingOrder returns the players' IDs in starting-rank order (index + 1 is
// the starting rank), following FIDE's initial ranking:
//
//  1. rating, highest first; unrated players (rating 0) after all rated ones;
//  2. title, in TitleOrder;
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
		if ta, tb := TitleOrder(a.Title), TitleOrder(b.Title); ta != tb {
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
