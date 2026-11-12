package category

import "errors"

// Errors returned by Criteria.Validate. Test for them with errors.Is.
var (
	// ErrInvalidRatingRange means MinRating is greater than MaxRating.
	ErrInvalidRatingRange = errors.New("min_rating must be less than or equal to max_rating")
	// ErrInvalidDateRange means BirthdayAfter is not before BirthdayBefore.
	ErrInvalidDateRange = errors.New("birthday_after must be before birthday_before")
	// ErrInvalidSex means Sex is set to something other than player.SexMale or
	// player.SexFemale.
	ErrInvalidSex = errors.New("sex must be 'M' or 'F'")
)
