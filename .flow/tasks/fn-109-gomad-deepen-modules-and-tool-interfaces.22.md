---
satisfies: [R1, R18]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.22 Repair the simulation-exploration target path so a real campaign completes

## Description
Prerequisite found by fn-109.1 on 2026-10-01: no fixture can complete a simulation-exploration campaign, locally or through the isolated coordinator, so R1's simulation leg cannot be proven. Three defects, all outside fn-109.1's Touches (evidence: .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task1-evidence.txt and task1-review.md):

1. `tools/gomad3/target/capability.go` `builtInSimulationLinknames` is stale against `tools/gomad3sim`: five SHA pins differ, `gomadProcessVolumeOperation` is unpinned, and `runtime_time_toolchain.go` has no entry.
2. The `gomad3sim` test variant imports `syscall`, which closure mode rejects.
3. With the pins corrected in a scratch overlay, the target rejects the Runner's plan: "simulation exploration candidate identity does not match".

Establish for each whether it is drift since the simulation track last qualified (find the last passing evidence and the commit that broke it) or a never-working path, fix the cause at its owner, and add a retained simulation-capable fixture target under the existing test tiers that completes a simulation-exploration campaign with committed logical executions and exact replay. Preserve exact first-party simulation pins and allowed bridge directives: re-pin only to the reviewed current sources, never loosen the policy or admit `syscall` generically. Then finish the simulation leg of the fn-109.1 transport test (tools/gomad3/runner/coordinator_transport_test.go) so it requires a completing campaign with the supplied bounds through real isolated execution.

**Touches:** tools/gomad3/target/capability.go and its tests, tools/gomad3sim/** (test variant only where the syscall import lives), tools/gomad3/runner/simulation_exploration*.go and the exploration-plan identity owner, tools/gomad3/runner/coordinator_transport_test.go, tools/gomad3/runner/testdata/**, a fixture under the existing conformance/testdata tree.

Quick: go test -count=1 ./runner/... ./target/... (in tools/gomad3); go test -count=1 -tags test_dep ./tools/gomad3sim/... (repo root); make -C tools/gomad3 validate test-host; one built-CLI `gomad explore` simulation-exploration run that completes.

## Acceptance
- Each of the three defects has an identified cause and owner; the fix is at the owner, with a regression test that fails before and passes after.
- A retained simulation-capable fixture completes a simulation-exploration campaign locally and through the isolated coordinator with distinguishable nonzero limits, committed logical executions and exact replay of a retained artifact.
- The fn-109.1 transport test's simulation leg requires that completing campaign.
- Simulation pins stay exact and policy is not widened; closure-mode analysis of the fixture reports supported.
- darwin/arm64 gates pass; linux/amd64 is recorded as not run. No commits, staging or worktrees.

## Done summary
A simulation-exploration campaign now completes on darwin/arm64, locally and through the isolated coordinator, in the default closure capability mode, and every retained artifact replays exactly. Four defects blocked it; three were reported, the fourth appeared once those were fixed. None is datable drift: the one retained commit that introduced the tree (88f48d6fc, 2026-08-23 squash) already carries all of them, and no gate exercised the path.

- **Stale pins** (owner `tools/gomad3/target/capability.go`): five SHA-256 pins re-pinned to the committed `tools/gomad3sim` sources, the second directive of `runtime_process_model.go` and an entry for `runtime_time_toolchain.go` added. The allowlist is still exact name, SHA and ordered directive list for the main module's own package; `builtInSimulationLinknameAllowed` and `forbiddenImport` are unchanged. `TestBuiltInSimulationLinknamesPinCurrentFirstPartySources` reads the sources, which the old self-comparing test never did.
- **Fixture under closure mode** (owner: the fixture choice): the gomad3sim white-box tests need `syscall` for host probes, so neither they nor the policy changed. New retained fixture `tools/gomad3sim/testdata/simulation_exploration` in the root module, where the bridges are admitted; it cannot live under `conformance/testdata`, a separate module. `TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests` requires zero findings for the fixture and only `import:syscall` findings for the test variant.
- **Candidate identity** (owner `tools/gomad3sim/exploration.go`): the target hashed under `gomad3-combined-frontier-*`, the Runner under `gomad3-simulation-exploration-*`; the four target domain strings now equal the Runner's. Recompute-and-compare is unchanged on both sides. A Runner-issued vector (`runner/testdata/simulation_exploration_plan_vector.json`) is checked by `TestSimulationExplorationPlanMatchesRetainedVector` and, from the target, by `TestExplorationIdentitiesMatchRunnerIssuedPlanVector`.
- **Runtime override proof** (fourth defect, same file): `validateExplorationEvidence` demanded a cluster decision for runtime overrides, which the cluster never decides and the Runner proves from the choice tape; it now skips that dimension, as `finishExplorationLocked` already did. `TestExplorationEvidenceProvesOnlyTheOverridesTheClusterDecides` keeps the other dimensions' proof.

`TestIsolatedRunnerCompletesSimulationStrategyWithSuppliedLimits` replaces the fn-109.1 simulation leg: both paths must end bounded-complete with the supplied bounds, commit and retain every logical execution, replay each artifact exactly, and retain both `route alpha` and `route beta`. The simulation bounds changed to runtime limit 1 and MaxExecutions 16 so the forced Scenario alternative always runs. Built CLI: 6 logical executions, 4 rounds, `dimension_depth_complete`, 4 artifacts replay `choice-replay=exact`.

Gates (darwin/arm64): `make -C tools/gomad3 validate` and `test-host` (45 packages), root `go test ./tools/gomad3sim/...`, gofmt, `go vet`, `make lint-code` for gomad3sim pass; runner, target and validate re-run green after the review edit, test-host not repeated. Not run: linux/amd64, full `make -C tools/gomad3 test`. No commits or staging.

Open, outside this task:
- `TestRootProcessSimulationUsesRunnerTransport` (`integration` tag, no Make or CI entrypoint) still exits 1. Its exploration subtest is now green; `TestScenarioChoicePlanRejectsChangedDecisionBeforeSelection` fails as before for a different cause in `tools/gomad3sim/controller.go` (read from code, not proven), and process-backend subtests intermittently hit the watchdog.
- A program importing `tools/gomad3sim` without `net` fails to link; a simulation-exploration target must hard-code the campaign Seed (fixture: 89); success artifacts with equal outcomes in one round share a directory while counted separately.
- The added allowlist directives are the ones the task named; the conductor should confirm that reading of "allowed bridge directives unchanged".
- Follow-up: memory capture for the cross-module identity drift was not written.

Detail: `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task22-evidence.txt`, `task22-review.md`.

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol) - round 1 NEEDS_WORK (one blocker, applied), round 2 SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (.toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./target/... rc=0; go test -count=1 -tags test_dep ./tools/gomad3sim/... rc=0), make -C tools/gomad3 validate (rc=0; re-run after the review edit rc=0), make -C tools/gomad3 test-host (rc=0, 45 packages ok; run before the review-round edit to coordinator_transport_test.go, not repeated), env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./target/... (rc=0, 14 packages, final tree), go test -count=1 -tags test_dep ./tools/gomad3sim/... (rc=0, final tree), GOWORK=off go vet -tags test_dep ./runner/... ./target/... (rc=0); go vet -tags test_dep,gomad3_toolchain ./tools/gomad3sim ./tools/gomad3sim/testdata/simulation_exploration (rc=0); gofmt -l (clean), make lint-code LINT_CODE_TARGETS=./tools/gomad3sim/... GOLANGCI_LINT_BASE_REV=HEAD (rc=0, 0 issues), built CLI: gomad explore --strategy=simulation-exploration --seeds 89 ... go-run ./tools/gomad3sim/testdata/simulation_exploration (exit 0, bounded_complete, 6 logical executions; gomad replay of 4 artifacts: reproduced=true choice-replay=exact), red before fix: TestBuiltInSimulationLinknamesPinCurrentFirstPartySources; TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests and TestIsolatedRunnerCompletesSimulationStrategyWithSuppliedLimits (go test -overlay with pre-edit capability.go); TestExplorationIdentitiesMatchRunnerIssuedPlanVector; TestExplorationEvidenceProvesOnlyTheOverridesTheClusterDecides; TestRootProcessSimulationUsesRunnerTransport/TestProcessExplorationConsumesExternalPlanAndPublishesRecord, UNGATED, still rc=1: -tags test_dep,integration -run TestRootProcessSimulationUsesRunnerTransport ./runner/internal/execution (exploration subtest green 3/3; TestScenarioChoicePlanRejectsChangedDecisionBeforeSelection fails as at baseline, different cause; intermittent process-backend watchdog), NOT RUN: linux/amd64 gates (darwin/arm64 host), NOT RUN: make -C tools/gomad3 test (full), gate receipts: none written (worktree dirty, commits forbidden)
- PRs: