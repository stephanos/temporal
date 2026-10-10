# Seed completion preparation slice

Source inspection supports a bounded correction with exactly two call-local assignments in `tools/gomad3/runner/seed_completion_characterization_test.go`. Both existing tables use scripted executors that never decode bootstrap bytes or execute the prepared target. Their fourteen leaves retain meaningful success, failure, cancellation, integrity and controller-statistics assertions. The correction is viable without changing helpers, production code, executors, assertions, fixture data, public defaults or runtime behavior. This is source-only feasibility evidence. It establishes no changed-source result or future pass count.

Root owns selection, admission, task creation, implementation dispatch, lifecycle, integration, review and commits. This report is the only authored file. Research used repository sources, retained logs, read-only Git comparison and hashes; it ran no Go, build, lint, vet, generator, toolchain, real-process or native tests. It follows the research workflow and Flow prose contract.

## Bound source and retained observation

The inspected PRIMARY is `/Users/stephan/Workspace/skunkworks/gomad/temporal` at HEAD `87d1927a90295dc267c33d6226a643a74c425f51`. The full current fn-109 spec hashes to `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. Its opening amendment removes format/byte compatibility obligations while preserving behavior, error precedence, transactions, resource lifetimes and existing comments. R5 at `.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md:329` preserves private fake failure coverage and usable public preparation/replay seams; R18/R19 at lines 418/425 preserve assertions and require source-bound verification. This proposal changes no existing expectation or datum.

The prior `completion-next-slice.md` report, SHA-256 `d84f4a9eb3185355ee75a21eb5edf87c2aa168abfccd21f9260e6d783a5ccb61`, excludes this file because shared `completionCampaign` changes would broaden its three-call slice. Task71 still limits its source scope to `completion_characterization_test.go` and three named consumers in `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.71.md:7`. The present report traces only the seed consumers and their dependencies. It does not enlarge task71 or duplicate its strategy survey. Tasks69/70/71 are already admitted owners; the dispatch records task69's isolated verified product pending commit and tasks70/71 TODO. None of that unintegrated work supplies a revised PRIMARY baseline or aggregate pass count.

All evidence paths in the following table are relative to `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-68/`.

| Evidence | SHA-256 | Meaning |
| --- | --- | --- |
| `ordinary-runner.log` | `3b75cacdac2bcc7016dc5f7a99f924f6b958f0cbc7d58e59afe77980b7b34b1e` | Actual terminal events and preparation failures |
| `run-binding.json` | `50780dd40849c42621f0f1d707132fb44b022b1aaf7fb9846da2315b79483d28` | Frozen execution HEAD `524c092a3f6cbf5834895ae5ba7821d6fa610924`, developmental Linux/aarch64, exact owner-spec hash |
| `ordinary-runner.json` | `46dd5ad24dd65e838d13d430a959dc28954d184de75b27d0ede796b871f80ec0` | Actual `go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner`, exit 1, 20.411523510003462 seconds |
| `source-before.json` and `source-after.json` | `f76076a82761a2a250eda3e97bf23c51d1c1520a5a53023b2c0897a31f756314` | Identical 1,230-path execution input manifests |
| `outcome-comparison.json` | `487dbedd092007fa33118bfccffe131b8afc42d1c7161ccd02f12a7c0cb8f0cd` | Frozen 673 named outcomes, 392 PASS, 269 FAIL, 12 SKIP |
| `postcapture-seal.json` | `c49a7baecc4435cadf0c702ea6bee06a1608b8245d2de52fe98086d207245f08` | Twenty explicitly sealed output members; postcapture evidence, not retroactive execution binding |

Read-only parsing of the raw terminal actions reproduced the 392/269/12 counts. `checkpoint-note.md` maps the isolated frozen task68 checkpoint to PRIMARY integration `c506713ce063759c5d24129d775d2fefc6314618`. A read-only Git diff against the frozen revision found no changes in the seven source files below. Their current hashes also match the retained manifest entries. This establishes unchanged inputs for this bounded trace, not whole-PRIMARY equivalence or qualification.

| Current source relative to `tools/gomad3/runner/` | SHA-256 |
| --- | --- |
| `seed_completion_characterization_test.go` | `b35b3ec2d623c5746cc58e8996ff033b8e584813e6308f5fe7e0a03fde76e2b6` |
| `preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |
| `completion_characterization_test.go` | `b865856c22c519d3b9af29b65cbc5cf0c72b288b5380875f6809801c7a794e3f` |
| `runner_test.go` | `d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96` |
| `runner.go` | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| `runner_local.go` | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |

## Exact eligible calls

Insert the same statement immediately before the existing `exploreWith` at current lines 213 and 256, after each configuration function has returned its final preparer, executor, limits and cancellation callback.

```go
config.dependencies = scriptedPreparationDependencies(t, config.Preparer, config.dependencies.executor)
```

