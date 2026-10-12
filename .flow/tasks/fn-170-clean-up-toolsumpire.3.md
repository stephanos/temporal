# fn-170-clean-up-toolsumpire.3 Fold internal/engine into interp and check; remove the alias facades

## Description
From the `tools/umpire` structure research (2026-10-12). `check/types.go` re-exports 27 types and 21 consts of `internal/engine` as aliases, and `interp/types.go` 8 more, so the engine's shape leaks through two facades: `check` is half facade, with 48 of its 94 exports aliases.

Do: move the table types (`engine/table.go`, `canonical.go`) into `interp`, and the claim evaluators (`search`, `progress`, `refine`, `compose`, `monitor`, `replay`, `keyclaims`) into `check`. Then delete `internal/engine` and the alias files. Keep `check`'s and `interp`'s behaviour and their callers' results identical (Go tests, lint, lowering and Case outputs byte-identical on the checked-in IR). Update `.plans/UMPIRE_MODULES.md` and the ownership tests (`ir/ownership_test.go`).

## Acceptance
- `internal/engine`, `check/types.go` and `interp/types.go` aliases gone; no exported alias of another package's type remains in `check`/`interp`.
- Go tests pass; outputs byte-identical; module map and ownership tests updated.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
