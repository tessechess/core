package category

// Rule pairs a category's ID with its criteria.
type Rule[CID comparable] struct {
	ID       CID      // the category's ID
	Criteria Criteria // what a player must satisfy to belong to the category
}

// Subject pairs a player's ID with what the rules need to know about them.
type Subject[SID comparable] struct {
	ID        SID       // the player's ID
	Candidate Candidate // the facts the rules are checked against
}

// Assign returns, for every subject, the IDs of the rules it matches, in the
// order the rules are given. ManualOnly rules are skipped. Every subject has an
// entry; it is nil when no rule matches.
func Assign[CID, SID comparable](rules []Rule[CID], subjects []Subject[SID]) map[SID][]CID {
	assignments := make(map[SID][]CID, len(subjects))
	for _, s := range subjects {
		var matched []CID
		for _, r := range rules {
			if r.Criteria.ManualOnly {
				continue
			}
			if r.Criteria.Matches(s.Candidate) {
				matched = append(matched, r.ID)
			}
		}
		assignments[s.ID] = matched
	}
	return assignments
}
