# fn-104-comparative-go-implementation-of-the.10 T9 Generated views with goldens

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Views (table, diagram, summary, diff) for both Go Models with 11 goldens; run.sh --views renders twice and diffs against the goldens.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/go/run.sh --views
- PRs: