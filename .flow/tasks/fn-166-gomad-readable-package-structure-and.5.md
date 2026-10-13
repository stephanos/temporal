---
satisfies: [R4, R6]
---
# fn-166-gomad-readable-package-structure-and.5 Move gomadtool under the native backend

## Description
Move cmd/gomadtool to backends/native/cmd/gomadtool, internal/compatibilitypack (with authoring) and internal/sourceinventory under backends/native, and fold internal/gomadtool/validation into gomadtool, all with Git renames; update the Makefile, owner tables and documentation.

## Acceptance
Root cmd/ holds only cmd/gomad; root internal/ holds no package imported only by the native backend; make targets using gomadtool pass; R6 checks pass.

## Done summary
gomadtool, compatibilitypack (with authoring) and sourceinventory live under backends/native; script validation folded into gomadtool; Make targets, docs and CI follow. Generated-code headers inside identity-bound files move with fn-166.6.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5cd08aa19b
- Tests: go test ./backends/native/cmd/gomadtool/... (failure set unchanged), make -C tools/gomad3 validate, lint-code-fast, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: