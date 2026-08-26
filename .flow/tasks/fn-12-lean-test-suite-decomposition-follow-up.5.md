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
Split the Query regression suite behind an import-only facade while preserving all 10 assertions, the public visibility guard, 47 definitions, nine existing explanatory comments, and the full semantic-string multiset. The facade maps visibility to `Visibility`; shared declarations (`id`, `source`, `phase`, `request`, `accepted`, `observed`, `role`, `value`, `initial`, `completed`, `requestValue`, `acceptedValue`, `observedValue`, `setup`, `transition`, `kernel`, `target`, `checkedProperty`, `checkedBehavior`, `completeness`, `bounds`, `exhaustivePolicy`, `searchPolicy`, `context`, `exhaustiveContext`, `declaration`, `errorKindOf`) to `Fixtures`; `summaryOf` plus the form assertion to `Forms`; `incompleteContext`, `noFiniteDomains`, and four assertions to `Completeness`; `invalidBounds`, `exactTrace`, `invalidExactBehavior`, and two assertions to `Validation`; and the canonical/digest helpers and semantic variants plus three assertions to `Identity`.

`Visibility` directly imports only `Umpire.Query`; each other concern imports `Fixtures`, which itself imports only `Umpire.Query`, so no child imports the test facade. All six leaves elaborate directly, `UmpireTests` and both full model aggregates build, the root Umpire regression passes, and structural/inventory/diff checks are green.

stage: impl-review - ran [2026-08-26T03:30:10Z..2026-08-26T03:33:00Z]
## Evidence
- Commits: 2a7de6d1062e2b07585ec7834bd8c0878b853373
- Tests: GATE_SKIPPED:build:green-receipt edf87325 - baseline reused from prior post-gate pass, GATE_SKIPPED:unittest:green-receipt edf87325 - baseline reused from prior post-gate pass, git diff --check, cd model && mise exec -- lake env lean Umpire/Query/Tests/Fixtures.lean, cd model && mise exec -- lake build Umpire.Query.Tests.Fixtures, cd model && mise exec -- lake env lean Umpire/Query/Tests/Visibility.lean, cd model && mise exec -- lake env lean Umpire/Query/Tests/Forms.lean, cd model && mise exec -- lake env lean Umpire/Query/Tests/Completeness.lean, cd model && mise exec -- lake env lean Umpire/Query/Tests/Validation.lean, cd model && mise exec -- lake env lean Umpire/Query/Tests/Identity.lean, cd model && mise exec -- lake build UmpireTests, Query assertion/comment/definition/semantic-string and import-boundary inventory checks, (cd model && mise exec -- lake build UmpireTests TemporalModelTests), make umpire-check-regression, git diff --check 0bb0781011268c5abf3b2679852f5baae5af60da..HEAD
- PRs: