# core

The tournament engine of [TournamentEssentials](https://github.com/tessechess/tessentials), an open-source chess tournament manager with an organizer dashboard, a public tournament website and an API. This module holds the pure FIDE rules (result codes, TRF generation, Swiss pairing through external engines, round-robin pairing via the Berger function, tiebreaks and standings, player categories) and is used by the backend API today and by a command-line tool later. It has no dependencies outside the Go standard library and can be used on its own.

Module path: `github.com/tessechess/core`, published from [github.com/tessechess/core](https://github.com/tessechess/core).

## Documentation

The reference is the Go doc comments: `go doc ./standings`, or [pkg.go.dev](https://pkg.go.dev/github.com/tessechess/core) once the module is public. [doc.go](doc.go) holds the module overview and a quick start. `doc_test.go` fails when an exported identifier has no doc comment.

## Rules

- **Pure rules and values only.** No database, HTTP, Gin, config or logging.
- **Imports nothing from `apps/`** and no third-party modules unless deliberately allowed. `make check-core` enforces this.
- **Values, not entities.** Core owns enums, small value types and the functions over them. Entities with identity or lifecycle (ids, owners, timestamps, `json`/`db` tags) stay in the apps. See `docs/ARCHITECTURE_REWORK.md` (section "Principle: core owns rules and values; apps own entities").
- **Tests live beside the code** (`*_test.go` in each package), unlike the backend's `tests/<module>_tests/` convention.

## Known limitations

Details are in the package documentation (`go doc`).

- **No team tiebreaks** ([#78](https://github.com/tessechess/tessentials/issues/78)): individual competitions only.
- **Acceleration is not implemented**: `Settings.Validate` rejects anything other than `none`.
- **Swiss pairing requires external engines**: the engine programs are not part of the module and JaVaFo must not be redistributed (round-robin schedules are generated natively via the Berger function).
- **Engine timeouts on non-Unix platforms** kill only the engine process, not its children ([#382](https://github.com/tessechess/tessentials/issues/382)).

## Packages

| Package | Contents |
|---|---|
| `result` | Game result codes (`1-0`, `0-1`, `0.5-0.5`, forfeits, byes, `*`, `void`) and the points each side receives |
| `player` | `Sex` and `Title` — the shared value types of `trf`, `pairing` and `category`; `Title.Order` (FIDE initial-ranking order) and `Title.FIDE` |
| `round` | `Game` — one game or bye of a round (`Round`, `Board`, `White`, `Black`, `Result`), with `IsBye` and `Points`; `Color` (`ColorWhite`, `ColorBlack`). The shared game type of `trf`, `pairing` and `standings`. Generic over the player ID |
| `category` | Category rules: `Criteria.Matches` / `Validate` (rating, birthday and sex conditions, `Validate` also rejects an invalid sex; unrated counts as under any max rating), `Assign` (players → categories, `ManualOnly` skipped) and `RankWithin` (category prize order by section rank). Generic over category and player IDs |
| `engine` | `PairingEngine` interface, bbpPairings and JaVaFo wrappers, and the runner (temp files, timeout; on Unix a timeout kills the whole process group, elsewhere only the engine process: see the package docs, #382). Callers supply the binary/jar paths |
| `pairing` | Pairs the next Swiss round (`Prepare` merges player-level fixed boards and builds the TRF; `Prepared.Run` runs an engine, parses its output by format, maps ranks to IDs, applies fixed boards; `Pair` does both) or generates a round-robin schedule upfront via the Berger function (`PairRoundRobin`, `RoundRobinTotalRounds`; for several cycles the caller picks a `CycleOrder`: plain table order, the FIDE C.05 recommendation, or Milvang's arrangement). History in and new boards out are `round.Game`s. `ApplyFixedBoards`. `Settings.Validate` / `ReferencedPlayers` check the arbiter settings (`Acceleration` is a typed value, not yet implemented; the first-player colour is `round.Color`). `StartingOrder` gives starting ranks in FIDE initial-ranking order (rating, title, name; arena titles count as untitled). Generic over the player ID |
| `standings` | FIDE tiebreaks (Buchholz and variants, Sonneborn-Berger, Koya, DE, ARO, TPR, PTP, APRO, APPO, wins, …), typed tiebreak keys (`TiebreakKey` constants such as `Buchholz`, `SonnebornBerger`) with the registry (`IsTiebreakKey`, `SupportsCut`, `SupportsThreshold`, `TiebreakKeys`), and standings: `Scores`, `Rank`, `Crosstable`, computed from `round.Game`s. The individual tiebreak functions are internal. Generic over the player ID |
| `trf` | FIDE TRF 1.6 generator (sex written as `m`/`w`, only FIDE titles) for pairing engines and parsers for bbpPairings / JaVaFo output. Player IDs are a type parameter (`Generate[ID comparable]`); completed rounds are passed as `round.Game`s + `CompletedRounds` |

## License

Core is released under the [Apache License 2.0](LICENSE); the copyright notice is in [NOTICE](NOTICE). The pairing engines are not part of this module and have their own terms: bbpPairings is Apache-2.0, JaVaFo is proprietary freeware that you download yourself.

Bug reports are welcome; code pull requests are not accepted yet, see [CONTRIBUTING.md](CONTRIBUTING.md).
