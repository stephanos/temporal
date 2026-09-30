# fn-104-comparative-go-implementation-of-the.6 T5 Port the worker and Nexus caller Models

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
worker/ and nexuscaller/ port the worker and the Nexus caller Model in the Lean file's order with its semantic comments: vocabulary, product and protocol machines, handlerWorker (restrict), the nexusCaller composition, 9 Properties, 7 Scenarios, 3 Limits, 9 Queries, functional/canary/exploratory sets. Step switches have no default arms; exhaustive and go-check-sumtype enforce every case. umpire.Check accepts every declaration.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./experiments/umpire-go/..., golangci-lint v2.13.1 --config=.github/.golangci.yml (0 issues), go-check-sumtype, exhaustive -default-signifies-exhaustive=false on model packages (clean)
- PRs: