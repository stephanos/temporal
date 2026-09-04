---
satisfies: [R1, R6, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.5 Deepen typed Observation authoring

## Description
Satisfy R1, R6, and R8 with typed constructors for repeated Observation profiles, rules, and mappings, then migrate the ordinary Nexus Observation while preserving the recently partitioned declaration/compiler boundary.

**Size:** M
**Files:** `model/Umpire/Observation/Declaration.lean`, `model/Umpire/Observation/Language.lean`, `model/Umpire/Observation/Tests/Check.lean`, `model/Temporal/Feature/Nexus/Observation.lean`, `model/Temporal/Feature/Nexus/ObservationTests.lean`
**Touches:** [model/Umpire/Observation/Declaration.lean, model/Umpire/Observation/Language.lean, model/Umpire/Observation/Tests/Check.lean, model/Temporal/Feature/Nexus/Observation.lean, model/Temporal/Feature/Nexus/ObservationTests.lean]

### Approach
- Build on the completed `fn-57` split: inert constructors belong with Declaration/Language and all validation remains in the single compiler/checker.
- Generalize only repeated semantic shapes already visible in `model/Temporal/Feature/Nexus/Observation.lean:95-161`; helpers return existing profile, rule, mapping, and disposition data.
- Keep source field identity, expected kind, explicit disposition, causal/order relation, provider reconciliation, Evidence bound, and closure visible at call sites.
- Preserve raw `checkObservation` and proof-taking `checkedObservation`; add representative invalid/missing/conflicting Evidence tests rather than a success-only fixture.
- Retain all existing comments and exact checker diagnostic precedence.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Declaration.lean` — inert authored vocabulary after fn-57.
- `model/Umpire/Observation/Language.lean:1-25` — public proof-taking authoring facade.
- `model/Temporal/Feature/Nexus/Observation.lean:95-161` — repeated ordinary mapping surface.
- `model/Umpire/Observation/Tests/Check.lean` — typed diagnostic patterns.
- `model/Umpire/Observation/Tests/Structure.lean` — ordering/closure failure patterns.

**Optional** (reference as needed):
- `.flow/specs/fn-57-partition-the-observation-authoring.md:17-50` — preserved ownership and checker contracts.
- `model/Temporal/Feature/Nexus/ObservationTests.lean` — current facade regressions.

### Acceptance
- [ ] Ordinary mapping code expresses typed fields, dispositions, ordering, providers/connectors, closures, and Evidence bounds without manual string-comparison boilerplate.
- [ ] Helpers produce only existing inert Observation data and do not validate, normalize, register, or default semantics.
- [ ] Invalid field/kind/reference, duplicates, missing dispositions, provider conflicts, invalid order/closure, over-limit input, and missing/ambiguous/conflicting Evidence fail closed with exact diagnostics.
- [ ] Existing Observation identity, fingerprint, canonical plan, accepted facts, Evidence Links, public imports, and comments remain exact except reviewed source-location corrections.
- [ ] Focused generic and Nexus Observation tests pass.

## Acceptance
- [ ] R1, R6, and R8 are satisfied through the existing Observation language/checker.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Temporal.Feature.Nexus.ObservationTests` passes.
- [ ] No callback, recursive authoring form, default disposition, or alternate checked-plan constructor is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
