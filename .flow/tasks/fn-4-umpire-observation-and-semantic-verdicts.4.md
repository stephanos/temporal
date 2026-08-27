---
satisfies: [R2, R3, R5, R6, R8]
---
# fn-4-umpire-observation-and-semantic-verdicts.4 Add one Temporal-owned synthetic observation profile

## Description
Define one synthetic Nexus lifecycle profile and demonstrate the full offline handoff for R2/R3/R5/R6/R8 without runtime imports.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Observation.lean`, `model/Temporal/Feature/Nexus/ObservationTests.lean`, `model/Temporal/Feature.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Observation.lean, model/Temporal/Feature/Nexus/ObservationTests.lean, model/Temporal/Feature.lean, model/TemporalModelTests.lean]

### Approach

- Map one BasicLifecycle synthetic evidence profile to current target vocabulary.
- Demonstrate compiled mapping, qualification, property verdicts, and strict summary.
- Author expected trace and derivations independently.
- Expose only the typed future adapter handoff.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:53-89` — normalized values.
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:194-259` — kernel and meanings.
- `model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean:13-34` — independent checks.
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean:71-97` — reusable Property use.
- `model/ARCHITECTURE.md:121-162` — ownership boundary.

## Acceptance
- [ ] Valid fixture produces the independent qualified trace and satisfied strict summary.
- [ ] Incomplete, ambiguous, conflicting, unsupported, and rejected-field fixtures return exact non-success outcomes.
- [ ] The profile emits only authorized BasicLifecycle vocabulary.
- [ ] No server, service, Go runtime, persistence, or Tool import is introduced.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests` passes.

## Done summary
Added one Temporal-owned synthetic BasicLifecycle Observation profile with a typed `EvidenceBundle` handoff through qualification, independent Property verdicts, and strict aggregation. Closed mapping predicates and independent fixtures cover the valid trace plus incomplete, ambiguous, conflicting, unsupported, rejected, empty, and unknown evidence without runtime imports.

baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests failed pre-edit: task target absent)
stage: impl-review - ran [2026-08-27T08:17:06Z..2026-08-27T08:19:24Z]
## Evidence
- Commits: 0e75e27f4dd0f1875df9ff6e7968bc78154b3ba9, 4a2287034ab190fcb79dba3440e105ae3b39e42b
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests failed pre-edit: task target absent), cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation, cd model && mise exec -- lake build Umpire.Observation.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests, make umpire-check-regression
- PRs: