---
satisfies: [R4]
---
# fn-181-gomad-wasm-operator-integration.4 Recover and resume interrupted typed cluster campaigns

## Description
Carry prepared scenario plans through existing campaign recovery/resume for R4.

**Size:** M
**Files:** tools/gomad/internal/runner/resume.go, recovery.go, backend_prepared.go, internal/campaign/resume_plan.go, tools/gomad/internal/backends/wasm/command/kill_resume_linux_test.go
**Touches:** [tools/gomad/internal/runner/**, tools/gomad/internal/backends/wasm/backend/**, tools/gomad/internal/backends/wasm/command/kill_resume_linux_test.go, tools/gomad/cmd/gomad/internal/cli/recover.go, tools/gomad/cmd/gomad/internal/cli/resume.go]

### Approach
- Reuse append-log commit/recovery and backend restoration. Persist the declared scenario, seed ordering and every prepared boot/input required for unexecuted seeds; observed replay transitions cannot replace the original plan.
- Resume consumes retained inputs and identities, never modified scenario files or ambient configuration. Reexecute only uncommitted work and retain committed results/artifacts.
- Extend TestInstalledOperatorKillResume with completed successful and failed outcomes plus an interrupted execution. Follow the memory fix: stop the process group, confirm quiescence, inspect its durable prefix, kill/reap descendants, then compare before resume.
- Compare logical seeds/outcomes/evidence with uninterrupted control, excluding host metrics and campaign paths. Test deletion/change of original scenario, compiler, sources and mounts.

### Investigation targets
**Required:**
- tools/gomad/internal/runner/resume.go
- tools/gomad/internal/runner/recovery.go
- tools/gomad/internal/runner/backend_prepared.go
- tools/gomad/internal/runner/internal/campaign/resume_plan.go
- tools/gomad/internal/backends/wasm/command/kill_resume_linux_test.go:29

### Quick commands
From tools/gomad: `go test -tags test_dep ./internal/runner ./internal/runner/internal/campaign`; run TestInstalledOperatorKillResume with documented installed-helper inputs on Linux.

## Acceptance
- [ ] Recover/resume preserves frozen committed prefix and matches uninterrupted logical results
- [ ] Remaining seeds execute from retained prepared scenario inputs with originals changed or unavailable
- [ ] Changed installation, corrupt log or missing inputs refuse without overwriting valid evidence
- [ ] Kill/cancellation controls reap all descendants and never report incomplete work as success

## Done summary
SIGKILL after two committed outcomes preserved the frozen prefix; recovery/resume matched the uninterrupted 24-outcome baseline using retained inputs. Corruption/changed-input refusals, journal ordering and descendant cleanup controls pass. Source review is SHIP at 18882a5f2a.

Evidence and exact candidate/commands are consolidated in `.flow/artifacts/fn180-current-batch-2026-10-16.md`, including final installed matrix `/tmp/fn181-classification-operator-20261016-7jBRX1/matrix.log`, portable closure checks and preserved lower-level qualification. The spec's final integrated qualification task remains open. No native qualification or fn-180 R6/fn-182 empirical completion is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 406e6bdf87, 86f25ef560, 46ff86c0e3, 18882a5f2a
- Tests: GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact go test -p=2 -tags test_dep -v -count=1 -timeout=15m -run '^TestInstalledOperatorTypedScenario(KillResume|DetachedReplay)$' ./internal/backends/wasm/command, go test -p=2 -tags test_dep -count=1 -v -timeout=5m -run '^(TestTypedScenarioCampaignInput|TestTypedScenarioInputRefusesInvalidFiles|TestExternalBackendRejectsCommandsBeforeResolvingInstallation|TestExternalBackendInstallationAvoidsNativeToolchainDiscovery|TestRecordedBackendPreservesEvidenceSchema|TestTypedScenarioCampaignRejectsUnsupportedSearch|TestCampaignRequestOwnsScenarioBytes|TestResumeProfileUsesValidatedBackendProvenance|TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures|TestSeedCompletionKeepsCampaignStatistics|TestFrontierScriptsCommitTheirOrderedOutcomes|TestPolicyStopDecidesOneOrderedCompletion|TestCampaignLogRecordsTheFrozenResumePlanAndRefusesChangedTargets|TestPreflightResumeReturnsValidatedLifecycleAndPlan|TestCampaignRecoveryValidatesLiveEvidenceBeforeReclaimingAnything|TestSpecCloneDetachesScenarioDeclaration)$' ./cmd/gomad/internal/cli ./internal/runner ./internal/runner/internal/campaign ./internal/backends/target, go test -tags test_dep ./internal/backends/wasm/cluster -run '^TestMarkerBoundary' -count=1 -v, make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT=/tmp/gomad-host-tools/golangci-lint ERRORTYPE=/tmp/gomad-host-tools/errortype GOLANGCI_LINT_FIX=false
- PRs: