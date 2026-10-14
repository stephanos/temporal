# fn-104-comparative-go-implementation-of-the.1 T0 Baseline the Lean side (measure.sh, results/lean-<commit>.json)

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
measure.sh for lean, go and scala; results in model/go/results/*-d08d20140.json. Lean cold rebuild after the relayout: 20 min; edit loop 191 s (Model) and about 590 s (pins, from the corpus).

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/go/measure.sh go, model/go/measure.sh lean, model/go/measure.sh scala
- PRs: