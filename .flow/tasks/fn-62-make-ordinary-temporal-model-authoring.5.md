---
satisfies: [R1, R6, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.5 Deepen typed Observation construction and migrate Nexus

## Description
Complete the uncovered Observation authoring stage using the current inert declaration owner and migrate its established Nexus consumer after `.2`/`.3`.

**Size:** M
**Files:** `model/Umpire/Observation/Declaration.lean`, Observation tests/imports, established Nexus Observation declaration/tests
**Touches:** [model/Umpire/Observation/Declaration.lean, model/Umpire/Observation/ImportTests.lean, model/Umpire/Observation/Tests/**, model/Temporal/Feature/Nexus/Observation.lean, model/Temporal/Feature/Nexus/ObservationTests.lean]

## Approach
- Compose typed profile/rule/mapping constructors from existing `ObservationFieldSpec` leaves. Keep fields, output kinds, dispositions, providers, order, closures, and limits explicit.
- Delegate validation to `checkObservation`; checked construction continues to require proof. Add no parallel validation or interpretation language.
- Migrate the established declaration and compare exact raw/helper checked values, fingerprint, accepted facts, Evidence links, order/closure support and sources.
- Extend existing compile/evaluation matrices to cover helper-produced declarations; preserve full diagnostic precedence, not just error presence.

## Investigation targets
**Required:**
- `model/Umpire/Observation/Declaration.lean:60` — typed leaf projections.
- `model/Umpire/Observation/Language.lean:12` — checked proof seam.
- `model/Umpire/Observation/Tests/Compilation.lean:186` — declaration rejection matrix.
- `model/Temporal/Feature/Nexus/Observation.lean` — migration.
- `model/Temporal/Feature/Nexus/ObservationTests.lean:113` — exact identity and Evidence tests.

## Acceptance
- [ ] Inert constructors and migrated Nexus preserve exact checked mapping meaning, field identities, fingerprint, sources and public imports.
- [ ] Compile failures cover missing/unknown/wrong-type fields, duplicate mappings/dispositions, absent dispositions, conflicting/unresolved providers, wrong output kind, contradictory/cyclic order, missing/duplicate closures, and invalid bounds.
- [ ] Evaluation failures cover over-limit, missing/ambiguous/conflicting Evidence, profile mismatch, rejected fields, and causal/order/closure failures, with unchanged facts/links and diagnostic precedence.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests` passes; public import, trust and lint checks pass with no new issues.
- [ ] Structural cost inventory names helper callees and traversals: at most one construction pass per explicit input collection, no nested rescan, normalization, or duplicate checkObservation pass; 10× independent declarations adds at most 10× wrapper work.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
