// Package category holds the rules for player categories (age, gender and
// rating groups such as "U12" or "Women"): which players a category matches,
// assigning categories to players, and ranking a category's players for its
// prize list.
//
// Categories and players are identified by caller-defined ID types. The
// package has no persistence.
package category

import "time"

// Sex is a player's recorded sex.
type Sex string

const (
	SexUnknown Sex = ""
	SexMale    Sex = "M"
	SexFemale  Sex = "F"
)

// Gender restricts a category to one sex.
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// Criteria is the rule data of a category. Every non-nil condition must hold.
type Criteria struct {
	BirthdayAfter  *time.Time // player must be born strictly after this date
	BirthdayBefore *time.Time // player must be born strictly before this date
	Gender         *Gender
	MinRating      *int
	MaxRating      *int
	ManualOnly     bool // never assigned automatically (see Assign)
}

// Candidate is what the rules need to know about a player.
type Candidate struct {
	Birthday *time.Time // nil when unknown
	Sex      Sex
	// Rating is 0 for an unrated player. An unrated player counts as below any
	// MaxRating (an unrated player is "under 1600") and fails any MinRating.
	Rating int
}

// Matches reports whether p satisfies every non-nil condition of c.
// A birthday condition with an unknown birthday does not match, nor does a
// gender condition with an unknown sex. ManualOnly is not checked here.
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

	if c.Gender != nil {
		switch p.Sex {
		case SexFemale:
			if *c.Gender == GenderMale {
				return false
			}
		case SexMale:
			if *c.Gender == GenderFemale {
				return false
			}
		default:
			return false
		}
	}

	return true
}

// Validate checks that the conditions are consistent: MinRating not above
// MaxRating, BirthdayAfter before BirthdayBefore.
func (c Criteria) Validate() error {
	if c.MinRating != nil && c.MaxRating != nil && *c.MinRating > *c.MaxRating {
		return ErrInvalidRatingRange
	}
	if c.BirthdayAfter != nil && c.BirthdayBefore != nil && !c.BirthdayAfter.Before(*c.BirthdayBefore) {
		return ErrInvalidDateRange
	}
	return nil
}
