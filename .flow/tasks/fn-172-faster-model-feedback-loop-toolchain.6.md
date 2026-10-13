---
satisfies: [R6]
---
# fn-172-faster-model-feedback-loop-toolchain.6 check.Check interprets each Model once: share or remove the replay binding and cut the GC share, with byte-identical receipts

## Description
Share the checker binding when witness replay uses the identical Model, retaining separate binding for deliberately different replay Models. Profile and remove redundant interpretation/allocation within the existing interpreter and replay surface, without changing search semantics, IR, IDs or fingerprints.

**Touches:** tools/umpire/check/**, tools/umpire/interp/**, tools/umpire/internal/engine/**, .flow/tasks/fn-172-faster-model-feedback-loop-toolchain.6.*

### Investigation targets
Required: tools/umpire/check/checking.go, tools/umpire/check/claims.go, tools/umpire/check/replay_test.go, tools/umpire/interp/machine.go, tools/umpire/internal/engine/table.go.

### Quick commands
- go test -tags test_dep ./tools/umpire/check -run '^(TestReplaySharesTheInterpretationOfTheSameModel|TestComposedProgressWitnessReconstructsTypedMemberStates|TestComposedProgressCountsASynchronizedStepOnceAndReplays|TestTheCounterModelIsAdmitted)$' -count=1
- make lint-code-fast GOLANGCI_LINT_BASE_REV=9792def9fcf7d6110a54e93adebf5c3a09af32df

Full model gate and complete check/interp package suite remain conductor-owned at final spec closure. The attempted broad pre-edit suite has inherited missing model/irgen fixture paths and was terminated under resource pressure; it is red/inconclusive, not waived as passing. Focused existing synthetic tests passed before implementation. Use preserved six sample IR artifacts to measure checker wall time and allocation before/after, compare complete Report bytes on representative samples, and require all six expected Query/progress verdicts after the fix. Inherited old Bank OOM and terminated capped runs are red/inconclusive observations, not green evidence.
## Acceptance
- [ ] R6: Check shares its interpreted Model with replay; a distinct replay Model still binds independently and rejects stale witnesses.
- [ ] Reduce measured checker allocation/GC without changing reachable exploration or IR shape.
- [ ] Complete Report bytes match the pre-edit reports on representative sample IR; all six samples produce unchanged expected Query/progress verdicts.
- [ ] Record before/after checker wall time and memory/allocation measured on the same environment and artifacts; preserve all IR bytes, fingerprints, Definition IDs and Case identities.
- [ ] Focused Go tests and changed-package lint pass; independent configured implementation review reaches SHIP before done.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
