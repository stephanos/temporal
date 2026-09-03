---
satisfies: [R2, R6]
---
# fn-49-centralize-observation-field-and.2 Migrate Observation fixtures and Nexus mappings

## Description
Use field specifications in the shared test fixture and both ordinary Temporal Observation mappings (R2).

**Size:** M
**Files:** `model/Umpire/Observation/Tests/Fixtures.lean`, `model/Temporal/Feature/Nexus/Observation.lean`, `model/Temporal/Feature/Nexus/ObservationTests.lean`, `model/Temporal/System/Nexus/Observation.lean`, `model/Temporal/System/Nexus/Tests.lean`
**Touches:** [model/Umpire/Observation/Tests/Fixtures.lean, model/Temporal/Feature/Nexus/Observation.lean, model/Temporal/Feature/Nexus/ObservationTests.lean, model/Temporal/System/Nexus/Observation.lean, model/Temporal/System/Nexus/Tests.lean]

### Approach
- Replace duplicated kind/field/type literals with named specs while leaving profile, rule, ordering, closure, and disposition lists explicit.
- Preserve source locations, IDs, comments, field order, checked mapping order, and independent expected traces.
- Assert checked-plan and canonical identity equivalence at each Temporal leaf.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Tests/Fixtures.lean:35-107,141` — minimal reusable profile and independent trace
- `model/Temporal/Feature/Nexus/Observation.lean:35-145` — ordinary single-kind declaration and mapping
- `model/Temporal/System/Nexus/Observation.lean:45-250` — multi-kind profile and repeated references
- `model/Temporal/Feature/Nexus/ObservationTests.lean` — Feature identity and behavior checks
- `model/Temporal/System/Nexus/Tests.lean` — composed System compatibility checks

## Acceptance
- [ ] Each migrated field identity/type is authored once and all prior declarations/references/dispositions remain explicit and equal.
- [ ] Feature and System checked plans, source provenance, statuses, field order, and comments are preserved.
- [ ] Canonical identities, fingerprints, artifacts, and independent expected traces do not drift.
- [ ] Focused Umpire and Temporal Observation tests pass.

## Done summary
Migrated shared Observation fixtures and Feature/System Nexus mappings to named ObservationFieldSpec values as the sole kind/field/type authority while preserving explicit lists, public identity projections, source locations, comments, field order, checked plans, canonical fingerprints, generated artifacts, and independent expected traces. Migrated the one portable-evaluation fixture consumer found by the full regression gate.
## Evidence
- Commits: 8baafcebc, 3ef68b3b9
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ObservationFaultTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model, make lint-code (exact inherited 1,379-finding baseline)
- PRs: