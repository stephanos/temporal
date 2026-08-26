---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-12-lean-test-suite-decomposition-follow-up.2 Split Umpire Behavior tests by concern

## Description
Split the Behavior suite into the approved admission, validation, canonicalization, and narrowing concerns behind its stable facade (R2-R7). This task owns only the Behavior test tree.

**Size:** M
**Files:** `model/Umpire/Behavior/Tests.lean`; new `model/Umpire/Behavior/Tests/{Fixtures,Admission,Validation,Canonicalization,Narrowing}.lean`
**Touches:** [model/Umpire/Behavior/Tests.lean, model/Umpire/Behavior/Tests/**]

## Approach
- Confirm the owned tree matches the fn-10 closure baseline, then map all 24 original assertions and five explanatory comments to their destination or approved removal.
- Follow the approved layout at `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:69-88`.
- Centralize only the base context, shared values/traces/occurrences, exact witness, constrained declaration, and admission helper in `Fixtures`.
- Remove only the reflexive canonical self-comparison at the current baseline location; preserve the other 23 assertions, semantic strings, and all existing comments exactly once.
- Add a short module comment to every new file, make the root directly import every concern, and keep every child independent of the root.

## Investigation targets
**Required** (read before coding):
- `model/Umpire/Behavior/Tests.lean:1-208` — base fixtures and admission behavior.
- `model/Umpire/Behavior/Tests.lean:209-350` — authoring, satisfiability, schedule, and occurrence validation.
- `model/Umpire/Behavior/Tests.lean:351-515` — canonicalization and narrowing laws, including the vacuous comparison at line 399.
- `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:69-88` — approved Behavior layout and preservation counts.

**Optional** (reference as needed):
- `model/Umpire/Behavior/Tests.lean:195-207` — attached admission comments.
- `model/Umpire/Behavior/Tests.lean:279` — attached validation comment.
- `model/Umpire/Behavior/Tests.lean:496-510` — attached narrowing comments.

## Key context
Preserve the established closed-computation proof style and use `rfl` only where definitional equality is the tested contract. This is a fresh-agent, serial current-branch task: stop for human direction on baseline drift, do not commit, and do not use a worktree.

## Acceptance
- [ ] `Behavior/Tests.lean` is import-only and directly imports `Admission`, `Validation`, `Canonicalization`, and `Narrowing`; no fixtures or concern module imports the facade.
- [ ] The evidence map identifies exactly the approved reflexive self-comparison as removed and accounts for the other 23 assertions and all five explanatory comments exactly once; semantic strings are unchanged.
- [ ] Admission, validation, canonicalization, and narrowing fixtures remain owned by their concern unless shared by multiple concerns; every new file has a short module comment.
- [ ] `Fixtures` and every concern module pass direct Lean elaboration, then `cd model && mise exec -- lake build UmpireTests` passes.
- [ ] No production behavior, public API, dependency, build target, documentation, generated file, commit, or worktree changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
