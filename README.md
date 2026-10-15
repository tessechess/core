# core

Shared tournament logic used by the backend API and, later, the CLI.

Module path: `github.com/ValkoM28/tournamentessentials/core` (placeholder; may be renamed if the project moves to a GitHub organization).

## Rules

- **Pure rules and values only.** No database, HTTP, Gin, config or logging.
- **Imports nothing from `apps/`** and no third-party modules unless deliberately allowed. `make check-core` enforces this.
- **Values, not entities.** Core owns enums, small value types and the functions over them. Entities with identity or lifecycle (ids, owners, timestamps, `json`/`db` tags) stay in the apps. See `docs/ARCHITECTURE_REWORK.md` (section "Principle: core owns rules and values; apps own entities").
- **Tests live beside the code** (`*_test.go` in each package), unlike the backend's `tests/<module>_tests/` convention.

## Packages

| Package | Contents |
|---|---|
| `result` | Game result codes (`1-0`, `0-1`, `0.5-0.5`, forfeits, byes, `*`, `void`) and the points each side receives |
| `engine` | `PairingEngine` interface, bbpPairings and JaVaFo wrappers, and the runner (temp files, timeout; on Unix a timeout kills the whole process group). Callers supply the binary/jar paths |
| `pairing` | Pairs the next round: `Prepare` (merges player-level fixed boards, builds the TRF) and `Prepared.Run` (runs an engine, parses its output by format, maps ranks to IDs, applies fixed boards); `Pair` does both. `ApplyFixedBoards`. Generic over the player ID |
| `standings` | FIDE tiebreaks (Buchholz and variants, Sonneborn-Berger, Koya, DE, ARO, TPR, PTP, APRO, APPO, wins, …), the tiebreak registry (`IsTiebreakKey`, `SupportsCut`, `SupportsThreshold`, `TiebreakKeys`), and standings: `Scores`, `Rank`, `Crosstable`, `BuildRoundResults`. Generic over the player ID |
| `trf` | FIDE TRF 1.6 generator for pairing engines and parsers for bbpPairings / JaVaFo output. Player IDs are a type parameter (`Generate[ID comparable]`) |

## Licence

Not decided yet (MIT or Apache-2.0). The repository is private; add a LICENSE file before publishing.
