# Task 74 retention deadline diagnosis

The ordinary candidate run reached failure-artifact publication and observed its campaign deadline after a successful artifact rename. The retained evidence locates the error boundary but does not establish why the campaign consumed its 10-second budget. Task 74's ordinary retention parent and `failures/choice-exploration` leaf remain failed. Passing focused runs cannot replace that result.

This investigation read the frozen candidate at `.worktrees/fn-109-74-retention-candidate` against base `305182879f727f0e925483c41b8b8f0679770621`. It ran no Go command, build, test, lint, generator, process termination, or Flow mutation. The only write is this report. It makes no native qualification or completion claim. Drafting follows `/home/agent/.codex/docs/flow-next/prose.md`.

## Proven mechanism

1. `runner/retention_characterization_test.go:489` installs the explicit scripted dependencies before `exploreWith(context.Background(), ...)`. Its `retentionCampaign` ultimately uses `testConfig` in `runner/runner_test.go:2504`, which supplies `OverallTimeout: 10 * time.Second` and `ExecutionTimeout: time.Second`. The retention policy table does not override those bounds. Each completion-order iteration constructs a fresh campaign, artifact root, executor and preparer.
2. `runner/runner.go:377` selects the local path. `runner/runner_local.go:62` creates a fresh `context.WithTimeout` before opening and preparing the campaign. That budget includes campaign setup, preparation, execution, publication and commits. The parent is `context.Background`, so an earlier package test cannot directly consume this campaign's deadline. The package wrapper's 600-second wall bound did not expire.
3. `runChoiceExplorationLocal` receives that context. `executeExplorationRound` creates a child cancellation context for the executors, then returns the completed round. `processExplorationCompletion` receives the campaign context, not the cancelled execution-round child. The artifact error therefore does not result from the execution helper's deferred round cancellation.
4. `runner/choice_exploration_campaign.go:396` calls `publishBoundedFailureArtifact` with the campaign context. `runner/runner.go:1082` installs it in `artifact.Store.Context`. `artifact.PublishArtifact` builds payloads and calls `Store.PublishArtifact`.
5. `artifact/store.go:170` checks the context before the rename loop. The reported `sync artifact store` prefix comes only from line 228, after `renameNoReplace(staging, finalPath)` succeeds. `syncDirectoryContext` at line 496 checks `ctx.Err()` before opening/syncing the directory, performs synchronous `directory.Sync()`, and checks `ctx.Err()` again if that operation succeeded. This message can mean the deadline became observable between the rename-loop check and the directory-sync entry, or during directory open/sync/close. It does not prove that the fsync syscall returned an error or took ten seconds.
6. Choice Exploration clones its summary for a round and installs it only after round commit. The observed counts `[1,0,1,1,0]` and sole ordinal-0 journal record show that round 0 committed and round 1 did not. They do not mean only one candidate executed. The second round's execution collection completed before this publication path. The log does not identify whether publication of ordinal 1 or ordinal 2 failed, or whether the nil-order or reverse-order campaign failed.

The store's deferred cleanup removes the old staging pathname after the rename, so it cannot undo the renamed artifact. This is still inside the uncommitted exploration round. The failing assertion prevents the fixed-identity projection log, and test temporary directories are cleaned up; the supplied evidence contains no surviving failed campaign journal or partial artifact tree to resolve the candidate further.

## Recorded comparison

All times below are the test JSON's leaf or parent elapsed values, not per-campaign measurements.

| Run | Table parent | Failure / choice leaf | Failure / simulation leaf | Result |
| --- | --- | --- | --- | --- |
| Before ordinary | Baseline table fails | 0.06s | Baseline table fails | Choice leaf fails preparation at the unsupported linux/arm64 boundary |
| After cold choice selection | 19.48s | 3.48s | Not selected | Six choice leaves and parent pass |
| After full table selection | 44.48s | 3.17s | 3.45s | 18 leaves and parent pass |
| After ordinary package | 56.74s | 12.50s | 5.67s | Failure / choice leaf and parent fail |

The failing leaf started at `2026-10-10T10:25:22.876018941Z`, emitted the assertion at `10:25:35.342474655Z`, and ended at `10:25:35.379674655Z`. Its message is `artifact_publication` / `sync artifact store: context deadline exceeded`. The previous leaf, failure / seed, passed in 2.46 seconds. Most earlier table leaves have comparable times in the ordinary and focused-table runs. This supports investigating a localized delay but does not identify its cause.

The current ordinary raw report has SHA-256 `4e160d741c02354081e88511d9c5204acfa3eca663d6f5fa373bccfdb325adb5`. Its 673 terminal named results contain 515 passes, 146 failures and 12 skips. The two source assignments let 17 selected names change from fail to pass; the remaining selected leaf and parent have the publication-deadline failure above, replacing their baseline preparation failure.

The after-ordinary, after-table and after-cold-choice binding files have equal source manifests, tool manifests, selected environment and inherited-environment value hashes. All three record source and tool stability across their command. They use the same Go/cache/temp settings and differ in the test-selection argument and run time. The binding explicitly leaves shared caches and unselected inherited environment nonhermetic; hashes do not measure resource contention or in-process state.

`TestMain` dispatches private modes and otherwise calls `m.Run()`. The Runner test sources contain no `t.Parallel()` call, and the ordinary JSON has no `pause` or `cont` events. The policy table's subtests execute serially, while each campaign can execute its two second-round candidates concurrently. Earlier package tests still differ from a focused run in process history, allocations, goroutine cleanup and filesystem activity.

The fixture's identified shared state is `explorationRankCache`. It only stores and loads each strategy's rank map. All three candidate selections enter their choice calibration at the first discard/choice leaf and reuse the map for the later failure leaf. The other fixture objects and artifact roots are created afresh. No cache mutation or altered timeout was found that explains the failed leaf. An exhaustive proof of absence of leaked process state is outside this read-only trace.

## Hypotheses and missing observations

- A slow filesystem operation, scheduler delay or host resource contention could consume the campaign budget. The error boundary and elapsed time are compatible with these explanations, but the retained logs contain no syscall duration, scheduling trace or contemporaneous resource sample.
- An earlier package test could leave activity or state that affects later timings. The matching environment bindings and fresh campaign ownership narrow that possibility; they do not rule out a goroutine/resource leak. There is no direct evidence of one here.
- A stalled reverse-order executor is a weaker explanation for this particular error. The round completed execution collection before artifact publication. The log still does not identify the failing completion-order iteration.

The worker's later observation of an external `make lint-code-fast` belongs to the later static gate. It is not evidence of overlapping load during `10:25:22Z` through `10:25:35Z`. No diagnosis in this report labels the timeout flaky or environmental.

## One proposed observation

After the worker's current shared lane is free, root may select one observational run with the same frozen source, environment, working directory and original ordinary command, adding only Go's execution-trace flag with a new evidence path. For example, the sole command change is `-trace=/absolute/new-evidence-path/ordinary.trace` in `go test -tags test_dep -count=1 -json ./runner`.

Retain the original failing log and all new raw outputs and bindings. The worker can inspect the trace with its pinned Go trace tool for syscall intervals and goroutine scheduling stacks under `artifact.syncDirectoryContext`, `os.(*File).Sync`, publication and campaign setup. `strace` is absent on the inspected PATH, so this proposal needs no tool installation or source instrumentation. The intended observation is where the ten-second budget was spent, not a replacement pass. A trace adds measurement overhead; a new failure may be perturbed, and a pass leaves the original cause unresolved. Missing context-creation timestamps and per-iteration labels may still prevent precise attribution. No deadline, assertion, adapter, strategy bound or test selection changes are proposed.

## Evidence bindings

Paths in the following table are relative to the candidate's `.flow/tmp/fn10974-evidence/`. The raw logs, receipts and matching binding files were hashed before substantive inspection and again before writing this report; both observations agreed.

| File | SHA-256 |
| --- | --- |
| before-ordinary.log | `143bf62334d70cc916a9af13149df8cf618326097665de2df3c64c4051a98e82` |
| before-ordinary.json | `a7d1f66d62fdb98c4e2464f6e88fd492de19c5af7b0b9669655a965c0201e310` |
| before-ordinary-binding.json | `509ba275803db75d239a1c38f0ce6b38c1590efb9e15b9db4422ec42c2315ae9` |
| after-ordinary.log | `4e160d741c02354081e88511d9c5204acfa3eca663d6f5fa373bccfdb325adb5` |
| after-ordinary.json | `31e9c217d8a24d20f6933e80a1b404368982dbec639ccfe8068cfb9443e41ced` |
| after-ordinary-binding.json | `0167e7f98d6f19f73b55dd7b813249c209e00c32fee2372d82b8e6b294572d80` |
| after-table.log | `5a7aeb4e5422d049312e34886ae21bc983ec907f4071803992aedd793c49c10f` |
| after-table.json | `64326dd1e823807cdef69942023355b7b855e0caf009ee06a96ef705cc2b0c6c` |
| after-table-binding.json | `9523963b313b3e9159a6625d5b402f726290ffddee865a2b986b54f24f432be7` |
| after-cold-choice-exploration.log | `d95e9cf0393362efda090208e0307890e21445fa1d3431b0b73c1068d91d68f6` |
| after-cold-choice-exploration.json | `37407657d454cee0386dd7a5cb6070092f79db044a7057742e448710dd0276d3` |
| after-cold-choice-exploration-binding.json | `ec095216940e237a47f42aa7557b0654a1229ce62e28429206e02fca6b52e514` |

For the source files below, the ordinary binding's pre-command hash equals the receipt's post-command hash and the live digest read during this investigation. These are retained gate before/after bindings, not a claim that this investigation created a new gate. Paths are relative to `tools/gomad3/`.

| Source | SHA-256 |
| --- | --- |
| runner/retention_characterization_test.go | `c86e26d2c016af55427faea8927c9b224a89778b4b5d70d81dc4257a8163f168` |
| runner/runner_test.go | `d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96` |
| runner/runner_local.go | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |
| runner/choice_exploration_campaign.go | `5744948588413558345e30b479a906baf74d87e3c33c171d15dffd7001ee88fa` |
| runner/preparation_fixture_test.go | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| runner/coordinator_transport_test.go | `1494b84cbfad58a395ad11904f2d4be26d23530e25062055271ba7e4d0201f21` |
| runner/runner.go | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| artifact/store.go | `f93601eccc225ab158f3c6f922b95e85deb4358de34cd7ff889b7d5533acddab` |
| artifact/publication.go | `44d9d007b0b09397654e62e24596f8321e14366a6797a6f31fc63ec1fb827671` |
