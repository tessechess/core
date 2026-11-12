// Package core is the root of a module of pure tournament logic: result codes,
// tiebreaks and standings, TRF generation and the pairing of Swiss rounds
// through external engines, and player categories. The root package has no
// code; import the packages below.
//
// It is the tournament engine of TournamentEssentials
// (https://github.com/tessechess/tessentials), an open-source chess
// tournament manager with a web dashboard for organizers and arbiters, a public
// tournament website and an API; the same code pairs rounds and computes
// standings there. It is published as a separate module so that other programs,
// such as command-line tools, can reuse it.
//
// The module uses only the Go standard library. It has no persistence and no
// network or HTTP code: callers load the data, pass it in as plain values and
// store what comes back. Players (and categories) are identified by a type the
// caller chooses, for example a string or a UUID, through generic parameters.
//
// # Packages
//
//   - [github.com/tessechess/core/result]: game result codes
//     ("1-0", "0.5-0.5", forfeits, byes, ...) and the points they award.
//   - [github.com/tessechess/core/player]: the Sex and Title
//     value types shared by the other packages.
//   - [github.com/tessechess/core/round]: Game, one game or
//     bye of a round, the type in which every package takes and returns games.
//   - [github.com/tessechess/core/standings]: tiebreaks,
//     standings and crosstables from the games of a section.
//   - [github.com/tessechess/core/pairing]: pairs the next
//     round (settings, fixed boards, starting ranks).
//   - [github.com/tessechess/core/trf]: the FIDE TRF 1.6
//     generator and the parsers of pairing-engine output, used by pairing.
//   - [github.com/tessechess/core/engine]: runs bbpPairings
//     and JaVaFo; the programs themselves are not part of the module.
//   - [github.com/tessechess/core/category]: player
//     categories (age, sex, rating) and their assignment.
//
// # Quick start
//
// Pair the next round of a section, then rank the players once the results
// are in (the games of finished rounds are kept by the caller):
//
//	eng := engine.NewBBPPairings("/opt/bbppairings/bbpPairings.exe", 30*time.Second)
//
//	boards, err := pairing.Pair(ctx, eng, pairing.Input[string]{
//		Tournament:      pairing.Tournament{Name: "Open", TotalRounds: 7},
//		Players:         players, // []pairing.Player[string], StartingRank set
//		History:         played,  // []round.Game[string] of finished rounds
//		CompletedRounds: 2,
//		Settings:        &pairing.Settings[string]{},
//	})
//	// boards are round.Game values for round 3; store them, play, then fill in
//	// each game's Result and add the games to played.
//
//	ranked := standings.Rank(standings.Input[string]{
//		Games:   played,
//		Ratings: ratings, // map[string]int, 0 or missing = unrated
//		Tiebreaks: []standings.TiebreakConfig{
//			{Key: standings.Buchholz},
//			{Key: standings.SonnebornBerger},
//		},
//	})
//
// # Known limitations
//
// See the Known limitations sections of packages standings and pairing, for
// example that the PTP and APPO tiebreaks can hang and that acceleration is
// ignored.
package core
