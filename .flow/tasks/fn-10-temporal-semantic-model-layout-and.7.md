---
satisfies: [R1, R2, R3, R4, R5, R6]
---
# fn-10-temporal-semantic-model-layout-and.7 Complete the clean target cutover and regression guards

## Description
Perform the final atomic integration cutover: expose only the approved Temporal aggregates, rename Lake test/executable targets, remove all temporary roots and the old directory, relocate the golden fixture, strengthen source and dependency guards, and update live model documentation (R1-R6).

**Size:** M
**Files:** `model/lakefile.toml`, `Makefile`, `model/README.md`, `model/Temporal.lean`, obsolete Temporal Umpire/root modules and the caller-closure fixture
**Touches:** [model/lakefile.toml, Makefile, model/README.md, model/Temporal.lean, model/NexusAutoClose.lean, model/TemporalUmpireTests.lean, model/Temporal/Umpire/**, model/Temporal/Feature/Nexus/testdata/**]

### Approach
- Make the production Temporal aggregate import Feature and System facades explicitly, excluding Tool executable code.
- Replace old Lake roots/targets with `TemporalModelTests` and `temporal-model-inspect`; remove the standalone auto-close target.
- Update only the repository-root Makefile, keeping `make umpire-check-regression` stable while retargeting builds, fixtures, stale-name checks, domain-purity scans, dependency-direction checks, deterministic output checks, and diagnostic checks.
- Guard every committed text artifact under `model/Umpire/**`, including Lean and JSON fixtures, against reverse imports and specific Temporal product prefixes; explicitly exclude generated build/runtime state and match leading whitespace plus import-name boundaries.
- Add import guards that reject Feature importing System, System importing Feature, and shared Configuration importing Callback or Matching; Tool may compose Feature and reusable examples.
- Move the caller-closure golden fixture to Feature ownership and update the README module map/direct CLI examples.
- Delete temporary facades, old roots, and old namespace/path references; do not rewrite historical records.

### Investigation targets
**Required** (read before coding):
- `model/lakefile.toml:1-23` — current default libraries, test root, and executable target
- `Makefile:123-126` — inspector and golden fixture integration variables
- `Makefile:1006-1077` — stable regression target, guards, builds, fixtures, and diagnostics
- `model/README.md:49-103` — live model ownership and usage documentation
- `model/Temporal.lean:1-4` — current production aggregate

**Optional** (reference as needed):
- `.plans/UMPIRE_DSL.md:1096-1258` — approved final layout and verification contract

### Acceptance
- [ ] Only the new Feature/System/Tool module roots, `TemporalModelTests`, and `temporal-model-inspect` remain in live build surfaces; old imports, namespaces, roots, targets, executable names, and directory paths are absent.
- [ ] Domain-purity guards scan all committed reusable text artifacts, including JSON fixtures, while excluding build/runtime state.
- [ ] Import guards reject whitespace-indented reverse imports, Feature/System boundary violations, shared Configuration imports of Callback/Matching, and Temporal-owned semantic/source prefixes without a broad ordinary-word ban.
- [ ] Both inspector scenarios emit byte-stable canonical output across repeated runs; fixture differences are limited to approved source paths.
- [ ] Unknown and invalid inspector requests retain canonical non-zero diagnostics with no artifact stdout.
- [ ] README usage and ownership descriptions match the final module/target layout.
- [ ] `make umpire-check-regression` passes from the repository root.
- [ ] No model-local Makefile or generated API drift/CI gate is introduced.
## Acceptance
- [ ] Clean module, namespace, Lake, executable, and fixture cutover is complete with no aliases.
- [ ] Stable make regression enforces domain purity and all artifact/diagnostic contracts.
- [ ] Live model documentation is current and the full regression command passes.

## Done summary
Completed the clean Temporal model cutover with Feature/System production facades, the TemporalModelTests and temporal-model-inspect targets, a byte-identical Feature-owned golden fixture, and no compatibility roots or old live paths. The stable root regression now guards all tracked reusable Umpire text plus Feature/System/Configuration import directions and verifies deterministic inspector artifacts and canonical error diagnostics (R1-R6).

baseline: green via receipt
GATE_SKIPPED:smoke:green-receipt 5708253f - baseline reused from prior post-gate pass
verification: make umpire-check-regression (green; 64 Lean jobs)
review fix: expanded product-prefix guards to cover qualified names, comments, and JSON source paths while allowing ordinary temporal terminology
memory capture: skipped(error: Flow memory is not initialized)
stage: impl-review - ran [2026-08-25T23:52:15Z..2026-08-25T23:59:35Z] | SHIP
## Evidence
- Commits: 517166a8709bd7a5a13dc1b77b259b30c32cb163, 03553666beb1dab648994d387ac55da998fc0e51
- Tests: GATE_SKIPPED:smoke:green-receipt 5708253f - baseline reused from prior post-gate pass, cd model && mise exec -- lake build Temporal UmpireTests TemporalModelTests temporal-model-inspect, make umpire-check-regression
- PRs: