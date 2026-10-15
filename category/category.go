// Package category holds the rules for player categories (age, sex and
// rating groups such as "U12" or "Women"): which players a category matches,
// assigning categories to players, and ranking a category's players for its
// prize list.
//
// Categories and players are identified by caller-defined ID types. The
// package has no persistence.
package category

import (
	"time"

	"github.com/ValkoM28/tournamentessentials/core/player"
)

// Criteria is the rule data of a category. Every non-nil condition must hold.
type Criteria struct {
	BirthdayAfter  *time.Time  // player must be born strictly after this date
	BirthdayBefore *time.Time  // player must be born strictly before this date
	Sex            *player.Sex // player must have this sex (player.SexMale or player.SexFemale)
	MinRating      *int
	MaxRating      *int
	ManualOnly     bool // never assigned automatically (see Assign)
}

// Candidate is what the rules need to know about a player.
type Candidate struct {
	Birthday *time.Time // nil when unknown
	Sex      player.Sex
	// Rating is 0 for an unrated player. An unrated player counts as below any
	// MaxRating (an unrated player is "under 1600") and fails any MinRating.
	Rating int
}

// Matches reports whether p satisfies every non-nil condition of c.
// A birthday condition with an unknown birthday does not match, nor does a
// sex condition with an unknown sex. ManualOnly is not checked here, and
// neither is the validity of c (see Validate); a sex condition that is not a
// valid sex matches nobody.
func (c Criteria) Matches(p Candidate) bool {
	// Rating conditions
	if c.MinRating != nil && p.Rating < *c.MinRating {
		return false
	}
	if c.MaxRating != nil && p.Rating > *c.MaxRating {
		return false
	}

	// Birthday conditions
	if c.BirthdayBefore != nil || c.BirthdayAfter != nil {
		if p.Birthday == nil {
			// Cannot evaluate date condition without birthday — skip this category.
			return false
		}
		if c.BirthdayBefore != nil && !p.Birthday.Before(*c.BirthdayBefore) {
			return false
		}
		if c.BirthdayAfter != nil && !p.Birthday.After(*c.BirthdayAfter) {
			return false
		}
	}

	if c.Sex != nil && (!c.Sex.Valid() || p.Sex != *c.Sex) {
		return false
	}

	return true
}

// Validate checks that the conditions are consistent: MinRating not above
// MaxRating, BirthdayAfter before BirthdayBefore, and Sex (if set) valid
// (player.Sex.Valid). Checks run in that order and the first failure is
// returned. Check criteria with Validate before using them: Matches does not.
func (c Criteria) Validate() error {
	if c.MinRating != nil && c.MaxRating != nil && *c.MinRating > *c.MaxRating {
		return ErrInvalidRatingRange
	}
	if c.BirthdayAfter != nil && c.BirthdayBefore != nil && !c.BirthdayAfter.Before(*c.BirthdayBefore) {
		return ErrInvalidDateRange
	}
	if c.Sex != nil && !c.Sex.Valid() {
		return ErrInvalidSex
	}
	return nil
}
