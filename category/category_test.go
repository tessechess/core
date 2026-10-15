package category_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ValkoM28/tournamentessentials/core/category"
	"github.com/ValkoM28/tournamentessentials/core/player"
)

func intPtr(v int) *int { return &v }
func date(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}
func gender(g category.Gender) *category.Gender { return &g }

func TestMatches(t *testing.T) {
	born2015 := date(2015, 6, 1)
	cases := []struct {
		name string
		c    category.Criteria
		p    category.Candidate
		want bool
	}{
		{"no conditions", category.Criteria{}, category.Candidate{}, true},

		// Rating
		{"min rating met", category.Criteria{MinRating: intPtr(1600)}, category.Candidate{Rating: 1600}, true},
		{"min rating missed", category.Criteria{MinRating: intPtr(1600)}, category.Candidate{Rating: 1599}, false},
		{"max rating met", category.Criteria{MaxRating: intPtr(1600)}, category.Candidate{Rating: 1600}, true},
		{"max rating exceeded", category.Criteria{MaxRating: intPtr(1600)}, category.Candidate{Rating: 1601}, false},
		{"unrated is under max rating (intended)", category.Criteria{MaxRating: intPtr(1600)}, category.Candidate{Rating: 0}, true},
		{"unrated fails min rating", category.Criteria{MinRating: intPtr(1000)}, category.Candidate{Rating: 0}, false},

		// Birthday (strict comparisons)
		{"born before cutoff", category.Criteria{BirthdayBefore: date(2016, 1, 1)}, category.Candidate{Birthday: born2015}, true},
		{"born on 'before' cutoff", category.Criteria{BirthdayBefore: date(2015, 6, 1)}, category.Candidate{Birthday: born2015}, false},
		{"born after cutoff", category.Criteria{BirthdayAfter: date(2014, 1, 1)}, category.Candidate{Birthday: born2015}, true},
		{"born on 'after' cutoff", category.Criteria{BirthdayAfter: date(2015, 6, 1)}, category.Candidate{Birthday: born2015}, false},
		{"unknown birthday fails birthday condition", category.Criteria{BirthdayAfter: date(2014, 1, 1)}, category.Candidate{}, false},
		{"unknown birthday is fine without birthday condition", category.Criteria{MaxRating: intPtr(2000)}, category.Candidate{Rating: 1500}, true},

		// Gender × sex
		{"female in women's category", category.Criteria{Gender: gender(category.GenderFemale)}, category.Candidate{Sex: player.SexFemale}, true},
		{"male in women's category", category.Criteria{Gender: gender(category.GenderFemale)}, category.Candidate{Sex: player.SexMale}, false},
		{"male in men's category", category.Criteria{Gender: gender(category.GenderMale)}, category.Candidate{Sex: player.SexMale}, true},
		{"female in men's category", category.Criteria{Gender: gender(category.GenderMale)}, category.Candidate{Sex: player.SexFemale}, false},
		{"unknown sex fails gender condition", category.Criteria{Gender: gender(category.GenderFemale)}, category.Candidate{}, false},

		// Combined
		{"U16 girls under 1600", category.Criteria{BirthdayAfter: date(2010, 1, 1), Gender: gender(category.GenderFemale), MaxRating: intPtr(1600)},
			category.Candidate{Birthday: born2015, Sex: player.SexFemale, Rating: 1450}, true},
		{"U16 girls under 1600, rating too high", category.Criteria{BirthdayAfter: date(2010, 1, 1), Gender: gender(category.GenderFemale), MaxRating: intPtr(1600)},
			category.Candidate{Birthday: born2015, Sex: player.SexFemale, Rating: 1700}, false},
		{"ManualOnly is not checked by Matches", category.Criteria{ManualOnly: true}, category.Candidate{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.c.Matches(c.p); got != c.want {
				t.Errorf("Matches = %v, want %v", got, c.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		c    category.Criteria
		want error
	}{
		{"empty", category.Criteria{}, nil},
		{"min equals max", category.Criteria{MinRating: intPtr(1600), MaxRating: intPtr(1600)}, nil},
		{"min above max", category.Criteria{MinRating: intPtr(1700), MaxRating: intPtr(1600)}, category.ErrInvalidRatingRange},
		{"dates in order", category.Criteria{BirthdayAfter: date(2010, 1, 1), BirthdayBefore: date(2012, 1, 1)}, nil},
		{"dates equal", category.Criteria{BirthdayAfter: date(2010, 1, 1), BirthdayBefore: date(2010, 1, 1)}, category.ErrInvalidDateRange},
		{"dates reversed", category.Criteria{BirthdayAfter: date(2012, 1, 1), BirthdayBefore: date(2010, 1, 1)}, category.ErrInvalidDateRange},
		{"rating checked before dates", category.Criteria{MinRating: intPtr(2), MaxRating: intPtr(1),
			BirthdayAfter: date(2012, 1, 1), BirthdayBefore: date(2010, 1, 1)}, category.ErrInvalidRatingRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.c.Validate(); !errors.Is(err, c.want) || (c.want == nil && err != nil) {
				t.Errorf("Validate = %v, want %v", err, c.want)
			}
		})
	}
	if got := category.ErrInvalidRatingRange.Error(); got != "min_rating must be less than or equal to max_rating" {
		t.Errorf("rating message changed: %q", got)
	}
	if got := category.ErrInvalidDateRange.Error(); got != "birthday_after must be before birthday_before" {
		t.Errorf("date message changed: %q", got)
	}
}

func TestAssign(t *testing.T) {
	rules := []category.Rule[string]{
		{ID: "u1600", Criteria: category.Criteria{MaxRating: intPtr(1600)}},
		{ID: "women", Criteria: category.Criteria{Gender: gender(category.GenderFemale)}},
		{ID: "manual", Criteria: category.Criteria{ManualOnly: true}},
		{ID: "all", Criteria: category.Criteria{}},
		{ID: "o2000", Criteria: category.Criteria{MinRating: intPtr(2000)}},
	}
	subjects := []category.Subject[int]{
		{ID: 1, Candidate: category.Candidate{Sex: player.SexFemale, Rating: 1500}},
		{ID: 2, Candidate: category.Candidate{Sex: player.SexMale, Rating: 1800}},
	}
	got := category.Assign(rules, subjects)
	want := map[int][]string{
		1: {"u1600", "women", "all"}, // rule order kept, ManualOnly skipped
		2: {"all"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Assign = %v, want %v", got, want)
	}

	// A subject matching nothing gets a nil entry (not missing, not empty).
	none := category.Assign([]category.Rule[string]{{ID: "o2000", Criteria: category.Criteria{MinRating: intPtr(2000)}}},
		[]category.Subject[int]{{ID: 3, Candidate: category.Candidate{Rating: 1000}}})
	v, ok := none[3]
	if !ok || v != nil {
		t.Errorf("no match: entry present=%v value=%#v, want present and nil", ok, v)
	}

	if got := category.Assign[string, int](nil, nil); len(got) != 0 {
		t.Errorf("no subjects: got %v", got)
	}
}

func TestRankWithin(t *testing.T) {
	members := []string{"c", "a", "x", "b"} // x has no section rank
	sectionRank := map[string]int{"a": 7, "b": 2, "c": 12, "z": 1}

	got := category.RankWithin(members, sectionRank)
	want := []category.Placing[string]{
		{ID: "b", SectionRank: 2, CategoryRank: 1},
		{ID: "a", SectionRank: 7, CategoryRank: 2},
		{ID: "c", SectionRank: 12, CategoryRank: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RankWithin = %+v, want %+v", got, want)
	}

	if got := category.RankWithin([]string{"x"}, sectionRank); len(got) != 0 {
		t.Errorf("unranked members only: got %+v, want empty", got)
	}
}
