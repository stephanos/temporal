---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.11 Elaborate Nexus3 identifiers from constructors, not whitelists

## Description
Step (b) of task .5's recorded sequence. Depends on step (a).

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Syntax.lean; model/Temporal/Feature/Nexus3/Authoring.lean
**Touches:** [model/Temporal/Feature/Nexus3/Syntax.lean, model/Temporal/Feature/Nexus3/Authoring.lean]

### Scope
Replace the four spelling whitelists with constructor-derived elaboration over the ordered data model step (a) delivered. Keep the Nexus3 success declaration character-for-character as it is today, so this step also moves no fixture bytes.

Task .5's block flags this as new ground: the repo has no `Lean.Elab.Command` elaborator that reads an inductive's constructors via `getConstInfoInduct`. Design that reader here; its diagnostics land in step (c).

This is what R8 (task .6, the `query ... all ...` verify form) follows directly — .6's own approach is small and sits on top of this step.

## Acceptance
- [ ] The four identifier whitelists are gone; elaboration derives admissible spellings from the declaring inductive's constructors.
- [ ] A renamed Action or an added transition elaborates instead of producing a whitelist miss.
- [ ] The Nexus3 success declaration is unchanged character-for-character and no fixture bytes move.
- [ ] `make umpire-check-case-runtime-conformance` green; `make lint-model` adds nothing to the 169 baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
