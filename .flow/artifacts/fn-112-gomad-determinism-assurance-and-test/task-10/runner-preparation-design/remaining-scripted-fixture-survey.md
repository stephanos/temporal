# Remaining scripted Runner fixture survey

The smallest new candidate is one preparation attachment in `runner_mode_unix_test.go`. Two other independent one-assignment candidates cover diagnostic-sidecar retention and the 10/100-job bounds fixture. All three preserve the existing final executor, real target copy/verification, fixture payloads and assertions. Root can admit any one separately; this report grants no implementation scope or expected pass count.

The survey excludes the already surveyed `retention_test.go`, `choice_exploration_divergence_test.go`, `completion_characterization_test.go`, `seed_completion_characterization_test.go` and `runner_test.go`. Their still-observed failures remain in the raw inventory. No future result from tasks69/70/71 or the proposed seed slice is subtracted. The three recommended sites do not call real target execution, runtime bootstrap decoding, resume, replay, minimize or native qualification. Ordinary filesystem operations and existing result-evidence decoders remain active where their assertions require them.

## Source and evidence boundary

Research read PRIMARY `/Users/stephan/Workspace/skunkworks/gomad/temporal` at HEAD `87d1927a90295dc267c33d6226a643a74c425f51`. The authoritative fn-109 spec SHA-256 remains `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. Its current R5/R18/R19 requirements at lines 329/418/425 and the opening format amendment preserve private failure coverage, original behavior/assertions and source-bound verification. `MILESTONES.md:38` retains the source-delivery order and native deferrals.

The research-directory search found eight existing reports. `fixture-research.md`/`seam-research.md` own the original seam; `next-scripted-slice.md`, `post-task66-next-slice.md`, `retained-success-next-slice.md`, `choice-divergence-next-slice.md`, `completion-next-slice.md` and `seed-completion-next-slice.md` own their named fixtures. None surveys the three files recommended here. The seed report was left unchanged at SHA-256 `68cd8095264fcf63a8d757724e00f774433570548de2665bd4b9405b890e8fc5`.

The only authored file is this report. No product, task metadata or Git state was changed. No Go, build, test, lint, vet, generator, toolchain, Flow, real-process test or native execution command ran. All shell reads used `login:false`, `env -u BASH_ENV bash -c` and an explicit PRIMARY `cd`; Git reads used `GIT_OPTIONAL_LOCKS=0`. This report follows the research workflow and Flow prose contract.

The retained evidence directory is `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-68/`.

| File | SHA-256 |
| --- | --- |
| `ordinary-runner.log` | `3b75cacdac2bcc7016dc5f7a99f924f6b958f0cbc7d58e59afe77980b7b34b1e` |
| `run-binding.json` | `50780dd40849c42621f0f1d707132fb44b022b1aaf7fb9846da2315b79483d28` |
| `ordinary-runner.json` | `46dd5ad24dd65e838d13d430a959dc28954d184de75b27d0ede796b871f80ec0` |
| `source-before.json` and `source-after.json` | `f76076a82761a2a250eda3e97bf23c51d1c1520a5a53023b2c0897a31f756314` |
| `outcome-comparison.json` | `487dbedd092007fa33118bfccffe131b8afc42d1c7161ccd02f12a7c0cb8f0cd` |

`run-binding.json` binds frozen execution HEAD `524c092a3f6cbf5834895ae5ba7821d6fa610924`, developmental Linux/aarch64 and the exact PRIMARY owner-spec hash. The ordinary command was `go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner`, exit 1, elapsed 20.411523510003462 seconds. Before/after execution source manifests match. `checkpoint-note.md` maps task68's isolated checkpoint to PRIMARY integration `c506713ce063759c5d24129d775d2fefc6314618`. A fresh read-only diff found no tracked Runner-tree changes between the frozen revision and the inspected PRIMARY working tree. This is bounded source equality; the checkout as a whole is not claimed identical or qualified.

## Actual observed failure map

The raw log has 673 named terminal events, consisting of 392 PASS, 269 FAIL and 12 SKIP. Grouping each actual FAIL by the source file defining its top-level test yields the following complete inventory. Parent events count where emitted. Counts describe that frozen run, not a forecast after admitted work. Mapping uses test definitions in the unchanged Runner tree; no slash-derived intermediate event was inserted.

| File under `tools/gomad3/runner/` | Observed FAIL events | Frontier |
| --- | ---: | --- |
| `choice_exploration_divergence_test.go` | 28 | Already surveyed; task70 and its explicit exclusions |
| `choice_exploration_divergence_unix_test.go` | 1 | Actual crash subprocess and resume |
| `completion_characterization_test.go` | 60 | Already surveyed; task71 |
| `coordinator_transport_test.go` | 3 | Real isolated execution/toolchain |
| `coverage_replay_test.go` | 1 | Compiler plus replay/minimize |
| `diagnostics_test.go` | 4 | One ranked fresh call; other paths use plan/shard/guidance, replay or resume |
| `environment_integration_test.go` | 4 | Actual target initialization/execution/replay |
| `executor_injection_characterization_test.go` | 5 | Intentional public/private/default replay, minimize, shard and resume boundaries |
| `guidance_identity_test.go` | 9 | Guided corpus/replayer behavior |
| `guided_selection_test.go` | 12 | Guidance, actual replay, plans/shards and resume |
| `inspect_test.go` | 3 | Two fresh simulation calls outside the top-three ranking; one minimize path |
| `minimize_operation_test.go` | 9 | Minimize and persisted resume behavior |
| `portable_plan_test.go` | 7 | Portable plan/shard/merge contracts, including resume |
| `preparation_owner_test.go` | 1 | Deliberate real adapter preparation |
| `replay_io_integration_test.go` | 3 | Real toolchain/preparation/replay |
| `replay_operation_test.go` | 22 | Replay validation/execution and connected World transport |
| `retention_characterization_test.go` | 49 | One ranked bounds call; other tables have shared calibration, resume or guidance dependencies |
| `retention_test.go` | 2 | Already surveyed; task69 |
| `runner_mode_unix_test.go` | 1 | Ranked ordinary Unix mode fixture |
| `runner_test.go` | 27 | Excluded from this survey, including prior admitted/surveyed slices |
| `seed_completion_characterization_test.go` | 16 | Already surveyed proposed seed slice |
| `watchdog_replay_test.go` | 2 | Actual target preparation and replay |
| Total | 269 | All currently observed FAIL events retained |

## Ranked one-assignment candidates

For each candidate, insert only the following statement at its specified existing call site. It retains the final outer executor and preparer instead of changing their construction.

```go
configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
```

| Rank | Actual terminal name copied from raw log | Existing call and placement | Raw diagnostic / FAIL lines |
| --- | --- | --- | --- |
| 1 | `TestRunEnforcesBatchModesIndependentOfUmask` | `runner_mode_unix_test.go:18`, after the original umask change and deferred restoration at 15/16 | 2364 / 2366 |
| 2 | `TestDiagnosticsRetainedWhenSuccessArtifactsAreDiscarded` | `diagnostics_test.go:57`, after Diagnostics, choice limit, evidence and coverage configuration at 53-56 | 1301 / 1303 |
| 3 | `TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs` | `retention_characterization_test.go:891`, inside this test's local `run` closure, after `configure(&config)` at 890 | 2204 / 2206 |

All three raw diagnostics explicitly report `deterministic I/O requires one of darwin/arm64, linux/amd64; host is linux/arm64`. They establish preparation RED, not a demonstrated defect in the downstream behavior. Each candidate can be admitted alone. Removing its single assignment from the eventual candidate must restore its entire file to the pre-edit bytes/hash. This source-only survey has not produced or executed a changed Go file or selective-removal proof.

### 1. Unix private modes

This 39-line fixture has one fresh local campaign and no helper-internal campaign. `runner_mode_unix_test.go:14` constructs `newFakePreparer` and `fakeExecutor` before setting the process umask to 0777. The attachment can occur after the existing deferred restoration. It does not change that global state's interval or the test's serial execution. The test must continue checking all seven real paths at lines 22-36: campaign root, failures, partials and executions directories at 0700, and campaign JSON, execution index and segment at 0600.

`runner_test.go:2332` shows the fake executor merely recording the request and returning its scripted result. It neither decodes bootstrap bytes nor runs the target. Real mode enforcement remains in `internal/campaign/campaign_journal.go:200`, `:253` and `:639`, including explicit chmod after directory creation; journal files and segments retain their existing explicit chmod calls. `fakePreparer.Prepare` at `runner_test.go:2012` still copies and chmods the target under the real prepared root, which `BeginPreparation` already creates privately. The Unix build constraint and umask operation test ordinary host filesystem behavior. They supply no patched-runtime or supported-native qualification claim. This ranks first because its single call restores one narrowly scoped filesystem assertion group with no additional evidence protocol or calibration.

### 2. Diagnostic sidecar with discarded successes

Retain the outer `*diagnosticExecutor` at `diagnostics_test.go:51`; passing its `delegate` instead would remove the Diagnostics-forwarding check at line 26 and the existing diagnostic result at lines 33-45. The delegate `explorationExecutor.Run` at `runner_test.go:2154` constructs the existing complete choice trace from fixed fixture identity. This test leaves Strategy at the seed default. Neither executor reads `request.IO.Config`, starts a process or invokes artifact replay.

The existing fixture decodes its diagnostic result bytes, which must remain unchanged. This is a result-evidence decoder, separate from runtime I/O bootstrap decoding. `runner.go:718` checks diagnostic decoding and record-count agreement; `runner_local.go:597` retains the sidecar before success retention; `diagnostics.go:24` decodes, syncs and publishes the real sidecar. `evidence.go:125` attaches its identity to execution evidence. The original assertions at `diagnostics_test.go:61`, `:68`, `:75` and `:82` require zero retained successful artifacts, both diagnostic references, matching sidecar digest, mode 0600 and a real journal execution with no success artifact. The proposed assignment preserves each condition and the original nil/panic sensitivity of those expressions.

Keep the other three failing tests in this file excluded. `TestDiagnosticsPlanShardAndGuidanceKeepTheProfile` uses plan/shard and a guided replayer at lines 94/98/104/105. `TestDiagnosticArtifactReplaysWithoutCollectingSidecar` calls `replayWith` at 117. `TestDiagnosticsResumeRestoresTheRecordedProfile` invokes the resume path at 174. A preparation assignment at their first call would not establish their remaining contracts.

### 3. Bounded active executions and retention

The local `run` closure at `retention_characterization_test.go:885` creates a new `pairedExecutor` and fresh matching fake preparer for every invocation. It applies the caller's retention configuration before the proposed attachment. This closure belongs only to the named test; changing it does not change `retentionCampaign`, `explorationRanks`, `unorderedRetentionCampaign` or any other test.

Keep the same `*pairedExecutor`. Its `Run` at line 845 ignores the execution request, synchronizes paired executions with a mutex/channel, returns the original bounded payload and complete empty transcript, and observes cancellation while waiting. It does not decode bootstrap, run a target or use World/replay machinery. The retained-success store still publishes and measures actual artifacts. `keepSuccesses` at line 262 only supplies the existing success limits and policy.

The initial two-job calibration at line 901 must still retain exactly two successes at line 902 and derive the existing byte limit at line 905. The source then loops over job sizes 10 and 100 and the existing discard, count-limit and byte-limit rows at lines 906-930. Preserve the maximum-active value 2, each row's attempt/retention counts, the capacity reason and the complete struct comparison at line 932. This is the directly relevant ordinary-source 10x control, but the report establishes no completed R19 measurement.

The retained log contains only the exact top-level FAIL name in the ranking table. The calibration fails before the `t.Run` statement at line 931, so no child terminal names are available to quote from that run. This report intentionally supplies no synthesized slash names or predicted child PASS counts. If admitted work reaches the unchanged loop, root must record its actual emitted names and account for newly reached source-owned events in its comparison. A rule that assumes every original preserved table already emitted all its names would be incorrect for this fixture. This additional evidence obligation places it third despite its single source insertion.

## Shared dependency trace and hashes

All three candidates use the existing `testConfig` at `runner_test.go:2504` and `newFakePreparer` at `:1981`. Target kind/source/argv already match; none overrides target metadata, arguments or adapters. `preparation_fixture_test.go:17` checks the explicit preparer/executor and request shape, calls the real preparer at 28 and `Prepared.Verify` at 35, then forwards the supplied executor unchanged. Its bootstrap at 41 returns the explicit synthetic marker. `preparation_dependencies.go:17` and `:24` retain lazy real defaults for all other callers.

Production flow still reaches preparation at `runner_local.go:250`, the final executor at `runner.go:712`, postexecution file verification at `runner_local.go:527`, ordinary assessment and the existing journal/publication transitions. No candidate reaches the real bootstrap decoder at `deterministicio/bootstrap.go:53`. A switch to a decoder or process executor would invalidate this eligibility conclusion. Default/public/isolated guards remain separately protected by the existing preparation and injection control tests named in `seed-completion-next-slice.md:98`.

| Current file under `tools/gomad3/runner/` | SHA-256 |
| --- | --- |
| `runner_mode_unix_test.go` | `ebdc20609fd89c246bf345e0df40f3c126b09765d898770dabe4389518228168` |
| `diagnostics_test.go` | `885b3df456c376522b484cfbdaa70e5f70ffe4d77a9c8af1d825740443606b68` |
| `retention_characterization_test.go` | `87253da780594e54139a2359ef5d2655ffbf4905b54eb1380c3700f3edaf8b4e` |
| `inspect_test.go` | `140e6688d3e1dddb8dbc1e85ae53de0de15fd676f655227efae6fa09282ca8df` |
| `preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |
| `runner_test.go` | `d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96` |
| `runner.go` | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| `runner_local.go` | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |
| `diagnostics.go` | `f07431ec3cdf4799c65d286b3fe72d93c0560885b734982ff3e05679722d0e29` |
| `evidence.go` | `05d6b71e29fe8eb3fba4100f74584a16241e6dbd7d4343dd4ce12366a53a3fe4` |
| `internal/campaign/campaign_journal.go` | `acd68ff0131f4d2d1f6050a0cb6e53e8a4ebc8e6a58cd5299a2cbaa7cc8b2f25` |

The three candidate file hashes match `source-before.json` entries at lines 869, 718 and 864. The read-only frozen-to-current Runner-tree comparison also covers their dependencies. That equality does not replace future candidate-bound execution or review.

## Rejected and unranked frontier

The other retention-characterization tables are not a safe outer-call-only group. `retentionCampaign` at line 143 calls `explorationRanks` at 146. Before returning the outer configuration, that shared helper performs its own campaign at line 169 and caches platform-dependent ordering at 183. Adding an assignment only at the first table's outer line 488 or the capacity table's lines 520/561 cannot reach those strategy assertions from a cold cache. Mutating the shared helper silently affects multiple consumers and exceeds the ranked one-call scope. `TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState` additionally resumes at 720 through `resumeRetentionCampaign:270`; `TestGuidedAdmissionReplaysBeforeTheCorpusAdvances` sets Guide/Replayer at 797/799. A larger explicit helper/calibration admission and its own source proof would be needed; no cache seeding, ordering map replacement or expectation adjustment is justified here.

The two fresh simulation inspection calls at `inspect_test.go:57` and `:159` use `simulationExplorationExecutor` and preserve artifact/campaign inspection assertions at 66 and 168/171. They remain an unranked two-call frontier because the three smaller one-call groups suffice for this bounded survey. Their separate simulation-plan/result/publication trace would need explicit admission and retained verification. The file's minimization test at 73 is excluded, so a whole-file or shared-helper migration would be inappropriate.

Actual compiler/process and operation boundaries remain excluded, regardless of a nearby fake executor. Concrete sources include `coverage_replay_test.go:34` building an instrumented target; `preparation_owner_test.go:13` using a nil preparer and `:27` running real adapter preparation; `coordinator_transport_test.go:497` selecting the patched toolchain; `environment_integration_test.go:72` launching a binary; `watchdog_replay_test.go:240` preparing a real target; and the replay/minimize operation calls in their dedicated fixture files. The crash fixture at `choice_exploration_divergence_unix_test.go:50` launches a test subprocess, checks actual SIGKILL at 119 and resumes at 127. That ordinary crash regression is not automatically a transferred native gate, but it still lies outside a fresh scripted-call correction.

Portable plans/shards and mixed public/private injection cases retain their intentional construction, toolchain and validation boundaries, including `portable_plan_test.go:239` and `executor_injection_characterization_test.go:145` resume operations. Guidance invokes its selected replayer at `guidance.go:103`; guided selection/identity tests therefore remain outside this no-replay slice. Synthetic bootstrap must never be passed off as a real frame or used to weaken their original validation.

Every recommendation remains pending root admission and candidate-bound unchanged RED, original downstream assertions, whole-file selective-removal proof, seam/public/default/error/isolated controls, applicable source standards/boundary checks and independent integrated review. Unexpected downstream failure requires a separate root decision; it does not authorize helper, assertion, datum, runtime, public/default, lint-policy or production changes. The frozen ordinary gate remains RED with its actual 392/269/12 outcomes. Native fn-128/fn-149 owners remain deferred and unverified; no qualification, soak, formal SHIP, Done, CI, PR or push claim follows.
