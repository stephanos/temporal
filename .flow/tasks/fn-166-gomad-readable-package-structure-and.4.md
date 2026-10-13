---
satisfies: [R3, R6]
---
# fn-166-gomad-readable-package-structure-and.4 Deepen runner and qualification packages

## Description
Merge runner/internal/exploration/* into one package; move inspection types and Inspect to runner/inspect; narrow runner/internal/campaign exports; merge qualification/workload into qualification. Update archcheck owner and edge tables.

## Acceptance
runner exported identifiers drop by at least half; exploration is one package; package count drops by at least five; R6 checks pass with compiled model and runtime identities unchanged.

## Done summary
Identity-neutral part: exploration is one package and runner/internal/campaign exports only what its importers use (119 to 72). runner/inspect and the qualification/workload merge need backends/wasm and simulation_broker edits and moved to fn-166.7 and .6.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a9e89aa16c, e16e795e8d, fdc02e77a0
- Tests: go test ./runner ./runner/internal/..., architecture selection, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: