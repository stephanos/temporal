---
satisfies: [R1, R6]
---
# fn-166-gomad-readable-package-structure-and.1 Empty the module root and remove stale package names

## Description
Move the ten root test files to their owners (architecture tests to internal/archcheck or puritycheck, native overlay ownership tests to backends/native, runner_consumer_test.go to runner, CI and gate-routing tests to tools/gomad3integration). Remove runner/internal/execution, internal/preparation and target/internal/gocommand from archcheck tables, tests and ARCHITECTURE.md; drop the module-root test-harness owner if unused.

## Acceptance
`find tools/gomad3 -maxdepth 1 -name '*.go'` is empty; moved tests pass from their new packages; no reference to the removed package paths remains outside Flow history; R6 checks pass with unchanged identities.

## Done summary
Module root holds no Go files; architecture tests live with internal/archcheck, native overlay ownership tests under backends/native, the Runner consumer test in runner; stale package names removed from archcheck tables, tests and ARCHITECTURE.md. Stale names inside the native boundary manifest stay with fn-128 because they feed the native boundary identity.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 33485603e1, 49d33d1f9a
- Tests: architecture selection (all platforms), make -C tools/gomad3 validate, lint-code-fast, make test-wasm, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: