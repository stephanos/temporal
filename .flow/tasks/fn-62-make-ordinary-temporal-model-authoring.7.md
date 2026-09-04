---
satisfies: [R1, R8, R9]
---
# fn-62-make-ordinary-temporal-model-authoring.7 Publish and verify the checked ordinary-authoring walkthrough

## Description
Complete R1, R8, and R9 with a small compiled teaching model, concise newcomer navigation, public API docs, and full compatibility/trust gates. This task integrates the completed APIs; it does not create a Markdown-only tutorial or another authoring layer.

**Size:** M
**Files:** `model/Umpire/Examples/OrdinaryFinite.lean`, `model/Umpire/Examples/OrdinaryFiniteTests.lean`, `model/UmpireTests.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/Umpire/Examples/OrdinaryFinite.lean, model/Umpire/Examples/OrdinaryFiniteTests.lean, model/UmpireTests.lean, model/README.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Add one minimal finite teaching model using only public Target, Property, Behavior, Query, Planning, and Observation facades; keep the expert Switch example as the `TransitionKernel` reference.
- Demonstrate raw typed failure inspection separately from proof-taking checked extraction, with no hidden native decision or unchecked Markdown claim.
- Show stable IDs, author source, ordered domains/encoders/enumerators/evidence, explicit provider composition, Target-owned outcomes, named stage Limits, Observation mapping, and model-owned Known Gaps in reader order.
- Update `model/README.md:68-179` to link the compiled source as the authority and distinguish illustrative Markdown from checked regressions.
- Update `model/Umpire/ARCHITECTURE.md:119-245,578-649` for ordinary/expert and checked boundaries; tasks `.1`-`.6` own docstrings for the public declarations they change.
- Run focused suites first, then aggregate model, regression, import, trust, formatting, and lint gates; preserve all existing comments touched by the work.

### Investigation targets
**Required** (read before coding):
- `model/README.md:68-179` — current newcomer map and missing checked journey.
- `model/Temporal/Feature/Nexus.lean:5-22` — ordinary facade and reading order.
- `model/Temporal/Feature/Nexus/LifecycleTests.lean:1-22` — current facade smoke-test pattern.
- `model/Temporal/Feature/Nexus/Operations/AsyncStartTests.lean:17-58` — semantic and artifact regression pattern.
- `model/Umpire/Examples/Switch.lean:1-120` — expert example boundary, not a template for ordinary authoring.

**Optional** (reference as needed):
- `model/Umpire/ARCHITECTURE.md:119-245` — existing public authoring contract reference.
- `.plans/LEAN_GUIDELINES.md:124-165` — module/declaration documentation rules.

### Acceptance
- [ ] The teaching source compiles and demonstrates the complete ordinary journey through public facades only; an author need not unfold representations or write dependent equality transport.
- [ ] Checked tests cover the happy path and invalid ID/reference, incomplete Target, invalid transition, invalid Query Limit, invalid Observation, and malformed Known Gap diagnostics.
- [ ] README and public docs explain ordinary versus expert authoring, raw versus checked APIs, explicit composition/outcomes/Limits/gaps, and the authoritative checked source reader order.
- [ ] Existing public imports, IDs, fingerprints, deterministic plans, artifacts, comments, warnings, and trust baseline are preserved except reviewed intentional deltas.
- [ ] Focused and aggregate Lake builds, model regressions, import boundaries, axiom audits, formatting, and `make lint-model` pass.

## Acceptance
- [ ] R1, R8, and R9 are satisfied by checked source and concise docs.
- [ ] `cd model && mise exec -- lake build UmpireTests TemporalModelTests` passes.
- [ ] `make umpire-build-model`, `make umpire-check-regression`, and `make lint-model` pass.
- [ ] No unchecked Markdown fence is cited as verification evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
