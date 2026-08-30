---
satisfies: [R1, R5]
---
# fn-46-export-lean-model-module-impact-index.1 Extract the shared ModelLint package loader

## Description
Extract the current effectful package inventory/build/OLean/reconciliation pipeline for R1 without changing lint policy.

**Size:** M
**Files:** `model/ModelLint.lean`, `model/ModelLint/PackageModules.lean`, `model/ModelLint/PackageModulesTests.lean`, `model/ModelLint/ImportGraphTests.lean`
**Touches:** [model/ModelLint.lean, model/ModelLint/PackageModules.lean, model/ModelLint/PackageModulesTests.lean, model/ModelLint/ImportGraphTests.lean]

### Approach
- Move source discovery, quiet Lake build, OLean lookup/read, region lifetime, capture, and reconciliation behind the parent spec's phase-explicit `ModelLint.PackageModules` interface.
- Capture child stdout/stderr: let `modelLint` replay it to original channels, and let the exporter discard successful build chatter while retaining failure diagnostics.
- Stop after discovery or build failure; collect/sort all independent source, per-module metadata, and reconciliation issues within phases before returning no result.
- Inject only focused process/metadata seams and keep loaded compacted regions alive until every consumer completes.
- Preserve diagnostic categories, build exclusions, import-policy checks, and existing comments.

### Investigation targets
**Required** (read before coding):
- `model/ModelLint.lean:15-82` — exact pipeline and diagnostics to extract.
- `model/Tools/LeanSourceInventory.lean:88-117,198-210` — validation/reconciliation and canonical root.
- `model/ModelLint/ImportGraph.lean:94-146,162-179` — default policy and diagnostics.
- `model/ModelLint/ImportGraphTests.lean:401-430` — executable fixture pattern.

### Key context
Lean 4.33.1 compiled environments/OLean metadata are the authority; do not parse source imports or OLean bytes directly.

### Quick commands
`cd model && mise exec -- lake -q build modelLintTests modelLint && mise exec -- lake exe modelLintTests`
## Acceptance
- [ ] `modelLint` and the future exporter consume the same package-loader result and output policy.
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
