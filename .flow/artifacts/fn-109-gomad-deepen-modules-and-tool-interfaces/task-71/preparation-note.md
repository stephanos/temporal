# Task 71 gate preparation

This is orchestration preparation only. Root selects the implementation worker, its isolated checkout and candidate, grants the serial lane, and owns integration, independent review, commits and Flow lifecycle. No Go, build, test, lint, vet, generator, toolchain or native command was run for this note. The sole authored path is this file. Preparation started in the primary checkout at `176a269e651c10f7e88bb8dc11fc03ef78bf2e57`; root integration advanced HEAD to `c506713ce063759c5d24129d775d2fefc6314618` during read-only inspection. These observations are not frozen gate evidence. The implementation worker must bind its actual candidate before execution.

Read `AGENTS.md`, `MILESTONES.md`, `tools/gomad3/README.md`, task 71 JSON/spec, [admission.md](admission.md) and [completion-next-slice.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/completion-next-slice.md). The research file's SHA-256 is `d84f4a9eb3185355ee75a21eb5edf87c2aa168abfccd21f9260e6d783a5ccb61`. Admission BASE is `2af15fcbc0052764f9804fd005dfe298189ef8c7`. Current `runner/completion_characterization_test.go` remains SHA-256 `b865856c22c519d3b9af29b65cbc5cf0c72b288b5380875f6809801c7a794e3f`, matching the researched and frozen combined66/67 file.

## Exact implementation boundary

Add exactly three existing statements, one immediately before each selected `exploreWith` call:

```go
configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
```

| Consumer | Placement in the unchanged file | Preserved behavior |
| --- | --- | --- |
| `TestCompletionFaultsKeepReasonPrecedenceAndEvidence` | After `completionCampaign` at line 342, before line 343 | Final preparer and outer fault executor; all seventeen faults, three strategies, classification/error precedence, seed statistics, counters, artifact/journal and partial-state assertions |
| `TestCancellationIsAHostFailure` | After both strategy branches, the final `blockingExecutor{}` replacement, progress cancellation registration and `TerminateGrace` setting; before line 374 | Blocking executor for every strategy, cancellation identity and precedence, zero target failures/artifacts, real seed resume-plan/partial inspection and existing exploration partial-state normalization |
| `TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy` | After `CollectExecutionEvidence = true`, before line 430 | Captured outer executor and mutex/last-result state, real World/semantic/choice/simulation assessment, campaign/round commits, journal inspection and existing diagnostic logging |

Removing only these three syntactic assignments must recover the entire BASE file byte-for-byte. Keep every existing body, assertion, helper, import, comment, payload and datum unchanged, including `completionCampaign`, `testConfig`, `preparation_fixture_test.go` and `seed_completion_characterization_test.go`. The marker bootstrap is for the scripted fixture; it does not satisfy the real bootstrap decoder or authorize process/runtime changes. Retain any newly reached original failure and return it to root for separate admission; it grants no authority to adjust assertions or fixtures.

## Actual frozen terminal inventory

[combined66/67 ordinary-runner.log](../combined-66-67/ordinary-runner.log), SHA-256 `f97441bc0b8f4ebc4da6673b0e3b097a9420fe6e5bd1c1871de83c6f333eb032`, contains Go JSON terminal events. [run-binding.json](../combined-66-67/run-binding.json), SHA-256 `45f31ddfd586858fda9d0b9afb76e421a0f989c659957ba0727fe23f55073bb5`, binds developmental Linux/aarch64 HEAD `f699252450b8e67f1edb50ed8e4cff4cb6e644c0`. The whole retained run has 673 named outcomes: 389 PASS, 272 FAIL, 12 SKIP; configured original-base lint retains RED50. These are historical observations only.

Filtering actual `pass`/`fail`/`skip` events by the three exact consumer names and descendants yields the following sixty emitted names, all FAIL. There are three parents and fifty-seven leaf executions: 51 fault leaves, three cancellation leaves and three projection leaves. No intermediate fault-row slash name emitted a terminal event. The inventory below is copied from terminal events, not synthesized from source rows.

```text
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_World/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_World/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_World/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/World_seed_mismatch/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/World_seed_mismatch/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/World_seed_mismatch/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_semantic_coverage/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_semantic_coverage/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_semantic_coverage/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_choice_trace/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_choice_trace/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_choice_trace/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/choice_trace_rejected_by_supervision/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/choice_trace_rejected_by_supervision/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/choice_trace_rejected_by_supervision/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/unterminated_choice_trace_rejected_by_supervision/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/unterminated_choice_trace_rejected_by_supervision/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/unterminated_choice_trace_rejected_by_supervision/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/missing_terminal_choice_frame/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/missing_terminal_choice_frame/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/missing_terminal_choice_frame/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_World,_semantic_coverage_and_choice_trace/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_World,_semantic_coverage_and_choice_trace/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_World,_semantic_coverage_and_choice_trace/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_semantic_coverage_and_choice_trace/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_semantic_coverage_and_choice_trace/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/malformed_semantic_coverage_and_choice_trace/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_World/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_World/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_World/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_semantic_coverage/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_semantic_coverage/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_semantic_coverage/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_choice_trace/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_choice_trace/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/watchdog_and_malformed_choice_trace/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution_and_malformed_World/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution_and_malformed_World/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution_and_malformed_World/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution_and_malformed_semantic_coverage/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution_and_malformed_semantic_coverage/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/cancelled_execution_and_malformed_semantic_coverage/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/supervision_failure_and_World_seed_mismatch/seed
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/supervision_failure_and_World_seed_mismatch/choice-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence/supervision_failure_and_World_seed_mismatch/simulation-exploration
TestCompletionFaultsKeepReasonPrecedenceAndEvidence
TestCancellationIsAHostFailure/seed
TestCancellationIsAHostFailure/choice-exploration
TestCancellationIsAHostFailure/simulation-exploration
TestCancellationIsAHostFailure
TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy/seed
TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy/choice-exploration
TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy/simulation-exploration
TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy
```

The research locates preparation-stage refusal before completion observations, including unsupported `linux/arm64` profile validation. Root must retain a new meaningful unchanged-source RED on the implementation worker's exact bound inputs before editing. Missing tools, compilation errors or timeout do not establish that RED. No successor PASS total is forecast.

## Proposed focused commands and unchanged controls

These are proposed commands only, requiring a root-granted exclusive lane. Run them from the root-selected isolated checkout with the existing stock Go 1.27.1 executable `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`, pinned and hashed before execution. Rebind its actual version/platform/settings and environment; an unavailable or different tool requires root reconciliation. Always use `test_dep` and `-count=1`. A 600-second external TERM watchdog with a 15-second kill grace surrounds each command; the bound is wall time and does not rely on logical Go test timeouts.

Exact unchanged-source RED selector:

```sh
task71_selected='^(TestCompletionFaultsKeepReasonPrecedenceAndEvidence|TestCancellationIsAHostFailure|TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy)$'
timeout --signal=TERM --kill-after=15s 600s /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner -run "$task71_selected"
```

Exact focused selector before and after attachment:

```sh
task71_focused='^(TestCompletionFaultsKeepReasonPrecedenceAndEvidence|TestCancellationIsAHostFailure|TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy|TestPreparationDependencies(ForwardRealFixtureInputs|OperationErrorsRemainUnchanged|FailuresStopAtOriginalStages|KeepRealDefaultsAndBootstrapGuard)|TestInjectionCharacterization(IsolatedPreparationDependencies|IsolatedExploreRejectsEverySubstitution)|TestRunPreparationFailureLeavesClassifiedPartial|TestLocal(CampaignPreparationShortCircuits|CompletionErrorPrecedence|FinalizationPreservesHostFailureBeforePublication|FinalizationClassifiesPublicationFailure|StoppedCancellationPreservesPriorFailure)|TestPortableProfilePublicGuardsRemainFirst)$'
timeout --signal=TERM --kill-after=15s 600s /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner ./deterministicio -run "$task71_focused"
```

The four preparation dependency controls retain forwarding of real fixture inputs, unchanged operation-error identity, original failure stages and real defaults/bootstrap refusal. The isolated preparation control and public profile guard remain mandatory; the added unchanged isolated-substitution, real preparation-error and local preparation/completion/finalization/cancellation controls match existing task68 controls. The selected tables retain their own negative fault/cancellation assertions and positive World/semantic/choice/simulation projections. No helper or additive fixture is required.

Affected host vet, standalone errortype and supported source-set vet use the same external bound and stock Go/environment; retain explicit argv, including `-tags test_dep`, existing analyzer `-style-check=false`, and `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64` / `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` for source-set checks. Use the already retained `/tmp/fn109-lint-tools.ZdNe1t50/errortype` after verifying its execution-time identity. Configured unfiltered Runner lint uses the already retained `/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --verbose --build-tags test_dep --timeout 10m --fix=false --config ../../.github/.golangci.yml ./runner` from `tools/gomad3`, under the external watchdog. Required root `make lint-code-fast` uses an explicitly bound worker BASE with `GOLANGCI_LINT_BASE_REV`, `GOLANGCI_LINT_FIX=false`, `ALL_TEST_TAGS=test_dep`, and the same pinned `LOCALBIN`/`GOLANGCI_LINT`/`ERRORTYPE`. Do not run fixes, build/download replacement tools, or substitute filtered lint for unfiltered findings. Root grants each shared gate and resolves missing required checkout inputs before retrying.

## Compact evidence and root gates

Reference the retained [task68 materialized wrapper](../task-68/run-control-materialized.sh), SHA-256 `4d67887f4dbd41949e2f5b08280d0c7a9a08fb1bc1f28080059473cec64fb0b8`, and [original wrapper](../task-68/run-control.sh), SHA-256 `f1b90cffe72b745327b5b20f3c4cc836cbc857613252bd6f08536903caf501ed`, instead of copying their policy into another report. The original authoritative materialized path is `/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/choice-fixtures/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-68/run-control-materialized.sh`; its fixed CWD/packet are task68-specific. Never invoke or mutate that historical wrapper to collect task71 evidence. Root authorizes a task71-bound execution adapter later.

Retain one compact receipt per actual command with candidate HEAD, complete before/after consumed-source manifests, control/admission/research identity, absolute CWD, tool and wrapper hashes, exported/effective Go settings, argv array, numeric exit, wall elapsed time and immutable raw-log hash. Reuse task68's documented clearing of ambient Go/CGO tuning, pinned offline proxy/cache paths, `GOENV=off GOWORK=off GOTOOLCHAIN=local`, explicit `GOFLAGS`, `TZ=UTC` and task-specific temporary directory only after revalidating actual locations. Shell entry uses `login:false`, `env -u BASH_ENV bash -c`, and explicit `cd` into the assigned checkout to prevent ambient CWD reset. Record reused-cache/full-installation/environment limitations without claiming hermetic qualification.

[task68 binding-review-amendment.md](../task-68/binding-review-amendment.md) discloses historical omission of Perl and checker bytes at execution. New preservation/lint/boundary checker receipts must bind the actual Perl executable, each checker script and exact inputs before execution and after completion. A later seal cannot retroactively provide that binding. Task71's whole-file checker must remove only the three selected assignments and compare all remaining bytes with the actual BASE; task68's three differently located call-site checker is not sufficient. Retain gofmt check, full-file reconstruction and whitespace check without rewriting any other source. This recovery boundary is not an encoded-output compatibility obligation.

For generated validation and both-source-set/package-architecture/private/public boundaries, retain current checks or compare every actual consumed input with its retained validated manifest and reference those original receipts. Task68's documented 554-path reconciliation uses 552 newer validated inputs and two earlier qualification-generator inputs; no whole-fingerprint PASS extrapolation is allowed. Any changed consumed input or uncovered requirement returns to root for a serial gate decision.

Root owns the future frozen ordinary Runner comparison against all 673 original terminal names, allowing outcome changes only for actual root-admitted fixture-scope names. Include the sixty names above, retain all unchanged originals, synthesize no intermediate slash names and forecast no candidate count. Root also compares all fifty complete original-base lint blocks, requiring zero introduced or removed findings. Preserve actual aggregate nonzero exits and standalone/integrated errortype reachability; a focused or diff-filtered pass cannot certify aggregate green. Fresh independent integrated source review and all remaining owned source acceptance govern completion.

The worker must not start any shared Go/build/lint/vet/generator lane without a new root grant. No production/public/default/metadata/bootstrap decoder, helper, assertion, replay/resume/minimize/guidance, simulation transport/runtime, adapter/installation, schema/generated-input or lint-policy change is admitted. Preserve dirty owner spec, `MILESTONES.md` including fn155, and unrelated user files. Native fn128/fn149 remain deferred and unverified; this note grants no native revival, full native test-host claim, soak bound, CI, PR or push authority.
