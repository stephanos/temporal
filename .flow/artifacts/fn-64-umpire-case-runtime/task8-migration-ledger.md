# fn-64 task 8 migration ledger

This is the concrete pre-deletion review artifact. Every path below exists in the shared staged baseline `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf` / tree `73fabb58da5c8ed4a456952d30be3b62c30d6377`. No deletion may begin until the official read-only `codex:gpt-5.6-sol:high` review records `SHIP` in Review history.

## Closed inventory

- Deletion manifest: **179 paths**: 124 files under 14 legacy Go roots, 6 legacy integration files, 3 proto sources, 8 generated Go API files, 36 Lean/model files, and 2 caller-only auxiliary outputs/fixtures.
- Deleted Go entry points: **307** total: **303 `Test*`** and **4 `Fuzz*`** across 45 files.
- Inherited failure identities: **10** total: 9 full-selector Umpire2/Umpire3 failures and 1 legacy-vocabulary false positive.
- Preserved boundaries: Umpire2/Umpire3, the public in-process Case facade, new execution/verification/Temporal Host packages, generic artifact v2 support, generic model Observation/ImplementationLink/Property APIs, generic regression-view machinery, and `model/Umpire/Promotion*`.

## Ownership decisions

| Surface | Status | Replacement / owner | Reason |
|---|---|---|---|
| PortableTestPlan proto and UmpireExecutor RPC | intentionally-retired | Case proto plus public in-process `tools/umpire` facade | R10 requires removal without compatibility reader, RPC, HTTP, or CLI replacement. |
| `evaluationcontract`, `portableevaluation`, and fixed `runevaluation` | replaced | generic Contract preparation/evaluator in `tools/umpire/verification` | The Case Contract is a deterministic monitor; Go no longer interprets model property patterns or invokes a fixed checker. |
| old `runtime`, `runner`, `executor`, and `internal/runtimeengine` | replaced | `tools/umpire/internal/execution` and public facade | Case execution centralizes preparation, scheduling, recording, typed values, cleanup, and verdict production. |
| `executorgrpc`, `executorhttp`, local Run Evaluation CLI | intentionally-retired | Task 8 | These surfaces exist only to transport the retired plan/checker and R10 forbids replacements. |
| `testplan` | replaced | `tools/umpire.Prepare` / `PreparedCase` | Case preparation replaces plan provenance, structure, limits, and authority admission. |
| `temporal/local` and `temporal/nexus` | replaced | generic `tools/umpire/temporal`, `server`, `worker`, and delivery packages | Host sessions and a Case Program replace borrowed-environment and caller-closure specialization. |
| caller-closure-specific Experimental, System Execution, and Portable Evaluation Lean lineage | replaced or intentionally-retired | Case compiler/ProtoJSON/runtime model | The new model emits Case IR; bespoke scenario models, fixtures, and fixed commands no longer own runtime behavior. |
| fn-5 checked-promotion source types and validation | preserved | `model/Umpire/Promotion.lean`, `model/Umpire/PromotionTests.lean`, compiled-source fixture | These modules are scenario-neutral; only Temporal caller candidate, binding, and command modules are deleted. |
| generic artifacts and regression views | preserved | `tools/umpire/artifact`, `internal/artifactv2`, regression generator/framework | Only caller-closure rows, fixture, and caller-only generated output are removed. |
| Umpire2/Umpire3 | preserved | existing owners | They are unrelated to the Case Runtime cutover and their inherited integration failures remain allowed. |

## Mixed consumers

| Path | Status | Edit / owner |
|---|---|---|
| `Makefile` | preserved | Remove retired commands/packages/fixtures from targets; point build and regression prerequisites at Case Runtime. |
| `proto/internal/buf.yaml` | preserved | Remove the deleted executor service from the internal-only breaking/lint exemptions. |
| `.github/workflows/umpire.yml` | preserved | Replace the deleted package-local command with Case Runtime package tests; Task 10 remains owner of broader CI reconciliation. |
| `tests/umpire4_testenv_test.go` | preserved | Keep `newUmpireTestEnvironment`; remove caller fixture loaders and old local/nexus factory return. |
| `tests/umpire4_async_nexus_case_test.go` | preserved | Adjust the helper call to the single TestEnv result; this is the replacement integration proof. |
| `tools/umpire/artifact/experiment_test.go` | preserved | Remove only caller-closure fixture rows; keep Switch round-trip/checksum and generic mutation coverage. |
| `tools/umpire/internal/artifactv2/artifact_test.go` | preserved | Remove only the deleted caller fixture from the accepted-fixture table. |
| `tools/umpire/cmd/umpire-gen-regression-views/catalog.go` | preserved | Remove caller manifest entry/output while retaining the Switch entry and generic validation. |
| `tools/umpire/cmd/umpire-gen-regression-views/render_test.go` | preserved | Remove caller-specific cases; retain generic rendering, path, determinism, and cleanup tests. |
| `tools/umpire/regression/generated_view_test.go` | preserved | Use the surviving Switch reference for working-directory independence. |
| `tools/umpire/regression/ci_workflow_test.go` | preserved | Update commands/docs expectations and remove deleted local-authority/generated-caller assumptions. |
| `model/lakefile.toml` | preserved | Remove deleted Experimental, Promote, RunEvaluation, and gen-tests targets. |
| `model/Temporal/System.lean` | preserved | Remove the deleted `Temporal.System.Execution` import. |
| `model/TemporalModelTests.lean` | preserved | Remove deleted model test imports/examples; retain Case Runtime and generic model suites. |
| `model/Temporal/System/Nexus/Tests.lean` | preserved | Retain Core checks; remove CallerClosure and old Observation imports/sections. |
| `model/Temporal/System/Nexus/ImplementationLink.lean` | preserved | Retain the scenario-neutral ordinary lifecycle link; remove only its caller-closure declarations. |
| `model/Temporal/System/Nexus/ImplementationLinkTests.lean` | preserved | Retain ordinary lifecycle link tests; remove only caller-closure tests. |
| `model/Temporal/ImplementationLinkTests/Nexus.lean` | preserved | Retain the ordinary Nexus composition tests; remove only caller-closure/old Observation dependencies and sections. |
| `model/Temporal/Feature/Nexus.lean` | preserved | Remove the retired Experimental/caller-closure facade comment while preserving the ordinary Nexus facade. |
| `model/Temporal/Feature/Nexus/Experimental/AutoClose.lean` | preserved | Keep the load-bearing AutoClose design artifact; it is scenario-neutral and not owned by caller closure. |
| `model/Temporal/Feature/Nexus/Experimental/{Exploration,ExplorationTests,VariationSpace,VariationSpaceTests}.lean` | preserved | Keep the ordinary Nexus basic-lifecycle exploration and variation-space model/tests; they do not import caller closure. |
| `model/Temporal/Tool/NexusDiscovery.lean` | preserved | Remove only the caller-closure query; keep generic discovery and ordinary Nexus query entries. |
| `model/Temporal/Tool/NexusDiscoveryTests.lean` | preserved | Update expectations after caller query removal; keep generic discovery coverage. |
| `model/Temporal/Tool/Inspect.lean` | preserved | Remove only caller-closure inspect scenario; keep generic inspection. |
| `model/Temporal/Tool/InspectTests.lean` | preserved | Remove caller-specific expectations; keep generic diagnostic/inspection tests. |
| `model/Temporal/Tool/SemanticInventory.lean` | preserved | Remove fixed Run Evaluation inventory row; retain generic inventory. |
| `model/SEMANTIC_INVENTORY.md` | preserved / Task 10 | Generated-document reconciliation is explicitly owned by Task 10; Task 8 edits only the active Lean inventory source. |
| `model/Temporal/Tool/Generated/Regressions.md` | preserved / Task 10 | The caller row becomes stale when Task 8 removes its active generator entry, and Task 10 owns generated-document reconciliation. |
| `model/ModelLint/ImportGraph.lean` | preserved | Remove deleted exception/module references; retain general import-graph policy. |
| `model/ModelLint/ImportGraphTests.lean` | preserved | Update fixtures/expectations for the removed exception. |
| `model/Temporal/API.lean` | replaced | Regenerate after proto removal so the UmpireExecutor method disappears. |
| `model/Temporal/API/Types.lean` | replaced | Regenerate after proto removal so legacy message/plan types disappear. |
| `model/TemporalExperimentalTests.lean` | preserved | Keep the aggregate build root and its surviving `NexusDiscoveryTests`, `InspectTests`, Exploration, and VariationSpace imports; remove only deleted caller, generation, fixed promotion, and fixed evaluation imports. |
| `.gitattributes` | preserved | Remove the linguist-generated rule for the deleted portable-evaluation fixture tree. |
| `.gitignore` | preserved | Remove the explicit exception for the deleted caller-closure generated integration test. |
| `model/ARCHITECTURE.md, model/README.md, model/Umpire/ARCHITECTURE.md` | preserved / Task 10 | Broader documentation and gate reconciliation is explicitly owned by Task 10; no active build dependency is retained. |
| `.plans/UMPIRE4_SPEC.md and historical Flow memory/specs` | preserved / Task 10 | Normative spec cleanup is Task 10; historical records remain evidence and are excluded from active-code cutover checks. |

## Deleted Test/Fuzz migration

A file row applies the displayed status, replacement/owner, and reason to every listed top-level entry point, except `tests/umpire4_caller_closure_test.go`, whose symbols are classified individually immediately below it.

| Deleted file | Top-level Test/Fuzz entries | Status | Replacement / owner | Reason |
|---|---|---|---|---|
| `tests/umpire4_caller_closure_generated_test.go` | `TestUmpireCallerClosurePortability` | replaced | `model/Umpire/Case/{CompilerTests,ProtoJSON}.lean`, `tools/umpire/prepare_test.go`, and `TestUmpireAsyncNexusCase` | The checked Case fixture crosses Lean, Go admission, and runtime without generated scenario Go code. |
| `tests/umpire4_caller_closure_test.go` | `TestUmpireCallerClosurePathTraversesEveryStageExactlyOnce`<br>`TestUmpireCallerClosurePathRejectsPreflightBeforeRunnerIO`<br>`TestUmpireCallerClosurePathRetainsIndependentOutcomes`<br>`TestUmpireCallerClosureEvaluationPreservesSemanticNonSuccess`<br>`TestUmpireCallerClosureReturnsAndPublishesOneExactOperationalSet`<br>`TestUmpireFaultedCallerClosureReturnsClosedFaultRealizationEvidence`<br>`TestUmpireCallerClosureParticipantRealizesOneForceClose`<br>`TestUmpireFaultedCallerClosureParticipantCompletesOneDuplicateObservation` | replaced | Per-symbol mapping below | This file mixes replaced generic seams and retired scenario-specific fault behavior; each symbol is classified separately. |
| `tests/umpire4_portable_executor_test.go` | `TestUmpirePortableCanaryExecutor` | replaced | `tools/umpire/{prepared_case,host_external}_test.go` and `TestUmpireAsyncNexusCase` | The public Go Case facade replaces HTTP PortableTestPlan execution; HTTP-specific assertions retire with the transport. |
| `tests/umpire4_portable_grpc_executor_test.go` | `TestUmpirePortableGRPCExecutor` | intentionally-retired | Task 8 public in-process `tools/umpire` facade | R10 explicitly removes the executor RPC with no compatibility reader or transport replacement. |
| `tests/umpire4_run_evaluation_negative_control_test.go` | `TestUmpireDuplicateDeliveryRunEvaluation` | replaced | `tools/umpire/verification/evaluator_failure_test.go` | The Case monitor failure corpus covers closed violations and incomplete evaluation without a fixed checker process. |
| `tests/umpire4_run_evaluation_test.go` | `TestUmpireCallerClosureRunEvaluation` | replaced | `tools/umpire/verification/evaluator_test.go` and `TestUmpireAsyncNexusCase` | The Case Evaluator and async proof replace fixed caller-closure Run Evaluation. |
| `tools/umpire/cmd/umpire-gen-tests-go/generate_test.go` | `TestRenderGeneratedRunnerTestMatchesTheCheckedInOrdinaryGoTest`<br>`TestRenderGeneratedRunnerTestPinsHermeticSubjectBeforeRuntimeIO`<br>`TestRunRegeneratesOnlyTheDeterministicGoTest`<br>`TestRunRejectsAnythingButTheGenerationGrammar` | replaced | `model/Umpire/Case/{CompilerTests,ProtoJSON}.lean` and `tools/umpire/prepare_test.go` | Case compilation produces portable Case ProtoJSON directly; a generated scenario-specific Go test is no longer an artifact boundary. |
| `tools/umpire/cmd/umpire-local-run-evaluation/main_test.go` | `TestRunPublishesSatisfiedSetBeforeWritingExactSummary`<br>`TestRunRejectsEveryNonExactArgumentGrammarBeforeChecking`<br>`TestRunRejectsUnsafeSetAndOutputRootsBeforeChecking`<br>`TestRunCanonicalizesSymlinkedAncestorsToPhysicalRoots`<br>`TestRunRejectsOverlappingSetAndOutputRootsBeforeAdmission`<br>`TestRunRejectsInvalidInputTreesBeforeCheckingOrPublication`<br>`TestRunReportsToolingFailuresAtTheirOwningBoundary`<br>`TestRunReportsSemanticInputRejectionBeforeInstalledPairResolution`<br>`TestRunReportsBrokenStdoutAfterKeepingAuthoritativePublication`<br>`TestRunReturnsTheSameRevalidatedDestinationForAnIdenticalRetry`<br>`TestRunDoesNotRepairAConflictingImmutableDestination`<br>`TestRunReportsActualOutputPermissionFailureWithoutPartialDestination`<br>`TestRunKeepsToolingStatusWhenStderrIsUnavailable`<br>`TestSummaryExitStatusRequiresAllThreeSuccessDimensions` | intentionally-retired | Task 8 public in-process `tools/umpire` facade | The installed fixed checker/controller pair and its command grammar have no Case Runtime replacement transport or CLI. |
| `tools/umpire/evaluationcontract/contract_test.go` | `TestPackProducesStableDeterministicContract`<br>`TestPackRejectsProtoJSONOutsideTheCanonicalVocabulary`<br>`TestAdmitRejectsOneFieldStructuralMutations`<br>`TestAdmitRejectsInvalidCoordinateBoundaries`<br>`TestAdmitRejectsCyclicEmitOrdering`<br>`TestAdmitRejectsChecksumUnknownFieldAndNoncanonicalWire`<br>`TestAdmitEnforcesCollectionLimitAtNAndNPlusOne`<br>`TestAdmitEnforcesContractByteLimitAtNAndNPlusOne`<br>`TestAdmitEnforcesExpressionDepthAtNAndNPlusOne`<br>`TestAdmitEnforcesTotalOperatorLimitAtNAndNPlusOne`<br>`TestAdmitEnforcesGlobalLimitMaxima` | replaced | `tools/umpire/verification/prepare_test.go` and `tools/umpire/internal/ir/{read,write}_test.go` | Contract admission and deterministic wire validation moved to generic Case Contract preparation and IR codecs. |
| `tools/umpire/evaluationcontract/fuzz_test.go` | `FuzzAdmitRejectsSingleByteContractMutations` | replaced | `tools/umpire/verification/prepare_test.go` and `tools/umpire/internal/ir/{read,write}_test.go` | Contract admission and deterministic wire validation moved to generic Case Contract preparation and IR codecs. |
| `tools/umpire/executor/executor_test.go` | `TestExecuteCompletesTheAdmittedContractThroughOneInterface`<br>`TestExecutePreservesIndependentStatusesAtTheEvidenceRecordBoundary`<br>`TestExecuteWaitsForExplicitSourceClosure`<br>`TestExecuteSequentialRunsReceiveFreshIdentities`<br>`TestExecuteRejectsOverlapBeforeRunnerIO`<br>`TestExecuteCancellationDoesNotExposeIdleBeforeCleanup`<br>`TestExecuteDeadlineIsInconclusiveAndReusableAfterCertainCleanup`<br>`TestExecuteAppliesTheContractDeadlineAcrossExecutionAndEvaluation`<br>`TestExecutePoisonsAfterUncertainCleanup`<br>`TestExecuteAdmissionFailuresAreTypedPreIOAndDoNotPoison`<br>`TestExecutePoisonsWhenAStartedRunLosesCleanupCertainty`<br>`TestExecuteKeepsPreStartRunnerFailuresInternalAndReusable` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/executor/portable_executor_test.go` | `TestPrepareLeanGeneratedModelPlansRetainExactArtifactBindings`<br>`TestPortableExecutorRunsExternalAndModelPlansThroughOnePipeline`<br>`TestPortableExecutorRejectsTenCallBurstWithoutQueueing`<br>`TestPortableExecutorCancellationAndCleanupPoisoningRemainAtTheExecutionSeam`<br>`TestPortableExecutorRejectsPostDispatchInvariantWithoutAResult`<br>`TestPortableExecutorPreservesCancellationDuringRuntimeAdmission` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/executor/portable_projection_test.go` | `TestProjectPortableExecutionProducesTheExistingRunnerInput`<br>`TestPortableModelBindingsMustMatchTheProjectedRunnerInput`<br>`TestPortableInputBindingCarriesRuntimeSlotsToTheCheckedRunner` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/executorgrpc/server_test.go` | `TestServerDelegatesOnePlanAndPreservesTypedResult`<br>`TestServerMapsPreResultFailuresToCanonicalStatuses`<br>`TestServerBoundsTransportValuesWithoutDispatchOrFabricatedResults`<br>`TestNewEnforcesPlanLimitAtTransportIngress` | intentionally-retired | Task 8 public in-process `tools/umpire` facade | R10 forbids a replacement RPC, HTTP transport, or CLI; transport-only behavior is deliberately removed. |
| `tools/umpire/executorhttp/handler_fuzz_test.go` | `FuzzHandlerWireSurfaceFailsClosed` | intentionally-retired | Task 8 public in-process `tools/umpire` facade | R10 forbids a replacement RPC, HTTP transport, or CLI; transport-only behavior is deliberately removed. |
| `tools/umpire/executorhttp/handler_test.go` | `TestNewConnectsTheResidentExecutor`<br>`TestHandlerExchangesCanonicalProtobuf`<br>`TestHandlerRejectsInvalidTransportBeforeExecution`<br>`TestHandlerAdmitsExactRequestLimitAndRejectsLimitPlusOne`<br>`TestHandlerPreservesToolingFailuresAsInconclusiveResults`<br>`TestHandlerRejectsExecutorAndResponseTransportFailures`<br>`TestHandlerPropagatesDeadlineToExecutorStatuses`<br>`TestHandlerBoundsSlowRequestBodyBeforeExecution`<br>`TestHandlerAdmitsExactResultLimitAndRejectsLimitPlusOne`<br>`TestHandlerCannotPublishSuccessAfterResponseEncodingDeadline`<br>`TestHandlerClientCancellationCannotPublishPartialSuccess`<br>`TestHandlerReusesOneExecutorSequentially`<br>`TestHandlerOverlapReturnsBusyBeforeRuntimeIO` | intentionally-retired | Task 8 public in-process `tools/umpire` facade | R10 forbids a replacement RPC, HTTP transport, or CLI; transport-only behavior is deliberately removed. |
| `tools/umpire/internal/runtimeengine/engine_test.go` | `TestRunMatchesIndependentExhaustivePhaseOracle`<br>`TestMatchingFaultBindingValuesAcceptsPortableSyntheticDuplicateKind`<br>`TestRunCleanupFailureDominatesEveryEarlierOutcome`<br>`TestRunConcreteCleanupFailureDominatesItsExpiredDeadline`<br>`TestRunStopsPreparationWhenFactoryContextIsTerminal`<br>`TestRunRecordsFactoryPreparationReceiptOutcomes`<br>`TestRunRecordsParticipantPreparationReceiptOutcomes`<br>`TestRunRecordsRejectedAndUnsupportedControlReceipts`<br>`TestRunAppliesCompoundOutcomePrecedence`<br>`TestRunDetachesIsolationAndCleanupFromCanceledParent`<br>`TestRunAdmitsOnlyOneRequestAtATime`<br>`TestExecutionAdmissionRejectsAlreadyCanceledContextWithoutConsumingSlot`<br>`TestRunRejectsAlreadyCanceledRequestBeforePreparationAndAllowsNextRequest`<br>`TestRunKeepsControlPartialWhenCancellationPreventsTheRequest`<br>`TestRunKeepsParticipantOutputPartialWhenPreparationStopsPrimaryCommands`<br>`TestRunRejectsMissingOrInvalidControlReceiptBeforeAdmission`<br>`TestRunRejectsDuplicateControlReceiptBeforeAdmission`<br>`TestRunAdmitsExactEvidenceCapacityBoundary`<br>`TestRunClosesAnExplicitHistoryCapacityReceiptAsPartial`<br>`TestOracleRowsRemainUnique` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/internal/runtimeengine/evidence_test.go` | `TestEvidenceAccumulatorRetainsExactlyNAndReportsNPlusOneBeforeAppend`<br>`TestEvidenceAccumulatorRejectsMutationsBeforeAppend`<br>`TestEvidenceAccumulatorEnforcesPhaseByteLimitBeforeAppend`<br>`TestEvidenceAccumulatorRejectsIllTypedOrUnboundAllowlistedFields`<br>`TestEvidenceAccumulatorRetainsRequestBindingsAndHashedSensitiveValues` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/portableevaluation/evaluator_test.go` | `TestEvaluateSatisfiedClosedEvidence`<br>`TestEvaluateViolatedClosedEvidence`<br>`TestEvaluateIncompleteClosureIsUnknown`<br>`TestEvaluateMissingEvidenceFromClosedSourceIsInconclusive`<br>`TestEvaluateConflictingCorrelation`<br>`TestEvaluateUnsupportedEvidenceType`<br>`TestEvaluateCanceled`<br>`TestEvaluateWorkLimitExactBoundary`<br>`TestEvaluateInputLimitsExactBoundaries`<br>`TestEvaluateResultLimitExactBoundary`<br>`TestNormalizeNaturalExactBoundary`<br>`TestEvaluateDoesNotMutateInputs`<br>`TestEvaluatePropertyPatternOutcomes`<br>`TestEvaluatePropertyDestinationVocabulary`<br>`TestDestinationPatternRequiresExactFingerprint`<br>`TestEvaluateRetainsExactBindingsAndClauseEvidence`<br>`TestEvaluateMissingRenameExactMapping`<br>`TestEvaluateRejectsCrossedArtifactBinding`<br>`TestEvaluateRejectsStaleRunAndClosure`<br>`TestEvaluateRejectsEvidenceKindCrossedWithAnotherSource`<br>`TestEvaluateRejectsMalformedAndMisorderedEvidence`<br>`TestEvaluateRejectsContractMutation`<br>`TestEvaluateRetainsRawEvidenceKnownGaps`<br>`TestEvaluateRejectsInvalidRawEvidenceChecksum`<br>`TestEvaluateRejectsResultLimitBelowMinimum`<br>`TestEvaluateRequiresEveryIndependentSuccessStatus`<br>`TestObservationExpressionOperators`<br>`TestObservationExpressionFailures`<br>`TestPresentRetainsConflictAndUnsupported`<br>`TestNormalizeFieldDispositionMatrix`<br>`TestSelectEmissionRejectsDuplicateAndContradictoryValues`<br>`TestApplyLinkRejectsDuplicateAndContradictoryMappings`<br>`TestImplementationLinkApplicationLimitExactBoundary`<br>`TestEvidenceLinkPreflightsResultGrowth`<br>`FuzzEvaluateFailsClosed` | replaced | `tools/umpire/verification/{prepare,evaluator,evaluator_failure}_test.go` and Case compiler tests | The deterministic monitor evaluator replaces property-pattern interpretation, link application, and portable-plan parity. |
| `tools/umpire/portableevaluation/parity_test.go` | `TestGeneratePortableEvaluationParityFixtures`<br>`TestPortableEvaluatorMatchesLeanRunEvaluationFixtures`<br>`TestLeanGeneratedPortablePlansUseSharedAdmissionAndRetainExactBindings`<br>`TestLeanGeneratedPortablePlanRejectsChecksumBindingSourceAndLimitMutations`<br>`TestLeanParityContractsCoverV1OperatorVocabulary`<br>`TestLeanParityContractsCoverFalseMissingAndTypeErrorBranches`<br>`TestCorrelationSlotsAllowOptionalReferencesAndRejectWhollyMissing`<br>`TestLeanParityContractsFailClosedOnCanonicalOrderAndCrossedPairs`<br>`TestLeanParityContractsEnforceExactWorkBoundary`<br>`TestLeanParityRawKnownGapIsRetainedAndInconclusive` | replaced | `tools/umpire/verification/{prepare,evaluator,evaluator_failure}_test.go` and Case compiler tests | The deterministic monitor evaluator replaces property-pattern interpretation, link application, and portable-plan parity. |
| `tools/umpire/portableevaluation/portable_test.go` | `TestEvaluatePortableUsesTheExistingEvaluatorForPlanLocalResults`<br>`TestEvaluatePortableAppliesDirectPlanTraceWithoutARenameLink`<br>`TestEvaluatePortablePreservesTrustworthyDecisionsAndInconclusiveEvidence`<br>`TestEvaluatePortableEnforcesWorkAndResultBoundaries` | replaced | `tools/umpire/verification/{prepare,evaluator,evaluator_failure}_test.go` and Case compiler tests | The deterministic monitor evaluator replaces property-pattern interpretation, link application, and portable-plan parity. |
| `tools/umpire/regression/catalog_generated_test.go` | `TestWorkflowNexusQueryExactActionCallerClosure` | intentionally-retired | Task 8; generic regression generator remains | This generated test owns only the deleted caller-closure query; the Switch generated view continues to cover the generic framework. |
| `tools/umpire/runevaluation/checker_test.go` | `TestMain`<br>`TestEncodeCheckerRequestUsesCanonicalProtocolEnvelope`<br>`TestCheckerRequestWriterPreservesCanonicalStringEncoding`<br>`TestDecodeCheckerResponseRequiresCanonicalClosedBindings`<br>`TestCheckerRequestWriterBoundsExactNAndNPlusOne`<br>`TestCheckerProcessRoundTripsTheExactClosedProtocol`<br>`TestCheckerProcessFailsClosedAndReapsEveryChild`<br>`TestCheckerProcessRejectsAnUnexecutableSiblingBeforeSpawn`<br>`TestCheckerProcessCancellationAndTimeoutReapTheChild`<br>`TestResolveCheckerSiblingRejectsUnsafeOrMissingTargets`<br>`TestResolveVerifiedCheckerSiblingRejectsChangedBytes`<br>`TestRunFixedCheckerRequiresInstalledDigest`<br>`TestCheckerProcessExecutesVerifiedBytesWhenSiblingChangesBeforeStart`<br>`TestCheckerProcessExecutesVerifiedSnapshotWhenReplacementIsAttempted`<br>`TestResolveCheckerSiblingRequiresTheFixedControllerName`<br>`TestBoundedCheckerCaptureNeverAllocatesTheLimitPlusOneByte`<br>`TestCheckerProcessSupportsConcurrentIndependentInvocations`<br>`FuzzDecodeCheckerResponse` | intentionally-retired | Task 8 in-process Evaluator | The subprocess sibling protocol, fixed binary digest, and fixed subject identity are ownership of the retired Run Evaluation command. |
| `tools/umpire/runevaluation/command_test.go` | `TestLocalRunEvaluationMakeTargetBuildsVerifiedPairAndPublishes`<br>`TestCheckerSignalContextCancelsOnTermination`<br>`TestLocalRunEvaluationMakeTargetValidatesInputsBeforeBuilding`<br>`TestLocalRunEvaluationMakeTargetDoesNotExposePairNameOverrides` | intentionally-retired | Task 8 in-process Evaluator | The subprocess sibling protocol, fixed binary digest, and fixed subject identity are ownership of the retired Run Evaluation command. |
| `tools/umpire/runevaluation/integration_test.go` | `TestRealCheckerSiblingIsDeterministic`<br>`TestRealCheckerSiblingAdmitsDuplicateDeliveryViolation`<br>`TestRealCheckerSiblingAdmitsExactAcceptedSet`<br>`TestRealCheckerCancellationPublishesNoPartialSet` | intentionally-retired | Task 8 in-process Evaluator | The subprocess sibling protocol, fixed binary digest, and fixed subject identity are ownership of the retired Run Evaluation command. |
| `tools/umpire/runevaluation/mutation_test.go` | `TestRawArtifactMutationFailsAtAdmission`<br>`TestCheckerRequestSeparatesRuntimeAndCheckedMappings`<br>`TestCheckerResponseRejectsConsistentCheckedProfileDriftAtTheProtocolBoundary`<br>`TestRealCheckerObservationMutationMatrix`<br>`TestRealCheckerDuplicateDeliveryMutationMatrix`<br>`TestRealCheckerDuplicateDeliveryIgnoresOrdinaryOperationalFacts`<br>`TestRealCheckerRejectsCrossedDuplicateDeliverySemanticClosure`<br>`TestDuplicateDeliveryResponseRejectsStrictNormalSemanticBindings`<br>`TestRealCheckerMisboundParticipantCancellationEvidenceIsSemanticConflict`<br>`TestRealCheckerPartialEvidencePublishesAnInMemoryResult` | replaced | `tools/umpire/verification/{evaluator,evaluator_failure}_test.go` and `tests/umpire4_async_nexus_case_test.go` | In-process Case evaluation preserves independent verdict/failure behavior without the fixed checker protocol. |
| `tools/umpire/runevaluation/result_test.go` | `TestConstructEvaluationPreservesResponseAndAddsExactTransportClosure`<br>`TestCheckWithCheckerRejectsResponseDriftWithoutASet`<br>`TestCheckWithCheckerAdmitsTheCompleteIndependentStatusMatrix`<br>`TestCheckWithCheckerAdmitsAcceptedNonAppliedImplementationLinkResults`<br>`TestAcceptedOutcomeChecksumIsStableAndSensitiveOnlyToSemanticContent`<br>`TestCheckWithCheckerPreservesExactKnownGapMembershipAndUnion`<br>`TestCheckWithCheckerAcceptsExactKnownGapOverlapAcrossPhases`<br>`TestCheckWithCheckerRejectsEverySemanticOutputInvariantClass` | replaced | `tools/umpire/verification/{evaluator,evaluator_failure}_test.go` and `tests/umpire4_async_nexus_case_test.go` | In-process Case evaluation preserves independent verdict/failure behavior without the fixed checker protocol. |
| `tools/umpire/runevaluation/run_evaluation_test.go` | `TestCheckWithCheckerConstructsOneAdmittedEvaluationSet`<br>`TestCheckWithCheckerRejectsNonExecutionBeforeChecking`<br>`TestCheckWithCheckerErrorsExposeStableClassification` | replaced | `tools/umpire/verification/{evaluator,evaluator_failure}_test.go` and `tests/umpire4_async_nexus_case_test.go` | In-process Case evaluation preserves independent verdict/failure behavior without the fixed checker protocol. |
| `tools/umpire/runevaluation/subject_test.go` | `TestCheckSubjectProvesLocalAndCISubjectParity`<br>`TestCheckSubjectRejectsIndependentArtifactAndModelMutations`<br>`TestCheckSubjectRejectsCanonicalByteAndGeneratedBindingDrift` | intentionally-retired | Task 8 in-process Evaluator | The subprocess sibling protocol, fixed binary digest, and fixed subject identity are ownership of the retired Run Evaluation command. |
| `tools/umpire/runner/runner_test.go` | `TestRunRejectsIncompleteInputBeforeAdapterConstruction`<br>`TestRunRejectsGeneratedDigestDriftBeforeAdapterConstruction`<br>`TestRunPassesTheExactAdmittedSetToTheAdapter`<br>`TestRunClassifiesAdapterPreflightAsNotStarted`<br>`TestRunClassifiesParticipantConstructionAsNotStarted`<br>`TestRunRejectsIncompleteAdapterBeforeParticipantConstruction`<br>`TestRunRejectsLimitNPlusOneBeforeAdapterConstruction`<br>`TestRunRejectsAuthorityLeakBeforeParticipantConstruction`<br>`TestRunResolvesAndEnforcesDeclaredRuntimeBindingSlotsBeforeDispatch`<br>`TestRunRequiresRuntimeBindingResolverBeforeDispatch`<br>`TestExecutionSurfaceExposesOnlyTheDigestBoundRunner` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/runtime/errors_test.go` | `TestPreflightErrorIs` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/runtime/request_test.go` | `TestCheckRequestAcceptsOneImmutableExactInput`<br>`TestOutputCopiesSchemaValidRunAndRawEvidence`<br>`TestOutputPreservesZeroAndEmptyArtifactValues`<br>`TestOutputLeavesUnsupportedRawEvidenceValuesOutsideCopyContract`<br>`TestDuplicateDeliveryInputSetIsCanonicalAndPreflightClosed`<br>`TestCheckRequestRejectsEachPreflightMutationBeforeIO`<br>`TestCheckedContractValuesEnforceBoundsAndCanonicalOrder`<br>`TestCheckedCollectionsPreserveShapeAndEnforceLimits`<br>`TestCheckRequestAcceptsEveryBoundedRunIdentity` | replaced | `tools/umpire/internal/execution/{runtime,scheduler,recorder,values,carrier}_test.go` and facade tests | Generic Case execution owns admission, scheduling, evidence recording, cleanup, concurrency, and typed values. |
| `tools/umpire/temporal/local/attached_test.go` | `TestAttachedFactoryRejectsNilOrIncompleteAuthority`<br>`TestAttachedFactoryRejectsAuthorityDriftBeforeResourceAcquisition`<br>`TestDistinctAttachedFactoriesPrepareConcurrentlyWithSeparateBindings`<br>`TestAttachedFactoryRejectsConcurrentPreparationUntilCleanup`<br>`TestAttachedFactoryOwnsOnlyFreshRunWorkers`<br>`TestAttachedIsolationScopesEachReusedNamespaceRun`<br>`TestAttachedCleanupCancellationRetainsOwnedWorker`<br>`TestAttachedWorkerStartCancellationReturnsBeforeBlockedStartAndCleansEventually`<br>`TestAttachedWorkerFailuresRetainOnlyAcquiredResources`<br>`TestAttachedWorkerStopCancellationReturnsBeforeBlockedStopAndClosesOnce`<br>`TestAttachedCleanupFailureRetainsOwnershipUntilClosed` | replaced | `tools/umpire/temporal/{host,artifact}_test.go`, server/worker session tests, and async Case integration | Case Temporal Host sessions own borrowed-cluster resources, route isolation, bounded cleanup, and worker lifecycle. |
| `tools/umpire/temporal/local/authority_test.go` | `TestFactoryRejectsInvalidPreparationBeforeAuthorityStart`<br>`TestFactoryFailsClosedWhenAuthorityStartReturnsNil`<br>`TestFactoryRetainsPartialAuthorityForCleanup` | replaced | `tools/umpire/temporal/{host,artifact}_test.go`, server/worker session tests, and async Case integration | Case Temporal Host sessions own borrowed-cluster resources, route isolation, bounded cleanup, and worker lifecycle. |
| `tools/umpire/temporal/local/environment_test.go` | `TestFactoryCancellationDoesNotStartAuthority`<br>`TestFactorySanitizesPartialStartupFailures`<br>`TestEnvironmentReceiptsUsePortableEvidenceKinds` | replaced | `tools/umpire/temporal/{host,artifact}_test.go`, server/worker session tests, and async Case integration | Case Temporal Host sessions own borrowed-cluster resources, route isolation, bounded cleanup, and worker lifecycle. |
| `tools/umpire/temporal/local/isolation_test.go` | `TestIsolationCollectionTransitions`<br>`TestIsolationCollectionDecision`<br>`TestIsolationCollectionInvalidationIsPermanent`<br>`TestEnvironmentIsolationOrchestration` | replaced | `tools/umpire/temporal/{host,artifact}_test.go`, server/worker session tests, and async Case integration | Case Temporal Host sessions own borrowed-cluster resources, route isolation, bounded cleanup, and worker lifecycle. |
| `tools/umpire/temporal/local/lifecycle_test.go` | `TestCleanupIsBoundedOrderedAndIdempotent`<br>`TestCleanupFailureRetainsOwnershipAndReturnsOnlyClosedCode`<br>`TestCanceledCleanupRetainsOwnershipAndCanBeRetried`<br>`TestLifecycleFactsHaveDistinctOperationIdentities`<br>`TestIsolationRequiresOneClosedOperationAndControlCollection`<br>`TestCleanupDeadlineReturnsTimeoutCompatibleReceipt`<br>`TestCleanupDeadlineReachedDuringStopReturnsTimeoutCompatibleReceipt`<br>`TestConcreteCleanupFailureDominatesExpiredDeadline` | replaced | `tools/umpire/temporal/{host,artifact}_test.go`, server/worker session tests, and async Case integration | Case Temporal Host sessions own borrowed-cluster resources, route isolation, bounded cleanup, and worker lifecycle. |
| `tools/umpire/temporal/local/profile_test.go` | `TestAuthorityUsesTheExactModelOwnedLocalProfile`<br>`TestRequiredCapabilitiesReturnsAnImmutableCopy` | replaced | `tools/umpire/temporal/{host,artifact}_test.go`, server/worker session tests, and async Case integration | Case Temporal Host sessions own borrowed-cluster resources, route isolation, bounded cleanup, and worker lifecycle. |
| `tools/umpire/temporal/nexus/binding_test.go` | `TestCheckRequestBindsTheExactCallerClosureProgram`<br>`TestCheckRequestBindsTheExactDuplicateDeliveryProgram`<br>`TestCallerClosureProgramVersionMatchesTheSystemModel`<br>`TestCheckRequestRejectsAnUnsupportedSetBeforeExecution` | replaced | `tools/umpire/temporal/{server,worker}/...` tests and `TestUmpireAsyncNexusCase` | Generic server/worker Hosts and the Case Program replace the bespoke caller-closure binding and synthetic evidence path. |
| `tools/umpire/temporal/nexus/configuration_test.go` | `TestCallerClosureInputSetIsStrictlyAdmitted`<br>`TestCallerClosureInputSetPassesLocalRuntimePreflight`<br>`TestCallerClosureInputSetRejectsNoncanonicalMemberBytes` | replaced | `tools/umpire/temporal/{server,worker}/...` tests and `TestUmpireAsyncNexusCase` | Generic server/worker Hosts and the Case Program replace the bespoke caller-closure binding and synthetic evidence path. |
| `tools/umpire/temporal/nexus/evidence_test.go` | `TestProjectTerminalHistoryDrainsTheIteratorAndClosesTheCausalChain`<br>`TestProjectTerminalHistoryRejectsEveryIncompleteOrCorruptClosure`<br>`TestValidateExecutionClosureAdmitsOnlyTheClosedMechanicalFourMemberSet`<br>`TestBindingRejectsCleanupLeakageWithStableClassification`<br>`TestFaultedExecutionFixtureIsTheExactClosedFourMemberSet`<br>`TestFaultedExecutionRejectsImpossibleReceiptAndObservationOrder`<br>`TestValidateExecutionClosureAdmitsFaultedEvidenceForEvaluation` | replaced | `tools/umpire/temporal/{server,worker}/...` tests and `TestUmpireAsyncNexusCase` | Generic server/worker Hosts and the Case Program replace the bespoke caller-closure binding and synthetic evidence path. |
| `tools/umpire/temporal/nexus/participant_test.go` | `TestParticipantAdmitsOnlyTheExactCheckedRequest`<br>`TestParticipantCleanupFactStaysInParticipantOutputSource`<br>`TestParticipantReceiptsRetainOnlyPortableRealizationEvidence`<br>`TestParticipantRejectsWrongCorrelationAndDuplicateCommandsBeforeAdapterIO`<br>`TestParticipantCancellationIsOperationalAndPerformsNoAdapterIO`<br>`TestParticipantCancellationBeforeRealizationIssuesNoControlRequest`<br>`TestRealizationContributesOneDuplicateObservationOnlyForTheFaultedProgram`<br>`TestDuplicateObservationCarriesSecondCancellationCoordinate`<br>`TestDuplicateObservationUsesPortableSyntheticKind`<br>`TestFaultedRealizationEmitsNoSyntheticObservationWithoutCompletedCancellation`<br>`TestFaultedRealizationRejectsAnUnstartedCancellationWithoutSyntheticObservation`<br>`TestWorkerReadinessCancellationEmitsNoReadinessClaim`<br>`TestSDKAndContextFailuresRemainOperationalReceipts`<br>`TestHandlerPanicsAreNonRetryable`<br>`TestHandlerBindsIdentityAndHandlesDuplicateCancellationIdempotently`<br>`TestCleanupReleasesEveryPartialPreparationExactlyOnce`<br>`TestCleanupFailureRetainsReleasedResourcesWithoutReacquiringThem` | replaced | `tools/umpire/temporal/{server,worker}/...` tests and `TestUmpireAsyncNexusCase` | Generic server/worker Hosts and the Case Program replace the bespoke caller-closure binding and synthetic evidence path. |
| `tools/umpire/temporal/nexus/runner_test.go` | `TestNewBindingRejectsAnIncompleteEnvironmentFactory`<br>`TestNewBindingRetainsOnlyTheSuppliedEnvironmentFactory`<br>`TestZeroBindingHasNoEnvironmentFactory` | replaced | `tools/umpire/temporal/{server,worker}/...` tests and `TestUmpireAsyncNexusCase` | Generic server/worker Hosts and the Case Program replace the bespoke caller-closure binding and synthetic evidence path. |
| `tools/umpire/testplan/authority_test.go` | `TestAuthorizeExternalPlanIsAlwaysPlanLocal`<br>`TestAuthorizeModelPlanRequiresExactHostProvenance`<br>`TestAuthorizeRejectsUnverifiedModelProvenance`<br>`TestScopeResultRejectsCallerOwnedAuthority`<br>`TestScopeResultAppliesExternalObligationPolicy` | replaced | `tools/umpire/{prepare,prepared_case}_test.go` and `tools/umpire/internal/execution/prepare_test.go` | Case preparation now owns version, structure, authorization profile, descriptor, and limit admission. |
| `tools/umpire/testplan/plan_test.go` | `TestGeneratedUnaryExecutorContract`<br>`TestPortablePlanSchemaHasNoOpaqueDocuments`<br>`TestSealAndAdmitCallerNeutralPlan`<br>`TestChecksumUsesDecodedPlanValue`<br>`TestAdmissionRejectsStructuralAndAuthorityMutations`<br>`TestAdmissionUsesCompleteKnownGapIdentity`<br>`TestAdmissionEnforcesIndependentLimitBoundaries`<br>`TestAdmissionRejectsDeclaredContentAndResultNPlusOne`<br>`TestAdmissionUsesExactDeclaredStructuralBounds` | replaced | `tools/umpire/{prepare,prepared_case}_test.go` and `tools/umpire/internal/execution/prepare_test.go` | Case preparation now owns version, structure, authorization profile, descriptor, and limit admission. |

### Per-symbol classification for the mixed caller-closure integration file

| Entry point | Status | Replacement / owner | Reason |
|---|---|---|---|
| `TestUmpireCallerClosurePathTraversesEveryStageExactlyOnce` | replaced | `TestUmpireAsyncNexusCase` and execution recorder/scheduler tests | The async proof plus generic execution tests cover the Case lifecycle and stage recording. |
| `TestUmpireCallerClosurePathRejectsPreflightBeforeRunnerIO` | replaced | `tools/umpire/{prepare,prepared_case}_test.go` | Case preparation validates before Host I/O. |
| `TestUmpireCallerClosurePathRetainsIndependentOutcomes` | replaced | execution runtime tests and verification failure tests | The Case result keeps execution and Contract evaluation dispositions independent. |
| `TestUmpireCallerClosureEvaluationPreservesSemanticNonSuccess` | replaced | `tools/umpire/verification/evaluator_failure_test.go` | The monitor failure corpus preserves violation and incomplete verdicts. |
| `TestUmpireCallerClosureReturnsAndPublishesOneExactOperationalSet` | replaced | `TestUmpireAsyncNexusCase` | The public facade returns the bounded Run and Verdict produced by the Case execution. |
| `TestUmpireFaultedCallerClosureReturnsClosedFaultRealizationEvidence` | intentionally-retired | Task 8 Case conformance corpus | Fault-realization records and fixed closed artifact sets are inventions of the deleted scenario adapter. |
| `TestUmpireCallerClosureParticipantRealizesOneForceClose` | replaced | `TestUmpireAsyncNexusCase` | The Case Program drives the SDK-triggered Nexus lifecycle through server and worker Hosts. |
| `TestUmpireFaultedCallerClosureParticipantCompletesOneDuplicateObservation` | intentionally-retired | Task 8 Case conformance corpus | Synthetic duplicate-observation injection belongs to the bespoke caller adapter and is absent from the closed Case scenario. |

## Inherited failure identity

Baseline command: `make umpire-check-live-tests`. Exit code was 0 under the repository inherited-failure policy. The raw suite failures below are all preserved, pre-existing Umpire2/Umpire3 identities; there was no Umpire4 or unclassified failure.

| Identity | Status | Owner / replacement | Reason |
|---|---|---|---|
| `TestUmpire2TestSuite` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire2TestSuite/TestProbeNexusDegraded` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire2TestSuite/TestPlanAndDriveKitchenSinkNexusOperation` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire2TestSuite/TestPlanAndDriveNexusOperationCHASM` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire2TestSuite/TestProbeNexusFlagged` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire2TestSuite/TestProbeNexusResilience` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire2TestSuite/TestProbeNexusRandomized` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire2TestSuite/TestProbeNexusExploration` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `TestUmpire3ParticipantProcessCrashAndRestartResumesRealSDKProgram` | preserved | Umpire2/Umpire3 existing owner | Observed before Task 8 deletion and allowed only as an exact full-selector inherited identity. |
| `tools/umpire/internal/ir/catalog.go:214: legacy-vocabulary fully.qualified token match` | preserved | Case IR / Task 10 regression gate owner | Known pre-Task-8 false positive recorded by Task 7; Task 8 must not claim it as green or delete Case IR. |

## Exact deletion manifest

Every path in this section is deleted as one of the ownership decisions above. The list is sorted within ownership groups and contains no duplicates.

### Legacy Go roots (124)

- `tools/umpire/evaluationcontract/contract.go`
- `tools/umpire/evaluationcontract/contract_test.go`
- `tools/umpire/evaluationcontract/errors.go`
- `tools/umpire/evaluationcontract/fuzz_test.go`
- `tools/umpire/evaluationcontract/validate.go`
- `tools/umpire/portableevaluation/README.md`
- `tools/umpire/portableevaluation/diagnostic.go`
- `tools/umpire/portableevaluation/evaluator.go`
- `tools/umpire/portableevaluation/evaluator_test.go`
- `tools/umpire/portableevaluation/link.go`
- `tools/umpire/portableevaluation/observation.go`
- `tools/umpire/portableevaluation/parity_test.go`
- `tools/umpire/portableevaluation/portable.go`
- `tools/umpire/portableevaluation/portable_test.go`
- `tools/umpire/portableevaluation/property.go`
- `tools/umpire/portableevaluation/testdata/any-operator/contract.pb`
- `tools/umpire/portableevaluation/testdata/any-operator/raw-evidence.json`
- `tools/umpire/portableevaluation/testdata/duplicate-delivery/contract.pb`
- `tools/umpire/portableevaluation/testdata/duplicate-delivery/lean-evidence.json`
- `tools/umpire/portableevaluation/testdata/duplicate-delivery/lean-result.json`
- `tools/umpire/portableevaluation/testdata/duplicate-delivery/raw-evidence.json`
- `tools/umpire/portableevaluation/testdata/normal/contract.pb`
- `tools/umpire/portableevaluation/testdata/normal/lean-evidence.json`
- `tools/umpire/portableevaluation/testdata/normal/lean-result.json`
- `tools/umpire/portableevaluation/testdata/normal/raw-evidence.json`
- `tools/umpire/portableevaluation/testdata/operator-branches.json`
- `tools/umpire/portableevaluation/testdata/portable-test-plan-v1/duplicate-delivery/plan.pb`
- `tools/umpire/portableevaluation/testdata/portable-test-plan-v1/normal/plan.pb`
- `tools/umpire/portableevaluation/testdata/portable-test-plan-v1/required-obligation/plan.pb`
- `tools/umpire/portableevaluation/testdata/run-branches/correlation-conflict/lean-evidence.json`
- `tools/umpire/portableevaluation/testdata/run-branches/correlation-conflict/lean-result.json`
- `tools/umpire/portableevaluation/trace.go`
- `tools/umpire/portableevaluation/work.go`
- `tools/umpire/runevaluation/README.md`
- `tools/umpire/runevaluation/checker.go`
- `tools/umpire/runevaluation/checker_snapshot_darwin.go`
- `tools/umpire/runevaluation/checker_snapshot_linux.go`
- `tools/umpire/runevaluation/checker_snapshot_unsupported.go`
- `tools/umpire/runevaluation/checker_test.go`
- `tools/umpire/runevaluation/command_test.go`
- `tools/umpire/runevaluation/integration_test.go`
- `tools/umpire/runevaluation/mutation_test.go`
- `tools/umpire/runevaluation/protocol.go`
- `tools/umpire/runevaluation/protocol_encoding.go`
- `tools/umpire/runevaluation/result.go`
- `tools/umpire/runevaluation/result_test.go`
- `tools/umpire/runevaluation/run_evaluation.go`
- `tools/umpire/runevaluation/run_evaluation_test.go`
- `tools/umpire/runevaluation/subject.go`
- `tools/umpire/runevaluation/subject_test.go`
- `tools/umpire/runevaluation/testdata/checker/main.go`
- `tools/umpire/runevaluation/testdata/checker/request.json`
- `tools/umpire/runevaluation/testdata/checker/response.json`
- `tools/umpire/runtime/README.md`
- `tools/umpire/runtime/engine.go`
- `tools/umpire/runtime/errors.go`
- `tools/umpire/runtime/errors_test.go`
- `tools/umpire/runtime/evidence.go`
- `tools/umpire/runtime/participant.go`
- `tools/umpire/runtime/request.go`
- `tools/umpire/runtime/request_test.go`
- `tools/umpire/runtime/runtime.go`
- `tools/umpire/runner/runner.go`
- `tools/umpire/runner/runner_test.go`
- `tools/umpire/executor/executor.go`
- `tools/umpire/executor/executor_test.go`
- `tools/umpire/executor/portable_executor.go`
- `tools/umpire/executor/portable_executor_test.go`
- `tools/umpire/executor/portable_projection.go`
- `tools/umpire/executor/portable_projection_test.go`
- `tools/umpire/executorgrpc/server.go`
- `tools/umpire/executorgrpc/server_test.go`
- `tools/umpire/executorhttp/handler.go`
- `tools/umpire/executorhttp/handler_fuzz_test.go`
- `tools/umpire/executorhttp/handler_test.go`
- `tools/umpire/testplan/authority.go`
- `tools/umpire/testplan/authority_test.go`
- `tools/umpire/testplan/errors.go`
- `tools/umpire/testplan/plan.go`
- `tools/umpire/testplan/plan_test.go`
- `tools/umpire/testplan/validate.go`
- `tools/umpire/temporal/nexus/binding.go`
- `tools/umpire/temporal/nexus/binding_test.go`
- `tools/umpire/temporal/nexus/configuration_test.go`
- `tools/umpire/temporal/nexus/evidence.go`
- `tools/umpire/temporal/nexus/evidence_test.go`
- `tools/umpire/temporal/nexus/output.go`
- `tools/umpire/temporal/nexus/participant.go`
- `tools/umpire/temporal/nexus/participant_test.go`
- `tools/umpire/temporal/nexus/runner.go`
- `tools/umpire/temporal/nexus/runner_test.go`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/artifacts/experiment.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/artifacts/runtime-configuration.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/manifest.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/artifacts/experiment-run.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/artifacts/experiment.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/artifacts/raw-evidence.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/artifacts/runtime-configuration.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/manifest.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-input-set/artifacts/experiment.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-input-set/artifacts/runtime-configuration.json`
- `tools/umpire/temporal/nexus/testdata/caller-closure-input-set/manifest.json`
- `tools/umpire/temporal/nexus/workflow.go`
- `tools/umpire/temporal/local/attached.go`
- `tools/umpire/temporal/local/attached_test.go`
- `tools/umpire/temporal/local/authority.go`
- `tools/umpire/temporal/local/authority_test.go`
- `tools/umpire/temporal/local/environment.go`
- `tools/umpire/temporal/local/environment_test.go`
- `tools/umpire/temporal/local/isolation.go`
- `tools/umpire/temporal/local/isolation_test.go`
- `tools/umpire/temporal/local/lifecycle_test.go`
- `tools/umpire/temporal/local/profile.go`
- `tools/umpire/temporal/local/profile_test.go`
- `tools/umpire/internal/runtimeengine/aliases_test.go`
- `tools/umpire/internal/runtimeengine/engine.go`
- `tools/umpire/internal/runtimeengine/engine_test.go`
- `tools/umpire/internal/runtimeengine/evidence.go`
- `tools/umpire/internal/runtimeengine/evidence_test.go`
- `tools/umpire/cmd/umpire-gen-tests-go/generate.go`
- `tools/umpire/cmd/umpire-gen-tests-go/generate_test.go`
- `tools/umpire/cmd/umpire-gen-tests-go/main.go`
- `tools/umpire/cmd/umpire-local-run-evaluation/main.go`
- `tools/umpire/cmd/umpire-local-run-evaluation/main_test.go`

### Legacy integration files (6)

- `tests/umpire4_caller_closure_generated_test.go`
- `tests/umpire4_caller_closure_test.go`
- `tests/umpire4_portable_executor_test.go`
- `tests/umpire4_portable_grpc_executor_test.go`
- `tests/umpire4_run_evaluation_negative_control_test.go`
- `tests/umpire4_run_evaluation_test.go`

### Proto sources and generated API (11)

- `proto/internal/temporal/server/api/umpire/v1/message.proto`
- `proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto`
- `proto/internal/temporal/server/api/umpire/v1/service.proto`
- `api/umpire/v1/message.go-helpers.pb.go`
- `api/umpire/v1/message.pb.go`
- `api/umpire/v1/portable_test_plan.go-helpers.pb.go`
- `api/umpire/v1/portable_test_plan.pb.go`
- `api/umpire/v1/service.pb.go`
- `api/umpire/v1/service.pb.mock.go`
- `api/umpire/v1/service_grpc.pb.go`
- `api/umpire/v1/service_grpc.pb.mock.go`

### Lean/model lineage (36)

- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureFault.lean`
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureFaultTests.lean`
- `model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotion.lean`
- `model/Temporal/Feature/Nexus/Experimental/CallerClosurePromotionTests.lean`
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean`
- `model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json`
- `model/Temporal/System/Execution.lean`
- `model/Temporal/System/Execution/LocalProfile.lean`
- `model/Temporal/System/Execution/LocalProfileTests.lean`
- `model/Temporal/System/Execution/Nexus.lean`
- `model/Temporal/System/Execution/NexusTests.lean`
- `model/Temporal/NexusExecutionIntegrationTests.lean`
- `model/Temporal/System/Nexus/CallerClosure.lean`
- `model/Temporal/System/Nexus/Observation.lean`
- `model/Temporal/System/Nexus/ObservationFaultTests.lean`
- `model/Temporal/Tool/PortableEvaluationContract.lean`
- `model/Temporal/Tool/PortableEvaluationContractTests.lean`
- `model/Temporal/Tool/GenerateTests.lean`
- `model/Temporal/Tool/GenerateTestsIOTestsMain.lean`
- `model/Temporal/Tool/GenerateTestsMain.lean`
- `model/Temporal/Tool/GenerateTestsTests.lean`
- `model/Temporal/Tool/RunEvaluation.lean`
- `model/Temporal/Tool/RunEvaluation/Protocol.lean`
- `model/Temporal/Tool/RunEvaluationMain.lean`
- `model/Temporal/Tool/RunEvaluationMutationTests.lean`
- `model/Temporal/Tool/RunEvaluationTests.lean`
- `model/Temporal/Tool/PromotionBinding.lean`
- `model/Temporal/Tool/PromotionBindingTests.lean`
- `model/Temporal/Tool/Promote.lean`
- `model/Temporal/Tool/PromoteMain.lean`
- `model/Temporal/Tool/PromoteTests.lean`
- `model/Temporal/Tool/PromoteTestsMain.lean`
- `model/Temporal/Tool/Fixtures/CallerClosurePromotionProposalV2.json`
- `model/Umpire/Artifact/PortableEvaluationContract.lean`
- `model/Umpire/Artifact/Tests/PortableEvaluationContract.lean`

### Caller-only auxiliary fixture/output (2)

- `tools/umpire/artifact/testdata/nexus-caller-closure-experiment-v2.json`
- `tools/umpire/regression/catalog_generated_test.go`

Manifest count: `179`. SHA-256 of newline-delimited manifest in section order: `d1055afe4aca241b918501110e2211de48c2ae095cb4d9e84304bd9e58948e28`.

## Pre-deletion closure checks

- [x] All 179 candidate paths have one owner decision and appear exactly once in the manifest.
- [x] All 307 deleted top-level Go Test/Fuzz entry points have a status, owner/replacement, and reason.
- [x] All 10 inherited failure identities are preserved and named exactly.
- [x] Mixed generic consumers have an explicit edit rather than directory-level deletion.
- [x] Umpire2/Umpire3, generic artifact/model functions, and scenario-neutral promotion types are outside the deletion manifest.
- [x] Official read-only ledger review verdict is `SHIP`.

## Review history

| Attempt | Backend | Verdict | Findings resolved |
|---|---|---|---|
| 1 | `codex:gpt-5.6-sol:high` | NEEDS_WORK | Preserved nine scenario-neutral/mixed Lean files; retained the experimental aggregate build root; classified `.gitattributes`/`.gitignore`; corrected Task 10 ownership. |
| 2 | `codex:gpt-5.6-sol:high` | NEEDS_WORK | Preserved both generated Markdown outputs for Task 10 and removed `Generated/Regressions.md` from the deletion manifest. |
| 3 | `codex:gpt-5.6-sol:high` | SHIP | No actionable ownership findings; verified 179 paths, 303 Test + 4 Fuzz entries, all 10 inherited identities, and all preserved boundaries. |
