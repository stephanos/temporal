# fn-105-comparative-scala-implementation-of-the.12 S11 Error corpus scala edits

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Scala edits for all twelve corpus cases and a scala side in model/go/corpus/run.py (compile, test, prove stages); results in results-scala.json and results-scala-prove.json. Found and fixed scala-cli exiting 0 on -Werror failures through Bloop (model/scala/scala.sh).

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: python3 model/go/corpus/run.py scala
- PRs: