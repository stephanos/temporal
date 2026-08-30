---
satisfies: [R1, R6]
---
# fn-51-shorten-ordinary-model-authoring.2 Migrate ordinary named Model Values

## Description
Replace eligible ordinary production and shared-positive-fixture Model Value literals with the Core constructor (R1, R6).

**Size:** L
**Files:** `model/Umpire/Examples/Switch.lean`, `model/Temporal/Feature/Nexus/Lifecycle/Target.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`, `model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean`, `model/Temporal/System/Nexus/Core.lean`, `model/Temporal/System/Nexus/CallerClosure.lean`, `model/Temporal/System/Execution/Nexus.lean`, `model/Umpire/Behavior/Tests/Fixtures.lean`, `model/Umpire/Observation/Tests/Fixtures.lean`, `model/Umpire/Planning/Tests/Fixtures.lean`, `model/Umpire/Query/Tests/Fixtures.lean`, `model/Umpire/Property/Tests/Fixtures.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean`
**Touches:** [model/Umpire/Examples/Switch.lean, model/Temporal/Feature/Nexus/Lifecycle/Target.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean, model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean, model/Temporal/System/Nexus/Core.lean, model/Temporal/System/Nexus/CallerClosure.lean, model/Temporal/System/Execution/Nexus.lean, model/Umpire/Behavior/Tests/Fixtures.lean, model/Umpire/Observation/Tests/Fixtures.lean, model/Umpire/Planning/Tests/Fixtures.lean, model/Umpire/Query/Tests/Fixtures.lean, model/Umpire/Property/Tests/Fixtures.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean]

### Approach
- Inventory author-written `ModelValue` literals across `model/Umpire` and `model/Temporal` before editing; classify every retained pair as a deliberate mutation, negative input, runtime/compiler-derived value, or a different record type.
- Migrate direct Definition-ID/value pairs in ordinary production declarations and shared positive fixtures, including Experimental CallerClosure, Variation Space assignments, System execution inputs, Observation/Query/Property fixtures, and their existing positive helpers.
- Keep record literals that intentionally mutate fields, assert raw invalid input, or are constructed by compiler/runtime projection.
- Preserve declaration names, order, source provenance, comments, and exact value strings.
- Use focused identity/fingerprint assertions before aggregate artifact checks.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:57-90` — domain-neutral ordinary values
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean:21-48` — Feature lifecycle values
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean:169-204` — experimental production values
- `model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean:262-280` — authored assignment values
- `model/Temporal/System/Nexus/Core.lean:92-131` — System lifecycle values
- `model/Temporal/System/Nexus/CallerClosure.lean:48-66` — CallerClosure values
- `model/Temporal/System/Execution/Nexus.lean:177-181` — authored execution action
- `model/Umpire/Behavior/Tests/Fixtures.lean:50-70` — shared fixture pattern
- `model/Umpire/Observation/Tests/Fixtures.lean:266-276` — shared positive trace values
- `model/Umpire/Planning/Tests/Fixtures.lean:25-45` — planner fixture pattern
- `model/Umpire/Query/Tests/Fixtures.lean:30-42` — Query fixture value helper
- `model/Umpire/Property/Tests/Fixtures.lean:145-165` — Property fixture value helper and trace
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean:170-180` — positive test helper
## Acceptance
- [ ] A repository-wide inventory covers author-written `ModelValue` pairs in production and shared positive fixtures; every eligible pair uses the named constructor.
- [ ] Every retained literal is classified as a deliberate mutation, negative input, runtime/compiler-derived construction, or non-`ModelValue` record rather than unexplained boilerplate.
- [ ] All names, values, comments, ordering, checked semantics, and fingerprints are unchanged.
- [ ] Switch, lifecycle, experimental, System execution, Behavior, Observation, Query, Property, and Planning focused tests pass.
- [ ] No artifact or generated-view drift is introduced.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
