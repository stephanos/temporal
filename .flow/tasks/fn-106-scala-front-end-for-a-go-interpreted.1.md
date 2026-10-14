# fn-106-scala-front-end-for-a-go-interpreted.1 IR schema and semantics (ir.proto, SEMANTICS.md, gen.sh)

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Proto IR with types, expression trees with match, actions and machines, positions; SEMANTICS.md; gen.sh builds protoc-gen-go from go.mod

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: model/scalav2/run.sh
- PRs: