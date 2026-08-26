---
satisfies: [R2, R4, R6]
---
# fn-32-add-umpire-refinement-and-the-first.3 Author the first isolated Nexus System and Feature refinement

## Description
Add the minimum pure System meaning and focused Nexus refinement leaf for R4 without moving implementation details into Feature.

**Size:** M
**Files:** `model/Temporal/System/Nexus/**`, `model/Temporal/System.lean`, `model/Temporal/Feature/Nexus/CallerClosure.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Temporal/System.lean]

### Approach
- Keep the existing Feature caller-closure declarations unchanged.
- Define only the pure mechanism vocabulary needed for the first correspondence.
- Put the cross-import exclusively in the family Refinement leaf.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/CallerClosure.lean:18-258` — canonical product meaning
- `model/Temporal/System/Configuration/Core.lean` — System-owned deep-module pattern
- `model/Temporal/System.lean` — System aggregate
- `model/Temporal.lean` — ordinary aggregate boundary

### Acceptance
- [ ] Feature and base System tests run independently.
- [ ] The focused leaf proves the declared correspondence.
- [ ] Feature has no System/Verify import and System mechanism code has no Feature import.

## Acceptance
- [ ] R4 positive correspondence passes.
- [ ] Import-direction mutations fail.
- [ ] Existing Feature identities/artifacts remain unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
