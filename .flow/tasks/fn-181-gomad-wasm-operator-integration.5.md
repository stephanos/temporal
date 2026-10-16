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
Qualified the documented clean-install operator fixtures: two-node durable crash/reopen, partition/heal, intentional replayable canary, source-free detached replay/tamper refusal and interrupted recovery/resume. Final installed matrix passed 42.063 s on production 18882a5f2a with helper a8dfd7fe and model d3aacafd; current broad backend/cluster/command/installation, portable CLI/runner/campaign, lint and named retained regressions pass. Independent integrated source review is SHIP. Only these named fixtures/host are qualified; fn-180 R6, fn-182 empirical acceptance and native execution remain explicitly open/deferred.

Candidate, exact commands, numeric exits, manifests, identities and review are consolidated in `.flow/artifacts/fn180-current-batch-2026-10-16.md`. Spec completion review follows before spec closure; no PR, push, CI or native qualification is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 406e6bdf87, 86f25ef560, 18882a5f2a
- Tests: GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact GOMAD_WASM_CONFORMANCE_HELPER=/tmp/gomad-fn180-conformance-helper-compact GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go GOFLAGS=-p=2 go test -tags test_dep ./internal/backends/wasm/backend ./internal/backends/wasm/cluster ./internal/backends/wasm/command ./internal/backends/wasm/installation -count=1, GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact go test -p=2 -tags test_dep -v -count=1 -timeout=15m -run '^TestInstalledOperatorTypedScenario(KillResume|DetachedReplay)$' ./internal/backends/wasm/command, go test -tags test_dep ./cmd/tools/lintcode -count=1, make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT=/tmp/gomad-host-tools/golangci-lint ERRORTYPE=/tmp/gomad-host-tools/errortype GOLANGCI_LINT_FIX=false
- PRs: