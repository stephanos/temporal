---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-12-lean-test-suite-decomposition-follow-up.5 Split Umpire Query tests by concern

## Description
Split Query tests into visibility, forms, completeness, validation, and identity concerns over shared fixtures (R2-R7). Keep visibility as a genuine public-facade test.

**Size:** M
**Files:** `model/Umpire/Query/Tests.lean`; new `model/Umpire/Query/Tests/{Fixtures,Visibility,Forms,Completeness,Validation,Identity}.lean`
**Touches:** [model/Umpire/Query/Tests.lean, model/Umpire/Query/Tests/**]

## Approach
- Confirm the owned tree matches the fn-10 closure baseline, then map all ten assertions, the visibility guard, and attached comments before moving them.
- Follow the approved layout at `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:132-151`.
- Place the shared model, checked property/behavior, completeness profile, declaration, and error projection in `Fixtures`.
- Make `Visibility` contain one direct import of `Umpire.Query`, with no fixtures or internal test import, so it remains a genuine public-facade check.
- Add a short module comment to every new file, make the root directly import every concern, and preserve canonical projections and semantic identities exactly.

## Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Tests.lean:1-189` — public visibility check and shared query fixtures.
- `model/Umpire/Query/Tests.lean:190-270` — query forms, completeness, and validation.
- `model/Umpire/Query/Tests.lean:271-380` — canonical projection and semantic identity checks.
- `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:132-151` — approved Query layout and ownership.

**Optional** (reference as needed):
- `model/Umpire/Planning/VisibilityTests.lean:1-21` — analogous public-facade visibility test.
- `model/UmpireTests.lean:1-10` — unchanged aggregate entry point.

## Key context
The visibility leaf must directly import only `Umpire.Query`; avoiding a fixtures import alone is insufficient if another internal test import exposes hidden declarations. This is a fresh-agent, serial current-branch task: stop for human direction on baseline drift, do not commit, and do not use a worktree.

## Acceptance
- [ ] `Query/Tests.lean` is import-only and directly imports `Visibility`, `Forms`, `Completeness`, `Validation`, and `Identity`; no fixtures or concern module imports the facade.
- [ ] The evidence map accounts for all ten Query assertions, the public visibility guard, existing comments, canonical projections, semantic identities, and fixture strings exactly once; every new file has a short module comment.
- [ ] `Query/Tests/Visibility.lean` has one direct import of only the public `Umpire.Query` facade and no fixtures or internal test imports.
- [ ] `Fixtures` and every concern module pass direct Lean elaboration, then `cd model && mise exec -- lake build UmpireTests` passes.
- [ ] No production query behavior, public API, dependency, build target, documentation, generated file, commit, or worktree changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
