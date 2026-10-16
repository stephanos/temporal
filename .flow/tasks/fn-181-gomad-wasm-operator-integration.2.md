---
satisfies: [R1, R2]
---
# fn-181-gomad-wasm-operator-integration.2 Connect seed campaigns, installation admission and CLI capabilities

## Description
Connect prepared scenarios to shared campaign execution/publication and the normal CLI for R1/R2.

**Size:** M
**Files:** tools/gomad/cmd/gomad/internal/cli/cli.go, capabilities.go, tools/gomad/internal/backends/backend.go, wasm/backend/lifecycle.go, wasm/backend/capabilities.go, tools/gomad/internal/runner/backend_prepared.go, related tests
**Touches:** [tools/gomad/cmd/gomad/internal/cli/**, tools/gomad/internal/backends/backend.go, tools/gomad/internal/backends/wasm/backend/**, tools/gomad/internal/backends/wasm/installation/**, tools/gomad/internal/backends/wasm/cluster/operator*, tools/gomad/internal/runner/**]

### Approach
- Reuse runExploreWith, Runner.Explore, shared commit/publication and backend preparation. Add only the backend-neutral seam required for typed cluster targets; Runner must not import concrete WASM packages.
- Add explicit scenario selection following existing CLI target/config conventions. Each seed starts a fresh fn-178 cluster; project outcomes and existing retained cluster evidence into shared campaign reporting/artifacts.
- Admit qualified cooperative installation profiles and advertise only executable support. The current broad Simulation capability also gates native simulation features: avoid enabling those by accident.
- Preserve seed ordering, failure policy, bounds, cancellation/reaping, human/JSON output and existing standalone choices. No cluster-global choice search or separate campaign framework.

### Investigation targets
**Required:**
- tools/gomad/cmd/gomad/internal/cli/cli.go:553
- tools/gomad/cmd/gomad/internal/cli/capabilities.go:83
- tools/gomad/internal/backends/backend.go:15
- tools/gomad/internal/backends/wasm/backend/lifecycle.go:37
- tools/gomad/internal/runner/backend_prepared.go
- tools/gomad/internal/backends/wasm/cluster/retained.go

### Quick commands
From tools/gomad: `go test -tags test_dep ./cmd/gomad/internal/cli ./internal/runner ./internal/backends/wasm/backend`; actual guest tests require the documented helper setup.

## Acceptance
- [ ] Built CLI explores a bounded cluster seed set, reports results/artifact paths and reaches the configured Rust owner
- [ ] Scripted controls prove deterministic commit order and failure policies; actual guest fixture proves execution
- [ ] Discovery lists implemented support only; unsupported stock/native simulation/guidance combinations remain refused
- [ ] Target failure, capacity, cancellation and infrastructure outcomes remain distinct and helpers are reaped

## Done summary
Connected typed scenarios to the existing CLI, ordered seed campaigns, installation admission and distinct terminal classifications. Genuine capacity/unsupported-before-marker, protocol/cancellation/reaping and scripted campaign controls pass; independent batch source review is SHIP at 18882a5f2a.

Evidence and exact candidate/commands are consolidated in `.flow/artifacts/fn180-current-batch-2026-10-16.md`, including final installed matrix `/tmp/fn181-classification-operator-20261016-7jBRX1/matrix.log`, portable closure checks and preserved lower-level qualification. The spec's final integrated qualification task remains open. No native qualification or fn-180 R6/fn-182 empirical completion is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 406e6bdf87, 86f25ef560, 46ff86c0e3, 18882a5f2a
- Tests: GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact go test -p=2 -tags test_dep -v -count=1 -timeout=15m -run '^TestInstalledOperatorTypedScenario(KillResume|DetachedReplay)$' ./internal/backends/wasm/command, go test -p=2 -tags test_dep -count=1 -v -timeout=5m -run '^(TestTypedScenarioCampaignInput|TestTypedScenarioInputRefusesInvalidFiles|TestExternalBackendRejectsCommandsBeforeResolvingInstallation|TestExternalBackendInstallationAvoidsNativeToolchainDiscovery|TestRecordedBackendPreservesEvidenceSchema|TestTypedScenarioCampaignRejectsUnsupportedSearch|TestCampaignRequestOwnsScenarioBytes|TestResumeProfileUsesValidatedBackendProvenance|TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures|TestSeedCompletionKeepsCampaignStatistics|TestFrontierScriptsCommitTheirOrderedOutcomes|TestPolicyStopDecidesOneOrderedCompletion|TestCampaignLogRecordsTheFrozenResumePlanAndRefusesChangedTargets|TestPreflightResumeReturnsValidatedLifecycleAndPlan|TestCampaignRecoveryValidatesLiveEvidenceBeforeReclaimingAnything|TestSpecCloneDetachesScenarioDeclaration)$' ./cmd/gomad/internal/cli ./internal/runner ./internal/runner/internal/campaign ./internal/backends/target, go test -tags test_dep ./internal/backends/wasm/cluster -run '^TestMarkerBoundary' -count=1 -v, make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT=/tmp/gomad-host-tools/golangci-lint ERRORTYPE=/tmp/gomad-host-tools/errortype GOLANGCI_LINT_FIX=false
- PRs: