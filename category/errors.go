package category

import "errors"

var (
	ErrInvalidRatingRange = errors.New("min_rating must be less than or equal to max_rating")
	ErrInvalidDateRange   = errors.New("birthday_after must be before birthday_before")
)
