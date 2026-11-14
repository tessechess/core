package standings

import (
	"sort"
	"testing"
)

// allKeys lists every exported TiebreakKey constant. Adding a key means adding
// it here, to the registry and to computeTiebreak / computeTeamTiebreak; the
// tests below fail if they disagree.
var allKeys = []TiebreakKey{
	Buchholz, ForeBuchholz, AverageOpponentBuchholz, SonnebornBerger, KoyaSystem,
	DirectEncounter, AverageRatingOfOpponents, TournamentPerformanceRating,
	PerfectTournamentPerformance, AveragePerfectPerformanceRatingOfOpponents,
	AveragePerfectPerformanceOfOpponents, Wins, GamesWon, GamesPlayedBlack,
	WinsWithBlack, ProgressiveScores, GamesElectedToPlay,
	MatchPoints, GamePoints, ExtendedMatchMatchSB, ExtendedMatchGameSB,
	ExtendedGameMatchSB, ExtendedGameGameSB, BoardCount, TopBoardResults,
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

func TestTiebreakKeys_ComputeTiebreakDispatch(t *testing.T) {
	teamOnly := map[TiebreakKey]bool{
		MatchPoints:          true,
		GamePoints:           true,
		ExtendedMatchMatchSB: true,
		ExtendedMatchGameSB:  true,
		ExtendedGameMatchSB:  true,
		ExtendedGameGameSB:   true,
		BoardCount:           true,
		TopBoardResults:      true,
	}
	individualOnly := map[TiebreakKey]bool{
		AverageRatingOfOpponents:                   true,
		TournamentPerformanceRating:                true,
		PerfectTournamentPerformance:               true,
		AveragePerfectPerformanceRatingOfOpponents: true,
		AveragePerfectPerformanceOfOpponents:       true,
		GamesPlayedBlack:                           true,
		WinsWithBlack:                              true,
	}

	for _, k := range allKeys {
		_, handled := computeTiebreak(TiebreakConfig{Key: k}, "p", map[string]*playerPairingInfo[string]{}, map[string]float64{}, map[string]int{})
		wantIndiv := k != DirectEncounter && !teamOnly[k]
		if handled != wantIndiv {
			t.Errorf("computeTiebreak(%q) handled = %v, want %v", k, handled, wantIndiv)
		}

		_, teamHandled := computeTeamTiebreak(
			TiebreakConfig{Key: k},
			"t",
			map[string]*teamPairingInfo[string, string]{},
			map[string]teamRawTotals{},
			resolveTeamScoring(nil, 4, TeamScoreMatchPoints),
		)
		wantTeam := k != DirectEncounter && k != TopBoardResults && !individualOnly[k]
		if teamHandled != wantTeam {
			t.Errorf("computeTeamTiebreak(%q) handled = %v, want %v", k, teamHandled, wantTeam)
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