The first placement follows `ctx, config := test.configure(t)` at line 212 in `TestSeedCompletionKeepsCampaignStatistics`. It covers ten rows. The second follows `injectedCompletionCampaign(...)` at line 255 in `TestSeedCompletionFaultsKeepCampaignStatistics`. It covers four rows. Keep `config.dependencies.executor` exactly as constructed, including each pointer's shared state. Do not substitute a generic fake, unwrap `faultExecutor.base`, or move either assignment into `injectedTestConfig`, `injectedCompletionCampaign`, `testConfig` or `completionCampaign`.

Only these two insertions are needed by the inspected source. A whole-file selective-removal proof is feasible: remove exactly these two admitted statements from the eventual candidate and compare the complete file to its recorded pre-edit bytes/hash. All helper definitions, imports, comments, table values, executors, calls and assertions can remain byte-identical. This research produced no changed Go file and did not execute that future candidate proof. If reaching the original assertions exposes another defect, retain the failure and return it to root for separate admission; this proposal authorizes no repair beyond the two assignments.

## Actual failed terminal names

Every name below is copied from an `Action:"fail"` event in the bound `ordinary-runner.log`. The line column identifies that terminal event. These are fourteen leaf executions and two failed parents. No intermediate slash name or additional test outcome is synthesized.

| Exact raw terminal name | Log line |
| --- | ---: |
| `TestSeedCompletionKeepsCampaignStatistics/successes` | 2768 |
| `TestSeedCompletionKeepsCampaignStatistics/distinct_and_duplicate_failures` | 2773 |
| `TestSeedCompletionKeepsCampaignStatistics/first_failure_cancels_active` | 2778 |
| `TestSeedCompletionKeepsCampaignStatistics/failure_budget` | 2783 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure` | 2788 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure_drains_a_cancelled_attempt` | 2793 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure_drains_a_success` | 2798 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure_drains_a_failure` | 2803 |
| `TestSeedCompletionKeepsCampaignStatistics/prepared_target_integrity` | 2808 |
| `TestSeedCompletionKeepsCampaignStatistics/campaign_cancelled_while_running` | 2813 |
| `TestSeedCompletionKeepsCampaignStatistics` | 2815 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/malformed_World` | 2822 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/supervision_rejected_the_choice_trace` | 2827 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/watchdog` | 2832 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/cancelled_execution` | 2837 |
| `TestSeedCompletionFaultsKeepCampaignStatistics` | 2839 |

Each leaf's preceding diagnostic reports `preparation.stageError{stage:"validation", ...}` where `observeCompletion` requires a HostError. The first table's diagnostics occur at raw lines 2766 through 2811, every five lines; the second table's diagnostics occur at 2820, 2825, 2830 and 2835. Those diagnostics establish preparation RED before the original comparisons at source lines 215 and 257. The printed pointer values do not identify the inner error. The source trace supplies the explanation. `internal/preparation/preparation.go:97` invokes the supplied preparer, then its unchanged default validation at 102 wraps a validation failure. `deterministicio/profile.go:293` invokes profile validation before target-shape checks, and line 283 owns unsupported-host refusal. No newly reachable downstream assertion has been executed by this research.

## Preparation, execution and assertions

`seed_completion_characterization_test.go:20` and `:25` only wrap the existing configuration/dependency pair. `runner_test.go:2504` returns a matching `KindGoRun`/source `.` request with the supplied preparer and executor. `newFakePreparer` at `runner_test.go:1981` creates a real 0500 target file with matching size/hash, argv `gomad3-target`, fixed build key and the default profile's target metadata. Its `Prepare` at line 2012 copies the actual bytes into the requested preparation root and returns that copied path. None of the selected rows changes target kind, argv, source, adapters or build metadata.

`preparation_fixture_test.go:17` requires explicit nonnil preparer/executor values, checks request relationships, invokes that preparer's real copy at line 28, verifies the returned file at line 35 and supplies an empty adapter list. `target/target.go:305` verifies compatibility and hashes the real prepared file, rejecting size/hash changes at line 317. The helper preserves the supplied executor unchanged at line 23. Its bootstrap at line 41 returns the explicitly synthetic marker. This is suitable because the selected executors never inspect `request.IO.Config`.

`runner_local.go:250` invokes the chosen private preparation operation after setting the journal-owned preparation path. `runner_local.go:84` chooses the existing injected executor; `runner.go:681` obtains bootstrap bytes and copies them into the execution request at 698 before calling that executor at 712. Actual seed scheduling, completion ordering, journals, artifact publication and the controller remain in production owners. `runner_local.go:355` orders completions; `:381` applies `controller.Complete` and synchronizes the statistics observed by the tests.

| Fixture path | Executor and unchanged behavioral observations |
| --- | --- |
| First table's success, distinct/duplicate and budget rows, source lines 86-141 | `fakeExecutor.Run` at `runner_test.go:2332` reads the seed environment and returns the row's existing result. Preserve counts, deduplicated artifact count, ordered journal entries, budget stopping and stop reasons. |
| First failure cancels active, lines 115-122 | `newFirstFailureExecutor(3)` retains its start barrier and shared pointer state. `Run` at `runner_test.go:2484` waits for all three admitted executions, fails seed 1 and returns cancellation for siblings. Preserve three attempts, one failure, two cancellations, one distinct failure and both partial records. |
| Four supervision rows, lines 143-185 | `scriptedSeedExecutor.Run` at source line 55 dispatches solely by seed. Keep the original `supervisionErr`, seed scripts and `cancelledByCampaign`. Their assertions distinguish unclassified attempts, drained cancellation/success/failure, artifact deduplication, reason/cause and partial recovery state. |
| Prepared target integrity, lines 188-195 | Keep `mutatingExecutor{}` as the final executor. `runner_test.go:2397` chmods and appends to `request.Command`, which is the real copied prepared path. `runner_local.go:527` then calls `Prepared.Verify` and records the existing `prepared_target_integrity` failure with one attempted execution. Replacing this executor or the copy/verify path would invalidate the negative control. |
| Campaign cancellation, lines 198-208 | Keep `blockingExecutor{}`, the 10 ms grace and `cancelOnProgress` callback configured before attachment. The executor at `runner_test.go:2417` waits on context; the callback at `:2514` cancels when Running equals 1. Preserve the original cancellation cause, one attempted/unclassified execution and starting/recoverable-failure partial states. |
| Fault statistics table, lines 225-259 | Keep the outer `faultExecutor` returned through `injectedCompletionCampaign`. `completion_characterization_test.go:56` calls its seed fake base, supplies the semantic transcript, applies the existing fault and returns its existing error. The base at line 78 supplies the complete choice trace. Unwrapping the base would remove all four intended fault observations. |

The second table deliberately corrupts a real World recording, supplies `execution.ErrChoiceTraceMalformed`, and marks watchdog/cancellation results while clearing their terminal choice traces. `runner.go:716` preserves the watchdog/cancellation exception to choice validation. `runner_local.go:508`, `:566` and `:579` preserve supervision, World and assessment precedence. `completion.go:25` decodes World evidence; `:49` decodes semantic coverage before projecting choices and classifying outcomes. All inputs and expected statistics remain unchanged.

The first table compares the complete `seedCompletionObservation` at line 215. Its `Completion` field includes reason/cause, five counters, opened artifact metadata, actual journal entries and partial states through `completion_characterization_test.go:128`; its `Statistics` field at source line 42 includes attempted, succeeded, failure, watchdog, replay-divergence, cancellation, distinct-failure and stop counters. The second table compares every `CampaignStatistics` field at line 257. It also calls `observeCompletion`, retaining its error-type, artifact-open and journal-read checks, but it does not compare that observation's other fields against expected values. This narrower second-table assertion should remain accurately described.

## Exclusions and remaining proof

No selected leaf launches a target, decodes the synthetic bootstrap, calls resume/replay/minimize, opens a guided corpus or enters an isolated coordinator. Artifact inspection and partial-journal observation in these tests are source-level publication checks; they do not establish artifact replay or crash-resume execution. The public decoder at `deterministicio/bootstrap.go:53` delegates to `deterministicio/internal/wire/wire_generated.go:61`, which checks frame length, magic, version, kind and checksum. The helper's marker cannot satisfy that contract. Any real-process, decoder, resume identity, replay/minimize, adapter/toolchain or native consumer therefore remains excluded. No new fixture interface or decoder exception is justified.

The focused controls should retain `TestPreparationDependenciesForwardRealFixtureInputs`, `TestPreparationDependenciesOperationErrorsRemainUnchanged`, `TestPreparationDependenciesFailuresStopAtOriginalStages` and `TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard` at `runner/preparation_dependencies_test.go:31`, `:135`, `:165` and `:205`; `TestInjectionCharacterizationIsolatedPreparationDependencies` at `runner/executor_injection_characterization_test.go:168`; and `TestPortableProfilePublicGuardsRemainFirst` at `deterministicio/profile_portable_test.go:162`. The default/bootstrap control distinguishes public, executor-only and prepare-only refusal. Public `Explore` at `runner.go:373` still supplies empty dependencies, and the isolated path at 383 still rejects injection. The proposed assignments affect none of these guards.

An admitted worker still needs candidate-bound unchanged RED, the original assertions after the two assignments, complete selective-removal proof, preserved helper/default/error/isolated/public controls, required affected standards and boundary checks, and independent integrated source review. Root's future ordinary comparison must use the actual integrated candidate and actual emitted terminal names. The frozen 392/269/12 counts and RED aggregate gate remain historical facts; no successor aggregate result is predicted. A failure beyond preparation would require separate root admission rather than changed table expectations. Applicable R5/R18/R19, ordinary source, lint, preservation, both-source-set and generated-validation requirements remain with their owners. Native qualification and soak evidence remain deferred and unverified under fn-128/fn-149, with no CI, PR or push authority.
