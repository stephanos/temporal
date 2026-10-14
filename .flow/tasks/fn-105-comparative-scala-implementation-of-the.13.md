# fn-105-comparative-scala-implementation-of-the.13 S12 RESULTS.md comparing Scala, Go and Lean

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
model/scala/RESULTS.md: parity, Stainless lemmas, three-way corpus with Stainless column, loop and size, authoring surface, trial, recommendation.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/scala/run.sh --views
- PRs: