---
satisfies: [R1, R2]
---
# fn-68-minimal-nexus3-success-demonstration.4 Derive Nexus3 metadata from authored declarations

## Description
Simplify Nexus3 authoring by deriving both declaration identities and required metadata from the authored model. Authors write states, transitions, and the success condition; they should not maintain a parallel definitions registry, handwritten IDs, identity overrides, or version-label strings for this demonstration. The current definitions list repeats provider, capability, law, and meaning information, including calls such as Authoring.metadata lifecycleProviderId .provider "temporal-nexus3-provider/v1". Generate the bookkeeping required by existing checkers from its owning declarations through the smallest existing authoring expansion/construction seam.

Use declaration names and named owners/members to derive internally consistent IDs. A declaration rename may change its ID and require fixture regeneration. Durable identity across renames, compatibility aliases, migration machinery, and user-facing ID overrides are not goals. Retain an explicit identity or version input only when an existing concrete consumer actually requires it, and document that dependency. Keep required wire-format versions distinct from unnecessary per-declaration version labels.

This user-approved scope amendment supersedes this task's previous requirement to preserve effective IDs and overrides, and the parent spec's pinned-ID/override requirements for the Nexus3 simplification performed here. Update affected documentation and tests to the simpler policy; do not retain obsolete identity machinery just to satisfy those earlier requirements.

Build on Task 1's success-only authoring surface and inspect its final implementation before choosing the smallest change. Preserve behavior, existing comments, capability/law checking, reference integrity, and Testpilot boundaries. Keep the change local to Nexus3 and necessary shared construction support; do not introduce a general metadata framework, a new semantic language, or migrate Nexus/Nexus2.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Authoring.lean, model/Temporal/Feature/Nexus3/Syntax.lean, model/Temporal/Feature/Nexus3/Nexus.lean, model/Temporal/Feature/Nexus3/Tests.lean, model/Temporal/Feature/Nexus3/Testpilot.lean (if Task 2 has begun), tests/testcore/testpilot/testdata/async-nexus-case.json (only if already generated), affected Nexus3 documentation.
**Touches:** [model/Temporal/Feature/Nexus3/*.lean, model/Temporal/Feature/Nexus3/*.md, tests/testcore/testpilot/testdata/async-nexus-case.json]

This task runs after Task 1 and before Tasks 2 and 3. Task 2 must consume the final derived identities; Task 3 regenerates fixtures only after this task completes.

## Acceptance
- [ ] Nexus3 authors declare behavior once. Required metadata and IDs are derived without a separately maintained definitions registry, including inside construction helpers.
- [ ] Remove handwritten IDs, override plumbing, pinned-ID expectations, and redundant temporal-nexus3-.../v1 labels from the demo where no concrete consumer requires them. Derive kinds, source information, and canonical descriptions from their owning declarations. Preserve required protocol versions.
- [ ] Derive IDs deterministically from declaration names and named owners/members, never source order. Renames may change IDs; no compatibility alias or migration is required. All generated references and Case provenance remain consistent.
- [ ] Preserve the exact scheduled → started → succeeded witness and existing rejection of unresolved, wrong-kind, duplicate, or conflicting references. Removing metadata boilerplate must not bypass semantic admission checks.
- [ ] Focused Lean tests cover deterministic derivation, reorder/comment stability, rename-driven identity changes, semantic fingerprint changes where consumed, and malformed/conflicting declaration rejection. Replace tests that require pinned IDs or overrides with tests for this simpler policy.
- [ ] Update affected documentation and downstream expectations. Regenerate affected fixtures through their owning generator when IDs or provenance change; runtime behavior remains unchanged.
- [ ] Run focused Nexus3 tests and applicable existing model/lint gates. Document any explicit identity/version input retained because of an actual consumer dependency.

## Done summary
Nexus3 now derives declaration IDs, owned member/relation/setup/occurrence identities, provider meanings, and checker metadata from macro-captured declaration names and typed model catalogs. User identity overrides, pinned-ID expectations, the parallel definitions registry, and redundant per-declaration `/v1` labels are removed while the exact witness, Known Gaps, semantic admission failures, syntax boundary, and 89-line consumer remain intact.

Tests add deterministic metadata derivation, declaration-order and documentation stability, rename coherence, semantic fingerprint sensitivity, and malformed, duplicate, wrong-kind, and conflicting definition rejection. Nexus3 documentation now states the rename/regeneration policy and identifies Umpire's required `DefinitionMetadata.version` field as the only retained version dependency; authors provide no identity or version input.

Baseline: focused Lean green; Go Testpilot inherited red because the local C toolchain cannot find `stddef.h`.

stage: impl-review - SHIP(receipt: /tmp/impl-review-receipt-fn68-task4.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: BASELINE_GREEN:(cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-testpilot), BASELINE_RED:mise exec -- go test -count=1 -tags test_dep ./tests/testcore/testpilot/... (inherited toolchain failure: stddef.h not found), RED:(cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests) -> /tmp/fn68-task4-red-derived-metadata.log, (cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-testpilot), (cd model && mise exec -- lake build TemporalModelTests), make lint-model, git diff --check -- model/Temporal/Feature/Nexus3/Authoring.lean model/Temporal/Feature/Nexus3/Syntax.lean model/Temporal/Feature/Nexus3/Nexus.lean model/Temporal/Feature/Nexus3/Tests.lean model/Temporal/Feature/Nexus3/Nexus.md model/Temporal/Feature/Nexus3/Integration.md
- PRs:
