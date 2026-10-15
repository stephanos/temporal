---
satisfies: [R5, R6]
---
# fn-181-gomad-wasm-operator-integration.5 Qualify and document the installed scenario workflow

## Description
Close R5/R6 with one integrated operator matrix and updated usage documentation.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/command/command_test.go, qualification/integration_test.go, CLI.md, QUALIFICATION.md, README.md, tools/gomad/README.md
**Touches:** [tools/gomad/internal/backends/wasm/command/**, tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/internal/backends/wasm/CLI.md, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/internal/backends/wasm/README.md, tools/gomad/README.md]

### Approach
- Reuse qualification receipts and cells for storage crash/reopen, partition/heal and declared cluster crash/restart. Exercise a clean installation through success, intentional failure, detached replay and interrupted recovery/resume.
- Bind actual candidate/helper/module/host and command exits. Support advertisement is not qualification for arbitrary hosts or workloads.
- Consolidate runnable installation/scenario examples and reconcile capability/replay/recovery limits. Preserve native hibernation and unsupported live-enumeration/global-choice boundaries.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/qualification/temporal_cluster_test.go
- tools/gomad/internal/backends/wasm/qualification/sqlite_durable_test.go
- tools/gomad/internal/backends/wasm/command/command_test.go
- tools/gomad/internal/backends/wasm/CLI.md
- tools/gomad/internal/backends/wasm/QUALIFICATION.md

### Quick commands
`make -C tools/gomad test-wasm GOMAD3_STOCK_GO=/absolute/pinned/go1.27.1` plus actual installed-operator/qualification selectors; verify they executed rather than skipped.

## Acceptance
- [ ] Frozen clean-install matrix executes R5 scenarios and failure/recovery controls
- [ ] Candidate-bound qualification records contain commands, exits, identities and resource limits
- [ ] Documented commands are exercised and capability tables match executable admission
- [ ] Actual WASM and required portable/source gates pass; unavailable gates remain open and native execution deferred

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
