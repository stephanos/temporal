---
satisfies: [R1, R2, R3, R4, R5, R6, R7]
---
# fn-12-lean-test-suite-decomposition-follow-up.7 Integrate and verify the decomposed Lean suites

## Description
Integrate the six suite-local splits and verify the complete closed-fn-10 inventory and regression contract (R1-R7). This downstream coordinator owns aggregate wiring and demonstrable cross-suite split defects only.

**Size:** M
**Files:** the six split test roots and concern trees; stable `model/UmpireTests.lean` and `model/TemporalModelTests.lean` only if missing fn-12 reachability requires reconciliation
**Touches:** [model/Umpire/CoreTests.lean, model/Umpire/CoreTests/**, model/Umpire/Behavior/Tests.lean, model/Umpire/Behavior/Tests/**, model/Umpire/Property/Tests.lean, model/Umpire/Property/Tests/**, model/Umpire/Planning/Tests.lean, model/Umpire/Planning/Tests/**, model/Umpire/Query/Tests.lean, model/Umpire/Query/Tests/**, model/Temporal/System/Configuration/Tests.lean, model/Temporal/System/Configuration/Tests/**, model/UmpireTests.lean, model/TemporalModelTests.lean]

## Approach
- Verify fn-10 is closed and anchor the final integration audit to its closure baseline; do not amend fn-10 or bypass the spec dependency.
- Recount the seven approved original large-suite surfaces: six roots split from the fixed inventory plus the already-decomposed Temporal owner aggregate.
- Review each root for direct concern coverage, each child for absence of facade imports, Query visibility for its direct-only public import, and the two read-only owner visibility/Callback files byte-for-byte.
- Reconcile the six declaration/comment maps, the two approved vacuous-test removals, focused `uniquenessProperty`, unchanged semantic strings, and short module comments.
- Preserve unrelated fn-11 aggregate edits. Change a stable aggregate only for demonstrated missing fn-12 reachability; reopen a suite task rather than silently repartitioning or weakening it here.
- Run `rg --no-ignore` over all six owned source trees so untracked new Lean leaves participate in trailing-whitespace checks, and mirror the root Makefile's forbidden import/namespace/semantic-prefix pattern over the five reusable Umpire test trees.
- Run the full aggregate, regression, and whitespace gates after every focused check is green.

## Investigation targets
**Required** (read before coding):
- `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:204-243` — sequencing, integration ownership, verification, and non-goals.
- `model/UmpireTests.lean:1-10` — stable reusable aggregate that must still reach every reusable suite facade.
- `model/TemporalModelTests.lean:1-4` — stable owner aggregate; preserve unrelated fn-11 imports.
- `Makefile:1014-1105` — full Umpire regression and domain/import guards.
- `AGENTS.md` — repository verification and implementation conventions.

**Optional** (reference as needed):
- `.flow/specs/fn-11-basic-nexus-umpire-dsl-showcases.md` — coordination scope if fn-11 has changed the Temporal aggregate.

## Key context
All six fresh suite workers must have completed serially before this task. The coordinator may fix aggregate reachability or another narrow integration defect, but a suite-local assertion or partition defect returns to that suite task. Make only task-scoped flow-next commits and do not use a worktree.

## Acceptance
- [ ] fn-10 is closed, its spec dependency remains recorded, and the final inventory accounts for six split roots plus the below-threshold owner-specific Temporal suites without modifying fn-10.
- [ ] Every import-only root directly covers every concern module, no child imports its facade, fixtures are suite-local, Query visibility imports only `Umpire.Query`, and Planning visibility plus Callback configuration tests remain byte-for-byte unchanged.
- [ ] The six declaration/comment maps reconcile exactly, except for the two approved reflexive removals; the Property uniqueness failure evaluates `uniquenessProperty`; semantic strings and new module comments satisfy the spec.
- [ ] Stable aggregates retain unrelated changes and reach every fn-12 facade; any aggregate edit is limited to a demonstrated missing import and no suite-local defect is silently rewritten here.
- [ ] `rg --no-ignore --glob '*.lean'` reports no trailing whitespace across all six split trees and no forbidden Temporal/Nexus imports, namespaces, or semantic prefixes across the five reusable Umpire test trees, including untracked new leaves.
- [ ] `(cd model && mise exec -- lake build UmpireTests TemporalModelTests)`, `make umpire-check-regression`, and `git diff --check` all pass.
- [ ] No production/public behavior, generated API or dynamic-config file, Lake/Make/CI wiring, documentation, dependency, unrelated commit, or worktree change is present; only task-scoped worker and lifecycle commits are allowed.

## Done summary
Audited the six decomposed Lean suites against the closed fn-10 baseline and found no integration correction necessary: facades, concern imports, declarations, comments, semantic strings, approved removals, stable aggregates, and read-only visibility/configuration guards all reconcile. Every split leaf and facade elaborates directly, and the terminal aggregate build, Umpire regression, structural/domain scans, and whitespace checks pass.

stage: impl-review - ran [2026-08-26T03:56:46Z..2026-08-26T04:04:39Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 31c773f8dc2a1d67419c06fabbfb7d252dd02187
- Tests: GATE_SKIPPED:build:green-receipt bc7a52ac - baseline reused from prior post-gate pass, GATE_SKIPPED:unittest:green-receipt bc7a52ac - baseline reused from prior post-gate pass, git diff --check (baseline), structural/inventory audit: direct facade coverage, no child facade imports, suite-local fixtures, Query public visibility, byte-stable Planning visibility and Callback configuration, exact declarations/comments/semantic strings, approved removals, uniquenessProperty, rg --no-ignore whitespace/domain guards, (cd model && for each of the 36 split leaves/facades; do mise exec -- lake env lean <module>; done), (cd model && mise exec -- lake build UmpireTests TemporalModelTests), make umpire-check-regression, git diff --check
- PRs:
