package standings

import (
	"sort"
	"testing"
)

// allKeys lists every exported TiebreakKey constant. Adding a key means adding
// it here, to the registry and to computeTiebreak; the tests below fail if the
// three disagree.
var allKeys = []TiebreakKey{
	Buchholz, ForeBuchholz, AverageOpponentBuchholz, SonnebornBerger, KoyaSystem,
	DirectEncounter, AverageRatingOfOpponents, TournamentPerformanceRating,
	PerfectTournamentPerformance, AveragePerfectPerformanceRatingOfOpponents,
	AveragePerfectPerformanceOfOpponents, Wins, GamesWon, GamesPlayedBlack,
	WinsWithBlack, ProgressiveScores, GamesElectedToPlay,
}

func TestTiebreakKeys_MatchConstants(t *testing.T) {
	got := TiebreakKeys()
	if len(got) != len(allKeys) {
		t.Fatalf("TiebreakKeys() has %d keys, constants list has %d", len(got), len(allKeys))
	}
	if !sort.SliceIsSorted(got, func(i, j int) bool { return got[i] < got[j] }) {
		t.Errorf("TiebreakKeys() is not sorted: %v", got)
	}
	for _, k := range allKeys {
		if !IsTiebreakKey(k) {
			t.Errorf("%q is not registered", k)
		}
	}
	if IsTiebreakKey("nope") || IsTiebreakKey("") {
		t.Error("unknown and empty keys must not be valid")
	}
}

func TestTiebreakKeys_ComputeTiebreakHandlesAllButDirectEncounter(t *testing.T) {
	for _, k := range allKeys {
		_, handled := computeTiebreak(TiebreakConfig{Key: k}, "p", map[string]*playerPairingInfo[string]{}, map[string]float64{}, map[string]int{})
		if want := k != DirectEncounter; handled != want {
			t.Errorf("computeTiebreak(%q) handled = %v, want %v", k, handled, want)
		}
	}
}

func TestTiebreakKeys_CutAndThresholdSupport(t *testing.T) {
	cut := map[TiebreakKey]bool{Buchholz: true, ForeBuchholz: true, AverageOpponentBuchholz: true}
	threshold := map[TiebreakKey]bool{KoyaSystem: true}
	for _, k := range allKeys {
		if SupportsCut(k) != cut[k] {
			t.Errorf("SupportsCut(%q) = %v, want %v", k, SupportsCut(k), cut[k])
		}
		if SupportsThreshold(k) != threshold[k] {
			t.Errorf("SupportsThreshold(%q) = %v, want %v", k, SupportsThreshold(k), threshold[k])
		}
	}
}
