---
satisfies: [R4]
---
# fn-22-deterministic-replay-semantic.3 Compile typed reduction candidates through checked authority

## Description
Define the private typed ReductionEdit and ReplayCandidate boundary for actions, ordering constraints, and requested faults. Implement the one exact Temporal Nexus candidate authority that reconstructs the authored fn-21 point, applies one semantic edit, and asks fn-16 checking/lowering/planning plus fn-18 admission to produce a new immutable input set. Never mutate JSON/wire structs or author an observed outcome. Canonicalize edit and candidate identities, dependencies, rejection reasons, and fixed enumeration order. Prove the no-fault edit compiles to the existing ordinary caller-closure expected count-one plan, while invalid sole-action or closure-breaking edits reject without execution. Return the closed non-applicable `configuration` class because fn-16/fn-18 expose no removable RuntimeConfiguration coordinate in this binding.

**Size:** M
**Files:** `tools/umpire/replay/candidate.go`, `tools/umpire/replay/candidate_test.go`, `model/Temporal/Feature/Nexus/CallerClosureReplayCandidate.lean`, `model/Temporal/Feature/Nexus/CallerClosureReplayCandidateTests.lean`
**Touches:** [tools/umpire/replay/candidate.go, tools/umpire/replay/candidate_test.go, model/Temporal/Feature/Nexus/CallerClosureReplayCandidate.lean, model/Temporal/Feature/Nexus/CallerClosureReplayCandidateTests.lean]

## Acceptance
Every executable candidate is newly produced by the checked authoring/planning boundary and passes fn-18 input-set admission. The fixed action/ordering/fault edit order and identities are stable across map/list order, and configuration is reported exactly once as non-applicable without an edit. The exact fault-removal candidate has the ordinary expected count-one plan; the observed count-two result is never an input. Invalid, duplicate, stale, crossing, or non-applicable edits are deterministic recorded rejections and cannot allocate a runtime or emit candidate bytes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
