// Package player defines the small value types that describe a player and are
// shared by trf, pairing and category: sex and FIDE title.
//
// Sex is "M" or "F" (or empty when not recorded) and a title is its FIDE
// abbreviation ("GM", "WIM", ...). Each package that writes a file format
// converts the values to that format itself (trf writes sex as "m"/"w").
package player

// Sex is a player's recorded sex.
type Sex string

const (
	// SexUnknown means the sex is not recorded. It is not a valid sex (see
	// Sex.Valid): conditions on sex never match it.
	SexUnknown Sex = ""
	// SexMale is stored as "M" and written to TRF files as "m".
	SexMale Sex = "M"
	// SexFemale is stored as "F" and written to TRF files as "w".
	SexFemale Sex = "F"
)

// Valid reports whether s is SexMale or SexFemale. SexUnknown is not valid: it
// means "not recorded", not a third value.
func (s Sex) Valid() bool { return s == SexMale || s == SexFemale }

// Title is a player's chess title.
type Title string

// FIDE titles, in FIDE's initial-ranking order (see Title.Order).
const (
	NoTitle Title = ""
	GM      Title = "GM"
	IM      Title = "IM"
	WGM     Title = "WGM"
	FM      Title = "FM"
	WIM     Title = "WIM"
	CM      Title = "CM"
	WFM     Title = "WFM"
	WCM     Title = "WCM"
)

// Arena titles are online-only titles. They are not FIDE titles: they rank as
// untitled and are not written to TRF files.
const (
	AGM Title = "AGM"
	AIM Title = "AIM"
	AFM Title = "AFM"
	ACM Title = "ACM"
)

// untitled is Order's value for players without a FIDE title.
const untitled = 8

// fideOrder lists the FIDE titles in the order used for the initial ranking.
var fideOrder = map[Title]int{
	GM: 0, IM: 1, WGM: 2, FM: 3, WIM: 4, CM: 5, WFM: 6, WCM: 7,
}

// FIDE reports whether t is one of the eight FIDE titles.
func (t Title) FIDE() bool {
	_, ok := fideOrder[t]
	return ok
}

// Order returns the position of t in FIDE's initial-ranking order: GM, IM,
// WGM, FM, WIM, CM, WFM, WCM (0..7), then untitled (8). Arena titles and any
// other value rank as untitled.
func (t Title) Order() int {
	if o, ok := fideOrder[t]; ok {
		return o
	}
	return untitled
}
