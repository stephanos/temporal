# fn-104-comparative-go-implementation-of-the.6 T5 Port the worker and Nexus caller Models

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
worker/ and nexuscaller/ port the worker and the Nexus caller Model in the Lean file's order with its semantic comments: vocabulary, product and protocol machines, handlerWorker (restrict), the nexusCaller composition, 9 Properties, 7 Scenarios, 3 Limits, 9 Queries, functional/canary/exploratory sets. Step switches have no default arms; exhaustive and go-check-sumtype enforce every case. umpire.Check accepts every declaration.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./experiments/umpire-go/..., golangci-lint v2.13.1 --config=.github/.golangci.yml (0 issues), go-check-sumtype, exhaustive -default-signifies-exhaustive=false on model packages (clean)
- PRs: