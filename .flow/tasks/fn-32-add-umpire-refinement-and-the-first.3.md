---
satisfies: [R2, R4, R6]
---

# fn-32-add-umpire-refinement-and-the-first.3 Author the first isolated Nexus System and Feature Implementation Link

## Description
Add the minimum pure System meaning and focused Nexus Implementation Link leaf for R4 without moving implementation details into Feature.

**Size:** M
**Files:** `model/Temporal/System/Nexus/**`, `model/Temporal/System.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Temporal/System.lean]

### Approach
- Import and consume the ordinary Feature Nexus lifecycle declarations unchanged; treat the Feature file as an investigation target, not a mutation target. AutoClose and CallerClosure remain experimental and outside this production seam.
- Define only the pure mechanism vocabulary needed for the first correspondence.
- Put the cross-import exclusively in the family Implementation Link leaf.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Lifecycle.lean` — canonical start, cancel, and successful-completion product meaning
- `model/Temporal/System/Configuration/Core.lean` — System-owned deep-module pattern
- `model/Temporal/System.lean` — System aggregate
- `model/Temporal.lean` — ordinary aggregate boundary

### Acceptance
- [ ] Feature and base System tests run independently.
- [ ] The focused leaf supplies the exact forward initial/step/coverage witness and proves the declared correspondence.
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
