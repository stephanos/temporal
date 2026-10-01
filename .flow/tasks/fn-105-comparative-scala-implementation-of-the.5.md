# fn-105-comparative-scala-implementation-of-the.5 S4 Canonical JSON, fingerprints, lowering

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Canonical JSON, fingerprints and lowering; target semantic and all per-query canonical strings byte-equal to Lean, 22 fingerprints equal.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/scala/run.sh --views
- PRs: