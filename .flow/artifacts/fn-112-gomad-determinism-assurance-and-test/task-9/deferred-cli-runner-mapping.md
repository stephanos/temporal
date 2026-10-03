# fn-112.9 CLI and Runner consolidation map

**Status (2026-10-03).** Applied on `gomad-fn112-9` over `59ca3d1739`, after fn-109 tasks 2 to 12
merged; `evidence.md` ("CLI and Runner part") records the result and `mapping.tsv` holds every
removed name. Re-anchored against the merged tree, the map below changed as follows:

- Applied: section 1 as `TestRunQualifySetForwardsFlags`, `TestRunMinimizeForwardsFlags`,
  `TestRunResumeForwardsCampaignAndClassifiesResult`, `TestRunQualifyForwardsFlagsAndClassifiesOutcome`
  (forwarding and outcome rows in one table) and `TestRunAnalyzeForwardsTargetAndClassifiesReport`
  (forwarding, status, cleanup and output rows). They use the merged `fakeInstallation` seam. fn-109's
  CLI characterization tests (`characterization_test.go`, `plan_characterization_test.go`) are unchanged.
- Applied: section 2 rows for preparation partials, isolated-runner responses, the canned
  coordinator helper, cancellation, resume rejections, replay before target start, and the
  supervision-rejected trace rows. The golden table is now `TestCampaignOptionsLegacyCharacterization`
  (125 rows), not `TestCampaignOptionsCharacterization`. It gains `all retention without bytes`, the
  one rejection with no golden row.
- Applied: section 3 as written.
- Not applied: `TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem` and
  `TestRunRejectsPreparedTargetMutationBeforeFailurePublication`. They assert published artifacts
  and partials through `exploreWith`, and `TestSeedCompletionKeepsCampaignStatistics` only sees
  controller statistics. `TestRunRejectsReservedDuplicateAndInvalidEnvironment` is unchanged: four of
  its six subcases (`GOMAD3_CHOICE_*`, `NOT-VALID`, `nul`) have no golden row.
- Left for later (the D26 lane owns these files now): `runner/internal/execution` simulation tests
  and `tools/gomad3sim`. In `simulation_time_test.go` the arbiter tests look at behavior. Its
  strict/forward pairs (for example `...ForwardActivationAdoptsCurrent` /
  `...StrictActivationRejectsFutureCurrent`) could become one table once D26 lands.

The original prepared map follows unchanged.

## Original map

Prepared on `gomad-next-b` at `a52fb184c` against `gomad-next-c` at `3e649dae3` (merge base
`7f2bd1ed8`). Nothing in this file is applied. The task's Approach says R10 follows fn-109
task 6, which rewrites the executor-injection tests; fn-109 task 6 has not started, and
`gomad-next-c` (fn-109 tasks 2 to 5) rewrites `cmd/gomad/internal/cli` and `runner` heavily.
Consolidating these files now would collide with both, so this map is the input for the
follow-up once fn-109 lands. Re-anchor every name against the merged tree first.

## 1. `cmd/gomad/internal/cli/cli_test.go`: forwarded-field tests

On this branch 17 `*Dependencies{...}` literals are injected into `run*With`. There is no
injected-dependency test for explore, replay, plan, inspect, or execute-shard here;
`gomad-next-c` adds `exploreDependencies`, `replayDependencies`, `campaignShardDependencies`
and pins them with whole-value `reflect.DeepEqual` tests in `characterization_test.go`
(`TestCharacterizeExploreRequestDefaultsAndWiring`, `TestCharacterizePlanRequestAndOutput`,
`TestCharacterizeReplayRequestOutputAndStatus`, `TestCharacterizeExecuteShardRequestOutputAndStatus`)
and `plan_characterization_test.go` (`TestCharacterizePlanRequestDefaults`). Those are already
per-command forwarding tables; adopt them rather than adding new ones.

| Command | Proposed table | Existing test -> row | Collision with gomad-next-c |
| --- | --- | --- | --- |
| qualify-set | `TestQualifySetForwardsFlags` | `TestRunQualifySetUsesCurrentExecutableAndPublicPaths` -> `/public paths and executable` (keep its JSON-schema stdout check as a row field); `TestRunQualifySetPassesShardToTheSet/runs one shard` -> `/shard and min free bytes` | low (`qualify_set.go` executable now from `app.executablePath`) |
| minimize | `TestMinimizeForwardsFlags` | `TestRunMinimizeUsesBoundedArtifactStoreAndCurrentInstallation` -> `/bounded store and installation` (keep `accepted=1`); `TestRunMinimizeResumesOnlyOnRequest/initial run` -> `/initial run`; `/resume` -> `/resume` | high: c changes `identity` to `install` and pins the whole `MinimizeSpec`; fold into c's `TestCharacterizeInstallationWiring` |
| resume | `TestResumeForwardsFlags` | `TestRunResumeUsesStoredBatchAndReportsResult` -> `/stored batch and installation` (explore-event/v3 output checks stay a row field) | high: c pins the whole `ResumeSpec` |
| qualify | `TestQualifyForwardsFlags` | `TestRunQualifyRepeatsOneSeedAndRetainsJSONReport` -> `/repeat one seed`; `TestRunQualifyReplaysEveryRetainedSuccess` -> `/success replay bounds`; `TestRunQualifyReplaysRepeatedTargetFailure` -> `/failure replay artifacts` | high: c's `TestCharacterizeInstallationWiring` (qualify) already pins the CampaignSpec and replay wiring |
| analyze | `TestAnalyzeForwardsFlags` | `TestRunAnalyzeEmitsSupportedJSONWithoutExecutingTarget` -> `/go-test build tag and args` | low |

The two qualify replay sources mainly assert report outcomes (`retained.Executions[i].Replay`,
status, classification); those assertions must move with them (row fields or an outcome
table `TestQualifyClassifiesOutcome`), or behavior is lost. `TestRunQualifySetPassesShardToTheSet`
keeps its rows `checks one shard`, `rejects index past count`, `rejects malformed shard`,
`rejects count past manifest` (status and error text) as `TestRunQualifySetValidatesShard`.

Left out (they assert output, status or error text, not forwarded fields):
compare-support (`TestRunCompareSupportMapsReviewAndIncomparableStatuses`,
`TestRunCompareSupportDistinguishesInvalidReportsFromIOFailures`); analyze
(`TestRunAnalyzeMapsUnsupportedInvalidAndInfrastructureStatuses`,
`TestRunAnalyzeClassifiesRealReadonlyModuleFailureAsInvalidInput`,
`TestRunAnalyzePreservesClassificationWhenCleanupFails`,
`TestRunAnalyzeReportsOutputFailuresAsInfrastructure`); recover (`TestRunRecoverReportsStableTextAndJSON`,
`TestRunRecoverDistinguishesInvalidInputFromInfrastructureFailure`,
`TestRunRecoverRepairsPublishedBatchPrivateState`); qualify outcomes
(`TestRunQualifyReportsNondeterministicEvidence`, `TestRunQualifyRequiresExplicitSuccessfulReplayBounds`,
`TestRunQualifyRetainsMissingSuccessfulReplayArtifact`, `TestRunQualifyRetainsReplayCancellation`,
`TestRunQualifyRetainsUnsupportedBoundary`, `TestRunQualifyRejectsUnboundedRepeat`; optional
second table `TestQualifyClassifiesOutcome`); `TestRunResumeClassifiesInvalidJournalAsInputError`;
`TestRunMergeSetMapsStatuses` (already a table). Not injection tests: `TestResolveExplore*`,
`TestExploreReporter*`, `TestRunInspect*`, `TestPrintInspection*`, `TestRunDoctor*`,
`TestClassifyExplore*`, `TestReportReplayResult*`, `TestRunExploreReportsFlagErrorsAsJSON`.
Any rewrite of `cli_test.go` must start from c's version (c also edits `TestRunDoctor*` and
`TestRunExploreReportsFlagErrorsAsJSON`).

## 2. Runner tests that drive a fake executor (all wait for fn-109.6)

Executors injected through `testConfig(..., executor, ...)`, `CampaignSpec.Executor`,
`ReplaySpec.Executor`, `MinimizeSpec.Executor`:

- `runner_test.go` `fakeExecutor`: 33 tests (`TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures`
  through `TestManifestForRunBindsIOProfileIdentity`); placeholder only in the three
  `TestRunPreparation*` tests.
- `explorationExecutor` and interrupt variants: 6 `TestRunChoiceExploration*` tests;
  `simulationExplorationExecutor`: 3 `TestRunSimulationExploration*` tests;
  `terminalErrorExecutor`: 3; `blockingExecutor`: 2; `resumeInterruptExecutor`,
  `choiceResumeInterruptExecutor`: 4; one test each for `outOfOrderExecutor`,
  `firstFailureExecutor`, `progressGatedExecutor`, `mutatingExecutor`, `minimizationExecutor`.
- Other files: `completion_characterization_test.go` (3), `replay_operation_test.go` (14),
  `minimize_operation_test.go` (10), `inspect_test.go` (3), `portable_plan_test.go` (8),
  `guided_selection_test.go` (8), `diagnostics_test.go` (4), and 13 more in smaller files.
  c adds `seed_completion_characterization_test.go` (`scriptedSeedExecutor`) and two
  injected-executor rows in `TestCampaignOptionsCharacterization`.

Proposed folds:

| New table | Old -> row | Needs fn-109.6 |
| --- | --- | --- |
| `TestRunPreparationFailureLeavesClassifiedPartial` | `TestRunPreparationFailureLeavesExplicitPartial` -> `/build failure`; `TestRunPreparationCancellationIsClassifiedSeparately` -> `/cancelled`; `TestRunPreparationOverallTimeoutIsClassifiedSeparately` -> `/overall timeout` | signature only |
| `TestIsolatedRunnerPreservesCoordinatorResponse` | `TestIsolatedRunnerPreservesUnsupportedTargetError`, `...MissingSemanticProbesError`, `...BoundedExecutionEvidence`, `TestIsolatedRunnerTransportsChoiceTraceConfiguration`, `TestIsolatedRunnerBoundsCoordinatorOutput` -> one row each | signature only |
| `TestCannedCoordinatorHelper` (env-keyed subprocess helper) | `TestUnsupportedTargetCoordinatorHelper`, `TestMissingSemanticProbesCoordinatorHelper`, `TestExecutionEvidenceCoordinatorHelper`, `TestChoiceTraceCoordinatorHelper`, `TestOversizedCoordinatorHelper`, `TestFastCoordinatorHelper` (SKIP-status names in `go test -json`) | no; c rewrote `TestChoiceTraceCoordinatorHelper` |
| `TestCancellationIsAHostFailure` | `TestRunCancellationIsAHostFailure` -> `/seed`; `TestExplorationCancellationIsAHostFailure/{choice-exploration,simulation-exploration}` | yes |
| `TestRunResumeRejectsChangedEvidence` | `TestRunResumeRejectsChangedRunnerIdentity` -> `/runner build`; `TestRunResumeRejectsTamperedRetainedSuccessArtifact` -> `/tampered retained success` | yes |
| `TestReplayDoesNotStartTarget` | `TestReplayVerifyOnlyDoesNotStartTarget`, `TestReplayRejectsUnavailableCompatibilityPackBeforeTargetStart`, `TestReplayRejectsChangedPayloadBeforeTargetStart`, `TestReplayRejectsDamagedSharedTargetBeforeTargetStart/*` | yes |
| `TestCompletionFaultsKeepReasonPrecedenceAndEvidence` | `TestRunClassifiesInvalidChoiceTraceTerminalEvidence/malformed` -> existing row `choice trace rejected by supervision`; `/unterminated` -> new row with `err: execution.ErrChoiceTraceUnterminated` | yes |
| c's `TestSeedCompletionKeepsCampaignStatistics` | `TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem`, `TestRunRejectsPreparedTargetMutationBeforeFailurePublication` | yes |

Validation tests c's `TestCampaignOptionsCharacterization` (125 golden rows) already covers,
deletable once each assertion maps to a row: `TestValidateConfigRequiresBoundedSingleSeedChoiceExploration/*`,
`TestValidateConfigRequiresBoundedSingleSeedSimulationExploration/*`,
`TestValidateConfigRejectsExplorationBoundsForSeedStrategy`, `TestRunRequiresBoundedChoiceTraceCapacity`,
`TestRunRequiresExplicitSuccessRetentionBounds`, `TestRunGuidanceRequiresCorpusAndSemanticCoverage`,
`TestExecutionEvidenceRequiresOneSeedAndSemanticCoverage`, and partly
`TestRunRejectsReservedDuplicateAndInvalidEnvironment` (check its `nul` and `GOMAD3_CHOICE_*`
subcases). c's table calls validation directly while some of these go through `Explore()`;
record that difference in the mapping.

## 3. Error text pinned in both completion test files

| Pinned text | `completion_test.go` row | `completion_characterization_test.go` row |
| --- | --- | --- |
| `decode World terminal: invalid character '\|' after object key:value pair` | `TestAssessWorldValidatesTheRecordAgainstItsSeed/malformed record` | `malformed World/*` (also a prefix in three combined rows) |
| `World record seed or schema does not match seed N` | `.../seed mismatch` (N=8) | `World seed mismatch/*` (N=7) |
| `I/O transcript has invalid length 21` | `TestAssessCompletionProjectsCoverageInOrderAndClassifies/malformed semantic coverage`, `.../malformed semantic coverage before malformed choices`, `.../malformed semantic coverage of a watchdog kill` | `malformed semantic coverage/*`, `malformed semantic coverage and choice trace/*`, `watchdog and malformed semantic coverage/*` |
| `project choice coverage: malformed choice trace\ninvalid choice terminal values` | `.../malformed choices` | `malformed choice trace/*` |

Keep the characterization copy: it runs through public `Explore` for all three strategies and
pins reason, counters, artifacts, journal, partials and joined causes on top of the text, and
it survives refactors of the private `assessWorld`/`assessCompletion`. In `completion_test.go`,
delete rows `malformed record` and `seed mismatch` (map to `malformed World/seed` and
`World seed mismatch/seed`), keep `no record`, `valid record`, `transition limit` and
`malformed record before seed mismatch` (the only pin of that precedence), and drop `cause`
from the four coverage error rows while keeping their `reason`. The kept copy injects
`faultExecutor` through `testConfig`, so this also waits for fn-109.6.
