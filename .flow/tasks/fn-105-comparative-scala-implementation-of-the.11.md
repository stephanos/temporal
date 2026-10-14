# fn-105-comparative-scala-implementation-of-the.11 S10 Stainless lemmas over the kernel

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Five Stainless lemmas over every state; 79 VCs valid; mutation rejected; dispatch tied to table rows by a runtime test.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/scala/run.sh --views
- PRs: