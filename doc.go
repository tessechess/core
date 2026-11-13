// Package core is the root of a module of pure chess tournament logic: game
// results and their points, Swiss pairing through external engines,
// round-robin schedules (the Berger function), tiebreaks, standings and
// crosstables, FIDE TRF generation, and player categories. The root package
// has no code; import the packages listed below.
//
// It is the heart of TournamentEssentials
// (https://github.com/tessechess/tessentials), an open-source chess tournament
// platform, responsible for pairing rounds and computing standings.
// It is published as a module of its own so that other programs, both within the
// TEssentials suite or outside can use the same basic logic.
//
// # Design
//
// The module is a set of functions over plain values. It uses only the Go
// standard library, and it has no persistence, no HTTP and no logging. Callers
// load their data, pass it in, and store what comes back; the same input gives
// the same output.
//
// Players are identified by a type the caller chooses, for example a string, an
// integer or a UUID, through a generic parameter (ID comparable) on the types
// and functions that need one. The module never inspects an ID; it only
// compares IDs and hands them back. Because of that, the types carry no
// serialization tags: converting them to and from a database or an API is
// up to the caller.
//
// # Concepts
//
// A game is a [github.com/tessechess/core/round.Game]: the round, the board,
// the player with white, the player with black and the result. The same type is
// what the caller passes in as the history of finished rounds and what the
// pairing functions return as the new round.
//
// A bye is a game with no black player: White holds the player who receives it
// and Black is nil. What the player scores depends on the result recorded on
// the game: 1-0 is a full-point bye, 0.5-0 a half-point bye and "0-0" a
// zero-point bye (or a withdrawal). A result is a
// [github.com/tessechess/core/result.Code], a string in the form
// "white_score-black_score"; "*" means the game is not played yet and "void"
// means the game does not count, and is left out of all calculations.
//
// Pairing engines and the TRF file format refer to players by starting rank
// (the pairing number, 1..N) instead of by ID. The caller assigns the starting
// ranks: [github.com/tessechess/core/pairing.StartingOrder] computes them in
// FIDE initial-ranking order (rating, then title, then name), and a
// round-robin takes the numbers as they are, since they are usually drawn by
// lot. Inside the module the ranks are converted to IDs and back, so the
// caller works with IDs only.
//
// # Packages
//
//   - [github.com/tessechess/core/result]: game result codes ("1-0",
//     "0.5-0.5", forfeits, byes, "*", "void") and the points each side
//     receives.
//   - [github.com/tessechess/core/round]: Game, one game or bye of a round,
//     the type in which every package takes and returns games.
//   - [github.com/tessechess/core/player]: the Sex and Title value types
//     shared by the other packages.
//   - [github.com/tessechess/core/pairing]: pairs the next Swiss round or
//     generates a round-robin schedule upfront; also the arbiter's settings
//     (byes, forbidden pairings, fixed boards, colour of the first player),
//     their validation, and the starting-rank order.
//   - [github.com/tessechess/core/engine]: runs bbpPairings and JaVaFo; the
//     programs themselves are not part of the module.
//   - [github.com/tessechess/core/trf]: the FIDE TRF 1.6 generator and the
//     parsers of pairing-engine output, used by pairing.
//   - [github.com/tessechess/core/standings]: scores, ranking, tiebreaks and
//     crosstables computed from the games of a section.
//   - [github.com/tessechess/core/category]: player categories (age, sex,
//     rating) and their assignment and prize order.
//
// # Quick start: Swiss pairing and standings
//
// Pair the next round of a section, then rank the players once the results
// are in. The caller keeps the games of finished rounds:
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
//	for _, s := range ranked {
//		fmt.Println(s.Rank, s.Score.ID, s.Score.Points)
//	}
//
// The arbiter's choices for a round go in [github.com/tessechess/core/pairing.Settings]:
// players withdrawn or given a half-point bye, late entries, forbidden
// pairings and fixed boards. Round 1 also needs the colour of the first player
// there. Check the settings with Settings. Validate when they are stored, and use
// Prepare instead of Pair to look at the TRF before an engine runs.
//
// # Quick start: round-robin
//
// A round-robin schedule is generated for the whole tournament at once and does
// not need an engine. Give every player a unique StartingRank from 1 to the
// number of players:
//
//	rounds, err := pairing.PairRoundRobin(players, 2, pairing.CycleOrderFIDE)
//	// rounds[0] is round 1, with one round.Game per board. A double round-robin
//	// has 2*(N-1) rounds for an even number of players N.
//
// With an odd number of players one player has a free round in each round. It
// appears as a bye game on the last board with the result "void", so it scores
// nothing and counts nowhere. For a double round-robin the order of the rounds
// of the first cycle matters for colours: [github.com/tessechess/core/pairing.CycleOrder]
// offers table order, the FIDE C.05 recommendation and Otto Milvang's
// arrangement. Use pairing.RoundRobinTotalRounds for the number of rounds
// before generating the schedule.
//
// # Quick start: categories
//
// A category (for example "U12" or "Women") is a rule over birthday, sex and
// rating. category.Assign puts every player into the categories they match, and
// category.RankWithin orders the members of a category by their rank in the
// section:
//
//	assigned := category.Assign(rules, subjects) // map[playerID][]categoryID
//
// # Pairing engines
//
// Swiss pairing is done by external programs that implement the FIDE Dutch
// system. Core builds their input, runs them and reads their answer, but does
// not ship them: bbpPairings is Apache-2.0 licensed and can be downloaded from
// its project page; JaVaFo is proprietary freeware the user can download.
// The caller supplies the paths when
// creating the engine (engine.NewBBPPairings, engine.NewJaVaFo). Another engine
// can be plugged in by implementing engine.PairingEngine.
//
// A run is limited by a timeout and by the context. See package engine for what
// happens to a runaway engine process on different platforms.
//
// # Errors
//
// Functions return errors that wrap sentinel values, so that callers can react
// to a cause with errors.Is: for example pairing.ErrInvalidSettings,
// pairing.ErrNotEnoughPlayers, pairing.ErrInvalidCycles and
// engine.ErrEngineNotFound.
//
// # Known limitations
//
// The module covers individual competitions. Team tiebreaks and team
// tournaments are not implemented, and neither is Swiss acceleration
// (pairing.Settings.Validate rejects it). Swiss pairing needs an external
// engine; there is no built-in one. There is no reader for complete TRF
// tournament files; trf generates the format for the engines and parses their
// output. The package documentation of standings, pairing and engine has the
// details.
//
// # License
//
// Core is released under the Apache License 2.0, see the LICENSE file. The
// pairing engines it can run have their own licenses.
package core
