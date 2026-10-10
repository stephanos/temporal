# Retention calibration and next scripted slice

The smallest source-supported retention-table candidate has **two syntactic preparation attachments** in `tools/gomad3/runner/retention_characterization_test.go`. Attach the existing scripted seam to the calibration campaign immediately before line 169 and to the first table's outer campaign immediately before line 488. This preserves the live platform-dependent ordering measurement, its cache, the original outer executor, all six policy rows, all three strategies, both completion orders and every assertion. It requires explicit admission of the shared calibration's consumers. This research authorizes no implementation and predicts no PASS count.

A helper-level attachment is possible without editing or substituting resume, replay or guidance execution. It is **not isolated from the setup of the resume characterization**. That test calls the same calibration through `retentionCampaign`. Root must admit and account for that shared setup effect if it selects the two-attachment candidate. If the scope instead requires zero effect on any resume consumer's setup, no existing shared-helper attachment satisfies it; do not disguise the dependency with cache seeding, helper cloning or new dispatch conditions.

The previous [remaining fixture survey](remaining-scripted-fixture-survey.md) already owns the one-call 10/100-job candidate at lines 885/891. It remains excluded here, as do its other ranked candidates. This report investigates only the previously unresolved retention-helper frontier.

## Source and evidence binding

All reads used PRIMARY `/Users/stephan/Workspace/skunkworks/gomad/temporal`, observed at HEAD `c4a3cc7e5b951003650af879943593e8f98c152c`. The requested research tier was `gpt-6-astra` at `high`; actual model/effort telemetry was not observed. The delegated research and prose skills were read directly. No further agent was spawned because this is already the delegated research pass.

The authoritative owner is `.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md`, SHA-256 `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. R5 at line 329 retains private fake-failure coverage and the usable preparation/replay seams. R18 at line 418 preserves shipped behavior, defaults, errors, transactions, comments and assertions. The opening amendment at lines 5-14 removes universal byte/format compatibility requirements, but does not authorize weakening these existing characterization assertions. R19 at line 425 still requires source-bound checks and separately owned qualification evidence. `MILESTONES.md:38` retains the immediate delivery order; lines 58-121 retain focused checks, serialized gates, root admission and fresh-worker ownership; lines 12-23 keep native fn-128/fn-149 work deferred and unverified.

The historical raw evidence is `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-69/ordinary-runner.log`. Its `run-binding.json` binds frozen execution HEAD `c668243e0ecab6e4080aa7dad0810ccc2cedb08f`, Linux/aarch64 and the same PRIMARY owner-spec hash. `ordinary-runner.json` records `go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner`, exit 1, elapsed 102.1600084239908 seconds. That command was historical and was not rerun here. Its timestamps are 2026-10-10; this report retains the recorded timestamps without relabeling the run as current.

Read-only JSON parsing found 673 named terminal events, comprising 394 PASS, 267 FAIL and 12 SKIP. Exactly 49 FAIL events belong to the five top-level tests in `retention_characterization_test.go`; another one belongs to its shared-helper consumer `TestRunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns`. The raw terminal names and line numbers appear below. No slash-derived intermediate event was synthesized.

| Retained evidence file under `combined-69/` | SHA-256 |
| --- | --- |
| `ordinary-runner.log` | `73ceada3c52c462a03f655c0be752b2c191d7a3925d0f30f624ef48aa0c438d9` |
| `run-binding.json` | `a9e9e28681a1db072ec390a1ee1ad2388d5e1c3118c937c9ed99e5d64b78da97` |
| `ordinary-runner.json` | `6fa4cd1da05b7d589abe568ab0ad179332f3e16df1c94e409807853a4aa41a36` |
| `source-before.json` and `source-after.json` | `406b032fddbc2289319419c13a4c9f7be33cd15491039e747ed12260412dee84` |

The current hashes of the following nine Runner files match their individual entries in that historical source manifest. This was checked by selecting the exact manifest entries with `jq` and passing their hashes to `sha256sum -c`. The historical before/after manifests also have equal hashes. This is equality for the listed source files, not a claim that the entire present checkout equals or is qualified by the historical checkout.

| File under `tools/gomad3/runner/` | Current SHA-256 |
| --- | --- |
| `retention_characterization_test.go` | `87253da780594e54139a2359ef5d2655ffbf4905b54eb1380c3700f3edaf8b4e` |
| `runner_test.go` | `d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96` |
| `completion_characterization_test.go` | `b865856c22c519d3b9af29b65cbc5cf0c72b288b5380875f6809801c7a794e3f` |
| `preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |
| `preparation_dependencies_test.go` | `c74d91de827cb171fb9565f697c6254f02b2091c69e51cbf916f76ce0921ecb9` |
| `executor_injection_characterization_test.go` | `b225b832599a605854e297b662c71d4c4e3684a903b40ccf6805c9ff7ef7ef0d` |
| `runner.go` | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| `runner_local.go` | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |

The earlier survey remains SHA-256 `25d2f660a363458d4954d319556d280600dce10d17278fd180c3da42d5c8b1a7`. `tools/gomad3/README.md` was read completely at SHA-256 `fb85ed4952fb925ca31768b516fa01285d73fa2738551d9781cd6264cda0f610`; `MILESTONES.md` was read completely at SHA-256 `91f2beff1e9a2e040a2efbcdad1e49eb16e071efaf14313ac56dfc3f72cf2544`.

## Calibration ownership and complete caller map

`tools/gomad3/runner/retention_characterization_test.go:143` first constructs the requested outer campaign through `unorderedRetentionCampaign` at line 145, then assigns `executor.order = explorationRanks(t, strategy)` at line 146. The constructor cannot return until that calibration succeeds. The `explorationRanks` seed branch at line 159 returns nil. Its exploration branches load the package-level `sync.Map` at line 162; a miss constructs a separate serial, successful campaign at line 166, retains novel successes at line 167, installs distinct rank probes at line 168 and executes at line 169.

The calibration reads real journal records at line 174, requires exactly one novel probe for each execution at line 175, maps the original probe's alternative to the committed selection ordinal at line 178, requires all three alternatives and root-first order at line 180, then stores the measured map at line 183. It does not derive the order from map iteration or a fixed ordering constant. Its identities bind the platform as the existing comment at lines 152-156 explains. Keep the declaration, key, Load/Store, probe list, journal derivation and assertions byte-for-byte unchanged for this proposed attachment-only scope.

Repository Go-source search found exactly these helper references. There are no other Go callers in `tools/gomad3`.

| Helper | Direct callers | Transitive consumers and effects |
| --- | --- | --- |
| `explorationRanks`, line 157 | `retentionCampaign:146` only | Shared calibration for all `retentionCampaign` callers below. Seed strategy performs no calibration. |
| `unorderedRetentionCampaign`, line 187 | `retentionCampaign:145`, `explorationRanks:166` | Builds both the outer campaign and the calibration campaign. Attaching here would silently give scripted preparation to every outer caller. |
| `retentionCampaign`, line 143 | This file lines 272, 481, 513, 557, 678; `runner_test.go:2714` | First retention table; both capacity call sites; interruption/resume table setup and resumed-executor construction; target-copy table. |
| `resumeRetentionCampaign`, line 270 | This file line 720 only | Builds a fresh ranked executor at line 272, discards the returned config/dependencies and invokes `exploreWith` with `ResumeCampaign` and executor-only dependencies at lines 274-277. |
| `explorationRankCache`, line 150 | Load at 162 and Store at 183 only | Existing process-local strategy cache. No reset, preload, replacement or new key is warranted. |

For the recommended candidate, the helper change can populate the cache where the old calibration failed. A later unmodified capacity or interruption test may therefore fail at its outer call/observation instead of inside calibration. That is a known shared setup effect, not restoration of that excluded test's contract. The resumed operation at lines 274-277 retains its existing real bootstrap dependency. The target-copy test at `runner_test.go:2709` also uses this setup, although the historical run stopped in its first seed campaign at line 2720 before it reached either exploration strategy. A helper edit cannot honestly be advertised as affecting only 19 named events.

## Ranked options and exact attachment sites

At each admitted site, the proposed statement is the same existing fixture operation, preserving the final outer executor from the current dependencies:

```go
configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
```

| Rank | Candidate | Exact insertion sites in original source | Total syntactic attachments | Boundary |
| --- | --- | --- | ---: | --- |
| 1 | First retention policy table plus calibration | `retention_characterization_test.go:169`, after `executor.shape = rankProbes(t, probes...)`; `:488`, after `executor.after = order` | 2 | One shared calibration call and one table-local outer call. Six policy rows, three strategies, both orders and every projection/assertion remain unchanged. |
| 2 | Capacity table plus calibration | Same helper `:169`; local `run` immediately before `:520`, after the optional reverse-order assignment; byte-bounded outer campaign immediately before `:561`, after its rank-probe assignment | 3 | Both capacity outer calls are required. The `bytes` row first measures through the local `run` at `:552`, then executes its bounded campaign at `:561`. |
| 3 | Both retention tables together | `:169`, `:488`, `:520`, `:561` | 4 | Union of ranks 1 and 2. Root may prefer two tasks; after rank 1 lands, rank 2 has two incremental attachments and must bind the new baseline. |

Rank 1 is the smallest candidate that restores the full first table's path from a cold cache without changing existing helper signatures, callers or assertions. A helper-only attachment leaves the outer campaign on real preparation. An outer-only attachment cannot return from the cold exploration calibration. A single constructor-level attachment would reach more callers, including the first phase of the interruption/resume table, and would be a broader admission despite its smaller textual diff.

Rank 2 retains the real byte measurement at lines 552-556, the byte limit derived from three successes, count and within-round expectations at lines 526-545, and both exact-limit and artifact-capacity branches at lines 564-573. None of those observations may be hard-coded to make a preparation correction pass.

The source also supports a separate two-attachment target-copy candidate at helper line 169 and `runner_test.go:2718`. It is not ranked ahead of the retention tables because it changes a second file and does not resolve their outer calls. It would preserve all three strategies, success/failure cases, committed-round assertions at lines 2722-2726 and the 16-artifact/one-inode assertion at lines 2734-2736. This is a possible later root choice, not part of the recommended scope.

If zero shared-helper impact is required, a seed-only conditional attachment to line 488 would reach only part of the existing table and introduce strategy-selective preparation behavior. That would not satisfy the complete table's stated characterization. No such conditional or helper clone is recommended. The separately researched 10/100-job closure is already the existing isolated-call alternative, and this report does not readmit or duplicate it.

## Why the calibrated executions remain scripted

`unorderedRetentionCampaign` creates one matching `newFakePreparer` at line 189 and one outer `*retentionExecutor` at line 191. `runner_test.go:1981` creates its real fixture target file with fixed bytes and identity. `fakePreparer.Prepare` at line 2012 copies that file into the requested prepared directory and chmods it. `testConfig` at line 2504 sets target kind/source/argv consistent with this preparer. The recommended helper and outer sites pass the original `config.Preparer` and `configDependencies.executor`; they do not replace either with the executor's base delegate.

`retentionExecutor.Run` at `retention_characterization_test.go:108` determines rank, preserves before/after/cancellation behavior, invokes its base at line 130 and adds the World record and shape at lines 132-134. Its seed base at line 195 is `fakeExecutor`, its choice base at line 210 is `explorationExecutor`, and its simulation base at line 219 is `simulationExplorationExecutor`. These bases are implemented at `runner_test.go:2332`, `:2154` and `:2196`. They inspect scripted environment/choice/simulation requests and construct result evidence; they neither launch the prepared file nor decode `request.IO.Config`. `completionWorldRecord` at `completion_characterization_test.go:111` uses the pure World model and retains its real recording encoding. The simulation plan JSON decoding at `retention_characterization_test.go:83` and `runner_test.go:2210` remains necessary fixture behavior; it is not a runtime bootstrap decoder.

The existing seam at `preparation_fixture_test.go:17` requires explicit preparer and executor, validates the request and prepared metadata, calls the original preparer at line 28 and real `Prepared.Verify` at line 35, and supplies only its explicit synthetic marker at line 41. Production still prepares the campaign at `runner_local.go:243`, sends the bootstrap bytes to the original executor at `runner.go:681`/`:712`, and validates observed choice evidence at `:717`. The seed completion path checks the real prepared file again at `runner_local.go:527`. The shared retention observer continues opening real artifacts and journals at `retention_characterization_test.go:294`/`:337`, checking exact-replay metadata, stored byte counts, ordinals, summary/journal agreement, failure references and canonical projections at lines 377-421. No artifact verifier, journal decoder, result decoder, publication path or filesystem is mocked by the proposed assignments.

## Explicit exclusions and unchanged guards

Do not attach preparation inside `retentionCampaign`, `unorderedRetentionCampaign`, `testConfig`, `newFakePreparer` or `resumeRetentionCampaign`. Do not change cache lifetime, rank derivation, the `completionStrategies` list, executor shapes, probe data, channel ordering, map contents, publication expectations, byte measurement, assertions or production code. Do not remove a failure branch after calibration makes it reachable.

`TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState` remains excluded from outer-call restoration. It changes transcript completeness at line 683, makes a real store temporarily unusable at lines 693-702, cancels at lines 704-710, checks committed state at lines 712-718, resumes at line 720 and checks resumed state at lines 721-725. An attachment at its initial line 710 would expose another operation boundary and would not restore the executor-only resume bootstrap. Its setup is an indirect calibration consumer and must be included in preservation observations, without claiming its original failure/resume assertions passed.

`TestGuidedAdmissionReplaysBeforeTheCorpusAdvances` has no retention helper/cache dependency. It sets Guide/Corpus/Replayer at lines 797-799 and runs at line 800. Its `replayRecorder.Replay` at line 743 opens a real artifact and observes corpus index timing at line 756. Even though that replayer is scripted, guidance/replay semantics and publication ordering need a separate explicit source admission. All five guided rows and their assertions stay outside this slice.

`TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs` remains the earlier survey's rank 3, not a new helper consumer. Its test-local run closure, initial measurement and later subtests remain outside this proposed change.

Preserve the default/public/private and failure controls in `preparation_dependencies_test.go:31`, `:135`, `:165` and `:205`. The real defaults in `preparation_dependencies.go:17`/`:24` remain lazy. Preserve the isolated substitution controls in `executor_injection_characterization_test.go:151` and `:168`; their callbacks must remain uncalled and coordinator execution rejected at the original stage. The raw combined-69 log actually emits PASS for these top-level names and their existing child events. Those are historical control results, not results for a future retention candidate.

The synthetic marker is never valid input to `deterministicio.DecodeBootstrapFrame` at `deterministicio/bootstrap.go:53`. Any newly discovered path that passes it to a real decoder, process executor, replay, resume or guidance operation invalidates this candidate's current scope and requires root reassessment.

## Actual historical events and diagnostic stages

The following is copied from parsed `Action:"fail"` records in the retained raw log. The leading integer is the raw JSONL line. All 49 retention-file events and the one additional shared-helper consumer are listed explicitly so a future comparison does not synthesize intermediate table names.

```text
1972 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/discard/seed
1977 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/discard/choice-exploration
1982 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/discard/simulation-exploration
1987 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/all/seed
1992 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/all/choice-exploration
1997 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/all/simulation-exploration
2002 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_probe_after_a_known_one/seed
2007 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_probe_after_a_known_one/choice-exploration
2012 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_probe_after_a_known_one/simulation-exploration
2017 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_probe_ahead_of_its_repeat/seed
2022 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_probe_ahead_of_its_repeat/choice-exploration
2027 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_probe_ahead_of_its_repeat/simulation-exploration
2032 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_choices/seed
2037 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_choices/choice-exploration
2042 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/novel_choices/simulation-exploration
2047 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/failures/seed
2052 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/failures/choice-exploration
2057 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/failures/simulation-exploration
2059 TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy
2066 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/count/seed
2071 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/count_inside_a_round/seed
2076 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/bytes/seed
2081 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/count/choice-exploration
2086 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/count_inside_a_round/choice-exploration
2091 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/bytes/choice-exploration
2096 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/count/simulation-exploration
2101 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/count_inside_a_round/simulation-exploration
2106 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy/bytes/simulation-exploration
2108 TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy
2115 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/incomplete_transcript_behind_a_kept_success/seed
2120 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/incomplete_transcript_behind_a_kept_success/choice-exploration
2125 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/incomplete_transcript_behind_a_kept_success/simulation-exploration
2130 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/incomplete_transcript_ahead_of_its_repeat/seed
2135 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/incomplete_transcript_ahead_of_its_repeat/choice-exploration
2140 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/incomplete_transcript_ahead_of_its_repeat/simulation-exploration
2145 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/unusable_success_store/seed
2150 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/unusable_success_store/choice-exploration
2155 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/unusable_success_store/simulation-exploration
2160 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/cancellation/seed
2165 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/cancellation/choice-exploration
2170 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/cancellation/simulation-exploration
2172 TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState
2179 TestGuidedAdmissionReplaysBeforeTheCorpusAdvances/exact_replay
2184 TestGuidedAdmissionReplaysBeforeTheCorpusAdvances/diverged_replay
2189 TestGuidedAdmissionReplaysBeforeTheCorpusAdvances/unverified_replay
2194 TestGuidedAdmissionReplaysBeforeTheCorpusAdvances/failed_replay
2199 TestGuidedAdmissionReplaysBeforeTheCorpusAdvances/incomplete_transcript
2201 TestGuidedAdmissionReplaysBeforeTheCorpusAdvances
2206 TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs
2753 TestRunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns
```

Counts are 19 for the first table, 10 for capacity, 13 for interruption/resume, 6 for guidance and 1 for bounds. The first table's seed diagnostic at raw line 1970 is `preparation.stageError{stage:"validation", ...}, want a host failure` from the observer at source line 489. Its exploration diagnostic at raw line 1975 reports `deterministic I/O requires one of darwin/arm64, linux/amd64; host is linux/arm64` from source line 481, before that outer constructor returns. Every first-table row has the same respective diagnostic stage; pointer addresses in `%#v` are not stable evidence.

Capacity seed rows report validation-stage errors at source call lines 525, 538 and 552; their exploration rows report the unsupported-host text at those same call sites because helper attribution hides the nested calibration location. The interruption/resume seed rows report validation-stage errors at source line 712 and exploration rows report unsupported-host text at line 678. Guided rows report unsupported-host text at line 806; the bounds fixture reports it at line 901; the target-copy test reports it at `runner_test.go:2720`. These are preparation-stage historical failures, not demonstrated retention-policy, resume or replay failures.

The calibration helper has no `t.Run` and therefore no independent historical terminal name. Its now-reachable work must be documented through its existing journal/order assertions and the real enclosing test event. Both proposed retention tables already emitted all their original child terminal names because their `t.Run` calls precede calibration. The second completion order, successful calibration records, capacity byte-bound run and later target-copy loop iterations are still newly reachable behavior even when they emit no new test name. The excluded 10/100-job test failed before its subtest loop, so its historical child event names do not exist and must not be invented. Any future candidate's actual newly emitted events must be captured and attributed to the unchanged source that becomes reachable.

## Limits and requirements for a later admission

This report establishes a source-supported boundary only. It has no changed-Go RED/GREEN evidence, no confirmed calibration success and no observed downstream retention result. The first table's policy/projection assertions, both capacity branches, cold-cache behavior on each exploration strategy, shared-cache reuse and excluded consumers' failure stages remain candidate-bound verification work. The shared map has no new reset or test hook. A fresh test process can exercise cold state without editing it; verification must not rely only on a prior test warming the cache.

A later admitted worker must preserve every original line other than the enumerated assignments, retain selective-removal proof for the whole edited file, execute unchanged original assertions and retain the public/default/bootstrap/failure/isolated controls. Root must account for all shared helper consumers in the outcome comparison, including unchanged failures whose diagnostic stage moves. Unexpected downstream failures require an explicit root scope decision, not changed expectations, broader helpers or product fixes inside this slice. Applicable lint, architecture, generated-input checks, both-source-set static checks and independent integrated source review remain owned requirements under the current spec and milestones; this research ran none of them.

Only this research artifact was authored. Shell reads used `login:false`, `env -u BASH_ENV bash -c` and an explicit PRIMARY `cd`. Git was used only to read HEAD with optional locks disabled. Flow usage and one bounded brief were read without lifecycle changes. Local source and retained raw evidence were sufficient; no external research, Go/env/tool probe, build, lint, vet, generator, timeout/native test, active worker workspace access, source edit, task minting, commit, CI, PR or push occurred. Neither the historical ordinary gate nor this report supplies a supported-native full-host pass, soak bound, formal SHIP or Done claim.
