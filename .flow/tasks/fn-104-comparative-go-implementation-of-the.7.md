# fn-104-comparative-go-implementation-of-the.7 T6 Nexus pin and row-level parity

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
nexuscaller/pins_test.go translates the Lean pins one to one with the Lean line cited; parity/ compares every dumped table, ID, refinement row, Query outcome and witness, and all 889 targets: all equal. One divergence, pinned and explained: over the whole protocol machine within four, Go visits 111 product states where Veil visits 171, because Lean keeps one fired bit per lowered clause group and Go one per Property; fn-88 exempts explored counts from backend comparison. Lean pins without a Go counterpart (assert_axioms, clause groups, reference path counts) are listed with reasons at the end of the pin file; Case pins move to T8.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./experiments/umpire-go/..., golangci-lint v2.13.1 --config=.github/.golangci.yml (0 issues), go-check-sumtype, exhaustive -default-signifies-exhaustive=false on model packages (clean)
- PRs: