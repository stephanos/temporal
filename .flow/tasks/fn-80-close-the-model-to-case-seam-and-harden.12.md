---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.12 Add located Nexus3 syntax diagnostics

## Description
Step (c) of task .5's recorded sequence. Depends on step (b).

**Size:** S
**Files:** model/Temporal/Feature/Nexus3/Syntax.lean; model/Temporal/Feature/Nexus3/Tests.lean
**Touches:** [model/Temporal/Feature/Nexus3/Syntax.lean, model/Temporal/Feature/Nexus3/Tests.lean]

### Scope
Add the located diagnostics for the five error classes task .5's block enumerates, each pinned by `#guard_msgs`: constructor with arguments, unknown constructor, duplicate `before + action`, unreachable terminal, and transition count over 256. Their message text needs designing — it is not inherited from anywhere.

Each diagnostic must point at the offending source coordinates, not at the macro.

## Acceptance
- [ ] All five error classes produce a located diagnostic at the offending source coordinates.
- [ ] Each is pinned by a `#guard_msgs` test so the message text cannot drift silently.
- [ ] A newcomer renaming an Action or adding a transition gets an actionable message, which is the Goal-section defect this requirement exists to fix.
- [ ] No fixture bytes move; `make lint-model` adds nothing to the 169 baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
