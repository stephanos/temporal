---
satisfies: [R5]
---
# fn-197-correlated-obligations.3 DSL completion: SEMANTICS, README and progress-notation describe the word

## Description
**Note 2026-10-18 (one-language series):** `export` is `lift` (fn-201). Run conformance against `owes` waits for fn-204's descriptor (or the fn-209 oracle), not fn-196's conformance replay.

**DSL completion phase (owner, 2026-10-17).** After .2. Add the word to SEMANTICS §11 (meaning: per-payload leads-to over recorded facts, owed set in state identity, the three violation kinds and the bound; writing it; refused), to the framework README's progress text, and to `model/design/progress-notation.md`'s "When a claim gets a word" table with the justification (reads later steps and correlates them by payload). If .1's note found that ReliableBroadcast's `someoneDelivered(current)` claim reads better as `owes`, rewrite it and pin its answer in `SampleClaims.test.scala`; otherwise record in the note why not.

**Size:** S

**Files:** `model/framework/SEMANTICS.md`, `model/framework/README.md`, `model/design/progress-notation.md`, optionally `model/framework/samples/ReliableBroadcast.scala` and `SampleClaims.test.scala`.

### Verification
- The docs lint (if any) and the samples suite pass.
- A reader of SEMANTICS §11 can write the fixture's claim without reading the source.
## Acceptance
- [ ] SEMANTICS §11, README and progress-notation.md describe owes/until and why it is a word
- [ ] ReliableBroadcast rewritten or the reason recorded

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: