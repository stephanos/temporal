---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-12-lean-test-suite-decomposition-follow-up.4 Split Umpire Planning tests by concern

## Description
Split Planning tests into outcomes, artifacts, and enumeration concerns over shared fixtures (R2-R7). Preserve the separate public-visibility regression unchanged.

**Size:** M
**Files:** `model/Umpire/Planning/Tests.lean`; new `model/Umpire/Planning/Tests/{Fixtures,Outcomes,Artifacts,Enumeration}.lean`
**Touches:** [model/Umpire/Planning/Tests.lean, model/Umpire/Planning/Tests/**]

## Approach
- Confirm the owned tree matches the fn-10 closure baseline, then map the ten Planning assertions and their comments before movement; record both separate visibility guards as read-only baseline evidence.
- Follow the approved layout at `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:112-130`.
- Put the deterministic model, checked query, incremental kernel, and planning runner in `Fixtures`; keep outcome, artifact, and enumeration-specific values local.
- Replace `Planning/Tests.lean` with direct imports for all concern modules without modifying `Planning/VisibilityTests.lean`.
- Add a short module comment to every new file and preserve artifact bytes, semantic identities, comments, and semantic source strings exactly.

## Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Tests.lean:1-238` — deterministic model, checked query, incremental kernel, and runner fixtures.
- `model/Umpire/Planning/Tests.lean:239-354` — query outcomes, invalid/absence/exhaustion, and unsatisfiable checks.
- `model/Umpire/Planning/Tests.lean:355-383` — artifact determinism, identity, and enumeration instrumentation.
- `model/Umpire/Planning/VisibilityTests.lean:1-21` — independent public-facade guards that must remain byte-for-byte unchanged.
- `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:112-130` — approved Planning layout.

## Key context
The visibility suite must continue importing only the public `Umpire.Planning` facade and must not gain access to internal fixtures. This is a fresh-agent, serial current-branch task: stop for human direction on baseline drift, do not commit, and do not use a worktree.

## Acceptance
- [ ] `Planning/Tests.lean` is import-only and directly imports `Outcomes`, `Artifacts`, and `Enumeration`; no fixtures or concern module imports the facade.
- [ ] The evidence map accounts for all ten Planning assertions, comments, artifact bytes, semantic identities, and fixture strings exactly once; every new file has a short module comment.
- [ ] `Planning/VisibilityTests.lean` remains byte-for-byte unchanged, separate, importing only `Umpire.Planning`, with both visibility guards intact.
- [ ] `Fixtures` and every concern module pass direct Lean elaboration, then `cd model && mise exec -- lake build UmpireTests` passes.
- [ ] No production planning behavior, public API, dependency, build target, documentation, generated file, commit, or worktree changes.

## Done summary
Split the reusable Planning regression suite behind its stable import-only facade into shared fixtures plus Outcomes, Artifacts, and Enumeration concerns. The declaration map at `.flow/tmp/fn12-4-planning-declaration-map.md` accounts for all ten assertions and comments, preserves semantic strings, and records the byte-identical separate visibility guards.

stage: impl-review - ran (model: gpt-5.6-sol)
## Evidence
- Commits: edf87325ba74b272d4b7c7dec063c98d42ca0bde
- Tests: GATE_SKIPPED:build:green-receipt 84754567 - baseline reused from prior post-gate pass, GATE_SKIPPED:unittest:green-receipt 84754567 - baseline reused from prior post-gate pass, git diff --check, cd model && mise exec -- lake env lean Umpire/Planning/Tests/Fixtures.lean, cd model && mise exec -- lake build Umpire.Planning.Tests.Fixtures, cd model && mise exec -- lake env lean Umpire/Planning/Tests/Outcomes.lean, cd model && mise exec -- lake env lean Umpire/Planning/Tests/Artifacts.lean, cd model && mise exec -- lake env lean Umpire/Planning/Tests/Enumeration.lean, cd model && mise exec -- lake build UmpireTests, Planning assertion/comment/definition/semantic-string and import-boundary inventory checks, Planning/VisibilityTests.lean byte-identity and visibility-guard preservation checks, (cd model && mise exec -- lake build UmpireTests TemporalModelTests), make umpire-check-regression, git diff --check 66c266a3e38ed59424268fea18fa71ca51d9dd07..HEAD
- PRs: