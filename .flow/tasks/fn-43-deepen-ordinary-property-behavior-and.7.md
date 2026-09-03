---
satisfies: [R7]
---
# fn-43-deepen-ordinary-property-behavior-and.7 Document and verify the deepened authoring boundaries

## Description
Finish R7 after every abstraction has landed: update the public conceptual guides, verify import boundaries, and run the complete model gates once. This is the single finalization task for documentation and cross-layer verification.

**Size:** S
**Files:** `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `model/Umpire/CoreTests/Primitives.lean`
**Touches:** [model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md, model/Umpire/CoreTests/Primitives.lean]

### Approach
- Update the authored-to-checked lifecycle and focused import guidance for the four checked facades and semantic constructors.
- Explain shared Definition identity utilities and DefinitionGraph while making clear that language modules own diagnostics.
- Explain KernelMorphism/ForwardSimulation as the reusable proof-bearing core below Link-owned indexing/coverage/Known Gaps.
- Explain CanonicalJson ordered construction/rendering while retaining the exact-byte artifact contract and narrow import boundary.
- Review learner examples and all changed module/public docstrings for plain-language completeness; preserve existing teaching text and comments.
- Run focused suites, the full model build, linting, and trust/axiom audits; regenerate only owner-required compatibility outputs and verify they are unchanged.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:38-77` — current checked lifecycle and Core/language ownership.
- `model/Umpire/ARCHITECTURE.md:246-280` — Implementation Link conceptual boundary.
- `model/Umpire/ARCHITECTURE.md:395-450` — JSON and typed-error invariants.
- `model/ARCHITECTURE.md:141-180` — model authoring lifecycle visible outside Umpire internals.
- `model/README.md:68-123` — learner path through Switch, Nexus Lifecycle, Operations, and Observation.
- `model/README.md:180-240` — artifact and Implementation Link explanations.

### Key context
- This is an internal API refactor: no changelog, runtime operations guide, external API spec, or ADR is required.
- Public `check*` functions are not deprecated; docs distinguish ordinary valid authoring from typed diagnostic recovery.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Umpire.Observation.Tests Umpire.ImplementationLink.Tests UmpireTests Temporal TemporalModelTests TemporalExperimentalTests temporal-model-inspect
cd .. && make umpire-build-model
make lint-model
```

## Acceptance
- [ ] All three learner/public architecture documents explain the checked lifecycle, semantic constructors, shared identity/graph ownership, forward simulation, and ordered JSON in plain language at the correct import boundary.
- [ ] Raw typed checkers remain documented and discoverable; no documentation calls them deprecated or hides their diagnostic role.
- [ ] Switch and Nexus walkthroughs read through the semantic APIs without losing existing teaching comments or authored documentation values.
- [ ] Focused suites, the full model build, and `make lint-model` pass; required canonical/generated compatibility checks report no unintended byte or artifact drift.
- [ ] Trust/axiom inspection for changed public declarations is recorded and shows no new unapproved dependency.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
