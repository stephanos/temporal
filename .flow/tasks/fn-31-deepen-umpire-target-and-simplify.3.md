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
- Replace routine provider/connector/extraction/planner assembly with the public Target path and opt the target into Target-owned finite planning once.
- Preserve the existing target-owned transition kernel and all fixtures.
- Preserve the existing `switch-role-domain/v1` and `switch-action-domain/v1` compatibility tokens verbatim at the Target finite-planning declaration.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:237-259` — current declaration, `composeTarget`, and checked extraction
- `model/Umpire/Examples/Switch.lean:409-548` — finite completeness, ordering, and planning plumbing
- `model/Umpire/Examples/SwitchTests.lean` — reference behavior

### Acceptance
- [ ] The example demonstrates ordinary authoring rather than framework plumbing.
- [ ] Existing semantic identities, plans, artifacts, and invalid cases are unchanged.
- [ ] Query derivation copies the existing role/action-domain tokens verbatim; ordinary query/planner code no longer threads them.

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
