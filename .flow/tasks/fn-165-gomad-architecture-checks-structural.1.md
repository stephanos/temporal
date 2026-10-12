---
satisfies: [R3, R9]
---
# fn-165-gomad-architecture-checks-structural.1 Build the structural checker and move inventory, owner and edge rules onto it

Touches: tools/gomad3/internal/archcheck, tools/gomad3/architecture_test.go

## Description
Create `tools/gomad3/internal/archcheck` with `Config`, `Discover` (moved unchanged from `internal/gomadtool/architecture/architecture.go`), `Check`, and `Finding`. Rules: owner membership and owner edges, module edges in the fn-164 direction (shared never imports `backends/*`, backends never import each other) with an allowlist naming today's violating shared packages, pure-file import allowlist, pure-syntax (no `go` statement, no cgo, `syscall` only for `Errno` and `E*`), and the time/select rule. Everything is `go list` plus `go/ast`; no `go/types`. Port `TestArchitectureInventoryFixtures`, `TestPackageArchitecture` and `TestHostPackageVet` in `tools/gomad3/architecture_test.go` to the new package so they stop importing the old one. The old package stays in place until fn-165.6; do not delete anything here. Keep Finding categories listed in the spec's API Contracts.

## Acceptance
- `internal/archcheck` exists with the API in the spec and no `go/types` import.
- Inventory fixtures, package-architecture and host-vet tests pass on both qualified platforms through `archcheck`.
- A test adds `import "os"` to a temporary pure file and gets `pure-import`; a `go` statement gets `pure-syntax`; an import from a shared package into `backends/` gets `module-edge`; removing an allowlist entry that still matches fails the test naming it.
- Structural check of the gomad3 module completes in under 5 seconds on the host platform.
- Package is at most 700 non-test lines.

## Done summary
internal/archcheck provides Discover/Check/Finding with owner, module-edge, pure-import, pure-syntax and time/select rules over go list and go/ast only; inventory, package-architecture and host-vet tests run through it on both qualified platforms; 652 non-test lines with an enforced 700-line per-package budget.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c6afac5097, 016ec47b74, d36476f98a
- Tests: root architecture/purity selection with GOMAD3_ARCH_ALL_PLATFORMS=1 plus ./internal/archcheck ./internal/puritycheck (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: