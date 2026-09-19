---
satisfies: [R3]
---
# fn-49-centralize-observation-field-and.3 Extract Observation structural analysis

## Description
Isolate the common sequence, causality, ordering-support, and closure calculations inside Observation Evaluation (R3).

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/Tests/Evaluation.lean`
**Touches:** [model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/Tests/Evaluation.lean]

### Approach
- Identify the smallest normalized structural facts/findings shared by raw records and accepted Evidence Link support.
- Extract pure internal analysis while leaving raw and accepted diagnostic construction at their callers.
- Add direct or adapter-level regressions for empty, single-source, multi-source, cycle, mixed-origin, and closure-count boundaries.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:424-552` — raw sequence, causality, and closure logic
- `model/Umpire/Observation/Evaluation.lean:1112-1325` — accepted ordering and closure provenance logic
- `model/Umpire/Observation/Tests/Evaluation.lean` — raw Evidence diagnostic precedence fixtures

**Optional** (reference as needed):
- `model/Umpire/Observation/Tests/EvidenceLink.lean:158-250` — multi-source support invariants
## Acceptance
- [ ] Common structural calculations have one Observation-owned implementation that produces one reusable normalized fact set per admission.
- [ ] Internal findings retain enough identity/order/context for both existing diagnostic vocabularies without re-sorting or re-normalizing in adapters.
- [ ] The extraction exposes no new public validation language or generic graph framework.
- [ ] Structural edge-case tests include a 10× evidence fixture, and code inspection confirms one analysis invocation with no added full traversal before caller migration continues.
## Done summary
Added one internal Observation structural analysis that canonicalizes ordering and closure facts, derives reusable closure expectations and stage-ordered identity-rich findings, and retains normalized per-link support with rule identity for accepted-boundary consistency checks. Direct tests cover empty, single-source, multi-source, cycle, mixed-origin, closure-count, link-local missing/reordered support, and 10x evidence inputs without routing production callers yet.
## Evidence
- Commits: 5c049a45ddc4c31c5235bacc790b686c2c94dd22, 654cd054755d2ed8f90104aa5cb8cd1f3a0af88e
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests.Evaluation, cd model && mise exec -- lake build Umpire.Observation, cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests Temporal.System.Nexus.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model, make lint-code (exact inherited 1,379-finding baseline)
- PRs: