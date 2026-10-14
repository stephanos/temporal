# fn-104-comparative-go-implementation-of-the.5 T4 Claims and search: properties, scenarios, queries, sets, coverage

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
claims.go and search.go: Property[S] and Scenario[S] are generic in their state type, so Find/Verify pair them at compile time; VerifyRefined takes a typed Via[S, PS] handle. Breadth-first product search (state, schedule progress, monitor) with a visited set, row then result order, first-discovered parent; free Scenarios; limit-reached distinct from not-found. coverage.go ports coverageTargets and the within sweep. set.go: sets, Check (stuck, evidence per fact, refinement, Query answers, canary silent steps). Unit tests cover find/not-found/verify/counterexample/limit-reached/transition-claim rejection/target cut.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./experiments/umpire-go/..., golangci-lint v2.13.1 --config=.github/.golangci.yml (0 issues), go-check-sumtype, exhaustive -default-signifies-exhaustive=false on model packages (clean)
- PRs: