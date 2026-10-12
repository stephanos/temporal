---
satisfies: [R1, R6]
---
# fn-166-gomad-readable-package-structure-and.1 Empty the module root and remove stale package names

## Description
Move the ten root test files to their owners (architecture tests to internal/archcheck or puritycheck, native overlay ownership tests to backends/native, runner_consumer_test.go to runner, CI and gate-routing tests to tools/gomad3integration). Remove runner/internal/execution, internal/preparation and target/internal/gocommand from archcheck tables, tests and ARCHITECTURE.md; drop the module-root test-harness owner if unused.

## Acceptance
`find tools/gomad3 -maxdepth 1 -name '*.go'` is empty; moved tests pass from their new packages; no reference to the removed package paths remains outside Flow history; R6 checks pass with unchanged identities.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
