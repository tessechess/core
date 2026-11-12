# core

The tournament engine of [TournamentEssentials](https://github.com/ValkoM28/TournamentEssentials), an open-source chess tournament manager with an organizer dashboard, a public tournament website and an API. This module holds the pure FIDE rules (result codes, TRF generation, Swiss pairing through external engines, tiebreaks and standings, player categories) and is used by the backend API today and by a command-line tool later. It has no dependencies outside the Go standard library and can be used on its own.

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

- **PTP and APPO tiebreaks can hang forever** with unrated opponents ([#379](https://github.com/ValkoM28/tournamentessentials/issues/379)); don't configure them yet.
- **Rating change against unrated opponents is wrong** ([#383](https://github.com/ValkoM28/tournamentessentials/issues/383)): they count as rating 0, but a game against an unrated player should not change the rating. ARO and TPR do leave unrated opponents out.
- **No team tiebreaks** ([#78](https://github.com/ValkoM28/tournamentessentials/issues/78)): individual competitions only.
- **Round 1 needs a first-player colour with bbpPairings** ([#376](https://github.com/ValkoM28/tournamentessentials/issues/376)).
- **Acceleration is validated but ignored** ([#377](https://github.com/ValkoM28/tournamentessentials/issues/377)).
- **Swiss pairing only**, through external engines (no round-robin generator); the engine programs are not part of the module and JaVaFo must not be redistributed.
- **Engine timeouts on non-Unix platforms** kill only the engine process, not its children ([#382](https://github.com/ValkoM28/tournamentessentials/issues/382)).

## Packages

| Package | Contents |
|---|---|
| `result` | Game result codes (`1-0`, `0-1`, `0.5-0.5`, forfeits, byes, `*`, `void`) and the points each side receives |
| `player` | `Sex` and `Title` — the shared value types of `trf`, `pairing` and `category`; `Title.Order` (FIDE initial-ranking order) and `Title.FIDE` |
| `round` | `Game` — one game or bye of a round (`Round`, `Board`, `White`, `Black`, `Result`), with `IsBye` and `Points`; `Color` (`ColorWhite`, `ColorBlack`). The shared game type of `trf`, `pairing` and `standings`. Generic over the player ID |
| `category` | Category rules: `Criteria.Matches` / `Validate` (rating, birthday and sex conditions, `Validate` also rejects an invalid sex; unrated counts as under any max rating), `Assign` (players → categories, `ManualOnly` skipped) and `RankWithin` (category prize order by section rank). Generic over category and player IDs |
| `engine` | `PairingEngine` interface, bbpPairings and JaVaFo wrappers, and the runner (temp files, timeout; on Unix a timeout kills the whole process group, elsewhere only the engine process: see the package docs, #382). Callers supply the binary/jar paths |
| `pairing` | Pairs the next round: `Prepare` (merges player-level fixed boards, builds the TRF) and `Prepared.Run` (runs an engine, parses its output by format, maps ranks to IDs, applies fixed boards); `Pair` does both. History in and new boards out are `round.Game`s. `ApplyFixedBoards`. `Settings.Validate` / `ReferencedPlayers` check the arbiter settings (`Acceleration` is a typed value; the first-player colour is `round.Color`). `StartingOrder` gives starting ranks in FIDE initial-ranking order (rating, title, name; arena titles count as untitled). Generic over the player ID |
| `standings` | FIDE tiebreaks (Buchholz and variants, Sonneborn-Berger, Koya, DE, ARO, TPR, PTP, APRO, APPO, wins, …), typed tiebreak keys (`TiebreakKey` constants such as `Buchholz`, `SonnebornBerger`) with the registry (`IsTiebreakKey`, `SupportsCut`, `SupportsThreshold`, `TiebreakKeys`), and standings: `Scores`, `Rank`, `Crosstable`, computed from `round.Game`s. The individual tiebreak functions are internal. Generic over the player ID |
| `trf` | FIDE TRF 1.6 generator (sex written as `m`/`w`, only FIDE titles) for pairing engines and parsers for bbpPairings / JaVaFo output. Player IDs are a type parameter (`Generate[ID comparable]`); completed rounds are passed as `round.Game`s + `CompletedRounds` |
