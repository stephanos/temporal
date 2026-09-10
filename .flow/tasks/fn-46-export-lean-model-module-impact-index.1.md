---
satisfies: [R1, R5]
---
# fn-46-export-lean-model-module-impact-index.1 Extract the shared ModelLint package loader

## Description
Extract the current effectful package inventory/build/OLean/reconciliation pipeline for R1 without changing lint policy.

**Size:** M
**Files:** `model/ModelLint.lean`, `model/ModelLint/PackageModules.lean`, `model/ModelLint/PackageModulesTests.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/Tools/LeanImportGraph/Metadata.lean`, focused adjacent metadata tests, minimal `model/lakefile.lean` test wiring if needed.
**Touches:** [model/ModelLint.lean, model/ModelLint/PackageModules.lean, model/ModelLint/PackageModulesTests.lean, model/ModelLint/ImportGraphTests.lean, model/Tools/LeanImportGraph/Metadata.lean, model/Tools/LeanImportGraph/MetadataTests.lean, model/lakefile.lean]

### Approach
- Move source discovery, build, region lifetime, capture and reconciliation orchestration behind `ModelLint.PackageModules`; reuse the existing `Tools.LeanImportGraph.Metadata.load` traversal and extend its lookup/read seam for accumulated independent failures. Quiet/captured build behavior is new work, not already provided by the current loader.
- Capture child stdout/stderr: let `umpire-lint` replay it to original channels, and let the exporter discard successful build chatter while retaining failure diagnostics.
- Stop after discovery failure, source issues or build failure. Continue independently known metadata nodes after lookup/read failures, sort qualified issues and return no partial result; inaccessible descendants cannot be claimed examined. On metadata success report reconciliation issues and architecture violations as current lint does.
- Inject only focused process/metadata seams and keep loaded compacted regions alive until every consumer completes.
- Preserve full external closure through validation, diagnostic categories, build exclusions, import-policy checks, existing comments and subsequent declaration-lint execution. Do not fold the exporter's stricter root-identity guard into existing lint discovery.
- Capture child streams without deadlock. Wire PackageModulesTests and any metadata tests into the existing umpire-lint-tests runner and actually execute them, rather than only compiling test modules.

### Investigation targets
**Required** (read before coding):
- `model/ModelLint.lean:16` — build sequence; graph phases at 41 and subsequent declaration lint below.
- `model/Tools/LeanImportGraph/Metadata.lean:17` — existing traversal, compacted regions and first-failure behavior.
- `model/Tools/LeanSourceInventory.lean:89` — sorted validation; confined current-root inventory at 205.
- `model/ModelLint/ImportGraph.lean` — current external-aware reconciliation and isolation policy.
- `model/ModelLint/ImportGraphTests.lean:639` — real external metadata controls; executable runner at 664.

### Key context
Lean 4.33.1 compiled environments/OLean metadata are the authority; do not parse source imports or OLean bytes directly.

Capture exact original lint success/controlled-violation diagnostics and source/build/metadata phase ordering before edits. Preserve external bridge, stale-owned skip, uniqueness and missing-OLean regressions. Use injected phase counters and lifetime ownership assertions for independent failure and region-release tests. No no-op region-count check as the sole lifetime proof.

### Quick commands
`cd model && mise exec -- lake -q build umpire-lint-tests umpire-lint && mise exec -- lake exe umpire-lint-tests`

Run the existing controlled-violation fixture with its expected nonzero exit and exact channels, and the shared-loader success path, before and after extraction. Record explicit suite execution and terminal results.

### Execution constraints
Preserve unrelated dirty source and existing comments; no staging, commits or pushes. Re-anchor delivered fn-75/76/78 and final fn-77 source before implementation; serialize shared source edits without adding artificial semantic dependencies. Run Lean jobs serially. Capture original touched bytes and relevant trust/metadata baselines before editing. New tests must run through registered roots. No new dependencies, import-policy relaxation, generated API drift/CI expansion, or cancellation work. Required nonfixing Go lint may retain only the exact verified inherited set; new findings and killed/missing-exit gates are failures.
## Acceptance
- [ ] `umpire-lint` and the future exporter consume the same package-loader result and output policy.
- [ ] Existing lint policy, success line, controlled-violation diagnostic, and child stream channel assignment remain unchanged.
- [ ] Discovery/build failures stop later phases; multiple simultaneous source/metadata/reconciliation issues are accumulated and sorted within their valid phase.
- [ ] Every failure returns no partial result and successful exporter mode emits no build chatter.
- [ ] Focused tests cover each effect boundary, multiple metadata failures, transcript replay/suppression, region lifetime, and comment preservation.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
