# fn-105-comparative-scala-implementation-of-the.2 S1 Framework core: Finite, actions, Machine, Table, restrict

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Finite derivation via Mirror, typed actions, machine blocks with ~> bindings, table build, restrict; tables match Lean.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/scala/run.sh --views
- PRs: