---
satisfies: [R5, R6]
---
# fn-194-parametric-models-constants-and-finite.3 DSL completion: Bank, Voting and TwoPhaseCommit take their world as parameters

## Description
**DSL completion phase (owner, 2026-10-17).** Rewrite the three drifted samples as parametric Models and move their concrete worlds into their test files: Bank takes its address type (`[Addr: Finite]`), `holder` and `supply`; Voting takes its values and quorum set; TwoPhaseCommit takes its resource-manager set. Each test file declares the enums and instantiates the Model with today's constants, so the intended answers in the native-migration inventory are reproduced unchanged on the default budget; a changed answer is a finding to fix, not a regeneration. The realization fixtures that name a sample Model are repointed at an instance so a realization binds to an instance (R6). The other three samples keep the object form.

**Size:** M

**Files:** `model/framework/samples/Bank.scala`, `Bank.test.scala`, `Voting.scala`, `Voting.test.scala`, `NativeVoting.test.scala`, `TwoPhaseCommit.scala`, `TwoPhaseCommit.test.scala`, the realization fixtures under `model/framework/realize` and `export/testdata` that reference these samples, `samples/testdata/native-migration/inventory.md`.

### Verification
- The samples native suite passes; the three samples' intended answers match the inventory.
- A second Voting instance with a different quorum set checks and reports a different graph size (one test).
- Quick: the samples native suite via the README Native checks recipe.

## Acceptance
- [ ] Bank, Voting and TwoPhaseCommit are parametric; their tests declare the world and instantiate them
- [ ] Intended answers unchanged on the default budget for the reproduced instances
- [ ] Realization fixtures compile against instances

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
