package pairing_test

import (
	"slices"
	"testing"

	"github.com/ValkoM28/tournamentessentials/core/pairing"
)

func TestTitleOrder(t *testing.T) {
	fide := []string{"GM", "IM", "WGM", "FM", "WIM", "CM", "WFM", "WCM"}
	for i, title := range fide {
		if got := pairing.TitleOrder(title); got != i {
			t.Errorf("TitleOrder(%q) = %d, want %d", title, got, i)
		}
	}
	untitled := pairing.TitleOrder("")
	if untitled <= pairing.TitleOrder("WCM") {
		t.Fatalf("untitled (%d) must rank after WCM", untitled)
	}
	for _, title := range []string{"AGM", "AIM", "AFM", "ACM", "NM", "gm", "unknown"} {
		if got := pairing.TitleOrder(title); got != untitled {
			t.Errorf("TitleOrder(%q) = %d, want untitled (%d)", title, got, untitled)
		}
	}
}

type entrant = pairing.Entrant[string]

func TestStartingOrder(t *testing.T) {
	cases := []struct {
		name    string
		players []entrant
		want    []string
	}{
		{"rating, highest first",
			[]entrant{{ID: "b", Rating: 1800}, {ID: "a", Rating: 2400}, {ID: "c", Rating: 2100}},
			[]string{"a", "c", "b"}},
		{"unrated after rated",
			[]entrant{{ID: "u", Rating: 0, LastName: "Aaron"}, {ID: "r", Rating: 1000, LastName: "Zed"}},
			[]string{"r", "u"}},
		{"title breaks rating ties in FIDE order",
			[]entrant{
				{ID: "none", Rating: 2400, LastName: "A"},
				{ID: "fm", Rating: 2400, Title: "FM", LastName: "B"},
				{ID: "wgm", Rating: 2400, Title: "WGM", LastName: "C"},
				{ID: "im", Rating: 2400, Title: "IM", LastName: "D"},
				{ID: "gm", Rating: 2400, Title: "GM", LastName: "E"},
				{ID: "wcm", Rating: 2400, Title: "WCM", LastName: "F"},
			},
			[]string{"gm", "im", "wgm", "fm", "wcm", "none"}},
		{"title does not beat rating",
			[]entrant{{ID: "gm", Rating: 2400, Title: "GM"}, {ID: "strong", Rating: 2500}},
			[]string{"strong", "gm"}},
		{"arena title ranks as untitled, then alphabetical",
			[]entrant{
				{ID: "agm", Rating: 2000, Title: "AGM", LastName: "Zimmer"},
				{ID: "plain", Rating: 2000, LastName: "Adams"},
				{ID: "cm", Rating: 2000, Title: "CM", LastName: "Young"},
			},
			[]string{"cm", "plain", "agm"}},
		{"name breaks rating and title ties",
			[]entrant{
				{ID: "zara", Rating: 2000, LastName: "Adams", FirstName: "Zara"},
				{ID: "bob", Rating: 2000, LastName: "Brown", FirstName: "Bob"},
				{ID: "alice", Rating: 2000, LastName: "Adams", FirstName: "Alice"},
			},
			[]string{"alice", "zara", "bob"}},
		{"unrated players: title then name",
			[]entrant{
				{ID: "u2", LastName: "Zed"},
				{ID: "u1", LastName: "Adams"},
				{ID: "wfm", Title: "WFM", LastName: "Young"},
			},
			[]string{"wfm", "u1", "u2"}},
		{"full ties keep input order",
			[]entrant{
				{ID: "first", Rating: 1500, LastName: "Same", FirstName: "Name"},
				{ID: "second", Rating: 1500, LastName: "Same", FirstName: "Name"},
				{ID: "third", Rating: 1500, LastName: "Same", FirstName: "Name"},
			},
			[]string{"first", "second", "third"}},
		{"empty", nil, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pairing.StartingOrder(c.players); !slices.Equal(got, c.want) {
				t.Errorf("StartingOrder = %v, want %v", got, c.want)
			}
		})
	}
}

func TestStartingOrder_DoesNotModifyInput(t *testing.T) {
	players := []entrant{{ID: "b", Rating: 1000}, {ID: "a", Rating: 2000}}
	pairing.StartingOrder(players)
	if players[0].ID != "b" || players[1].ID != "a" {
		t.Errorf("input reordered: %v", players)
	}
}
