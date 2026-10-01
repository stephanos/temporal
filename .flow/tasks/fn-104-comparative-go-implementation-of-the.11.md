# fn-104-comparative-go-implementation-of-the.11 T10 Error corpus on both sides

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Twelve-case corpus (cases.json with lean, go and scala edits) and run.py harness; all three sides run; Go gap (duplicate Property names) found and fixed in umpire.Check.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: python3 model/go/corpus/run.py go, python3 model/go/corpus/run.py lean, python3 model/go/corpus/run.py scala
- PRs: