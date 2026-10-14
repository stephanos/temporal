# fn-106-scala-front-end-for-a-go-interpreted.4 Parity with Lean, diagnostics tests, run.sh gate

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Tables, IDs, 1152 refinement rows and target fingerprint equal Lean; run.sh re-lifts and requires the checked-in IR to be current

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/scalav2/run.sh
- PRs: