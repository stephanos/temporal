---
satisfies: [R1, R6]
---
# fn-51-shorten-ordinary-model-authoring.2 Migrate ordinary named Model Values

## Description
Replace eligible ordinary production and shared-fixture Model Value literals with the Core constructor (R1, R6).

**Size:** M
**Files:** `model/Umpire/Examples/Switch.lean`, `model/Temporal/Feature/Nexus/Lifecycle/Target.lean`, `model/Temporal/System/Nexus/Core.lean`, `model/Temporal/System/Nexus/CallerClosure.lean`, `model/Umpire/Behavior/Tests/Fixtures.lean`, `model/Umpire/Planning/Tests/Fixtures.lean`
**Touches:** [model/Umpire/Examples/Switch.lean, model/Temporal/Feature/Nexus/Lifecycle/Target.lean, model/Temporal/System/Nexus/Core.lean, model/Temporal/System/Nexus/CallerClosure.lean, model/Umpire/Behavior/Tests/Fixtures.lean, model/Umpire/Planning/Tests/Fixtures.lean]

### Approach
- Migrate only direct Definition-ID/value pairs; keep record literals that intentionally mutate fields or demonstrate raw invalid input.
- Preserve declaration names, order, source provenance, comments, and exact value strings.
- Use focused identity/fingerprint assertions before aggregate artifact checks.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:57-90` — domain-neutral ordinary values
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean:21-48` — Feature lifecycle values
- `model/Temporal/System/Nexus/Core.lean:92-131` — System lifecycle values
- `model/Temporal/System/Nexus/CallerClosure.lean:48-66` — CallerClosure values
- `model/Umpire/Behavior/Tests/Fixtures.lean:50-70` — shared fixture pattern
- `model/Umpire/Planning/Tests/Fixtures.lean:25-45` — planner fixture pattern

## Acceptance
- [ ] Eligible ordinary Model Value pairs use the named constructor; retained literals have an existing negative/custom purpose.
- [ ] All names, values, comments, ordering, checked semantics, and fingerprints are unchanged.
- [ ] Switch, lifecycle, System, Behavior, and Planning focused tests pass.
- [ ] No artifact or generated-view drift is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
