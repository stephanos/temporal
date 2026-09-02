---
satisfies: [R3]
---
# fn-40-centralize-plannerpolicy-constructors.3 Refresh canonical artifacts and generated views

## Description
Refresh the complete canonical fixture/checksum and generated-view surface affected by the seed-17 Query identity migration, then run the full model gates (R3).

**Size:** M
**Files:** affected files under `model/Umpire/Examples/Fixtures/`, `model/Umpire/Examples/testdata/`, `model/Umpire/Artifact/Tests/Fixtures/`, `model/Temporal/Feature/Nexus/Fixtures/`, `model/Umpire/Examples/Generated/`, `model/Umpire/Artifact/Tests/Codecs.lean`, generated/checksum consumers under `tools/umpire/`, and the approved semantic-inventory/documentation ownership unblock
**Touches:** [model/Umpire/Examples/Fixtures/**, model/Umpire/Examples/testdata/**, model/Umpire/Artifact/Tests/**, model/Temporal/Feature/Nexus/Fixtures/**, model/Temporal/Feature/Nexus/Experimental/*Tests.lean, model/Temporal/Tool/NexusDiscoveryTests.lean, model/Umpire/Examples/Generated/**, model/Temporal/Tool/SemanticInventory.lean, model/Umpire/SemanticInventory/**, model/Umpire/Planning/Tests/KnownGaps.lean, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, tools/umpire/artifact/**, tools/umpire/cmd/umpire-gen-regression-views/**, tools/umpire/cmd/umpire-local-run-evaluation/**, tools/umpire/regression/**, tools/umpire/internal/artifactv2/**]

### Approach
- Produce fresh canonical Query and ExperimentSpec bytes from the migrated model declarations and update every byte/checksum fixture as a complete owned set.
- Regenerate the managed Go and Markdown regression views with the repository generator; do not hand-maintain generated output.
- Update independent artifact-v2 canonical fixtures and checksum assertions reached by the changed Switch artifact identity.
- Search for obsolete pre-migration fingerprints/checksums to catch consumers outside the obvious fixture directories.
- Unblock the full regression gate by restoring Umpire's reusable ownership boundary: publish the Run Evaluation admitted-input row from the Temporal leaf aggregator and keep Temporal adapter documentation on the Temporal architecture surface.
- Run focused byte-equality checks followed by the repository model regression and import/documentation lint gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/SwitchTests.lean:14-18` — canonical Switch Query/artifact fixtures
- `model/Temporal/Feature/Nexus/OperationsTests.lean:17-30` — canonical Lifecycle-derived fixtures
- `model/Umpire/Tests/MigrationCompatibility.lean:214-225` — relocated-layout artifact equality
- `model/Umpire/Artifact/Tests/Codecs.lean:17-29` — v2 canonical byte fixtures
- `Makefile:1019-1030` — regression-view generation/check entry points
- `model/README.md:248-288` — focused and full model verification commands

### Key context
- Changing a meaning-bearing Query seed changes query fingerprints, nested DrivePlan/ExperimentSpec checksums, and generated views even when the selected trace is unchanged. Partial fixture refreshes are invalid.

### Acceptance
- [ ] All checked Query/artifact fixtures contain the post-migration identities and valid recomputed checksums.
- [ ] `make umpire-check-regression-views` reports no generated-view drift after regeneration.
- [ ] No stale pre-migration fingerprint or checksum remains in model/tool consumers.
- [ ] `make umpire-check-regression` and `make lint-model` pass.
## Acceptance
- [ ] Complete canonical fixture and generated-view sets satisfy R3.
- [ ] Focused byte/checksum tests, regression-view drift check, full regression, and model lint pass.

## Done summary
Regenerated the complete seed-17 Query, ExperimentSpec, artifact-closure, checksum, mirrored-fixture, and regression-view sets without changing shortest or exhaustive traces. The approved inherited unblock moved the Run Evaluation admitted-input descriptor and Temporal exploration documentation to their Temporal-owned surfaces, restoring the reusable Umpire ownership gate.

baseline: red via task-2 handoff (canonical identity/checksum/generated-view bytes were intentionally stale); direct `make umpire-check-legacy-vocabulary` was green before regeneration.

lint-code: inherited red. Exact `make lint-code` reported 1,379 findings after applying an unrelated import auto-fix, which was restored; `GOLANGCI_LINT_FIX=false make lint-code` reported 1,386 rather than the stated 1,385, while the base-scoped run reported zero task-diff golangci findings and only the known `tools/umpire/runtime/errors.go:60` `et:unw+` failure.

stage: impl-review - ran [2026-09-02T23:42:55Z..2026-09-02T23:49:49Z] (Codex SHIP; session 01a06481-3c4e-7cf2-87c5-8991b538c920; 0 findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3853e51aa9c40770e049e2ea62971c268dcd6536
- Tests: baseline: red via task-2 handoff (canonical Query/artifact/checksum/generated-view bytes stale after the seed-17 identity migration), make umpire-check-legacy-vocabulary, cd model && mise exec -- lake build Umpire.Examples.SwitchTests Umpire.ExecutionHandoffTests Umpire.Artifact.Tests.Codecs Umpire.Artifact.Tests.Goldens Umpire.Artifact.Tests.Set Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.VariationSpaceTests Temporal.Tool.NexusDiscoveryTests, cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.ExplorationTests, focused Go byte/checksum tests: ./tools/umpire/artifact ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/cmd/umpire-local-run-evaluation ./tools/umpire/regression (green with physical TMPDIR), make umpire-check-regression-views, authoritative normalized trace comparison: four shortest Query/artifact traces unchanged after identity fields were removed; exhaustive caller-closure inspector bytes unchanged, stale identity scan: 36 retired fingerprints/checksums searched across model and tools, zero matches, cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests, make umpire-check-semantic-inventory, cd model && mise exec -- lake build UmpireTests, cd model && mise exec -- lake build Umpire.Examples.Switch Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, make umpire-check-regression, make lint-model, INHERITED_RED: make lint-code reported 1,379 after an unrelated auto-fix; the auto-fix was restored, INHERITED_RED: GOLANGCI_LINT_FIX=false make lint-code reported 1,386 findings rather than the stated 1,385, with zero findings in task-touched Go files, INHERITED_RED: base-scoped GOLANGCI_LINT_FIX=false make lint-code reported zero task-diff golangci findings, then tools/umpire/runtime/errors.go:60:9 et:unw+, GATE_SKIPPED:unittest:green-receipt 3853e51a - baseline reused from prior post-gate pass, GATE_SKIPPED:smoke:green-receipt 3853e51a - baseline reused from prior post-gate pass, post-review: cd model && mise exec -- lake build Umpire.Examples.Switch Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, post-review: make lint-model, impl-review codex SHIP session 01a06481-3c4e-7cf2-87c5-8991b538c920 receipt /tmp/impl-review-receipt-fn-40-centralize-plannerpolicy-constructors.3.json
- PRs:
