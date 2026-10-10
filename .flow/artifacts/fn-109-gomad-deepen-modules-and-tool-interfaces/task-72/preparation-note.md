# Task 72 gate preparation

Task 72 remains TODO and unassigned. This note grants no implementation or shared execution lane; task 70 currently holds the Go lane. Requested routing is Sol/high. Root owns worker/workspace selection, actual BASE/candidate, cache/tool/environment bindings, serialized gates, integration, independent review, commits and lifecycle. This preparation agent ran only bounded reads and authored this note; no product, Flow state or Git mutation, tool probe, Go/build/lint/vet/generator, native, CI, PR or push operation ran.

Read the complete applicable Flow-Next skill, used `flowctl usage` then task 72 `show --json`/`cat`, and read [admission.md](admission.md) plus the complete [seed-completion-next-slice.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/seed-completion-next-slice.md), SHA-256 `68cd8095264fcf63a8d757724e00f774433570548de2665bd4b9405b890e8fc5`. AGENTS.md and MILESTONES.md were read completely; the complete Gomad README was already read by this preparation agent in the preceding task 69 preparation. Retained research supplies the source trace; this note does not redo it.

## Exact source boundary

Proposal BASE is PRIMARY `87d1927a90295dc267c33d6226a643a74c425f51`, with `tools/gomad3/runner/seed_completion_characterization_test.go` SHA-256 `b35b3ec2d623c5746cc58e8996ff033b8e584813e6308f5fe7e0a03fde76e2b6`. Root must bind the actual dispatched BASE before execution. Exactly two syntactic inserts are admitted:

```go
config.dependencies = scriptedPreparationDependencies(t, config.Preparer, config.dependencies.executor)
```

Insert immediately before the existing `exploreWith` in `TestSeedCompletionKeepsCampaignStatistics`, after `ctx, config := test.configure(t)` has returned final configuration. Insert immediately before the existing `exploreWith` in `TestSeedCompletionFaultsKeepCampaignStatistics`, after `injectedCompletionCampaign(...)` has returned final configuration. Both use the final preparer and outer executor. Preserve all fourteen rows, every datum/assertion, shared pointer state, first-failure barrier, scripted supervision/drain results, `mutatingExecutor{}`, `blockingExecutor{}`, progress cancellation and the outer `faultExecutor`; do not unwrap its base. Helpers, imports, comments, metadata and every excluded consumer stay unchanged.

The first table retains the entire observation comparison, including counters, stop cause/reason, real copied-target integrity, opened artifacts, journal entries, partial states and statistics. The second compares every `CampaignStatistics` field and retains `observeCompletion`'s error-type/artifact-open/journal-read checks; it does not assert that observation's other fields. Neither fixture executes the prepared target or decodes the synthetic marker. An unexpected original downstream failure returns to root for separate admission.

Retain the entire actual BASE file before editing. A task-bound, function-scoped checker must remove exactly one admitted statement at each location, reject every other edit, then compare the reconstructed entire file byte-for-byte with BASE. Record candidate/BASE/reconstructed hashes and numeric comparison exit. Whole-file equality preserves all helpers and excluded table bytes; selected snippets or gofmt equality do not establish it. Bind the actual checker, Perl if used, and BASE bytes before/after execution. Require separate empty pinned `gofmt -d` output and `git diff --check` success.

## Exact focused selections

Original RED regex, selecting both tables and all existing children:

```text
^(TestSeedCompletionKeepsCampaignStatistics|TestSeedCompletionFaultsKeepCampaignStatistics)$
```

Control regex, retaining the prior positive attachment selections plus forwarding/error/stage/default/bootstrap/isolated/public and preparation/local-phase controls:

```text
^(TestRun(CountsASharedTargetInFullAgainstTheSuccessByteLimit|RetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts|ChoiceExploration(ExecutesRootAndEveryNonSelectedRank|DivergingPrefixRetainsCompletedRound|ExpandsCompleteTargetFailures)|ReportsPreparationProgressAndCompletedCounts|ReportsPeriodicProgressWhileTargetIsRunning|ReportsPassiveSemanticCoverage|FailsWhenRequiredSemanticProbeIsMissing|RetainsOnlyProbeNovelSuccessesWithinExplicitBounds|RetainsOnlyChoiceNovelSuccessesAndRecordsTheirFeatures|MergesParallelCompletionsInSelectionOrdinalOrder|FailsClosedWhenSuccessRetentionCountIsExhausted|RejectsSuccessfulRetentionWithoutReplayTranscript|CollectsBoundedQualificationEvidenceForOneSeed|PreparesOnceBoundsParallelismAndGroupsMatchingFailures|PublishesConnectedWorldBundle|ClassifiesConnectedWorldDeadlock|CountsConnectedWorldReplayDivergence|RejectsInvalidConnectedWorldBeforePublication|RejectsPreparedTargetMutationBeforeFailurePublication|PreparationFailureLeavesClassifiedPartial)|TestPreparationDependencies(ForwardRealFixtureInputs|OperationErrorsRemainUnchanged|FailuresStopAtOriginalStages|KeepRealDefaultsAndBootstrapGuard)|TestInjectionCharacterization(IsolatedPreparationDependencies|IsolatedExploreRejectsEverySubstitution)|TestWaitForProgressStart(ObservesPreparationFailure)?|TestLocal(CampaignPreparationShortCircuits|CompletionErrorPrecedence|FinalizationPreservesHostFailureBeforePublication|FinalizationClassifiesPublicationFailure|StoppedCancellationPreservesPriorFailure)|TestPortableProfilePublicGuardsRemainFirst)$
```

Final regex is the exact union, retaining both tables and the same controls:

```text
^(TestSeedCompletion(KeepsCampaignStatistics|FaultsKeepCampaignStatistics)|TestRun(CountsASharedTargetInFullAgainstTheSuccessByteLimit|RetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts|ChoiceExploration(ExecutesRootAndEveryNonSelectedRank|DivergingPrefixRetainsCompletedRound|ExpandsCompleteTargetFailures)|ReportsPreparationProgressAndCompletedCounts|ReportsPeriodicProgressWhileTargetIsRunning|ReportsPassiveSemanticCoverage|FailsWhenRequiredSemanticProbeIsMissing|RetainsOnlyProbeNovelSuccessesWithinExplicitBounds|RetainsOnlyChoiceNovelSuccessesAndRecordsTheirFeatures|MergesParallelCompletionsInSelectionOrdinalOrder|FailsClosedWhenSuccessRetentionCountIsExhausted|RejectsSuccessfulRetentionWithoutReplayTranscript|CollectsBoundedQualificationEvidenceForOneSeed|PreparesOnceBoundsParallelismAndGroupsMatchingFailures|PublishesConnectedWorldBundle|ClassifiesConnectedWorldDeadlock|CountsConnectedWorldReplayDivergence|RejectsInvalidConnectedWorldBeforePublication|RejectsPreparedTargetMutationBeforeFailurePublication|PreparationFailureLeavesClassifiedPartial)|TestPreparationDependencies(ForwardRealFixtureInputs|OperationErrorsRemainUnchanged|FailuresStopAtOriginalStages|KeepRealDefaultsAndBootstrapGuard)|TestInjectionCharacterization(IsolatedPreparationDependencies|IsolatedExploreRejectsEverySubstitution)|TestWaitForProgressStart(ObservesPreparationFailure)?|TestLocal(CampaignPreparationShortCircuits|CompletionErrorPrecedence|FinalizationPreservesHostFailureBeforePublication|FinalizationClassifiesPublicationFailure|StoppedCancellationPreservesPriorFailure)|TestPortableProfilePublicGuardsRemainFirst)$
```

These are selections, not task 72 results or predictions that every control passes on the future candidate. Root must confirm the admitted candidate includes the selected prior attachments and preserve their actual baseline outcomes. Any later sibling controls require root's actual integrated selection, not guessed PASS results. Public/executor-only/prepare-only bootstrap refusal, isolated injection rejection and error precedence remain substantive controls.

Retain fresh unchanged-source preparation RED for both original tables before either insert, and record unchanged control outcomes on the same bound inputs. A missing tool, compilation failure, timeout or absent terminal row outcome is inconclusive. After attachment, execute every original row/assertion and compare all actually emitted control outcomes. Historical combined68 emitted exactly fourteen failed leaves and two failed parents. Reference the exact sixteen names/line bindings in admission and research rather than duplicating their table. These are not sixteen independent cases and authorize no invented slash intermediary or successor PASS count.

## Serial command and capture recipe

Reference [task 69 run-control-v3.sh](../task-69/run-control-v3.sh), supplied SHA-256 `751acf75e99657e5e0a82e043724ee1a6b7dbd9cccebbc08afefc5028a447b6e`, for capture conventions. It hardcodes task 69's retained-success cwd and packet: never invoke it for task 72. A dispatched worker must author/rebind its own small adapter and checkers, retain their pre/post execution identities, and reference prior receipts without copying the whole wrapper or histories. The supplied identity is a reference, not a task 72 tool probe or executable attestation.

All shell launches use `login:false`, outer `env -u BASH_ENV bash -c`, and an explicit absolute `cd`; `BASH_ENV` otherwise resets cwd. This PRIMARY-only launch form preserves the child argv without interpolation:

```sh
env -u BASH_ENV bash -c 'cd /Users/stephan/Workspace/skunkworks/gomad/temporal && export SANDBOX_START_DIR="$PWD" && exec "$@"' bash CHILD_ARGV
```

`CHILD_ARGV` is a placeholder for the actual argv array. Root must replace PRIMARY with the exact assigned worker cwd before launching its task-bound adapter. Before the measured child, that adapter clears ambient Go/CGO/C-tool and Make settings, fixes the assigned cache/offline-proxy/TMPDIR settings, records exported/effective settings, and pins `GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS= TZ=UTC`. Use the existing Go 1.27.1 installation `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64` and existing pinned lint tools only after root binds their availability, versions and bytes. No download, build or tool substitution is authorized by this note.

Apply `timeout --signal=TERM --kill-after=15s 900s` to each measured child and retain numeric exit/signal/timeout status. The Go test internal timeout is an additional bound. Under the bound and pinned environment, proposed child argv are:

| Gate | Child argv |
| --- | --- |
| Original RED | Pinned `bin/go -C tools/gomad3 test -tags test_dep -count=1 -timeout=8m -json ./runner -run` plus the exact original RED regex as one argv element. |
| Before/after controls | Pinned `bin/go -C tools/gomad3 test -tags test_dep -count=1 -timeout=8m -json ./runner ./deterministicio -run` plus the exact control regex. |
| Final focused | The same pinned two-package test command plus the exact final regex. This covers the after-control observation without a duplicate control run. |
| Format/preservation | Pinned `bin/gofmt -d tools/gomad3/runner/seed_completion_characterization_test.go`, then task-specific whole-file reconstruction and `git diff --check`, each with an actual receipt. |
| Affected host vet | Pinned `bin/go -C tools/gomad3 vet -tags test_dep ./runner`. |
| Affected errortype | Pinned `bin/go -C tools/gomad3 vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false ./runner`. |
| Supported source-set vet | Separate `env GOOS=darwin GOARCH=arm64 CGO_ENABLED=0` and `env GOOS=linux GOARCH=amd64 CGO_ENABLED=0` prefixes to the affected pinned vet argv. |
| Unfiltered baseline/final Runner lint | In the actual worker's explicit `tools/gomad3` cwd, `/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --verbose --build-tags test_dep --timeout 10m --fix=false --config ../../.github/.golangci.yml ./runner`. |
| Required fast lint | `make lint-code-fast GOLANGCI_LINT_BASE_REV=ACTUAL_WORKER_BASE GOLANGCI_LINT_FIX=false ALL_TEST_TAGS=test_dep LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50 GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype`. Root must bind the actual worker BASE; proposal `87d1927...` is not an automatic substitute. |
| Current affected architecture/private/public gates when reconciliation is unavailable | Pinned `bin/go -C tools/gomad3 test -tags test_dep -count=1 -timeout=8m -json -run '^(TestPackageArchitecture\|TestRunnerExecutionInjectionIsPrivate\|TestRunnerRequestsCompileInExternalModule\|TestExactModuleEdges\|TestPureModulesHaveNoHostEffects)$' .`; use the displayed regex's plain pipes as one argv element. |

All tests include `test_dep` and `-count=1`; no integration tag applies. `bin/go`/`bin/gofmt` mean the absolute pinned installation above, not ambient PATH tools. Prepare the actual needed source cones before the first capture, including the full `tests` tree for generated source inventory and `tests/mixedbrain/go.mod`/`go.sum`. Sparse missing-source failure before analysis supplies no standards result. Serialize all gates, freeze inputs during each, and await every command handle before handing off the lane.

## Receipt and acceptance bindings

The first capture must inventory the complete actual consumed-source set, including all generated-inventory inputs under `tests`, module files, Make/lint configuration, task/admission/research bytes, BASE source, authoritative PRIMARY owner spec and task-bound launch/wrapper/checker bytes. Retain deduplicated source/tool/environment manifests and one immutable raw log plus JSON receipt per command. Bind actual Perl and every checker script/input before execution and after completion; also bind every other invoked checker/launcher/tool executable, pinned Go/gofmt/lint/errortype and effective environment. A later hash/seal cannot repair omitted execution-time identities.

Each receipt carries actual HEAD/product diff, absolute cwd, exact argv array, UTC start/end, wall elapsed time, numeric exit/signal/timeout, pre/post source/tool/wrapper/checker/environment identities and raw hash. Preserve both raw environments if an explicitly disclosed ephemeral-field comparison is needed; do not normalize away meaningful configuration differences. Keep original failure receipts immutable. Reused caches and incomplete installation/header/libc/environment identities remain limits, not a hermetic execution claim.

Generated validation, architecture/private/public boundaries and broader both-supported-source-set requirements need current checks or genuine unaffected-input reconciliation to valid retained receipts for every actually consumed input. If a prior manifest omitted a consumed input, find a valid retained binding or execute the owned current gate; do not infer coverage from whole-fingerprint equality, a focused PASS or a postcapture hash. The full tests-tree inventory must exist from the first task 72 capture. Root selects any still-required generated checks and their lane; no generator ran here.

Root selects the actual immediately preceding frozen ordinary baseline at execution time. Combined68's retained sixteen failed terminal names establish historical RED only; unintegrated/task 69 results supply no invented immediate baseline for task 72. Compare actual terminal names from actual baseline and integrated candidate, allowing changes only within root-admitted scopes and preserving unrelated originals. Invent no parent/intermediate names or PASS totals. Compare all fifty complete original-base lint blocks with header/source/caret and exact line mapping, requiring zero introduced/removed findings. Unfiltered affected Runner RED6 and aggregate RED50 stay explicit unless actual current gates establish otherwise. Standalone errortype and diff-filtered fast lint establish no aggregate green or configured analyzer reachability by themselves.

Stock linux/arm64 host observations are developmental ordinary source coverage. Darwin/arm64 and Linux/amd64 cross-source vet selects supported source sets; it does not execute their native hosts. Full native test-host, runtime, replay and soak evidence stays deferred/unverified with fn-128/fn-149. This preparation revives neither and grants no native/CI/PR/push authority. Root's independent integrated review and all still-owned acceptance remain necessary; a verified RED checkpoint is not Done/SHIP.

Unresolved bindings: assigned worker and exact physical cwd; actual worker BASE/HEAD/product bytes and original table hash; lane grant after current task 70 handles terminate; pinned executable versions/hashes; actual caches/offline proxy/TMPDIR and effective environment; first complete consumed-source/generated-inventory set; task-specific wrapper/checker/Perl/input pre/post identities; valid inherited boundary receipts or current gates; root-selected actual immediate ordinary baseline and complete RED50 comparison inputs; independent integrated review candidate. All executions and outcomes for task 72 remain pending.
