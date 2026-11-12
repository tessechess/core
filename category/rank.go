package category

import "sort"

// Placing is a player's place in a category's prize list.
type Placing[ID comparable] struct {
	ID           ID  // the player's ID
	SectionRank  int // rank in the section standings
	CategoryRank int // 1 = best player of the category
}

// RankWithin orders a category's members by their section rank and numbers
// them 1..n. Members without a section rank (not in the standings) are left out.
func RankWithin[ID comparable](members []ID, sectionRank map[ID]int) []Placing[ID] {
	placings := make([]Placing[ID], 0, len(members))
	for _, id := range members {
		rank, ok := sectionRank[id]
		if !ok {
			continue
		}
		placings = append(placings, Placing[ID]{ID: id, SectionRank: rank})
	}
	sort.Slice(placings, func(i, j int) bool {
		return placings[i].SectionRank < placings[j].SectionRank
	})
	for i := range placings {
		placings[i].CategoryRank = i + 1
	}
	return placings
}
