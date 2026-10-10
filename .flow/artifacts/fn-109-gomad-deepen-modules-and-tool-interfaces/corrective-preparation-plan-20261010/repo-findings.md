# Six fresh-campaign preparation attachments

The six call-site attachments form a cohesive M-sized correction using the existing `scriptedPreparationDependencies`. They restore explicit preparation for existing scripted executors across diagnostics, retention bounds and simulation inspection. Source inspection found no selected executor that decodes the synthetic bootstrap or launches its prepared file. This is a planning finding, with no executed successor result or PASS forecast.

The retention slice requires task 74's admitted calibration source to be integrated and rebound first. Its two assignments are absent from the inspected PRIMARY. The successor must preserve those assignments and add exactly six further assignments, with no constructor, helper, assertion or production change. Formal task 74 completion is a separate root decision from this source prerequisite.

## Exact anchors

All source paths below are relative to `tools/gomad3/runner/`; line numbers name PRIMARY HEAD `305182879f727f0e925483c41b8b8f0679770621` before task 74 integration.

| File and existing function | Immediately before | Final state that must precede attachment |
| --- | --- | --- |
| `diagnostics_test.go`, `TestDiagnosticsRetainedWhenSuccessArtifactsAreDiscarded` at 49 | `exploreWith` at 57 | Diagnostics, choice limit, execution evidence and semantic-choice coverage at 53-56; retain outer `diagnosticExecutor` |
| `retention_characterization_test.go`, `TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy` at 509, local `run` | `exploreWith` at 520 | `configure`, `rankProbes`, conditional `reverseRankOrder` at 515-519; retain outer `retentionExecutor` |
| Same test, `bytes` subtest | `exploreWith` at 561 | Measured byte limit, `KeepSuccessesAll`, `rankProbes` at 556-560 |
| `retention_characterization_test.go`, `TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs` at 878, local `run` | `exploreWith` at 891 | One-minute overall timeout and final `configure` at 889-890; use final `config.Preparer` and outer `pairedExecutor` |
| `inspect_test.go`, `TestOpenReportsSimulationExplorationEvidence` at 44 | `exploreWith` at 57 | All six simulation dimension limits and other bounds at 49-55; retain failing `simulationExplorationExecutor` |
| `inspect_test.go`, `TestOpenReportsSimulationExplorationBoundsAndRemainingWork` at 146 | `exploreWith` at 159 | All six simulation dimension limits and other bounds at 151-157; retain successful `simulationExplorationExecutor` |

The two guessed inspection names in the dispatch do not name these tests. The actual names above are the BASE57/159 controls. Attach through the existing helper with final `config.Preparer` and `configDependencies.executor`. Removing only the six successor assignments must reconstruct all three immediately preceding files, including task 74's retained changes.

## Existing execution and observation graph

`testConfig` (`runner_test.go:2504`) creates the existing `CampaignSpec` and executor-only private dependencies. `scriptedPreparationDependencies` (`preparation_fixture_test.go:17`) preserves that executor, checks the explicit preparer and prepared target shape, calls the actual preparer, invokes `Prepared.Verify`, and returns the existing conspicuously synthetic bootstrap marker. `newFakePreparer` and `fakePreparer.Prepare` (`runner_test.go:1981,2012`) still create/copy the real mode-0500 fixture file with recorded size and hash. No fake profile or identity source is needed.

`exploreWith` (`runner.go:377`) enters `runLocal` (`runner_local.go:51`), which retains request validation, journal lifecycle, preparation, executor selection and strategy dispatch. `localCampaign.prepareTarget` (`runner_local.go:243`) still records the real plan. `runSeed` (`runner.go:647`) creates partial outputs before requesting bootstrap, forwards the marker only in `execution.Spec.IO.Config`, runs the final outer executor at 712, then validates choice and diagnostic evidence at 716-723. Existing preparation/bootstrap callbacks, output lifetime and error precedence remain unchanged.

The selected executor branches are exhaustive for this proposal.

- `diagnosticExecutor.Run` (`diagnostics_test.go:24`) delegates to `explorationExecutor.Run` (`runner_test.go:2154`), checks diagnostics, decodes its original diagnostic bytes and supplies its complete empty transcript. `runSeed` performs a second real diagnostic decode and checks record-count agreement. `retainDiagnosticTrace` (`diagnostics.go:24`) preserves decoding, private directory/file publication, hash, sync and cleanup. The test reads the sidecar, checks SHA identity and mode 0600, reopens the campaign and checks one journaled execution without a successful artifact (`diagnostics_test.go:61-83`). Keep all of these assertions and its original fixture data.
- `retentionExecutor.Run` (`retention_characterization_test.go:108`) preserves rank mapping, completion channels, before/after callbacks and World recording. Its bases are `fakeExecutor.Run`, `explorationExecutor.Run` and `simulationExplorationExecutor.Run` (`runner_test.go:2332,2154,2196`). They consume seed, choice prefix or simulation-plan data and return fixture results. `completionWorldRecord` (`completion_characterization_test.go:111`), `rankProbes` and `semanticTranscript` retain actual World/transcript encoding. None reads `IO.Config` or invokes a process executor.
- `pairedExecutor.Run` (`retention_characterization_test.go:845`) ignores the request and uses its original mutex/channel pairing and bounded payload. Preserve the real two-success calibration at 901-905 and its byte calculation, then all discard/count/bytes cases at 906-935 for both 10 and 100 jobs. Width must remain 2; bounded retention expects 5 attempts and 3 retained successes. The attachment belongs after `configure`, so calibration and every row receive the same explicit treatment.
- Both inspection tests use `simulationExplorationExecutor.Run` and its actual simulation-plan JSON parsing, canonical scenario decision and recorded simulation payload (`runner_test.go:2196`). The failing case retains the original artifact and invokes `Inspect`; the successful case retains the actual two-execution campaign. `Inspect` (`inspect.go:361`) opens/validates artifacts, lifecycle, simulation journals and campaigns; `projectArtifact` and `projectCampaign` (`inspect.go:676,770`) project their real records. Neither selected inspection invokes Replay, Minimize, Resume or bootstrap decoding. Preserve the artifact's six nonempty identities, plan/record sizes and 128-MiB record limit, plus campaign logical executions 2, pending 0, scenario limit 1, implementation/chain identities, two journal records and forced depth 1 (`inspect_test.go:65-67,167-172`).

The capacity table covers all three `completionStrategies` (`completion_characterization_test.go:30`). Preserve count, count-inside-a-round and bytes rows for seed and both exploration strategies, including their distinct commit points and reversed parallel completion. Its bytes row first measures three actual retained successes through local `run`, divides the real total by three, then checks both exact-bound exhaustion and typed `artifact.CapacityError.Maximum` against remaining bytes (`retention_characterization_test.go:551-572`). `observeRetention` and `journaledExecutions` (337,294) preserve real artifact opening, exact-replay metadata, ordinal and stored-byte agreement, success order/count/total checks, failure references and actual journal decoding. Reading exact-replay metadata does not execute replay.

## Complete shared calibration and collateral map

| Existing shared owner | All direct callers and consequences |
| --- | --- |
| `explorationRanks` at 157 | Only `retentionCampaign:146`. Seed returns nil. Both exploration strategies use the strategy-keyed process-local map or execute live calibration. Preserve Load/Store, measured ordering, root-first/all-alternative assertions and cache lifetime. |
| `unorderedRetentionCampaign` at 187 | `retentionCampaign:145` and calibration at 166. No attachment belongs in either constructor. |
| `retentionCampaign` at 143 | This file at 272, 481, 513, 557, 678; `runner_test.go:2714`. These are resumed-executor construction, task 74's first policy table, the two selected capacity callers, interruption/resume setup and target-copy setup. |
| `resumeRetentionCampaign` at 270 | Only interruption/resume test at 720. Its executor-only resume at 274-277 retains real bootstrap behavior and receives no attachment. |
| Selected capacity local `run` at 511 | Count at 525, count-inside-round at 538, live bytes measurement at 552, each across all three strategies. The second selected call is the subsequent byte-bound campaign at 561. |
| Selected bounds local `run` at 885 | Two-success measurement at 901 and each existing row at 932 for 10/100 jobs. No retention calibration/cache dependency. |

Task 74 owns the calibration at 169 and first policy-table outer call at 488. Its shared setup admission already covers possible diagnostic-stage movement in capacity, interruption/resume and target-copy consumers. A successor must reobserve the unchanged first policy table including both completion orders and equality at 495, interruption/resume at 608, and `TestRunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns` (`runner_test.go:2709`). It must attribute new outcomes to the exact source and predecessor calibration. It cannot claim that this six-site batch restores the unmodified interruption/resume or target-copy outer calls.

`TestGuidedAdmissionReplaysBeforeTheCorpusAdvances` at 765 uses `testConfig` directly and has no retention-cache edge. Diagnostics' plan/shard/guidance, replay and resume tests at 87,114,160 remain excluded controls. `TestOpenReportsMinimizationLineageAndBounds` (`inspect_test.go:71`) remains excluded. No synthetic bootstrap may reach a real decoder/process or an unadmitted resume/replay/guidance path; an observed crossing or newly reached behavioral failure returns to root without a repair to assertions or data.

The unchanged helper's existing direct attachment sites are `runner_test.go:50,109,146,180,194,217,267,296,421,434,454,606,642,677,704,724,829,1112,1192`; `retention_test.go:107,139`; `completion_characterization_test.go:343,375,432`; `choice_exploration_divergence_test.go:69,144,251`; `seed_completion_characterization_test.go:213,257`; `runner_mode_unix_test.go:18`; and `preparation_dependencies_test.go:84,172,221`. This lexical caller inventory supports leaving the helper untouched; those callers gain no new behavior from six call-local assignments.

## Controls, documentation and limits

Retain the full assertions in `TestPreparationDependenciesForwardRealFixtureInputs`, `TestPreparationDependenciesOperationErrorsRemainUnchanged`, `TestPreparationDependenciesFailuresStopAtOriginalStages`, and `TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard` (`preparation_dependencies_test.go:31,135,165,205`). These cover real copied target/output inputs and closing, argument/result/error identity, original prepare/bootstrap failure stages and callback counts, and public/executor-only/prepare-only default guards. Also retain `TestInjectionCharacterizationIsolatedExploreRejectsEverySubstitution` and `TestInjectionCharacterizationIsolatedPreparationDependencies` (`executor_injection_characterization_test.go:151,168`) including zero callbacks/process starts, injection-before-invalid-seed rejection and missing-resume preflight precedence. The public profile control is in `tools/gomad3/deterministicio/profile_portable_test.go:162`, `TestPortableProfilePublicGuardsRemainFirst`; preserve all four guards and its existing qualified-host skip condition.

This task needs a bounded correction admission and evidence pointers for fn-109 R5/R18/R19, fn-109.63 and fn-112.10 source acceptance. Root may update task/milestone tracking to reflect the actual new owner and correct test names. README, SPEC and ARCHITECTURE need no product-contract or interface edit for six private test assignments. Preserve the existing README diagnostic retention, retention bounds and inspection contracts. The owner spec's format-compatibility waiver does not authorize assertion rewrites here; selective-removal reconstruction proves source scope only.

Exclude all new seams, profiles, identities, conditional dispatch, helper/cache/reset/instrumentation changes, fixture payloads, comments, imports, assertions, storage/public/default/runtime/schema/policy changes and additional attachments. The current admission must be rebound after task 74 source integration. Root must select the actual immediate frozen baseline, observe cold calibration and existing cache reuse without instrumentation, retain every actual terminal and diagnostic movement, and admit execution separately. This research ran no Go, test, build, lint, vet, compiler or generator command, repeated no terminal classification, and investigated no replay/minimize identity or native transport. Task 74 retains the exclusive execution lane. Native fn-128/fn-149 remain deferred and unverified; fn-155.1 retains separate first-platform proof. No task, lifecycle, Git, index, branch, native, CI, PR or push mutation occurred.

## Read binding

PRIMARY was `/Users/stephan/Workspace/skunkworks/gomad/temporal`. HEAD remained `305182879f727f0e925483c41b8b8f0679770621` from entry through the final source check. The following hashes were recorded during inspection and checked again before report writing; none moved. Initial working-tree inspection and final product diff showed no tracked product change. Hashes recorded during inspection are not a claim of a pre-first-read execution receipt.

| Input | SHA-256 |
| --- | --- |
| `AGENTS.md` | `8d634df5cbbbffd7dbada06e32b4d20d879707273f8be07bdf253b387211e6f3` |
| `MILESTONES.md` | `4ce9e9723be8162e9c579a6ccefdd3ac4169d2d58ecb126cae918912f62642ea` |
| `tools/gomad3/README.md` | `fb85ed4952fb925ca31768b516fa01285d73fa2738551d9781cd6264cda0f610` |
| Dirty fn-109 owner spec | `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c` |
| Task 74 markdown | `4bcec36f22eba4fe6a22785cc57a95c484b6cc987338f797d2cab9e1d3e646f2` |
| Task 74 `admission.md` | `f681dca0dd895098772a3f01c20076933c2ea475fb72548f912bc3014adf0e4f` |
| Task 74 `source-reconciliation-20261010.md` | `ce8a6cba43523c029594448de61f9863740829d7912c916aa9f63c8d56f76d0c` |
| Task 74 `execution-admission-20261010.md` | `dd49119b8d7ad8bc0d69d70a66b6adb38d913f276be5fe0e9f16d42e7151a709` |
| First-read fn-112.10 `runner-frontier-after-task73-20261010.md` | `0c16ae4b3ed06d6f5a27bc5c1161b56c5ecac98ead019b4999d8c4bb66e7a0cc` |
| `diagnostics_test.go` | `885b3df456c376522b484cfbdaa70e5f70ffe4d77a9c8af1d825740443606b68` |
| `retention_characterization_test.go` | `87253da780594e54139a2359ef5d2655ffbf4905b54eb1380c3700f3edaf8b4e` |
| `inspect_test.go` | `140e6688d3e1dddb8dbc1e85ae53de0de15fd676f655227efae6fa09282ca8df` |
| `preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `preparation_dependencies_test.go` | `c74d91de827cb171fb9565f697c6254f02b2091c69e51cbf916f76ce0921ecb9` |
| `preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |
| `runner.go` | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| `runner_local.go` | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |
| `runner_test.go` | `d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96` |
| `completion_characterization_test.go` | `bd9ad5807a84c484dab8ed3d5f1e2ed1863b8a985bb0e3841125271b697bcd82` |
| `executor_injection_characterization_test.go` | `b225b832599a605854e297b662c71d4c4e3684a903b40ccf6805c9ff7ef7ef0d` |
| `diagnostics.go` | `f07431ec3cdf4799c65d286b3fe72d93c0560885b734982ff3e05679722d0e29` |
| `inspect.go` | `94d69de99dbb7a30c9a307dfd4a2e51b35f24b49aaf0ae0d1da5f77766af7359` |
| `tools/gomad3/deterministicio/profile_portable_test.go` | `6dfd6c349c6c6a05995d5f9343541e0529ab7eb2e453607de3026c02dd4932c4` |

The complete README, MILESTONES, AGENTS, dirty owner spec, task/admissions and `/home/agent/.codex/docs/flow-next/prose.md` were read. The prose contract hashes to `b40a9f92c3e31af9b917e19df844cf6bf7c323a438a058eaec5debb1d152e42c`; its skill hashes to `2e6110252e7f4c374b99a2aaa465af621dc18b79487dec962b84db3a20cf748a`. These two prose hashes were recorded at the final check only. Read-only discovery initially guessed three nonexistent filenames and returned shell errors; corrected paths above were then read. No missing source was treated as equal. Requested research route was gpt-6-astra/high, with parent-reported JudgeTier session fallback `unavailable(no_key)` and unverified actual model telemetry. No tier retry occurred.
