---
satisfies: [R2, R3, R5]
---
# fn-31-deepen-umpire-target-and-simplify.3 Migrate the domain-neutral Switch teaching target

## Description
Move the minimum Umpire teaching example to the ordinary Target interface and prove compatibility (R2, R3, R5).

**Size:** S
**Files:** `model/Umpire/Examples/Switch.lean`, `model/Umpire/Examples/SwitchTests.lean`
**Touches:** [model/Umpire/Examples/Switch.lean, model/Umpire/Examples/SwitchTests.lean]

### Approach
- Replace routine provider/connector/extraction/planner assembly with the public Target path.
- Preserve the existing target-owned transition kernel and all fixtures.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:180-280` — current provider/target assembly
- `model/Umpire/Examples/Switch.lean:350-570` — checked extraction and planning plumbing
- `model/Umpire/Examples/SwitchTests.lean` — reference behavior

### Acceptance
- [ ] The example demonstrates ordinary authoring rather than framework plumbing.
- [ ] Existing semantic identities, plans, artifacts, and invalid cases are unchanged.

## Acceptance
- [ ] R2/R3 are demonstrated by the domain-neutral example.
- [ ] R5 whole-value and byte fixtures pass.
- [ ] Switch remains independent of Temporal.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
