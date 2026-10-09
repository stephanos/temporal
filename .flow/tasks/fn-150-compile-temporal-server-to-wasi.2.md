---
satisfies: [R1, R2, R3, R4]
---
# fn-150-compile-temporal-server-to-wasi.2 Replace SQLite driver and publish three WASI draft PRs

## Description
Split the compilation work into a gh stack based on gomad: Prometheus patch update, replace modernc SQLite with ncruces on all platforms, then WASI build target. Preserve native persistence behavior and publish three draft PRs.

## Acceptance
Three draft PRs linked by gh stack with the requested dependency order; native SQLite persistence tests, focused tests, WASI compilation and lint pass; driver runtime compatibility and remaining WASI limitations documented.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
