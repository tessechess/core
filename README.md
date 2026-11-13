# TEssentials core library

The tournament engine of [TournamentEssentials](https://github.com/tessechess/tessentials), an open-source chess tournament platform. This module holds the pure FIDE rules (result codes, TRF generation, Swiss pairing through external engines, round-robin pairing via the Berger function, tiebreaks and standings, player categories) and is used by different tools in the TEssentials ecosystem. It has no dependencies outside the Go standard library and can be used on its own.

Module path: `github.com/tessechess/core`, published from [github.com/tessechess/core](https://github.com/tessechess/core).

## Documentation

[pkg.go.dev](https://pkg.go.dev/github.com/tessechess/core)

## Design principles

- **Pure functions over plain values.** Nothing here touches a database, the network or anything else, just pure functions other programs are calling. The same input always gives the same output.
- **Standard library only.** The module has no third-party dependencies, so adding it to a project adds nothing else to your dependency tree.
- **Your types, not ours.** Players and games are identified by a type parameter (`[ID comparable]`), so you can use database ids, UUIDs or strings. 
- **Rules only, no records.** The module covers FIDE rules and calculations. It has no tournament, player or user records, no persistence and no authorization.
- **Tests live beside the code.** Each package has its `*_test.go` files next to the source; run them all with `go test ./...`.

## Implementation status

Core covers individual competitions. Details of each area are in the package documentation (`go doc`).

| Area | Status | Notes |
|---|---|---|
| Result codes and scoring | Implemented | Wins, draws, forfeits, byes, unplayed games and void games, with the points each side receives |
| Round-robin pairing | Implemented | Berger tables for any number of players, any number of cycles, and three round orders for double round-robins (plain, FIDE C.05, Milvang). The caller assigns the pairing numbers. |
| Swiss pairing | Implemented through external engines | Uses bbpPairings or JaVaFo, which you install yourself and point core to; they are not part of this module. There is no native Swiss pairing. |
| Swiss acceleration | Not implemented | `Settings.Validate` rejects anything other than `none` |
| Tiebreaks | Implemented, individual only | 17 FIDE tiebreaks, including Buchholz and its variants, Sonneborn-Berger, Koya, direct encounter, performance ratings and progressive scores. Team tiebreaks are not implemented. |
| Standings and crosstables | Implemented | Scores, ranking with tiebreaks, and the crosstable, all computed from the list of games |
| TRF | Partly implemented | Generates TRF 1.6 for the pairing engines and parses their output. There is no reader for complete TRF tournament files. |
| Player categories | Implemented | Rating, birthday and sex conditions, assignment and prize order within a category |
| Team tournaments | Not implemented | Pairing, scoring and tiebreaks for teams |
| Engine timeouts on non-Unix platforms | Limited | A timeout kills only the engine process, not its children |

## Packages

Import paths are `github.com/tessechess/core/<package>`.

| Package | What it does | Main entry points |
|---|---|---|
| `result` | Game result codes (`1-0`, `0-1`, `0.5-0.5`, forfeits, byes, `*` for not yet played, `void`) and the points each side receives | `Code` |
| `round` | The shared game type: one game or bye of a round, with its board, colours and result | `Game`, `Color` |
| `player` | Player value types shared by the other packages: sex and FIDE title, including the title order used for the initial ranking | `Sex`, `Title` |
| `pairing` | Swiss pairing of the next round through an engine, round-robin schedules (single or multiple cycles, with a choice of round order), fixed boards, validation of the arbiter's settings, and starting ranks in FIDE initial-ranking order | `Pair`, `Prepare`, `PairRoundRobin`, `RoundRobinTotalRounds`, `StartingOrder`, `Settings.Validate` |
| `engine` | Wrappers that run bbpPairings or JaVaFo (temporary files, timeout). You supply the paths to the programs. (licenses of external engines apply) | `PairingEngine`, `NewBBPPairings`, `NewJaVaFo` |
| `trf` | Generates FIDE TRF 1.6 input for the pairing engines and parses the engines' output | `Generate`, `ParseSimplePairs`, `ParseTRFRound` |
| `standings` | Scores, ranking and crosstables computed from the games, with the FIDE tiebreaks selected by typed keys. The registry says which keys accept a cut or a threshold. | `Scores`, `Rank`, `Crosstable`, `TiebreakKey`, `TiebreakKeys` |
| `category` | Prize categories: which players match the rating, birthday and sex conditions of a category, and the prize order within it | `Assign`, `RankWithin`, `Criteria`, `Rule` |

## License

Core is released under the [Apache License 2.0](LICENSE); the copyright notice is in [NOTICE](NOTICE). The pairing engines are not part of this module and have their own terms: bbpPairings is Apache-2.0, JaVaFo is proprietary freeware that you download yourself.

Bug reports are welcome; code pull requests are not accepted yet, see [CONTRIBUTING.md](CONTRIBUTING.md).
