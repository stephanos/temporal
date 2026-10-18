---
satisfies: [R5, R6]
---
# fn-196-the-scala-checker-is-the-oracle.4 IR and plugin phase: retire the Go check and the IR evaluator, the gate and conformance replay run on the Scala checker

## Description
**Phase 3 (IR and plugin, batched), on the measurement note's go.** Delete the Go `check` package and the Go explorer's use as a checker, retire the IR evaluator task fn-190.3 and re-cut fn-190.5 to delete the Go check rather than the Scala runtime; a Go test that only exercised the Go check goes with it, one that exercised lowering or export stays. The gate's sample check runs the Scala checker through the suites. Conformance replay of a recorded Run against the Model runs in Scala. The six samples' intended answers on the default budget match the inventory on the Scala checker; a changed answer is a finding to fix.

**Size:** L

**Files:** `tools/umpire/check/` (deleted), `tools/umpire/explore` and `interp` where used as a checker, `tools/common/formal/conformance` (replay moves), `model/framework/check/Samples.scala` and its suite, `Makefile` targets, `.flow` tasks fn-190.3 and fn-190.5.

### Verification
- `go build ./...` and the Go test suites of the remaining packages pass; no package imports the deleted check.
- The samples check in the gate reports the inventory's answers from the Scala checker.
- Quick: the README Native checks recipe plus `mise exec -- go test ./tools/umpire/... ./common/testing/testpilot/...`.

## Acceptance
- [ ] Go check package and IR evaluator retired; fn-190.3/.5 re-cut
- [ ] Gate's sample check and conformance replay run on the Scala checker
- [ ] Intended answers match the inventory

## Done summary
Blocked:
The committed fn-196.3 measurement decision is no-go on immediate Go check/interpreter retirement. Its source checkpoints45000a5680/b1ea944f7d/2319050f0f and closure946b550a6d retain native refinement/replacement, rule-disjointness, recorded-Run/backend replay, lowering and consumer migration obligations. This task is also in the explicitly held Phase3 IR/Go lane. Do not start or force it merely because the task dependency on .3 is done. Release only after the native prerequisites have evidence, a new go decision is recorded, and the owner-authorized Phase3 scope permits execution. No deletion or runtime migration is complete.
## Evidence
- Commits:
- Tests:
- PRs:
