---
satisfies: [R1]
---
# fn-165-gomad-architecture-checks-structural.6 Delete the old analyzer and enforce the line budget

Touches: tools/gomad3/internal/gomadtool/architecture, tools/gomad3/architecture_test.go

## Description
Remove `tools/gomad3/internal/gomadtool/architecture` with all its tests and the fixtures that only it used. Add a test in gomad3 that counts non-test and test lines of `internal/archcheck` plus `internal/puritycheck` and fails above 2,500 and 1,500 respectively. Update references in `tools/gomad3/README.md`, `AGENTS.md` sections and the Makefile that name the old package or its tests. Confirm `fn-153.15` (migrate architecture effect fixtures) is annotated as superseded by this task.

## Acceptance
- `internal/gomadtool/architecture` no longer exists and `go build ./... && go vet ./...` passes in gomad3 and gomad_wasm.
- The budget test passes at the delivered size and fails when a generated 3,000-line file is dropped into `internal/puritycheck`.
- `grep -r gomadtool/architecture` over the repository returns no match outside Flow history.
- Full `go test ./...` in gomad3 with the architecture tests included is at least 50 seconds faster than the recorded baseline for the three replaced tests.

## Done summary
The old analyzer is deleted with no remaining references outside Flow history; archcheck+puritycheck are 2184 production / 743 test lines under enforced 2500/1500 and per-package budgets; the replaced tests run in seconds instead of the 86s baseline.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c6afac5097, d36476f98a, 1520d5bf56
- Tests: root architecture/purity selection with GOMAD3_ARCH_ALL_PLATFORMS=1 plus ./internal/archcheck ./internal/puritycheck (0), Budget tests including oversize rejection (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: