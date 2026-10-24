---
satisfies: [R6]
---
# fn-195-modelsuite-a-models-test-file-holds-its.4 DSL completion: the six samples' tests are ModelSuites with Quint's runs

## Description
**DSL completion phase (owner, 2026-10-17).** Rewrite the six samples' test files as `ModelSuite`s: Quint's runs as `examples` (Bank's `sendTest` and `getAllBalancesTest`, TwoPhaseCommit's `successTest`, Kettle's boil from room temperature with `from = room`), the hand-written reachability finds as `explorations`, the negative controls as `refuted`, one seeded `sample` per sample, and the former `.live` witnesses as `conformance` tests; the native predicate truth tables become plain tests of the pure helpers or go. The intended answers in the native-migration inventory are reproduced; a changed answer is a finding to fix.

**Size:** M

**Files:** `model/framework/samples/*.test.scala` (six, plus `NativeInputs.test.scala`, `NativeHistory.test.scala`, `NativeVoting.test.scala` folded in or kept as plain tests), `samples/testdata/native-migration/inventory.md`.

### Verification
- The samples native suite passes; the inventory's answers are reproduced on the default budget.
- Each sample has at least one example, one exploration and, where a realization exists, one conformance test.

## Acceptance
- [ ] Six ModelSuites with Quint's runs as examples, finds as explorations, controls as refuted
- [ ] Intended answers reproduced on the default budget

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: