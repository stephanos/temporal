# Replay fixture identity and bootstrap boundary, 2026-10-10

The existing campaign preparation seam cannot restore these replay/minimize fixtures. Replay additionally requires a pinned-toolchain identity and host-guarded adapter verification; replay and minimization construct bootstrap frames directly. A bounded solution needs a newly admitted private contract separating native admission from detached artifact validation and profile encoding. No existing owner currently supplies that contract. This research authorizes no implementation.

Research used the requested gpt-6-astra/high route; actual execution telemetry is unknown. JudgeTier was supplied as session, jev unavailable/no_key. HEAD at the initial and completed source inspection was `655dc3d6914e3c5c4322fed2e12069ef383e352b`; the final publication check observed `84f9198d58969509e55fd97364d364c7d8145fca`. All consumed-file hashes below remained unchanged across that movement. The governing dirty fn-109 spec remained SHA-256 `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. Its 2026-10-09 amendment removes byte/format equality requirements while preserving capabilities, guards, error precedence, transactions, lifetimes and workloads. It grants no identity substitution or replay fixture admission ([spec, lines 3-34](../../../specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md)).

## Actual blocking path

Source links below use repository-relative files; line numbers name the inspected revision.

| Stage | Current mechanism and preservation |
| --- | --- |
| Fixture publication | [replay_operation_test.go:610-679](../../../../tools/gomad3/runner/replay_operation_test.go) reads the real executable, its build information and bytes, then calls `target.ReadToolchainIdentity(toolchainRoot(t))`. Target SHA/size and BuildInfo are genuine. The fixed runner digest, outputs, choices and simulation record are fixture data. The helper does not prepare a target. |
| Installed identity | [target.go:357-399](../../../../tools/gomad3/target/target.go) validates the installation, invokes its Go command for GOVERSION/GOOS/GOARCH/CGO_ENABLED, requires CGO_ENABLED=0 and the actual host platform, and returns the installation build key. Inventing a root, key or platform is not equivalent evidence. |
| Replay preflight | [replay_operation.go:429-509](../../../../tools/gomad3/runner/replay_operation.go) opens/validates the artifact, matches the real default I/O profile, checks replay mode, decodes captured mounts and choices, checks host platform, reads and compares installed identity, validates World and seed, extracts linked/guarded capabilities when applicable, reads real retained build information, rejects coverage, and validates durations. An injected executor changes none of this. |
| Adapter admission | [replay_operation.go:90-95](../../../../tools/gomad3/runner/replay_operation.go) verifies compatibility packs, then calls Default().VerifyAdapters. [adapter_registry.go:121-151](../../../../tools/gomad3/deterministicio/adapter_registry.go) first invokes the supported-host guard, even for an empty adapter list. After that it checks nonnil, sorted/unique and exact module/version/sum identities. This is a third barrier beyond identity and bootstrap. |
| Execution assembly | Replay copies the verified target, checks its build information again, builds the World plan, constructs the bootstrap at line 181, restores captured mounts/transcript, and calls the executor at line 239. Verify-only returns at line 114 after adapter and tape validation, before bootstrap/execution. |
| Minimize | [minimize_operation.go:283-420](../../../../tools/gomad3/runner/minimize_operation.go) runs the same preflight, requires exact simulation failure and choice tape, copies the retained target and opens the real bounded workspace. Each trial directly calls BootstrapFrame at line 429. Its default replayer forwards the complete dependencies to replayWith at line 533; a supplied replayer has its existing separate contract. |

The resolved historical log is `.worktrees/fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/after-ordinary.log`, SHA-256 `10ca80cc190d8f29f0aca9440cf6eb44a11bf5d1433963f33920c85e8bfc26cd`. Its JSON terminal names were inspected without executing tests. The existing [frontier report](runner-frontier-after-task73-20261010.md) retains 673 terminals, 498 PASS/163 FAIL/12 SKIP and 42 missing-toolchain terminals. Those 42 cannot be classified collectively as native requirements. No new PASS or coverage claim follows from this research.

## Required identities and real validation

The independent expected toolchain identity must retain Go version, build key and platform; expected identity cannot simply be copied from whichever artifact is being checked. The fixture's actual executable bytes, size and projected build information must continue through publication, reopening, copying and build-info validation. Closure fixtures do not justify bypassing linked/guarded executable manifest extraction.

The profile must remain the actual Default identity, inventory and implementation digest. Adapter and compatibility checks must still reject changed or unavailable identities. Choice replay binds target SHA, build key, platform and controller identity; it decodes retained traces, projects the tape and compares tape digest/decision count ([replay_operation.go:361-406](../../../../tools/gomad3/runner/replay_operation.go)). World snapshots, transitions, seed and terminal data retain their existing validation before execution and comparison afterward. Captured mount descriptors and payloads retain actual DecodeCapturedInputs validation and bounded replay snapshots. Bootstrap encodes the real profile inventory/implementation, target digest, runner digest, canonical argv digest and seed ([bootstrap.go:25-79](../../../../tools/gomad3/deterministicio/bootstrap.go)).

The fake replay executor only records the request and returns its scripted result ([replay_operation_test.go:570-575](../../../../tools/gomad3/runner/replay_operation_test.go)). The minimization executor actually decodes the simulation plan and builds choice/simulation evidence, but never decodes the I/O bootstrap or starts the target. Its replayer opens the real artifact and returns a scripted exact-match result ([minimize_operation_test.go:450-568](../../../../tools/gomad3/runner/minimize_operation_test.go)). Those assertions remain valuable orchestration coverage but establish no native exact replay.

## Complete shared-helper consumer boundary

The publisher family is `replayArtifact → publishReplayArtifact → publishReplayArtifactWithWorldAndCompatibility`, plus `publishReplayArtifactWithCompatibility` and `publishReplayArtifactForTarget`, converging on `publishReplayArtifactForTargetAndCompatibility`. The transitive minimization family adds `minimizationParent`, `otherMinimizationParent`, `minimizationSpec`, `uninterruptedMinimization` and `interruptedMinimization`. A repository search found the following consumers. This inventory is not an implementation allowlist.

| File and exact tests | Mechanism and proposed disposition |
| --- | --- |
| replay_operation_test.go:35,69,87,120,135,150,170,261,458,500,526 | Scripted: TestReplayVerifiesThenRunsStoredTargetWithoutRebuilding; TestReplayAutomaticallySuppliesExactChoiceTape; TestReplayAutomaticallySuppliesExactSimulationExplorationTape; TestReplayReportsChangedSimulationExplorationRecord; TestReplayReportsTypedChoiceDivergenceBeforeOrdinaryComparison; TestReplayPreservesInfrastructureErrorJoinedWithChoiceDivergence; TestReplayDoesNotStartTarget; TestReplayRunsAnArtifactCopiedOutOfItsStore; TestReplayReportsFirstObservableDivergence; TestReplayRejectsUnexpectedWorldRecord; TestReplayPreflightValidatesConnectedWorldRecord. Candidate portable scope after a new boundary is admitted. |
| replay_operation_test.go:170 | Preserve all nine existing table cases: verify only, unavailable compatibility pack, changed payload, and altered/truncated/missing shared target for verify-only=false/true. Keep zero-executor-call assertions and real corruption. |
| minimize_operation_test.go:26,76,178,211,247,277,313,364,396 | All nine TestMinimize functions use the publisher directly or through minimizationParent. Preserve exact reduction/lineage, missing-tape rejection, resumed attempt order, independent parent state, foreign-run rejection, damaged accepted-artifact refusal, concurrent lock exclusion, final-publication validation and interrupted-publication recovery. They exercise real artifact/workspace operations with scripted candidate/replay results. Existing table children must remain; no new emitted outcomes are assumed. |
| executor_injection_characterization_test.go:35,59,91 | TestInjectionCharacterizationReplayRequiresSupervisorOnlyForTheProcessExecutor; TestInjectionCharacterizationMinimizeRequiresSupervisorOnlyForTheProcessExecutor; TestInjectionCharacterizationMinimizeDefaultReplayerUsesItsExecutor. Mixed private/default controls. Preserve the public Replay/verify-only assertions and their native setup. Add explicitly private portable controls separately if admitted; do not silently convert the public calls or weaken their expected errors. |
| diagnostics_test.go:114 | TestDiagnosticArtifactReplaysWithoutCollectingSidecar. Scripted replay, real retained environment, no sidecar collection; separate from the fresh-campaign diagnostic-sidecar proposal. |
| inspect_test.go:71 | TestOpenReportsMinimizationLineageAndBounds. Scripted minimization followed by real Inspect. |
| runner_test.go:2778 | TestMinimizeKeepsOneTargetCopyInItsOutputRoot. Transitive minimization helper, real sharing/interrupt/resume and wrapping sharedTargetReplayer. Storage assertions must remain; no storage implementation change follows. |
| watchdog_replay_test.go:138 | TestWatchdogDiagnosticReplayRejectsUnsupportedChoiceEvidence. Shared publisher plus real republication and scripted replay rejection; candidate portable scope. |
| replay_operation_test.go:286,324 | TestReplayRejectsFirstDivergentWorldTransitionBeforeTargetMutation and TestReplayExecutesMatchingWorldPlanThroughChildTransport actually call execution.Run through recordReplayIOTranscript and replay with empty dependencies. Keep native process, runtime I/O and World transport unchanged. |
| coverage_replay_test.go:26 | TestReplayAndMinimizeRejectRetainedCoverageBinary first compiles a covered binary with the pinned Go command, then uses replayArtifact. Preserve real compilation/build-info rejection. It is not a fixture-only conversion. |

TestReplayEnvironmentExcludesChoiceControlVariables, TestReplayReportsIOTranscriptOrdinalBeforeOutcome, TestReplayReportsFinalChoiceTraceMismatch and TestReplayBuildInfoRejectsMatchingCoverageInstrumentation do not call this publisher. They remain unchanged.

TestWatchdogDiagnosticReplayUsesCapturedInputs uses a different builder, watchdogReplayInput, which actually calls target.Prepare, constructs a real frame, executes until the watchdog observes target output and captures mount inputs; public Replay executes it again after source removal ([watchdog_replay_test.go:19,184-299](../../../../tools/gomad3/runner/watchdog_replay_test.go)). The three replay_io_integration_test.go tests also prepare native targets; retain all of these native paths.

## Smallest source-supported option and required decision

[executionDependencies](../../../../tools/gomad3/runner/preparation_dependencies.go) has only executor, prepare and bootstrap. Nil delegates lazily to the original operation; injected() protects isolated campaigns. Replay ignores prepare/bootstrap, minimization ignores bootstrap, and both call the direct preflight. [internal/preparation](../../../../tools/gomad3/internal/preparation/preparation.go) owns fresh adapter preparation, target preparation and validation. Calling it for retained replay would rebuild or change the wrong contract. Extending the campaign marker helper globally would affect the native consumers above.

A useful existing split lives inside deterministicio. BootstrapFrame does admission then private encodeBootstrapFrame; VerifyAdapters does admission then private adapterRegistry.verify. [profile_portable_test.go:121-140](../../../../tools/gomad3/deterministicio/profile_portable_test.go) uses the real private encoder and public decoder, and its adapter tests use the actual registry. Runner cannot access those private operations. No existing opaque fixture context combines them with independently expected toolchain identity.

The bounded next owner decision is whether to introduce one repo-private retained-execution binding contract, with two explicit construction paths: normal installed/native resolution, and deliberately supplied fixture inputs for same-package scripted execution. Its owner should retain real profile evidence, adapter checks and bootstrap encoding together, using the existing deterministicio mechanics; Runner should continue owning artifact open/World/choice/mount/build-info validation. The fixture constructor needs an expressly approved source and meaning for the independent expected toolchain identity. Without a real supplied identity or a new, clearly distinguished fixture-identity contract, this research cannot recommend fabricating a build key or presenting an ordinary executable as a patched target. Reading the artifact's own identity back as the expectation would make the mismatch check vacuous.

This is a semantic design admission, not a two-line dependency attachment. It should define the contract before choosing the exact package/API shape. A chain of verifier callbacks returning nil, a replacement preflight returning an already accepted artifact, a made-up profile, an automatic unsupported-host fallback, or public guard removal would fail preservation. If a future private fixture path needs a frame, it should use the real encoder and verify the real decoder against all bound fields. The existing scripted marker is only request plumbing evidence and must never enter a real decoder or process.

The current child path decodes the bounded bootstrap request, installs IOConfig and launches the target ([bootstrap_unix.go:35-83](../../../../tools/gomad3/runner/internal/execution/bootstrap_unix.go)); activated overlay I/O calls gomadwire.DecodeBootstrap ([gomadio.go:28-40](../../../../tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/gomadio.go)). A passing fake executor does not exercise either path.

The new owner must name its exact initial consumer slice and preserve these controls:

- Public/default and executor-only calls retain the current installed-identity failures, adapter host guard, supervisor requirements and lazy ordering. Verify-only still validates before returning. Minimize still checks output root and attempt budget before preflight.
- Artifact corruption/profile/mount/choice/platform rejection precedes identity lookup; identity mismatch precedes World/build-info validation; compatibility precedes adapter admission; bootstrap stays at its current execution stage. Original error wrapping and cleanup precedence survive.
- Explicit fixture inputs reject mismatched expected identity, missing/changed/unsorted adapters, changed target/build-info/coverage, malformed World/choices/mounts and malformed bootstrap; injected operation failure does not execute the target.
- Every newly injectable operation participates in isolated-campaign rejection, with executor nil and existing resume-preflight precedence. Existing preparation/default/isolated controls remain ([preparation_dependencies_test.go](../../../../tools/gomad3/runner/preparation_dependencies_test.go), [executor_injection_characterization_test.go:163-230](../../../../tools/gomad3/runner/executor_injection_characterization_test.go)).
- Default minimizer replay retains all dependencies, while explicitly supplied ArtifactReplayer retains its existing behavior. No executor-type heuristic switches into fixture mode.

Task 65 expressly excludes replay/minimize owners and only admits its first six fresh-campaign calls ([task 65](../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.65.md)). Task 74's later one-test-file work cannot admit this contract. A dated new owner is required before implementation, followed by source tests, default/public/isolation controls, architecture/generated checks as implicated by extraction, lint, frozen ordinary evidence and independent review. The current format waiver does not waive those requirements.

The true native work remains the two World child-transport tests, actual covered/linked target preparation, I/O/mount/watchdog executions, public installed-toolchain controls, patched bootstrap/runtime behavior and each platform's qualification. fn-128/fn-149 remain deferred and unverified. Neither the old portable profile proof nor this source survey supplies a new test-host pass, native qualification, soak bound or CI/PR/push authority.

## Consumed-file binding

The following SHA-256 values were captured during inspection and checked again after the source investigation; every listed first/post value matched. Some first hashes followed preliminary search/read, so this is a bounded observed stability check, not an assertion that a filesystem snapshot covered every earlier read. No product edits or Go/build/lint/vet/generator/test/native commands were run. Only this report was written.

```text
8d634df5cbbbffd7dbada06e32b4d20d879707273f8be07bdf253b387211e6f3  AGENTS.md
fb85ed4952fb925ca31768b516fa01285d73fa2738551d9781cd6264cda0f610  tools/gomad3/README.md
7f8bbd53d0f7de462c49cb354a448a4efaeb3b043d88fd3e6a59f4c7954167d7  MILESTONES.md
851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c  .flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md
82d6419dbf1b9393dd8ab5dc02225a12bc04e9c1b41565f25eb8408093ba7bbf  .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.65.md
24dbae80d5175671d6b89c163e35df3ab459c17afc5662ef2ad73110538bdb49  tools/gomad3/runner/replay_operation.go
bf076fd378e6f58777b0d00a34127713df9f0a45868c108ec8e9486716c5b7ed  tools/gomad3/runner/replay_operation_test.go
758cc37786e21093a9e4c1833eab3feb10e571c2aa9bb029024d52b71359f346  tools/gomad3/runner/minimize_operation.go
20b09aa279bad6391044609e3b77fd047113b212f2be3cbd20b45cbcac2a3275  tools/gomad3/runner/minimize_operation_test.go
4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56  tools/gomad3/runner/preparation_dependencies.go
c74d91de827cb171fb9565f697c6254f02b2091c69e51cbf916f76ce0921ecb9  tools/gomad3/runner/preparation_dependencies_test.go
c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e  tools/gomad3/runner/preparation_fixture_test.go
b225b832599a605854e297b662c71d4c4e3684a903b40ccf6805c9ff7ef7ef0d  tools/gomad3/runner/executor_injection_characterization_test.go
84bcb2df3a4db9001a806199eb04e451cb4f3ee01889b5241d3d81db86b42c65  tools/gomad3/runner/coverage_replay_test.go
885b3df456c376522b484cfbdaa70e5f70ffe4d77a9c8af1d825740443606b68  tools/gomad3/runner/diagnostics_test.go
140e6688d3e1dddb8dbc1e85ae53de0de15fd676f655227efae6fa09282ca8df  tools/gomad3/runner/inspect_test.go
2ec97e04725e893b6435621546c4bffb66bf22fc7f515a6a28540c1266962e42  tools/gomad3/runner/watchdog_replay_test.go
a1c7ea227188bb4231a1d30ff21c3f575cc09bf3b2087e11a945e5f4e6d3fe2b  tools/gomad3/runner/replay_io_integration_test.go
d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96  tools/gomad3/runner/runner_test.go
6f7a7904efaf956712678ee996fac80e9189b5963714125a2a778d6e530cd744  tools/gomad3/deterministicio/bootstrap.go
d068900a3b76bc91e5b94f0d67e00e67e225c50c0f3d23cbf92ce992ef0c5f21  tools/gomad3/deterministicio/profile.go
308a7ca06057995978dd750b5073c0307c8053040748743b905b88d0625face9  tools/gomad3/deterministicio/adapter_registry.go
6dfd6c349c6c6a05995d5f9343541e0529ab7eb2e453607de3026c02dd4932c4  tools/gomad3/deterministicio/profile_portable_test.go
bf5a1c8e193650913220fd3a1de6ae2aa0dbf264f1a77a1c77bb52bb9a848bf7  tools/gomad3/target/target.go
52f02f1dc0d1f90910eb9c7e409093e1731e01c91467084c040f4369417247f6  tools/gomad3/internal/preparation/preparation.go
190a8e7dd4e5ffa3a66344406f3a6c1216711a012479741623cef0d7c7b58b91  tools/gomad3/runner/internal/execution/bootstrap_unix.go
5e6b5e9609372bd7402ed6f1a4c87e8e2bf9745dd4b9d5b52d16274879198509  tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/gomadio.go
0c16ae4b3ed06d6f5a27bc5c1161b56c5ecac98ead019b4999d8c4bb66e7a0cc  .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-frontier-after-task73-20261010.md
10ca80cc190d8f29f0aca9440cf6eb44a11bf5d1433963f33920c85e8bfc26cd  .worktrees/fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/after-ordinary.log
```
