# fn-104-comparative-go-implementation-of-the.4 T3 Refinement and composition

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
refine.go ports Umpire.Command.deriveRefinement exactly (outcome names, start map, carriers preferring the same-named product action, stutters, rejection text). compose.go ports the composition: reachable rows only, _-joined state keys, <member>_<class> own actions, sync names, first member's outcome, facts in member order, compose-<name> owner, one-field member named by the member. Unit tests: match, stutter, missing product step, fact subset, start mismatch, sync semantics, malformed sync ref. Parity: all 1,152 refinement rows and the 316-state/1,468-row composition equal Lean.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./experiments/umpire-go/..., golangci-lint v2.13.1 --config=.github/.golangci.yml (0 issues), go-check-sumtype, exhaustive -default-signifies-exhaustive=false on model packages (clean)
- PRs: