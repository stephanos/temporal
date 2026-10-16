# fn-170-clean-up-toolsumpire.4 Tidy: fold ir/paths, one duration conversion, export package name, over-exports, doc drift

## Description
From the `tools/umpire` structure research (2026-10-12). Small tidy-ups, each verified by the Go tests and byte-identical outputs:
1. Fold `ir/paths` into `ir`, and delete the five one-line wrappers in `ir/outputs.go`. Only `interp` tests and `cmd/umpire-repeat` need paths without `ir`; give them what they need.
2. Delete `lower/duration.go`'s `durationMilliseconds` (raw arithmetic, which differs from `realization/duration.go`'s checked conversion) and call `testpilot/duration`. Check that every caller keeps its error behaviour.
3. Rename the `export` package from `backends` to `export`, and document `make umpire-check-backends` as its entry point (it has no non-test caller).
4. Prune over-exported identifiers: `lint` exports 50 and its only caller uses 9. Unexport, or delete if unused, the research's candidates: `check.{ClaimClassification,HoleEdge,HoleReach,ResourceBound}`, `lower.{Entry,ErrNotGenerated,ExpectedClaim}`, `ir.WaiverSuffix`, `interp.Interpretation`, `realization.Mapping`, and `lint.{Cell,Coverage,Judged,Kinds,Modality,Model,Of,Result,Rule,Table,Tally,Verdict}`. Re-check each against current callers first.
5. Fix doc drift. `README.md` says `--kind model` writes `model/cases`, but the kind writes beside each feature (`cmd/umpire-gen-cases/main.go:31`). Add the missing `ir/paths` and `internal/runtimecel` rows. `.plans/UMPIRE_MODULES.md` names `lint.Forward` and `conformance.Admits`, which do not exist.
6. Optional, size only: split `lower/lower.go`'s accounting (lines ~1,056–1,680) into `accounting.go`, and consider splitting `ir/validate.go` (1,407 lines) by declaration kind.

## Acceptance
- `ir/paths` folded and wrappers gone; one duration conversion; `export` package named `export`; unused exports removed or unexported; README and module map match the code.
- Go tests pass; outputs byte-identical.


## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
