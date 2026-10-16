---
satisfies: [R3]
---
# fn-181-gomad-wasm-operator-integration.3 Inspect and replay newly produced cluster campaign artifacts

## Description
Extend existing retained cluster replay/inspection for task .2 artifacts, implementing R3 without another replay engine.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/command/command.go, command_test.go, cluster/retained.go, cluster/replay.go, tools/gomad/internal/runner/inspect/inspect.go
**Touches:** [tools/gomad/internal/backends/wasm/command/**, tools/gomad/internal/backends/wasm/cluster/retained*, tools/gomad/internal/backends/wasm/cluster/replay*, tools/gomad/internal/runner/inspect/**]

### Approach
- Feed new payloads through RetainedClusterReplay, VerifyRetained and ReplayBoundRetained. Preserve schema/identity refusal ordering.
- Resolve every boot, captured file and recovery image from retained bytes; preserve per-incarnation tape completeness and shared lifecycle/network/storage evidence. Do not claim a global exact-choice tape.
- Extend inspection only for missing scenario/boot metadata. Preserve fn-180 metadata if present, without creating a feature dependency.
- Extend detached operator tests to remove source/compiler/mounts and preparation caches before replay, retaining the compatible installed helper.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/command/command.go:56
- tools/gomad/internal/backends/wasm/command/command_test.go:211
- tools/gomad/internal/backends/wasm/cluster/retained.go:382
- tools/gomad/internal/backends/wasm/cluster/replay.go
- tools/gomad/internal/runner/inspect/inspect.go

### Quick commands
From tools/gomad: `go test -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/cluster ./internal/runner/inspect` with helper inputs for actual replay gates.

## Acceptance
- [ ] Success/failure artifacts inspect and replay without original source/compiler/mounts/preparation cache
- [ ] Missing/tampered scenario, module, boot/image payloads and incompatible installed identities fail closed
- [ ] Evidence and completeness reports match the per-incarnation replay contract
- [ ] Existing standalone and historical schema refusals remain intact

## Done summary
New success/canary/network artifacts inspect and replay after source/compiler/mount/preparation inputs become unavailable. All 18 missing/tampered retained-payload controls pass in the final installed matrix; standalone and historical refusals remain covered. Source review is SHIP at 18882a5f2a.

Evidence and exact candidate/commands are consolidated in `.flow/artifacts/fn180-current-batch-2026-10-16.md`, including final installed matrix `/tmp/fn181-classification-operator-20261016-7jBRX1/matrix.log`, portable closure checks and preserved lower-level qualification. The spec's final integrated qualification task remains open. No native qualification or fn-180 R6/fn-182 empirical completion is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 406e6bdf87, 86f25ef560, 46ff86c0e3, 18882a5f2a
- Tests: GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact go test -p=2 -tags test_dep -v -count=1 -timeout=15m -run '^TestInstalledOperatorTypedScenario(KillResume|DetachedReplay)$' ./internal/backends/wasm/command, go test -p=2 -tags test_dep -count=1 -v -timeout=5m -run '^(TestTypedScenarioCampaignInput|TestTypedScenarioInputRefusesInvalidFiles|TestExternalBackendRejectsCommandsBeforeResolvingInstallation|TestExternalBackendInstallationAvoidsNativeToolchainDiscovery|TestRecordedBackendPreservesEvidenceSchema|TestTypedScenarioCampaignRejectsUnsupportedSearch|TestCampaignRequestOwnsScenarioBytes|TestResumeProfileUsesValidatedBackendProvenance|TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures|TestSeedCompletionKeepsCampaignStatistics|TestFrontierScriptsCommitTheirOrderedOutcomes|TestPolicyStopDecidesOneOrderedCompletion|TestCampaignLogRecordsTheFrozenResumePlanAndRefusesChangedTargets|TestPreflightResumeReturnsValidatedLifecycleAndPlan|TestCampaignRecoveryValidatesLiveEvidenceBeforeReclaimingAnything|TestSpecCloneDetachesScenarioDeclaration)$' ./cmd/gomad/internal/cli ./internal/runner ./internal/runner/internal/campaign ./internal/backends/target, go test -tags test_dep ./internal/backends/wasm/cluster -run '^TestMarkerBoundary' -count=1 -v, make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT=/tmp/gomad-host-tools/golangci-lint ERRORTYPE=/tmp/gomad-host-tools/errortype GOLANGCI_LINT_FIX=false
- PRs: