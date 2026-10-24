# Test entrypoint disposition index

Part of [inventory.md](inventory.md). Source: `dc76222c8a3a6a9e67170e019b9f7f7eefd8cc87`. Named contracts are source assertions, not executed mutations. See [coverage.json](coverage.json) for whole-source hashes, declaration-to-EOF context spans and disposition reasons. Conversion is a future obligation, not completed coverage.

## tools/gomad/internal/backends/wasm/cluster/clock_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestClockConformanceCorpusIsCurrent`:708 | convert-to-Rust-corpus | Violation of Clock Conformance Corpus Is Current |
| `TestClockConformanceDriftPinIsCurrent`:790 | obsolete | Violation of Clock Conformance Drift Pin Is Current |
| `TestClockConformanceGenerationIsDeterministic`:814 | convert-to-Rust-corpus | Violation of Clock Conformance Generation Is Deterministic |

## tools/gomad/internal/backends/wasm/cluster/testdata/service/output_capacity_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestOutputCapacity`:8 | kept-Go | Missing output-producing guest prevents Rust output-capacity policy checks |

## tools/gomad/internal/backends/wasm/cluster/world_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWorldConformanceCorpusIsCurrent`:715 | convert-to-Rust-corpus | Violation of World Conformance Corpus Is Current |
| `TestWorldConformanceDriftPinIsCurrent`:799 | obsolete | Violation of World Conformance Drift Pin Is Current |
| `TestWorldConformanceFinalWireRoundTrips`:823 | convert-to-Rust-corpus | Violation of World Conformance Final Wire Round Trips |

## tools/gomad/internal/model/capture/capture_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCaptureCachesRegularFileContents`:11 | kept-Go | Violation of Capture Caches Regular File Contents |
| `TestCaptureCachesMissingMountedPath`:46 | kept-Go | Violation of Capture Caches Missing Mounted Path |
| `TestCaptureReturnsSortedDirectoryEntries`:65 | kept-Go | Violation of Capture Returns Sorted Directory Entries |
| `TestCaptureRejectsSymlinksHardLinksAndTraversal`:86 | kept-Go | Violation of Capture Rejects Symlinks Hard Links And Traversal |
| `TestCaptureEnforcesLimitsBeforePublishingEntry`:110 | kept-Go | Violation of Capture Enforces Limits Before Publishing Entry |
| `TestCaptureSerializesConcurrentFirstLookup`:130 | kept-Go | Violation of Capture Serializes Concurrent First Lookup |

## tools/gomad/internal/model/capture/config_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestParseMappingsNormalizesSourcesAndTargets`:10 | kept-Go | Violation of Parse Mappings Normalizes Sources And Targets |
| `TestParseMappingsRejectsUnsafeOrOverlappingMappings`:28 | kept-Go | Violation of Parse Mappings Rejects Unsafe Or Overlapping Mappings |
| `TestParseMappingsRejectsNonDirectoryAndSymlinkRoots`:57 | kept-Go | Violation of Parse Mappings Rejects Non Directory And Symlink Roots |

## tools/gomad/internal/model/capture/inventory_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCaptureReadOnlyMountInventoryBindsCompleteTree`:9 | kept-Go | Violation of Capture Read Only Mount Inventory Binds Complete Tree |

## tools/gomad/internal/model/capture/persistence_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestArtifactRoundTripBuildsHostIndependentReplay`:13 | kept-Go | Violation of Artifact Round Trip Builds Host Independent Replay |
| `TestDecodeCapturedInputsRejectsCorruptPayload`:58 | kept-Go | Violation of Decode Captured Inputs Rejects Corrupt Payload |
| `TestDecodeCapturedInputsAcceptsAuthenticatedOrdinaryWhitespace`:74 | kept-Go | Violation of Decode Captured Inputs Accepts Authenticated Ordinary Whitespace |
| `TestEncodeCapturedInputsRejectsInvalidOriginalString`:112 | kept-Go | Violation of Encode Captured Inputs Rejects Invalid Original String |
| `TestDecodeCapturedInputsRejectsStrictSyntaxSchemaAndDecimalErrors`:120 | kept-Go | Violation of Decode Captured Inputs Rejects Strict Syntax Schema And Decimal Errors |

## tools/gomad/internal/model/capture/uint64_string_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestUint64StringJSONUsesStrictDecimalStrings`:10 | kept-Go | Violation of Uint64 String JSONUses Strict Decimal Strings |

## tools/gomad/internal/model/capture/wire_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestServeReturnsCapturedFileAndUnmountedStatus`:15 | kept-Go | Violation of Serve Returns Captured File And Unmounted Status |
| `TestServeRejectsOversizedAndOutOfOrderRequests`:54 | kept-Go | Violation of Serve Rejects Oversized And Out Of Order Requests |
| `TestReadResponseRejectsInvalidProtocolFields`:82 | kept-Go | Violation of Read Response Rejects Invalid Protocol Fields |
| `TestReadResponseRejectsInvalidChildFields`:113 | kept-Go | Violation of Read Response Rejects Invalid Child Fields |

## tools/gomad/internal/model/choice/creation_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCreationMetadataCodec`:9 | convert-to-Rust-corpus | Violation of Creation Metadata Codec |
| `TestCreationMetadataRequiresCompleteAlternativeSet`:29 | convert-to-Rust-corpus | Violation of Creation Metadata Requires Complete Alternative Set |
| `TestPreviousVersionTraceRemainsReadable`:37 | convert-to-Rust-corpus | Violation of Previous Version Trace Remains Readable |

## tools/gomad/internal/model/choice/diagnostic_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBuildDiagnosticTraceUsesBoundedExistingWire`:12 | convert-to-Rust-corpus | Violation of Build Diagnostic Trace Uses Bounded Existing Wire |
| `TestDiagnosticDiffFindsFirstOrdinalAndFields`:55 | convert-to-Rust-corpus | Violation of Diagnostic Diff Finds First Ordinal And Fields |
| `TestDiagnosticDiffValidatesEntireInputsBeforeReporting`:81 | convert-to-Rust-corpus | Violation of Diagnostic Diff Validates Entire Inputs Before Reporting |
| `TestDiagnosticSessionCollectsOnlyCompleteBoundedTrace`:103 | convert-to-Rust-corpus | Violation of Diagnostic Session Collects Only Complete Bounded Trace |

## tools/gomad/internal/model/choice/internal/wire/wire_generated_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestChoiceWireRecordRoundTrip`:9 | convert-to-Rust-corpus | Violation of Choice Wire Record Round Trip |
| `TestChoiceWireRejectsInvalidSelectionReadinessAndOrigin`:47 | convert-to-Rust-corpus | Violation of Choice Wire Rejects Invalid Selection Readiness And Origin |
| `TestChoiceWireTapeHeaderRoundTrip`:79 | convert-to-Rust-corpus | Violation of Choice Wire Tape Header Round Trip |
| `TestChoiceWireHeaderAndTerminalRoundTrip`:108 | convert-to-Rust-corpus | Violation of Choice Wire Header And Terminal Round Trip |
| `TestChoiceWireDiagnosticHeaderRoundTrip`:144 | convert-to-Rust-corpus | Violation of Choice Wire Diagnostic Header Round Trip |
| `TestChoiceWireDiagnosticRecordRoundTrip`:177 | convert-to-Rust-corpus | Violation of Choice Wire Diagnostic Record Round Trip |

## tools/gomad/internal/model/choice/interrupted_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestProjectInterruptedReplayPlanValidatesObservedPrefix`:11 | convert-to-Rust-corpus | Violation of Project Interrupted Replay Plan Validates Observed Prefix |

## tools/gomad/internal/model/choice/readiness_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestProjectReplayPlanCarriesSelectReadinessOntoPollDecisions`:66 | convert-to-Rust-corpus | Violation of Project Replay Plan Carries Select Readiness Onto Poll Decisions |
| `TestProjectReplayPlanRejectsResultsThatMisnameTheirPollDecisions`:130 | convert-to-Rust-corpus | Violation of Project Replay Plan Rejects Results That Misname Their Poll Decisions |
| `TestProjectReplayPlanLeavesSingleCaseSelectsUnmatched`:164 | convert-to-Rust-corpus | Violation of Project Replay Plan Leaves Single Case Selects Unmatched |
| `TestDecodeTraceRejectsReadinessOnADecision`:184 | convert-to-Rust-corpus | Violation of Decode Trace Rejects Readiness On ADecision |
| `TestChoiceReadersRejectOtherWireVersions`:192 | convert-to-Rust-corpus | Violation of Choice Readers Reject Other Wire Versions |

## tools/gomad/internal/model/choice/session_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestSessionCollectsCompleteTrace`:11 | convert-to-Rust-corpus | Violation of Session Collects Complete Trace |
| `TestSessionRejectsInvalidTerminalStates`:58 | convert-to-Rust-corpus | Violation of Session Rejects Invalid Terminal States |
| `TestSessionReturnsValidatedOverflowTrace`:94 | convert-to-Rust-corpus | Violation of Session Returns Validated Overflow Trace |
| `TestSessionReplayPlanIsInheritedReadOnly`:133 | convert-to-Rust-corpus | Violation of Session Replay Plan Is Inherited Read Only |

## tools/gomad/internal/model/choice/tape_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestReplayPlanConformance`:89 | convert-to-Rust-corpus | Violation of Replay Plan Conformance |

## tools/gomad/internal/model/choice/tape_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCanonicalDecisionIsIndependentOfPhysicalAlternativeOrder`:11 | convert-to-Rust-corpus | Violation of Canonical Decision Is Independent Of Physical Alternative Order |
| `TestCanonicalDecisionRejectsInvalidAlternativeIdentities`:29 | convert-to-Rust-corpus | Violation of Canonical Decision Rejects Invalid Alternative Identities |
| `TestProjectDecisionTapeCopiesAndBindsCompleteTrace`:51 | convert-to-Rust-corpus | Violation of Project Decision Tape Copies And Binds Complete Trace |
| `TestProjectReplayPlanOmitsNonBranchingDecisions`:104 | convert-to-Rust-corpus | Violation of Project Replay Plan Omits Non Branching Decisions |
| `TestValidateDivergenceTerminalBindsExpectedTapeDecision`:139 | convert-to-Rust-corpus | Violation of Validate Divergence Terminal Binds Expected Tape Decision |
| `TestValidateDivergenceTerminalAcceptsRankPrefix`:182 | convert-to-Rust-corpus | Violation of Validate Divergence Terminal Accepts Rank Prefix |
| `TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace`:207 | convert-to-Rust-corpus | Violation of Project Decision Tape Rejects Observation Only And Legacy Trace |
| `TestBuildRankPrefixBindsCanonicalRankWithoutParentTraceIdentity`:220 | convert-to-Rust-corpus | Violation of Build Rank Prefix Binds Canonical Rank Without Parent Trace Identity |
| `TestBuildForcedRankPrefixReconstructsSelectedOverride`:263 | convert-to-Rust-corpus | Violation of Build Forced Rank Prefix Reconstructs Selected Override |
| `TestBuildRankPrefixRejectsInvalidOverrideTargets`:292 | convert-to-Rust-corpus | Violation of Build Rank Prefix Rejects Invalid Override Targets |

## tools/gomad/internal/model/choice/trace_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestProjectExposesCanonicalChoiceFeatures`:11 | convert-to-Rust-corpus | Violation of Project Exposes Canonical Choice Features |
| `TestProjectBoundsUniqueAdjacentChoicePairs`:48 | convert-to-Rust-corpus | Violation of Project Bounds Unique Adjacent Choice Pairs |
| `TestImplementationIdentityAndProjectionAreCanonical`:77 | convert-to-Rust-corpus | Violation of Implementation Identity And Projection Are Canonical |
| `TestImplementationIdentityBindsToolchainBuildKey`:97 | convert-to-Rust-corpus | Violation of Implementation Identity Binds Toolchain Build Key |
| `TestDecodeTraceValidatesRecordsAndSummarizes`:116 | convert-to-Rust-corpus | Violation of Decode Trace Validates Records And Summarizes |
| `TestDecodeTraceRejectsMalformedEvidence`:136 | convert-to-Rust-corpus | Violation of Decode Trace Rejects Malformed Evidence |
| `TestDecodeTraceReportsOverflowSeparately`:164 | convert-to-Rust-corpus | Violation of Decode Trace Reports Overflow Separately |
| `TestProjectAcceptsValidatedOverflowWhileProjectCompleteRemainsStrict`:176 | convert-to-Rust-corpus | Violation of Project Accepts Validated Overflow While Project Complete Remains Strict |

## tools/gomad/internal/model/clock/identity_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBrokerImplementationBindsEverySharedClockSource`:9 | convert-to-Rust-corpus | Violation of Broker Implementation Binds Every Shared Clock Source |

## tools/gomad/internal/model/clock/simulation_broker_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBrokerClockRejectsInactiveParticipantWithoutAdvancing`:8 | convert-to-Rust-corpus | Violation of Broker Clock Rejects Inactive Participant Without Advancing |

## tools/gomad/internal/model/clock/simulation_time_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestSimulationTimeArbiterAdvancesEveryParticipantToEarliestDeadline`:10 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Advances Every Participant To Earliest Deadline |
| `TestSimulationTimeArbiterAcceptsForwardTickEpoch`:63 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Accepts Forward Tick Epoch |
| `TestSimulationTimeArbiterForwardActivationAdoptsCurrent`:83 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Forward Activation Adopts Current |
| `TestSimulationTimeArbiterStrictActivationRejectsFutureCurrent`:106 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Strict Activation Rejects Future Current |
| `TestSimulationTimeArbiterForwardTickEpochIsOrderIndependent`:125 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Forward Tick Epoch Is Order Independent |
| `TestSimulationTimeArbiterRejectsForwardTickEpochPastItsDeadline`:189 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Rejects Forward Tick Epoch Past Its Deadline |
| `TestSimulationTimeArbiterCancelsAnEpochWhenExternalWorkArrives`:208 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Cancels An Epoch When External Work Arrives |
| `TestSimulationTimeArbiterDoesNotAdvancePastExternalWorkOrInactiveParticipants`:256 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Does Not Advance Past External Work Or Inactive Participants |
| `TestSimulationTimeArbiterExcludesAnExternallyBlockedParticipant`:320 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Excludes An Externally Blocked Participant |
| `TestSimulationTimeArbiterSettlesWhenLastRunnableParticipantBlocksExternally`:356 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Settles When Last Runnable Participant Blocks Externally |
| `TestSimulationTimeArbiterDoesNotSettleWhileExternalRequestIsHandled`:393 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Does Not Settle While External Request Is Handled |
| `TestSimulationTimeArbiterAdvancesAfterForwardedRequestIsDelivered`:431 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Advances After Forwarded Request Is Delivered |
| `TestSimulationTimeArbiterForwardsExternalRequestAtomically`:460 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Forwards External Request Atomically |
| `TestSimulationTimeArbiterTransfersExternalArrivalAtomically`:498 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Transfers External Arrival Atomically |
| `TestSimulationTimeArbiterWaitsForDeliveredExternalWorkToBeConsumed`:544 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Waits For Delivered External Work To Be Consumed |
| `TestSimulationTimeArbiterRejoinsAfterExternalWorkArrives`:596 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Rejoins After External Work Arrives |
| `TestSimulationTimeArbiterActivatesRestartAtCurrentTime`:652 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Activates Restart At Current Time |
| `TestSimulationTimeArbiterRejectsForwardTickEpochInStrictMode`:693 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Rejects Forward Tick Epoch In Strict Mode |
| `TestSimulationTimeArbiterForwardTickEarlyResponsesDoNotRewind`:712 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Forward Tick Early Responses Do Not Rewind |
| `TestSimulationTimeArbiterMalformedForwardTickRequestDoesNotAdvance`:753 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Malformed Forward Tick Request Does Not Advance |
| `TestSimulationTimeArbiterAdoptsForwardedModelCurrent`:772 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Adopts Forwarded Model Current |
| `TestSimulationTimeArbiterRejectsInvalidForwardedModelCurrentWithoutMutation`:794 | convert-to-Rust-corpus | Violation of Simulation Time Arbiter Rejects Invalid Forwarded Model Current Without Mutation |

## tools/gomad/internal/model/cluster/controller_resolution_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestResolveFaultLifecycleStates`:11 | convert-to-Rust-corpus | Violation of Resolve Fault Lifecycle States |
| `TestResolveFaultLifecycleTargets`:55 | convert-to-Rust-corpus | Violation of Resolve Fault Lifecycle Targets |
| `TestResolveFaultOuterControls`:105 | convert-to-Rust-corpus | Violation of Resolve Fault Outer Controls |

## tools/gomad/internal/model/cluster/exploration_stock_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestExplorationIdentitiesMatchRunnerIssuedPlanVector`:14 | convert-to-Rust-corpus | Violation of Exploration Identities Match Runner Issued Plan Vector |

## tools/gomad/internal/model/cluster/exploration_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestExplorationPlanCanonicalRoundTripAndDetachedInput`:9 | convert-to-Rust-corpus | Violation of Exploration Plan Canonical Round Trip And Detached Input |
| `TestExplorationPlanRejectsChangedForcedDecisionIdentity`:30 | convert-to-Rust-corpus | Violation of Exploration Plan Rejects Changed Forced Decision Identity |
| `TestExplorationForcedMismatchProducesValidDivergenceEvidence`:43 | convert-to-Rust-corpus | Violation of Exploration Forced Mismatch Produces Valid Divergence Evidence |
| `TestNonExplorationDivergenceRejectsExplorationEvidence`:60 | convert-to-Rust-corpus | Violation of Non Exploration Divergence Rejects Exploration Evidence |
| `TestExplorationEvidenceProvesOnlyTheOverridesTheClusterDecides`:76 | convert-to-Rust-corpus | Violation of Exploration Evidence Proves Only The Overrides The Cluster Decides |
| `TestRuntimeExplorationOverrideIsCompletedByHostController`:95 | convert-to-Rust-corpus | Violation of Runtime Exploration Override Is Completed By Host Controller |

## tools/gomad/internal/model/cluster/fault_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestFaultPlanCanonicalRoundTripAndDetachedInputs`:11 | convert-to-Rust-corpus | Violation of Fault Plan Canonical Round Trip And Detached Inputs |
| `TestFaultPlanIdentityChangesWithEverySemanticField`:39 | convert-to-Rust-corpus | Violation of Fault Plan Identity Changes With Every Semantic Field |
| `TestFaultPlanIdentityBindsPriorTargetReference`:72 | convert-to-Rust-corpus | Violation of Fault Plan Identity Binds Prior Target Reference |
| `TestFaultMatchingTreatsEmptyFieldsAsWildcards`:86 | convert-to-Rust-corpus | Violation of Fault Matching Treats Empty Fields As Wildcards |
| `TestFaultPlanRejectsInvalidShapesAndNonCanonicalInput`:98 | convert-to-Rust-corpus | Violation of Fault Plan Rejects Invalid Shapes And Non Canonical Input |
| `TestFaultPlanCapacity`:126 | convert-to-Rust-corpus | Violation of Fault Plan Capacity |

## tools/gomad/internal/model/cluster/implementation_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestImplementationIdentityBindsAllSemanticSources`:13 | convert-to-Rust-corpus | Violation of Implementation Identity Binds All Semantic Sources |

## tools/gomad/internal/model/cluster/inspect_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestInspectClusterRecordProducesStableJSONAndTextProjection`:10 | convert-to-Rust-corpus | Violation of Inspect Cluster Record Produces Stable JSONAnd Text Projection |
| `TestInspectClusterRecordRejectsInvalidEvidence`:56 | convert-to-Rust-corpus | Violation of Inspect Cluster Record Rejects Invalid Evidence |

## tools/gomad/internal/model/cluster/internal/wire/network_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRuntimeNetworkConfigExplicitLimits`:12 | convert-to-Rust-corpus | Violation of Runtime Network Config Explicit Limits |
| `TestRuntimeNetworkWireRecordRoundTrip`:40 | convert-to-Rust-corpus | Violation of Runtime Network Wire Record Round Trip |
| `TestRuntimeNetworkWireRejectsMalformedInput`:83 | convert-to-Rust-corpus | Violation of Runtime Network Wire Rejects Malformed Input |

## tools/gomad/internal/model/cluster/internal/wire/process_generated_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestGeneratedProcessWaitVector`:11 | convert-to-Rust-corpus | Violation of Generated Process Wait Vector |
| `TestGeneratedProcessFrameKinds`:22 | convert-to-Rust-corpus | Violation of Generated Process Frame Kinds |

## tools/gomad/internal/model/cluster/internal/wire/process_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestProcessFrameRejectsMalformedValuesAndInvalidSemantics`:10 | convert-to-Rust-corpus | Violation of Process Frame Rejects Malformed Values And Invalid Semantics |
| `TestProcessFrameRoundTripAndStrictDecode`:29 | convert-to-Rust-corpus | Violation of Process Frame Round Trip And Strict Decode |
| `TestProcessActivationFramesAreValid`:40 | convert-to-Rust-corpus | Violation of Process Activation Frames Are Valid |
| `TestProcessExplorationFramesAreValid`:46 | convert-to-Rust-corpus | Violation of Process Exploration Frames Are Valid |

## tools/gomad/internal/model/cluster/internal/wire/test_helpers_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/cluster/internal/wire/volume_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRuntimeVolumeWireConfigProjectsMountedCapacitiesAndReplay`:10 | convert-to-Rust-corpus | Violation of Runtime Volume Wire Config Projects Mounted Capacities And Replay |
| `TestRuntimeVolumeWireTypedReplayErrorPreservesPartialRecord`:30 | convert-to-Rust-corpus | Violation of Runtime Volume Wire Typed Replay Error Preserves Partial Record |
| `TestRuntimeVolumeWireRecordRoundTrip`:45 | convert-to-Rust-corpus | Violation of Runtime Volume Wire Record Round Trip |
| `TestRuntimeVolumeWireEnumerationAndFrontierRoundTrip`:78 | convert-to-Rust-corpus | Violation of Runtime Volume Wire Enumeration And Frontier Round Trip |
| `TestRuntimeVolumeWireRejectsMalformedInput`:113 | convert-to-Rust-corpus | Violation of Runtime Volume Wire Rejects Malformed Input |

## tools/gomad/internal/model/cluster/listener_history_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRealNetworkListenerRebindHistoryRoundTrips`:13 | convert-to-Rust-corpus | Violation of Real Network Listener Rebind History Round Trips |
| `TestListenerHistoryEqualityGroupsRequireEarlierEntriesClosed`:89 | convert-to-Rust-corpus | Violation of Listener History Equality Groups Require Earlier Entries Closed |

## tools/gomad/internal/model/cluster/oracle_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestStateInvariantFailureIdentityIsSeedIndependent`:9 | convert-to-Rust-corpus | Violation of State Invariant Failure Identity Is Seed Independent |
| `TestSoftAssertionOracleRetainsStaticInstanceSignature`:25 | convert-to-Rust-corpus | Violation of Soft Assertion Oracle Retains Static Instance Signature |
| `TestExactHistoryAndDuplicateLostOracles`:46 | convert-to-Rust-corpus | Violation of Exact History And Duplicate Lost Oracles |
| `TestEventualConvergenceAndOracleEvidenceBounds`:71 | convert-to-Rust-corpus | Violation of Eventual Convergence And Oracle Evidence Bounds |
| `TestHistoryValidationRejectsMutableOrAmbiguousOperations`:92 | convert-to-Rust-corpus | Violation of History Validation Rejects Mutable Or Ambiguous Operations |

## tools/gomad/internal/model/cluster/record_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestClusterRecordCanonicalRoundTripIncludesLimitsAndLimitations`:16 | convert-to-Rust-corpus | Violation of Cluster Record Canonical Round Trip Includes Limits And Limitations |
| `TestClusterRecordRejectsInvalidNetworkLimitsBeforeIdentity`:60 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Invalid Network Limits Before Identity |
| `TestClusterRecordRejectsOmittedNetworkLimits`:91 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Omitted Network Limits |
| `TestClusterRecordReplayBindsNetworkLimits`:114 | convert-to-Rust-corpus | Violation of Cluster Record Replay Binds Network Limits |
| `TestClusterRecordNetworkEndpointOccupancy`:137 | convert-to-Rust-corpus | Violation of Cluster Record Network Endpoint Occupancy |
| `TestClusterRecordNetworkOccupancyAcrossConnections`:180 | convert-to-Rust-corpus | Violation of Cluster Record Network Occupancy Across Connections |
| `TestClusterRecordNetworkSnapshotRejectsInvalidEndpointPorts`:201 | convert-to-Rust-corpus | Violation of Cluster Record Network Snapshot Rejects Invalid Endpoint Ports |
| `TestClusterRecordNetworkListenersUseCanonicalEndpointOrder`:219 | convert-to-Rust-corpus | Violation of Cluster Record Network Listeners Use Canonical Endpoint Order |
| `TestDecodeClusterRecordRejectsRetiredNetworkFields`:284 | convert-to-Rust-corpus | Violation of Decode Cluster Record Rejects Retired Network Fields |
| `TestClusterRecordRejectsPreviousNetworkFormat`:313 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Previous Network Format |
| `TestClusterRecordNetworkHeldTimeoutVocabulary`:324 | convert-to-Rust-corpus | Violation of Cluster Record Network Held Timeout Vocabulary |
| `TestClusterRecordRejectsMalformedHeldTimeoutTransitions`:344 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Malformed Held Timeout Transitions |
| `TestClusterRecordRejectsInconsistentOutputBeforeIdentity`:380 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Inconsistent Output Before Identity |
| `TestProcessClusterEvidenceRejectsInProcessLeakDiagnostics`:408 | convert-to-Rust-corpus | Violation of Process Cluster Evidence Rejects In Process Leak Diagnostics |
| `TestDecodeClusterRecordRejectsUnknownAndTrailingData`:438 | convert-to-Rust-corpus | Violation of Decode Cluster Record Rejects Unknown And Trailing Data |
| `TestClusterRecordRejectsInvalidLifecycleAndTerminalEnums`:445 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Invalid Lifecycle And Terminal Enums |
| `TestValidateSpecRejectsReplayPlanContentWithStaleIdentity`:480 | convert-to-Rust-corpus | Violation of Validate Spec Rejects Replay Plan Content With Stale Identity |
| `TestClusterRecordRejectsCorruptVolumeTransitionAndSnapshotIdentities`:493 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Corrupt Volume Transition And Snapshot Identities |
| `TestClusterRecordV5IncludesControllerEvidenceAndStaticIdentities`:538 | convert-to-Rust-corpus | Violation of Cluster Record V5 Includes Controller Evidence And Static Identities |
| `TestClusterRecordRejectsCorruptControllerEvidenceBeforeOuterIdentity`:598 | convert-to-Rust-corpus | Violation of Cluster Record Rejects Corrupt Controller Evidence Before Outer Identity |

## tools/gomad/internal/model/cluster/registry_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBootRegistry`:13 | convert-to-Rust-corpus | Violation of Boot Registry |
| `TestBootRegistryConcurrentRegistration`:33 | convert-to-Rust-corpus | Violation of Boot Registry Concurrent Registration |

## tools/gomad/internal/model/cluster/runtime_domain_fingerprint_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRuntimeDomainBridgeFingerprints`:17 | convert-to-Rust-corpus | Violation of Runtime Domain Bridge Fingerprints |

## tools/gomad/internal/model/cluster/runtime_domain_stock_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRunFailsClosedWithoutPinnedRuntime`:10 | convert-to-Rust-corpus | Violation of Run Fails Closed Without Pinned Runtime |
| `TestRunRejectsUnavailableProcessBackendBeforeScenario`:31 | convert-to-Rust-corpus | Violation of Run Rejects Unavailable Process Backend Before Scenario |

## tools/gomad/internal/model/cluster/runtime_volume_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRuntimeVolumeTypedReplayError`:10 | convert-to-Rust-corpus | Violation of Runtime Volume Typed Replay Error |

## tools/gomad/internal/model/cluster/scenario_control_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestScenarioChoicePlanCanonicalRoundTripAndDetachedInput`:9 | convert-to-Rust-corpus | Violation of Scenario Choice Plan Canonical Round Trip And Detached Input |
| `TestScenarioChoicePlanRejectsInvalidRankBoundOverrides`:28 | convert-to-Rust-corpus | Violation of Scenario Choice Plan Rejects Invalid Rank Bound Overrides |

## tools/gomad/internal/model/cluster/scenario_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestScenarioSequenceAndRepeatRecordStableActions`:13 | convert-to-Rust-corpus | Violation of Scenario Sequence And Repeat Record Stable Actions |
| `TestScenarioChooseIsDomainSeparatedAndReplayable`:37 | convert-to-Rust-corpus | Violation of Scenario Choose Is Domain Separated And Replayable |
| `TestScenarioBoundedParallelHonorsLimitAndReturnsStableFirstError`:60 | convert-to-Rust-corpus | Violation of Scenario Bounded Parallel Honors Limit And Returns Stable First Error |
| `TestScenarioConstructorsRejectInvalidInputsBeforeExecution`:101 | convert-to-Rust-corpus | Violation of Scenario Constructors Reject Invalid Inputs Before Execution |

## tools/gomad/internal/model/cluster/spec_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestValidateSpec`:13 | convert-to-Rust-corpus | Violation of Validate Spec |
| `TestDecodeSpecRejectsRetiredNetworkDeliveryLimit`:20 | convert-to-Rust-corpus | Violation of Decode Spec Rejects Retired Network Delivery Limit |
| `TestDecodeSpecRejectsPreviousNetworkFormat`:29 | convert-to-Rust-corpus | Violation of Decode Spec Rejects Previous Network Format |
| `TestDefaultLimitsExplicitNetworkDefaults`:38 | convert-to-Rust-corpus | Violation of Default Limits Explicit Network Defaults |
| `TestValidateSpecNetworkLimits`:59 | convert-to-Rust-corpus | Violation of Validate Spec Network Limits |
| `TestValidateSpecNetworkLimitsPreserveValidationPrecedence`:94 | convert-to-Rust-corpus | Violation of Validate Spec Network Limits Preserve Validation Precedence |
| `TestDecodeSpecNetworkLimits`:106 | convert-to-Rust-corpus | Violation of Decode Spec Network Limits |
| `TestValidateSpecRejectsInvalidStructure`:138 | convert-to-Rust-corpus | Violation of Validate Spec Rejects Invalid Structure |
| `TestSpecJSONFieldNames`:225 | convert-to-Rust-corpus | Violation of Spec JSONField Names |
| `TestValidateSpecAllowsZeroSeed`:233 | convert-to-Rust-corpus | Violation of Validate Spec Allows Zero Seed |
| `TestDecodeSpec`:239 | convert-to-Rust-corpus | Violation of Decode Spec |
| `TestDecodeSpecAllowsMaximumMountShape`:261 | convert-to-Rust-corpus | Violation of Decode Spec Allows Maximum Mount Shape |
| `TestValidateSpecCapacityErrors`:292 | convert-to-Rust-corpus | Violation of Validate Spec Capacity Errors |
| `TestValidateSpecRejectsInvalidFaultReferences`:351 | convert-to-Rust-corpus | Violation of Validate Spec Rejects Invalid Fault References |
| `TestValidateSpecRejectsInvalidReplayPlan`:377 | convert-to-Rust-corpus | Violation of Validate Spec Rejects Invalid Replay Plan |
| `TestCapacityErrorSupportsErrorsIs`:416 | convert-to-Rust-corpus | Violation of Capacity Error Supports Errors Is |

## tools/gomad/internal/model/cluster/test_helpers_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/cluster/wasm_inspection_codec_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWASMInspectionBinaryBoundaryRoundTripAndTruncation`:12 | convert-to-Rust-corpus | Violation of WASMInspection Binary Boundary Round Trip And Truncation |
| `TestWASMInspectionBinaryBoundaryRejectsMalformedMaterial`:30 | convert-to-Rust-corpus | Violation of WASMInspection Binary Boundary Rejects Malformed Material |
| `TestWASMInspectionBinaryWitnessProjectsAllRawCounters`:73 | convert-to-Rust-corpus | Violation of WASMInspection Binary Witness Projects All Raw Counters |
| `TestWASMInspectionBinaryEmptyClockRemainsBase64InControlJSON`:93 | convert-to-Rust-corpus | Violation of WASMInspection Binary Empty Clock Remains Base64 In Control JSON |
| `TestWASMInspectionBinaryWitnessAuthenticatesUnprojectedRawBytes`:110 | convert-to-Rust-corpus | Violation of WASMInspection Binary Witness Authenticates Unprojected Raw Bytes |
| `TestWASMInspectionBinaryLifecyclePreservesEveryAction`:117 | convert-to-Rust-corpus | Violation of WASMInspection Binary Lifecycle Preserves Every Action |
| `TestWASMInspectionBinaryEvidenceRequiresVersionedDiscriminators`:161 | convert-to-Rust-corpus | Violation of WASMInspection Binary Evidence Requires Versioned Discriminators |

## tools/gomad/internal/model/cluster/wasm_inspection_commitment_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWASMInspectionCommitmentsUseOwnerDomainsAndLengthFraming`:14 | convert-to-Rust-corpus | Violation of WASMInspection Commitments Use Owner Domains And Length Framing |
| `TestWASMInspectionCommitmentEncodingAndFramingRefusals`:36 | convert-to-Rust-corpus | Violation of WASMInspection Commitment Encoding And Framing Refusals |
| `TestWASMInspectionCommitmentsKeepEveryBoundaryWithinCapacity`:73 | convert-to-Rust-corpus | Violation of WASMInspection Commitments Keep Every Boundary Within Capacity |

## tools/gomad/internal/model/cluster/wasm_inspection_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWASMInspectionRejectsIncompleteMetadata`:15 | convert-to-Rust-corpus | Violation of WASMInspection Rejects Incomplete Metadata |
| `TestWASMInspectionTargetAndStopResolution`:22 | convert-to-Rust-corpus | Violation of WASMInspection Target And Stop Resolution |
| `TestWASMInspectionRegistryAvailabilityAndNames`:57 | convert-to-Rust-corpus | Violation of WASMInspection Registry Availability And Names |
| `TestWASMInspectionAuthenticatesCoordinatesAndMaterial`:183 | convert-to-Rust-corpus | Violation of WASMInspection Authenticates Coordinates And Material |
| `TestWASMInspectionPrefixVerificationRejectsPerturbations`:218 | convert-to-Rust-corpus | Violation of WASMInspection Prefix Verification Rejects Perturbations |
| `TestWASMInspectionIncludesExecutedNeighborsAndExcludesFutureGuests`:249 | convert-to-Rust-corpus | Violation of WASMInspection Includes Executed Neighbors And Excludes Future Guests |
| `TestWASMInspectionRetainsCompleteTerminalCuts`:275 | convert-to-Rust-corpus | Violation of WASMInspection Retains Complete Terminal Cuts |
| `TestWASMInspectionSealedPrefixHasLiveIncarnationsAndExactRegistry`:290 | convert-to-Rust-corpus | Violation of WASMInspection Sealed Prefix Has Live Incarnations And Exact Registry |
| `TestWASMInspectionPrefixDoesNotConsumeFutureTerminalNetworkEncoding`:300 | convert-to-Rust-corpus | Violation of WASMInspection Prefix Does Not Consume Future Terminal Network Encoding |
| `TestWASMInspectionPrefixAuthenticatesConsumedNetworkEncodingAndBytes`:332 | convert-to-Rust-corpus | Violation of WASMInspection Prefix Authenticates Consumed Network Encoding And Bytes |
| `TestWASMInspectionCapacityRefusalPreservesOrdinaryEvidence`:379 | convert-to-Rust-corpus | Violation of WASMInspection Capacity Refusal Preserves Ordinary Evidence |
| `TestWASMInspectionStrictWitnessDecodeAndClone`:390 | convert-to-Rust-corpus | Violation of WASMInspection Strict Witness Decode And Clone |
| `TestWASMInspectionOutputMaterialMatchesRetainedOutput`:404 | convert-to-Rust-corpus | Violation of WASMInspection Output Material Matches Retained Output |

## tools/gomad/internal/model/cluster/wasm_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWASMWorldEncodingAdmissionAndOpaqueBytes`:11 | convert-to-Rust-corpus | Violation of WASMWorld Encoding Admission And Opaque Bytes |
| `TestWASMLiveEnumerationFailsClosedAfterHandleValidation`:31 | convert-to-Rust-corpus | Violation of WASMLive Enumeration Fails Closed After Handle Validation |
| `TestWASMShutdownDoesNotReserveModeledLifecycle`:69 | convert-to-Rust-corpus | Violation of WASMShutdown Does Not Reserve Modeled Lifecycle |
| `TestWASMLifecycleRejectsBeforeExternalEffects`:81 | convert-to-Rust-corpus | Violation of WASMLifecycle Rejects Before External Effects |

## tools/gomad/internal/model/cmd/protocolgen/main_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestProtocolgenExitStatuses`:43 | kept-Go | Violation of Protocolgen Exit Statuses |
| `TestModuleRootSkipsNestedModules`:86 | kept-Go | Violation of Module Root Skips Nested Modules |

## tools/gomad/internal/model/cmd/sqlite-native/main_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestActualStorageInputRejectsWrongBoundsAndChangedPayload`:13 | convert-to-Rust-corpus | Violation of Actual Storage Input Rejects Wrong Bounds And Changed Payload |

## tools/gomad/internal/model/generation/protocol/generate_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestSharedSetIsCurrent`:14 | kept-Go | Violation of Shared Set Is Current |
| `TestSharedSetStaysInTheParentModule`:24 | kept-Go | Violation of Shared Set Stays In The Parent Module |
| `TestSharedOutputsCarryTheProtocolgenHeader`:42 | kept-Go | Violation of Shared Outputs Carry The Protocolgen Header |
| `TestGenerateWritesAndChecksEverySharedOutput`:95 | kept-Go | Violation of Generate Writes And Checks Every Shared Output |
| `TestGenerateReportsSchemaAndTemplateErrors`:158 | kept-Go | Violation of Generate Reports Schema And Template Errors |
| `TestSharedSetIncludesLinkedChoiceCodec`:174 | kept-Go | Violation of Shared Set Includes Linked Choice Codec |

## tools/gomad/internal/model/generation/protocol/process_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestReadProcessSchemaRejectsInvalidDefinition`:10 | kept-Go | Violation of Read Process Schema Rejects Invalid Definition |

## tools/gomad/internal/model/generation/protocol/protocol_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestChoiceImplementationIdentityBindsEveryParentInput`:12 | kept-Go | Violation of Choice Implementation Identity Binds Every Parent Input |
| `TestReadSimulationModelSchemaPinsBoundedOperationVocabulary`:46 | kept-Go | Violation of Read Simulation Model Schema Pins Bounded Operation Vocabulary |
| `TestReadSimulationModelSchemaRequiresDeclaredStallTimeout`:59 | kept-Go | Violation of Read Simulation Model Schema Requires Declared Stall Timeout |
| `TestReadWorldSnapshotSchemaPinsBoundedTransitionVocabulary`:107 | kept-Go | Violation of Read World Snapshot Schema Pins Bounded Transition Vocabulary |
| `TestWorldSnapshotTemplateFollowsSchemaFieldAndTagOrder`:152 | kept-Go | Violation of World Snapshot Template Follows Schema Field And Tag Order |
| `TestRustWorldSnapshotWriterFollowsSchemaFieldAndTagOrder`:186 | kept-Go | Violation of Rust World Snapshot Writer Follows Schema Field And Tag Order |
| `TestReadSchemaRejectsTrailingData`:224 | kept-Go | Violation of Read Schema Rejects Trailing Data |

## tools/gomad/internal/model/generation/protocol/volume_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeSchemaRejectsIncompleteAndAmbiguousLayouts`:10 | kept-Go | Violation of Volume Schema Rejects Incomplete And Ambiguous Layouts |

## tools/gomad/internal/model/iowire/wire_generated_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBootstrapGoldenVector`:11 | kept-Go | Violation of Bootstrap Golden Vector |
| `TestBootstrapSeedAtRuntimeOffset`:28 | kept-Go | Violation of Bootstrap Seed At Runtime Offset |
| `TestMountGoldenVectors`:37 | kept-Go | Violation of Mount Golden Vectors |
| `TestTranscriptGoldenVectors`:75 | kept-Go | Violation of Transcript Golden Vectors |
| `TestDecodersRejectReservedBytesAndEnums`:118 | kept-Go | Violation of Decoders Reject Reserved Bytes And Enums |
| `TestReadMountResponseRejectsFileWithChildren`:150 | kept-Go | Violation of Read Mount Response Rejects File With Children |
| `TestDecodersRejectEveryTruncation`:167 | kept-Go | Violation of Decoders Reject Every Truncation |
| `TestMountDecodersRejectOversizedLengthsBeforeAllocation`:198 | kept-Go | Violation of Mount Decoders Reject Oversized Lengths Before Allocation |

## tools/gomad/internal/model/network/adapter_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRequestErrorSerializationIsClosedAndPreservesMessages`:20 | convert-to-Rust-corpus | Violation of Request Error Serialization Is Closed And Preserves Messages |
| `TestRequestStallTimeoutMessageIsDistinctFromCallerDeadline`:38 | convert-to-Rust-corpus | Violation of Request Stall Timeout Message Is Distinct From Caller Deadline |
| `TestProjectedWaitBoundaryPreservesUnrelatedNativePanic`:47 | convert-to-Rust-corpus | Violation of Projected Wait Boundary Preserves Unrelated Native Panic |
| `TestModelRejectsInvalidExplicitNetworkLimits`:85 | convert-to-Rust-corpus | Violation of Model Rejects Invalid Explicit Network Limits |
| `TestModelHeldTimeoutReplayAdmission`:99 | convert-to-Rust-corpus | Violation of Model Held Timeout Replay Admission |
| `TestModelRejectsUnknownTransitionBeforeCommit`:131 | convert-to-Rust-corpus | Violation of Model Rejects Unknown Transition Before Commit |
| `TestModelReplayPreflightRejectsUnboundSnapshot`:146 | convert-to-Rust-corpus | Violation of Model Replay Preflight Rejects Unbound Snapshot |
| `TestModelReplayPreflightChecksSealedOccupancyAndOwnsInputs`:184 | convert-to-Rust-corpus | Violation of Model Replay Preflight Checks Sealed Occupancy And Owns Inputs |
| `TestStreamDelayDrainHalfClose`:247 | convert-to-Rust-corpus | Violation of Stream Delay Drain Half Close |
| `TestPartitionQueuedHoldBlockedDialCancelAndHeal`:284 | convert-to-Rust-corpus | Violation of Partition Queued Hold Blocked Dial Cancel And Heal |
| `TestQueueBackpressureCapacityDeadlineAndStaleDomain`:331 | convert-to-Rust-corpus | Violation of Queue Backpressure Capacity Deadline And Stale Domain |
| `TestGeneratedOperationSequencesExactNativeReplay`:388 | convert-to-Rust-corpus | Violation of Generated Operation Sequences Exact Native Replay |
| `TestDelayedReadDeadlineAndReset`:531 | convert-to-Rust-corpus | Violation of Delayed Read Deadline And Reset |
| `TestPendingRequestOwnsBytesAndCapacityFailures`:557 | convert-to-Rust-corpus | Violation of Pending Request Owns Bytes And Capacity Failures |
| `TestRevokeDiscardsDeadIncarnationReadinessAndFinishRejectsHandles`:603 | convert-to-Rust-corpus | Violation of Revoke Discards Dead Incarnation Readiness And Finish Rejects Handles |

## tools/gomad/internal/model/network/corpus_admission_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/network/corpus_healed_epoch_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNetworkHealedEpochLiteralGuards`:12 | convert-to-Rust-corpus | Violation of Network Healed Epoch Literal Guards |

## tools/gomad/internal/model/network/corpus_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestListenerHistoryCorporaExactReplay`:507 | convert-to-Rust-corpus | Violation of Listener History Corpora Exact Replay |
| `TestNetworkCorpusIsCurrent`:1020 | convert-to-Rust-corpus | Violation of Network Corpus Is Current |
| `TestNetworkCorpusSerializedReplay`:1100 | convert-to-Rust-corpus | Violation of Network Corpus Serialized Replay |
| `TestNetworkCorpusDetachesMutableInputs`:1119 | convert-to-Rust-corpus | Violation of Network Corpus Detaches Mutable Inputs |

## tools/gomad/internal/model/network/stall_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestModelExecutionClockCeilingRejectsAdmissionAtomically`:11 | convert-to-Rust-corpus | Violation of Model Execution Clock Ceiling Rejects Admission Atomically |
| `TestModelExecutionClockCeilingAllowsStrictExpiryBoundary`:61 | convert-to-Rust-corpus | Violation of Model Execution Clock Ceiling Allows Strict Expiry Boundary |
| `TestModelClockCeilingRejectsCallerDeadlineAndAdvanceBeforeMutation`:97 | convert-to-Rust-corpus | Violation of Model Clock Ceiling Rejects Caller Deadline And Advance Before Mutation |
| `TestNativeDeadlineArithmeticIncludesAbsoluteEpoch`:123 | convert-to-Rust-corpus | Violation of Native Deadline Arithmetic Includes Absolute Epoch |
| `TestModelClockCeilingTighteningChecksAllAdmittedTimes`:140 | convert-to-Rust-corpus | Violation of Model Clock Ceiling Tightening Checks All Admitted Times |
| `TestModelPartitionHoldsExistingAndNewFIFO`:182 | convert-to-Rust-corpus | Violation of Model Partition Holds Existing And New FIFO |
| `TestModelAutonomousStallExpiryUsesReservedTerminalSlot`:238 | convert-to-Rust-corpus | Violation of Model Autonomous Stall Expiry Uses Reserved Terminal Slot |
| `TestModelStallOrderingIdleAndEpochInvalidation`:287 | convert-to-Rust-corpus | Violation of Model Stall Ordering Idle And Epoch Invalidation |
| `TestModelStallOverflowRejectsBeforeWriteOrTopologyMutation`:368 | convert-to-Rust-corpus | Violation of Model Stall Overflow Rejects Before Write Or Topology Mutation |
| `TestModelFullCloseCancelsHeldSuffixButPreservesReceivedTail`:397 | convert-to-Rust-corpus | Violation of Model Full Close Cancels Held Suffix But Preserves Received Tail |
| `TestModelCloseAndHalfClosePreserveEnabledDelayAndHealedFIFO`:430 | convert-to-Rust-corpus | Violation of Model Close And Half Close Preserve Enabled Delay And Healed FIFO |
| `TestModelFullCloseBeforePartitionDoesNotStrandDelayedEOF`:477 | convert-to-Rust-corpus | Violation of Model Full Close Before Partition Does Not Strand Delayed EOF |
| `TestModelStalledPartialWritePreservesAcceptedPrefix`:501 | convert-to-Rust-corpus | Violation of Model Stalled Partial Write Preserves Accepted Prefix |
| `TestModelExplicitResetSettlesHeldReservation`:525 | convert-to-Rust-corpus | Violation of Model Explicit Reset Settles Held Reservation |
| `TestModelFullCloseBeforePartitionRetainsReceivedTail`:545 | convert-to-Rust-corpus | Violation of Model Full Close Before Partition Retains Received Tail |
| `TestModelRejectedPartitionAfterFullClosePreservesEnabledQueue`:574 | convert-to-Rust-corpus | Violation of Model Rejected Partition After Full Close Preserves Enabled Queue |
| `TestModelHalfCloseBeforePartitionRetainsDelayedFIFO`:607 | convert-to-Rust-corpus | Violation of Model Half Close Before Partition Retains Delayed FIFO |
| `TestModelSimultaneousPairExpiryReplay`:635 | convert-to-Rust-corpus | Violation of Model Simultaneous Pair Expiry Replay |
| `TestModelPartitionedDialStallIndependentFromCallerDeadline`:684 | convert-to-Rust-corpus | Violation of Model Partitioned Dial Stall Independent From Caller Deadline |
| `TestModelPartitionedDialHealRepartitionAndRedundantDisable`:715 | convert-to-Rust-corpus | Violation of Model Partitioned Dial Heal Repartition And Redundant Disable |
| `TestModelPendingDialReservationAndOverflowRejectBeforeWork`:759 | convert-to-Rust-corpus | Violation of Model Pending Dial Reservation And Overflow Reject Before Work |
| `TestModelPendingDialRevocationReleasesReservationAndStaleWork`:784 | convert-to-Rust-corpus | Violation of Model Pending Dial Revocation Releases Reservation And Stale Work |
| `TestModelSimulationSelfDialRejectsBeforePendingAdmission`:813 | convert-to-Rust-corpus | Violation of Model Simulation Self Dial Rejects Before Pending Admission |
| `TestModelPayloadFreeHistoryReexecutesActualBytesAndRejectsLengthMismatch`:831 | convert-to-Rust-corpus | Violation of Model Payload Free History Reexecutes Actual Bytes And Rejects Length Mismatch |
| `TestModelReceivedTailSurvivesPartitionAndRepartitionDelaysSuffix`:878 | convert-to-Rust-corpus | Violation of Model Received Tail Survives Partition And Repartition Delays Suffix |
| `TestModelHeldAdmissionReservesTerminalHistoryAtomically`:921 | convert-to-Rust-corpus | Violation of Model Held Admission Reserves Terminal History Atomically |
| `TestModelPartitionReservationFailureLeavesQueuedBytesAndTopology`:950 | convert-to-Rust-corpus | Violation of Model Partition Reservation Failure Leaves Queued Bytes And Topology |

## tools/gomad/internal/model/softassert/softassert_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestObserveStaticMessagesFromBothStreams`:8 | convert-to-Rust-corpus | Violation of Observe Static Messages From Both Streams |
| `TestObserveIncompleteEvidenceNeverPasses`:49 | convert-to-Rust-corpus | Violation of Observe Incomplete Evidence Never Passes |

## tools/gomad/internal/model/strictjson/decode_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestDecodeRetainsStdlibNumericDuration`:11 | kept-Go | Violation of Decode Retains Stdlib Numeric Duration |
| `TestDecodeRejectsStrictSyntaxAndTypedUnknownMembers`:54 | kept-Go | Violation of Decode Rejects Strict Syntax And Typed Unknown Members |
| `TestDecodeAllowsMapAndRawProjectionMembers`:82 | kept-Go | Violation of Decode Allows Map And Raw Projection Members |
| `TestDecodeAcceptsOrdinaryJSONSpellingAndCaseAliases`:96 | kept-Go | Violation of Decode Accepts Ordinary JSONSpelling And Case Aliases |
| `TestDecodeAcceptsStdlibByteArrayEncoding`:107 | kept-Go | Violation of Decode Accepts Stdlib Byte Array Encoding |
| `TestDecodeRetainsTypedNumericBounds`:119 | kept-Go | Violation of Decode Retains Typed Numeric Bounds |
| `TestDecodeAcceptsPairedAndRejectsUnpairedUTF16Surrogates`:126 | kept-Go | Violation of Decode Accepts Paired And Rejects Unpaired UTF16 Surrogates |
| `TestDecodeReportsLegacySyntaxErrors`:139 | kept-Go | Violation of Decode Reports Legacy Syntax Errors |
| `TestDecodeFirstReportsTypedErrorsBeforeTrailingData`:148 | kept-Go | Violation of Decode First Reports Typed Errors Before Trailing Data |
| `TestCheckSyntaxPreservesLibraryErrorsAndCallerDepth`:175 | kept-Go | Violation of Check Syntax Preserves Library Errors And Caller Depth |

## tools/gomad/internal/model/strictjson/identity_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestImplementationIdentityBindsSourceAndDependencyPin`:11 | kept-Go | Violation of Implementation Identity Binds Source And Dependency Pin |
| `TestDependencyPinMatchesModuleFiles`:38 | kept-Go | Violation of Dependency Pin Matches Module Files |

## tools/gomad/internal/model/volume/conformance_state_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestConformanceRecoveryInodeHeadroomMatchesActualGoRebuild`:14 | convert-to-Rust-corpus | Violation of Conformance Recovery Inode Headroom Matches Actual Go Rebuild |
| `TestConformanceStateExportsFullRecoveryState`:94 | convert-to-Rust-corpus | Violation of Conformance State Exports Full Recovery State |
| `TestConformanceStateDoesNotMutateOrAliasModel`:132 | convert-to-Rust-corpus | Violation of Conformance State Does Not Mutate Or Alias Model |
| `TestConformanceStateRetainsSyncedOrphanAndActualOwnership`:165 | convert-to-Rust-corpus | Violation of Conformance State Retains Synced Orphan And Actual Ownership |

## tools/gomad/internal/model/volume/identity_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestImplementationIdentityDoesNotDispatchMissingSourceError`:40 | convert-to-Rust-corpus | Violation of Implementation Identity Does Not Dispatch Missing Source Error |
| `TestImplementationIdentityBindsModelAndHostAdapters`:47 | convert-to-Rust-corpus | Violation of Implementation Identity Binds Model And Host Adapters |
| `TestImplementationIdentityRejectsMissingCompiledSources`:69 | convert-to-Rust-corpus | Violation of Implementation Identity Rejects Missing Compiled Sources |
| `TestImplementationInventoryNamesEveryEmbeddedSource`:76 | convert-to-Rust-corpus | Violation of Implementation Inventory Names Every Embedded Source |

## tools/gomad/internal/model/volume/internal/hashwire/hash_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestProjectedHasherMatchesSHA256AcrossBlockBoundaries`:8 | convert-to-Rust-corpus | Violation of Projected Hasher Matches SHA256 Across Block Boundaries |

## tools/gomad/internal/model/volume/metadata_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestMetadataReturnsDetachedAttributesWithoutPayload`:13 | convert-to-Rust-corpus | Violation of Metadata Returns Detached Attributes Without Payload |
| `TestMetadataRejectsClosedAndStaleHandles`:57 | convert-to-Rust-corpus | Violation of Metadata Rejects Closed And Stale Handles |
| `TestMetadataAllocationsDoNotCopyLargeFile`:83 | convert-to-Rust-corpus | Violation of Metadata Allocations Do Not Copy Large File |

## tools/gomad/internal/model/volume/owners_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCapturedSnapshotOwnsInputsAndDetachedLookupResults`:27 | convert-to-Rust-corpus | Violation of Captured Snapshot Owns Inputs And Detached Lookup Results |
| `TestCapturedSnapshotBoundsAndPerFilesystemLookupAccounting`:62 | convert-to-Rust-corpus | Violation of Captured Snapshot Bounds And Per Filesystem Lookup Accounting |
| `TestOwnedClockAndRecorderPreserveNativeReferenceState`:93 | convert-to-Rust-corpus | Violation of Owned Clock And Recorder Preserve Native Reference State |
| `TestRecorderBoundAndFaultRejectBeforeMutation`:139 | convert-to-Rust-corpus | Violation of Recorder Bound And Fault Reject Before Mutation |

## tools/gomad/internal/model/volume/recovery_image_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCrashRecoveryImageAdmissionCorpus`:16 | convert-to-Rust-corpus | Violation of Crash Recovery Image Admission Corpus |
| `TestCrashRecoveryImagePreservesOriginalEnumeration`:48 | convert-to-Rust-corpus | Violation of Crash Recovery Image Preserves Original Enumeration |
| `TestCrashRecoveryImageRefusesMalformedPendingState`:83 | convert-to-Rust-corpus | Violation of Crash Recovery Image Refuses Malformed Pending State |

## tools/gomad/internal/model/volume/types_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamedMirrorsMatchAuthoritativeWireTypes`:13 | convert-to-Rust-corpus | Violation of Named Mirrors Match Authoritative Wire Types |

## tools/gomad/internal/model/volume/volume_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVisibleDataFileSyncAndDirectorySync`:108 | convert-to-Rust-corpus | Violation of Visible Data File Sync And Directory Sync |
| `TestCrashEnumerationIsResumableAndSnapshotsAreDetached`:146 | convert-to-Rust-corpus | Violation of Crash Enumeration Is Resumable And Snapshots Are Detached |
| `TestCrashEnforcesWriteResizeAndNamespaceOrdering`:195 | convert-to-Rust-corpus | Violation of Crash Enforces Write Resize And Namespace Ordering |
| `TestDurableCloneDoesNotRevokeOwnerOrShareMutations`:255 | convert-to-Rust-corpus | Violation of Durable Clone Does Not Revoke Owner Or Share Mutations |
| `TestFaultObserverAndCapacityRejectBeforeMutation`:290 | convert-to-Rust-corpus | Violation of Fault Observer And Capacity Reject Before Mutation |
| `TestCrashEnumerationWorkBudgetResumesAndRejectsStaleCursor`:323 | convert-to-Rust-corpus | Violation of Crash Enumeration Work Budget Resumes And Rejects Stale Cursor |
| `TestLifecycleRestartDropsVolatileAndRevokesOldFS`:346 | convert-to-Rust-corpus | Violation of Lifecycle Restart Drops Volatile And Revokes Old FS |

## tools/gomad/internal/model/volume/work_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestTinyWorkResumesWithExactlyTheUnrestrictedStatesAndOrder`:10 | convert-to-Rust-corpus | Violation of Tiny Work Resumes With Exactly The Unrestricted States And Order |
| `TestCrashEnumerationRejectsZeroWork`:65 | convert-to-Rust-corpus | Violation of Crash Enumeration Rejects Zero Work |

## tools/gomad/internal/model/wasi/broker_lifecycle_corpus_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBrokerLifecycleCorpusIsCurrent`:251 | convert-to-Rust-corpus | Violation of Broker Lifecycle Corpus Is Current |
| `TestBrokerLifecycleCorpus`:296 | convert-to-Rust-corpus | Violation of Broker Lifecycle Corpus |

## tools/gomad/internal/model/wasi/broker_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBrokerOrdersAreCurrent`:87 | convert-to-Rust-corpus | Violation of Broker Orders Are Current |
| `TestBrokerOrdersRustNetworkEvidenceRebuildsGoDocuments`:118 | convert-to-Rust-corpus | Violation of Broker Orders Rust Network Evidence Rebuilds Go Documents |

## tools/gomad/internal/model/wasi/closed_cluster_corpus_support_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/wasi/cluster_network_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestClusterNetworkRustEvidenceRebuildsTheGoBrokerDocument`:75 | convert-to-Rust-corpus | Violation of Cluster Network Rust Evidence Rebuilds The Go Broker Document |
| `TestClusterNetworkDecoderRefusesMalformedEvidence`:95 | convert-to-Rust-corpus | Violation of Cluster Network Decoder Refuses Malformed Evidence |
| `TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses`:155 | convert-to-Rust-corpus | Violation of Cluster Network Frame Refuses What The Go Broker Refuses |

## tools/gomad/internal/model/wasi/corpus_assertion_preservation_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestBrokerLifecycleCorpusPreservesIdleReturn`:12 | convert-to-Rust-corpus | Violation of Broker Lifecycle Corpus Preserves Idle Return |
| `TestBrokerLifecycleCorpusPreservesCreationCapacityIdentity`:37 | convert-to-Rust-corpus | Violation of Broker Lifecycle Corpus Preserves Creation Capacity Identity |
| `TestBrokerLifecycleCorpusDistinguishesSameTextCapacityErrors`:67 | convert-to-Rust-corpus | Violation of Broker Lifecycle Corpus Distinguishes Same Text Capacity Errors |
| `TestNetworkClusterCorpusPreservesExactResourceExhaustion`:93 | convert-to-Rust-corpus | Violation of Network Cluster Corpus Preserves Exact Resource Exhaustion |

## tools/gomad/internal/model/wasi/debug_output_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestDebugHeapProfileOpensInPprof`:33 | convert-to-Rust-corpus | Violation of Debug Heap Profile Opens In Pprof |
| `TestDebugLibraryRecordsPreserveFailuresAndRawStrings`:73 | convert-to-Rust-corpus | Violation of Debug Library Records Preserve Failures And Raw Strings |

## tools/gomad/internal/model/wasi/environment_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceRightsOffsetsDirectoryCookiesAndClosure`:34 | convert-to-Rust-corpus | Violation of Namespace Rights Offsets Directory Cookies And Closure |
| `TestNamespaceRenameRejectsMissingDirectoryDestinationWithoutMutation`:103 | convert-to-Rust-corpus | Violation of Namespace Rename Rejects Missing Directory Destination Without Mutation |
| `TestCapturedNamespaceNeverReopensHostAndInfersAbsence`:118 | convert-to-Rust-corpus | Violation of Captured Namespace Never Reopens Host And Infers Absence |
| `TestCapturedNamespaceAllowsStockReadOnlyOpenWithoutMutation`:152 | convert-to-Rust-corpus | Violation of Captured Namespace Allows Stock Read Only Open Without Mutation |
| `TestCapturedNamespaceRejectsWritableDescendantAndAllowsAncestor`:210 | convert-to-Rust-corpus | Violation of Captured Namespace Rejects Writable Descendant And Allows Ancestor |
| `TestCapturedArgumentsEnvironmentAndEntropy`:243 | convert-to-Rust-corpus | Violation of Captured Arguments Environment And Entropy |
| `TestEnvironmentRejectsAmbiguousInputsAndMalformedCalls`:275 | convert-to-Rust-corpus | Violation of Environment Rejects Ambiguous Inputs And Malformed Calls |
| `TestEnvironmentRejectsInvalidUTF8IdentityInputs`:299 | convert-to-Rust-corpus | Violation of Environment Rejects Invalid UTF8 Identity Inputs |
| `TestEnvironmentPreservesMultibyteConfigAndRawRuntimeFilenames`:327 | convert-to-Rust-corpus | Violation of Environment Preserves Multibyte Config And Raw Runtime Filenames |
| `TestExploratoryClockAndPollDoNotUseWallTimeOrYieldTicks`:356 | convert-to-Rust-corpus | Violation of Exploratory Clock And Poll Do Not Use Wall Time Or Yield Ticks |
| `TestCapacityRejectsMutationAndBoundsAllGrowingState`:386 | convert-to-Rust-corpus | Violation of Capacity Rejects Mutation And Bounds All Growing State |
| `TestPathOpenRequiresCreateRightAndPreservesDirectoryRequirement`:442 | convert-to-Rust-corpus | Violation of Path Open Requires Create Right And Preserves Directory Requirement |
| `TestOfferedOutputChargesRejectedWritesAndEntropyBufferReservation`:465 | convert-to-Rust-corpus | Violation of Offered Output Charges Rejected Writes And Entropy Buffer Reservation |
| `TestEnvironmentRejectsWireFrameCapacityBeforeRecording`:485 | convert-to-Rust-corpus | Violation of Environment Rejects Wire Frame Capacity Before Recording |
| `TestCapacityIncludesCapturedStdinInFilesystemBudget`:501 | convert-to-Rust-corpus | Violation of Capacity Includes Captured Stdin In Filesystem Budget |

## tools/gomad/internal/model/wasi/helper_corpus_closed_refusal_literals_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/wasi/helper_corpus_closed_refusal_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestHelperCorpusClosedRefusalsRejectWrongStage`:8 | convert-to-Rust-corpus | Violation of Helper Corpus Closed Refusals Reject Wrong Stage |
| `TestHelperCorpusClosedNonnegativeCasesDoNotAcquireRefusalGuards`:45 | convert-to-Rust-corpus | Violation of Helper Corpus Closed Nonnegative Cases Do Not Acquire Refusal Guards |
| `TestHelperCorpusClosedRefusalBoundaries`:67 | convert-to-Rust-corpus | Violation of Helper Corpus Closed Refusal Boundaries |
| `TestHelperCorpusClosedClusterRefusalsSelectAffectedInstance`:153 | convert-to-Rust-corpus | Violation of Helper Corpus Closed Cluster Refusals Select Affected Instance |

## tools/gomad/internal/model/wasi/helper_corpus_refusal_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestHelperCorpusRefusalLiteralsRejectWrongStageMessages`:143 | convert-to-Rust-corpus | Violation of Helper Corpus Refusal Literals Reject Wrong Stage Messages |
| `TestHelperCorpusAdmissionRefusalsRejectWrongStageMessages`:180 | convert-to-Rust-corpus | Violation of Helper Corpus Admission Refusals Reject Wrong Stage Messages |
| `TestHelperCorpusStorageUnknownFieldTermination`:205 | convert-to-Rust-corpus | Violation of Helper Corpus Storage Unknown Field Termination |

## tools/gomad/internal/model/wasi/helper_corpus_residual_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestHelperCorpusResidualLiteralGuards`:153 | convert-to-Rust-corpus | Violation of Helper Corpus Residual Literal Guards |

## tools/gomad/internal/model/wasi/helper_corpus_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestHelperCorpus`:47 | convert-to-Rust-corpus | Violation of Helper Corpus |
| `TestHelperCorpusWriteRawLinesPreserveLiteralKeyOrderAndBytes`:228 | convert-to-Rust-corpus | Violation of Helper Corpus Write Raw Lines Preserve Literal Key Order And Bytes |
| `TestHelperCorpusInputFrames`:568 | convert-to-Rust-corpus | Violation of Helper Corpus Input Frames |

## tools/gomad/internal/model/wasi/helper_corpus_wire_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestHelperCorpusRawWirePreservesDuplicateKeys`:8 | convert-to-Rust-corpus | Violation of Helper Corpus Raw Wire Preserves Duplicate Keys |
| `TestHelperCorpusRawWireRefusesExtraFrames`:32 | convert-to-Rust-corpus | Violation of Helper Corpus Raw Wire Refuses Extra Frames |

## tools/gomad/internal/model/wasi/helper_model_divergence_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestHelperOrdinaryDivergenceDoesNotRequireInspection`:11 | convert-to-Rust-corpus | Violation of Helper Ordinary Divergence Does Not Require Inspection |
| `TestHelperRejectsUnsealedSuccessfulInspection`:26 | convert-to-Rust-corpus | Violation of Helper Rejects Unsealed Successful Inspection |

## tools/gomad/internal/model/wasi/helper_model_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCapturedEntropySharedGoldenVectors`:22 | convert-to-Rust-corpus | Violation of Captured Entropy Shared Golden Vectors |
| `TestHelperInputCarriageLiteralAndMaximumFragmentation`:44 | convert-to-Rust-corpus | Violation of Helper Input Carriage Literal And Maximum Fragmentation |
| `TestHostileHelperCallbackReplyLeavesEnvironmentByteIdentical`:118 | convert-to-Rust-corpus | Violation of Hostile Helper Callback Reply Leaves Environment Byte Identical |
| `TestHelperBridgeModelLiteralPayloadAndState`:179 | convert-to-Rust-corpus | Violation of Helper Bridge Model Literal Payload And State |
| `TestHelperBridgeModelRejectsStateBeforeMutationAndKeepsOperationError`:220 | convert-to-Rust-corpus | Violation of Helper Bridge Model Rejects State Before Mutation And Keeps Operation Error |
| `TestHelperBridgeSplitOutputStateAdmission`:252 | convert-to-Rust-corpus | Violation of Helper Bridge Split Output State Admission |
| `TestHelperBridgeModelPreservesConcreteStorageCrashAndTransportInterrupt`:284 | convert-to-Rust-corpus | Violation of Helper Bridge Model Preserves Concrete Storage Crash And Transport Interrupt |
| `TestHelperBridgeTransportChunkOrderAndWriteErrorPrecedence`:314 | convert-to-Rust-corpus | Violation of Helper Bridge Transport Chunk Order And Write Error Precedence |
| `TestHelperBridgeReplayTraceFrameBoundary`:366 | convert-to-Rust-corpus | Violation of Helper Bridge Replay Trace Frame Boundary |
| `TestHelperModelRejectsRetiredNetworkEvidence`:430 | convert-to-Rust-corpus | Violation of Helper Model Rejects Retired Network Evidence |

## tools/gomad/internal/model/wasi/helper_profile_stderr_capture_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestHelperCorpusRejectsLLVMProfileStderrOnSuccessfulExit`:10 | convert-to-Rust-corpus | Violation of Helper Corpus Rejects LLVMProfile Stderr On Successful Exit |

## tools/gomad/internal/model/wasi/helper_profile_stderr_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/wasi/identity_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestImplementationIdentityBindsStorageEvidenceSources`:9 | convert-to-Rust-corpus | Violation of Implementation Identity Binds Storage Evidence Sources |
| `TestImplementationIdentityBindsCompiledModelDigest`:37 | convert-to-Rust-corpus | Violation of Implementation Identity Binds Compiled Model Digest |
| `TestImplementationIdentityFailsClosedOnMissingSource`:44 | convert-to-Rust-corpus | Violation of Implementation Identity Fails Closed On Missing Source |
| `TestImplementationIdentityBindsStrictJSONOwner`:50 | convert-to-Rust-corpus | Violation of Implementation Identity Binds Strict JSONOwner |

## tools/gomad/internal/model/wasi/measure_environment_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/wasi/measure_process_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/wasi/namespace_conformance_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/wasi/namespace_host_adapter_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceHostAdapterConformanceCorpus`:172 | convert-to-Rust-corpus | Violation of Namespace Host Adapter Conformance Corpus |
| `TestNamespacePublicCorpusRejectsChangedExpectedBytes`:195 | convert-to-Rust-corpus | Violation of Namespace Public Corpus Rejects Changed Expected Bytes |

## tools/gomad/internal/model/wasi/namespace_image_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceImageConformanceCorpus`:36 | convert-to-Rust-corpus | Violation of Namespace Image Conformance Corpus |

## tools/gomad/internal/model/wasi/namespace_image_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceImageLiteralAndWritableRootRefusal`:19 | convert-to-Rust-corpus | Violation of Namespace Image Literal And Writable Root Refusal |
| `TestNamespaceImageCanonicalCreationOrderAndFragmentation`:55 | convert-to-Rust-corpus | Violation of Namespace Image Canonical Creation Order And Fragmentation |
| `TestNamespaceImagePreservesMetadataAndWriterRefusals`:109 | convert-to-Rust-corpus | Violation of Namespace Image Preserves Metadata And Writer Refusals |
| `TestNamespaceImageCapturedOwnershipAndPreflightPriority`:144 | convert-to-Rust-corpus | Violation of Namespace Image Captured Ownership And Preflight Priority |
| `TestNamespaceImagePreInstantiationInvalidReceipt`:206 | convert-to-Rust-corpus | Violation of Namespace Image Pre Instantiation Invalid Receipt |
| `TestNamespaceImageCarriageFollowsAuthorizationAndReplay`:221 | convert-to-Rust-corpus | Violation of Namespace Image Carriage Follows Authorization And Replay |

## tools/gomad/internal/model/wasi/namespace_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceConformanceCorpus`:84 | convert-to-Rust-corpus | Violation of Namespace Conformance Corpus |
| `TestNamespaceFilesystemCounterResultDoesNotAliasModel`:785 | convert-to-Rust-corpus | Violation of Namespace Filesystem Counter Result Does Not Alias Model |
| `TestNamespaceOwnedCallbackRefusedBeforeAdoptingRemoteState`:796 | convert-to-Rust-corpus | Violation of Namespace Owned Callback Refused Before Adopting Remote State |
| `TestNamespaceUnsupportedTranscriptAdmissionCounters`:814 | convert-to-Rust-corpus | Violation of Namespace Unsupported Transcript Admission Counters |
| `TestNamespaceUnsupportedTranscriptExactAdmissionCounters`:838 | convert-to-Rust-corpus | Violation of Namespace Unsupported Transcript Exact Admission Counters |
| `TestNamespaceHelperOutputRefusesInconsistentOwnership`:874 | convert-to-Rust-corpus | Violation of Namespace Helper Output Refuses Inconsistent Ownership |
| `TestNamespacePartialEvidenceResultDoesNotAliasTransport`:887 | convert-to-Rust-corpus | Violation of Namespace Partial Evidence Result Does Not Alias Transport |

## tools/gomad/internal/model/wasi/namespace_transcript_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceTranscriptConformanceCorpus`:12 | convert-to-Rust-corpus | Violation of Namespace Transcript Conformance Corpus |

## tools/gomad/internal/model/wasi/namespace_transcript_public_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespacePublicTranscriptRejectsChangedExpectedBytes`:188 | convert-to-Rust-corpus | Violation of Namespace Public Transcript Rejects Changed Expected Bytes |

## tools/gomad/internal/model/wasi/namespace_transcript_reservation_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceTranscriptReservationRejectsChangedNormalBytes`:62 | convert-to-Rust-corpus | Violation of Namespace Transcript Reservation Rejects Changed Normal Bytes |

## tools/gomad/internal/model/wasi/namespace_wire_conformance_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNamespaceWireConformanceCorpus`:13 | convert-to-Rust-corpus | Violation of Namespace Wire Conformance Corpus |

## tools/gomad/internal/model/wasi/network_adapter_corpus_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNetworkAdapterCorpusIsCurrent`:455 | convert-to-Rust-corpus | Violation of Network Adapter Corpus Is Current |
| `TestNetworkAdapterRuntimeIdleCallbackVersionAndDeadline`:500 | convert-to-Rust-corpus | Violation of Network Adapter Runtime Idle Callback Version And Deadline |
| `TestNetworkAdapterCorpus`:517 | convert-to-Rust-corpus | Violation of Network Adapter Corpus |

## tools/gomad/internal/model/wasi/network_cluster_corpus_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNetworkClusterCorpusIsCurrent`:487 | convert-to-Rust-corpus | Violation of Network Cluster Corpus Is Current |
| `TestNetworkClusterCorpus`:547 | convert-to-Rust-corpus | Violation of Network Cluster Corpus |

## tools/gomad/internal/model/wasi/network_v2_readiness_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNetworkV2ReadinessProbePreservesLegacyUndefinedOpcode`:22 | convert-to-Rust-corpus | Violation of Network V2 Readiness Probe Preserves Legacy Undefined Opcode |
| `TestNetworkV2ReadinessProbe`:34 | convert-to-Rust-corpus | Violation of Network V2 Readiness Probe |
| `TestNetworkV2ReadinessProbeAdvancesDeadline`:63 | convert-to-Rust-corpus | Violation of Network V2 Readiness Probe Advances Deadline |
| `TestNetworkV2ReadinessProbeBounds`:79 | convert-to-Rust-corpus | Violation of Network V2 Readiness Probe Bounds |
| `TestNetworkV2ReadinessProbeRejectsInvalidUnchanged`:99 | convert-to-Rust-corpus | Violation of Network V2 Readiness Probe Rejects Invalid Unchanged |

## tools/gomad/internal/model/wasi/network_v2_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestNetworkV2StateRejectsInvalidProfile`:56 | convert-to-Rust-corpus | Violation of Network V2 State Rejects Invalid Profile |

## tools/gomad/internal/model/wasi/protocol_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestTransportInputValidationPrecedenceAndMutation`:12 | convert-to-Rust-corpus | Violation of Transport Input Validation Precedence And Mutation |
| `TestPublicCallRevalidatesChangedInput`:107 | convert-to-Rust-corpus | Violation of Public Call Revalidates Changed Input |
| `TestValidatedFrameOwnsBytesAndOperationIsLazy`:132 | convert-to-Rust-corpus | Violation of Validated Frame Owns Bytes And Operation Is Lazy |
| `TestValidatedFrameRejectsInvalidUTF8`:168 | convert-to-Rust-corpus | Violation of Validated Frame Rejects Invalid UTF8 |
| `TestValidatedNamespaceMatchesPublicReference`:176 | convert-to-Rust-corpus | Violation of Validated Namespace Matches Public Reference |

## tools/gomad/internal/model/wasi/runtime_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestCooperativeFrozenClockPreservesStartupAndStockPolicy`:18 | convert-to-Rust-corpus | Violation of Cooperative Frozen Clock Preserves Startup And Stock Policy |
| `TestCooperativeEnvironmentSeparatesRuntimeControlFromWASIObservations`:35 | convert-to-Rust-corpus | Violation of Cooperative Environment Separates Runtime Control From WASIObservations |
| `TestRuntimeTransportForcesLogicalAlternativesAndConsumesWholeTape`:101 | convert-to-Rust-corpus | Violation of Runtime Transport Forces Logical Alternatives And Consumes Whole Tape |
| `TestRuntimeHeaderV2CreationSitesMatchHelperGoldenAndReplayExactly`:250 | convert-to-Rust-corpus | Violation of Runtime Header V2 Creation Sites Match Helper Golden And Replay Exactly |
| `TestRuntimeDecisionRefusesMalformedHeaderV2Layouts`:340 | convert-to-Rust-corpus | Violation of Runtime Decision Refuses Malformed Header V2 Layouts |

## tools/gomad/internal/model/wasi/storage_projection_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestSQLiteCrashCatalogueUsesOriginalEnumeration`:15 | convert-to-Rust-corpus | Violation of SQLite Crash Catalogue Uses Original Enumeration |

## tools/gomad/internal/model/wasi/storage_selection_order_corpus_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestStorageCrashValidationOrderCorpus`:43 | convert-to-Rust-corpus | Violation of Storage Crash Validation Order Corpus |

## tools/gomad/internal/model/wasi/storage_selection_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestStorageCrashContextAdmission`:15 | convert-to-Rust-corpus | Violation of Storage Crash Context Admission |
| `TestStorageCrashValidationVolumeOrder`:56 | convert-to-Rust-corpus | Violation of Storage Crash Validation Volume Order |
| `TestStorageCrashContextLargeChunkTransport`:98 | convert-to-Rust-corpus | Violation of Storage Crash Context Large Chunk Transport |
| `TestStorageCrashContextEvidenceRefusal`:162 | convert-to-Rust-corpus | Violation of Storage Crash Context Evidence Refusal |
| `TestStorageCrashContextEqualIsExact`:227 | convert-to-Rust-corpus | Violation of Storage Crash Context Equal Is Exact |

## tools/gomad/internal/model/wasi/temporal_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestTemporalConfigPreservesPrototypeBytes`:16 | convert-to-Rust-corpus | Violation of Temporal Config Preserves Prototype Bytes |
| `TestTemporalConfigPoliciesPreserveCompleteBytes`:52 | convert-to-Rust-corpus | Violation of Temporal Config Policies Preserve Complete Bytes |

## tools/gomad/internal/model/wasi/transcript_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestTranscriptUsesVersionedBinaryRecords`:12 | convert-to-Rust-corpus | Violation of Transcript Uses Versioned Binary Records |
| `TestTranscriptClockRunFoldsAndPreservesDigestModes`:24 | convert-to-Rust-corpus | Violation of Transcript Clock Run Folds And Preserves Digest Modes |
| `TestTranscriptClockRunChargesCapacityOnce`:50 | convert-to-Rust-corpus | Violation of Transcript Clock Run Charges Capacity Once |
| `TestTranscriptClockRunDecoderBoundsCountsAndPositions`:72 | convert-to-Rust-corpus | Violation of Transcript Clock Run Decoder Bounds Counts And Positions |
| `TestTranscriptClockRunDigestBindsFrozenCountsKeysAndPositions`:104 | convert-to-Rust-corpus | Violation of Transcript Clock Run Digest Binds Frozen Counts Keys And Positions |
| `TestTranscriptDecoderRejectsMalformedRecords`:129 | convert-to-Rust-corpus | Violation of Transcript Decoder Rejects Malformed Records |
| `TestTranscriptRejectedOperationKeepsExactValidatedInput`:172 | convert-to-Rust-corpus | Violation of Transcript Rejected Operation Keeps Exact Validated Input |
| `TestTranscriptEmptyOperationPreservesUnsupportedRecord`:193 | convert-to-Rust-corpus | Violation of Transcript Empty Operation Preserves Unsupported Record |

## tools/gomad/internal/model/wasi/volume_conformance_boundary_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeStorageEnvelopeCorpus`:37 | convert-to-Rust-corpus | Violation of Volume Storage Envelope Corpus |
| `TestVolumeStorageCollectionCorpus`:154 | convert-to-Rust-corpus | Violation of Volume Storage Collection Corpus |
| `TestVolumeStorageRunAdmissionCorpus`:270 | convert-to-Rust-corpus | Violation of Volume Storage Run Admission Corpus |
| `TestVolumeRecoveredConstructorCorpus`:332 | convert-to-Rust-corpus | Violation of Volume Recovered Constructor Corpus |

## tools/gomad/internal/model/wasi/volume_conformance_cases_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeConformanceMigrationContracts`:17 | convert-to-Rust-corpus | Violation of Volume Conformance Migration Contracts |

## tools/gomad/internal/model/wasi/volume_conformance_core_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeCoreAPIContracts`:17 | convert-to-Rust-corpus | Violation of Volume Core APIContracts |
| `TestVolumeErrorMappingCorpus`:409 | convert-to-Rust-corpus | Violation of Volume Error Mapping Corpus |

## tools/gomad/internal/model/wasi/volume_conformance_faults_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeFaultDecisionCorpus`:13 | convert-to-Rust-corpus | Violation of Volume Fault Decision Corpus |

## tools/gomad/internal/model/wasi/volume_conformance_image_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeImageIdentityCorpus`:18 | convert-to-Rust-corpus | Violation of Volume Image Identity Corpus |
| `TestVolumeImageAdmissionCorpus`:85 | convert-to-Rust-corpus | Violation of Volume Image Admission Corpus |

## tools/gomad/internal/model/wasi/volume_conformance_recovery_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeRecoveryAdmissionCorpus`:15 | convert-to-Rust-corpus | Violation of Volume Recovery Admission Corpus |

## tools/gomad/internal/model/wasi/volume_pending_absence_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumePendingOpenNamespaceAbsenceLiteralGuards`:71 | convert-to-Rust-corpus | Violation of Volume Pending Open Namespace Absence Literal Guards |
| `TestVolumePendingOpenNamespaceAbsenceCorpus`:113 | convert-to-Rust-corpus | Violation of Volume Pending Open Namespace Absence Corpus |

## tools/gomad/internal/model/wasi/volume_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestStorageErrorMappingIsClosed`:30 | convert-to-Rust-corpus | Violation of Storage Error Mapping Is Closed |
| `TestVolumeConformanceCorpus`:133 | convert-to-Rust-corpus | Violation of Volume Conformance Corpus |
| `TestVolumeConformanceInvalidRecoveryImages`:172 | convert-to-Rust-corpus | Violation of Volume Conformance Invalid Recovery Images |
| `TestVolumeCoreVectors`:724 | convert-to-Rust-corpus | Violation of Volume Core Vectors |

## tools/gomad/internal/model/wasi/volume_transport_corpus_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestVolumeTransportComparisonGuards`:15 | convert-to-Rust-corpus | Violation of Volume Transport Comparison Guards |
| `TestVolumeTransportCorpus`:48 | convert-to-Rust-corpus | Violation of Volume Transport Corpus |

## tools/gomad/internal/model/world/benchmark_test.go

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/internal/model/world/capacity_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWorldIdentityAndTransitionExhaustionDoNotWrap`:9 | convert-to-Rust-corpus | Violation of World Identity And Transition Exhaustion Do Not Wrap |
| `TestWorldQueueCapacityRollbackPreservesEventSequence`:53 | convert-to-Rust-corpus | Violation of World Queue Capacity Rollback Preserves Event Sequence |
| `TestWorldTransitionCapacityIsTerminalWithoutMutation`:86 | convert-to-Rust-corpus | Violation of World Transition Capacity Is Terminal Without Mutation |
| `TestWorldPayloadCapacityIncludesTranscriptCopies`:106 | convert-to-Rust-corpus | Violation of World Payload Capacity Includes Transcript Copies |
| `TestWorldRejectsEveryZeroLimit`:125 | convert-to-Rust-corpus | Violation of World Rejects Every Zero Limit |
| `TestWorldStringLimitCoversResourceComponents`:144 | convert-to-Rust-corpus | Violation of World String Limit Covers Resource Components |

## tools/gomad/internal/model/world/codec_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestDecodeSnapshotRejectsMalformedEnvelope`:10 | convert-to-Rust-corpus | Violation of Decode Snapshot Rejects Malformed Envelope |
| `TestMinimumLimitSnapshotRoundTrips`:53 | convert-to-Rust-corpus | Violation of Minimum Limit Snapshot Round Trips |
| `TestPopulatedHighCardinalitySnapshotRoundTrips`:71 | convert-to-Rust-corpus | Violation of Populated High Cardinality Snapshot Round Trips |

## tools/gomad/internal/model/world/codec_wire_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWorldJSONWireCompatibility`:99 | convert-to-Rust-corpus | Violation of World JSONWire Compatibility |
| `TestWorldEncodersRejectInvalidOriginalStrings`:282 | convert-to-Rust-corpus | Violation of World Encoders Reject Invalid Original Strings |
| `TestWorldDecodersRetainStrictAndExactWireRefusal`:328 | convert-to-Rust-corpus | Violation of World Decoders Retain Strict And Exact Wire Refusal |
| `TestDecodeTransitionsPreservesExactWireAndRefusesNonCanonicalInput`:401 | convert-to-Rust-corpus | Violation of Decode Transitions Preserves Exact Wire And Refuses Non Canonical Input |

## tools/gomad/internal/model/world/compressed_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestSnapshotUsesOneBinaryEnvelope`:18 | convert-to-Rust-corpus | Violation of Snapshot Uses One Binary Envelope |
| `TestCompressedSnapshotRoundTripSummary`:50 | convert-to-Rust-corpus | Violation of Compressed Snapshot Round Trip Summary |
| `TestSnapshotRoundTripsConfiguredLongString`:96 | convert-to-Rust-corpus | Violation of Snapshot Round Trips Configured Long String |
| `TestCompressedSnapshotRejectsMalformedExpandedWire`:121 | convert-to-Rust-corpus | Violation of Compressed Snapshot Rejects Malformed Expanded Wire |
| `TestCompressedWireReaderRejectsAllocationBeyondRemainingBytes`:166 | convert-to-Rust-corpus | Violation of Compressed Wire Reader Rejects Allocation Beyond Remaining Bytes |
| `TestCompressedSnapshotRejectsCorruptedEnvelopeAndGzip`:203 | convert-to-Rust-corpus | Violation of Compressed Snapshot Rejects Corrupted Envelope And Gzip |
| `TestCompressedSnapshotCopiedHandlesConsumeOnce`:274 | convert-to-Rust-corpus | Violation of Compressed Snapshot Copied Handles Consume Once |
| `TestCompressedSnapshotCapacityFailureConsumesHandle`:300 | convert-to-Rust-corpus | Violation of Compressed Snapshot Capacity Failure Consumes Handle |
| `TestCompressedSnapshotRejectsActiveRecordingWithoutFreezingModel`:324 | convert-to-Rust-corpus | Violation of Compressed Snapshot Rejects Active Recording Without Freezing Model |
| `TestCompressedSnapshotInvalidProfileDoesNotFreezeModel`:347 | convert-to-Rust-corpus | Violation of Compressed Snapshot Invalid Profile Does Not Freeze Model |
| `TestCompressedSnapshotConfiguredCapsRejectBeforeTransfer`:370 | convert-to-Rust-corpus | Violation of Compressed Snapshot Configured Caps Reject Before Transfer |

## tools/gomad/internal/model/world/concurrency_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWorldConcurrentRegistrationAllocatesEveryIDOnce`:8 | convert-to-Rust-corpus | Violation of World Concurrent Registration Allocates Every IDOnce |
| `TestWorldCancelQuiesceRaceLinearizes`:46 | convert-to-Rust-corpus | Violation of World Cancel Quiesce Race Linearizes |

## tools/gomad/internal/model/world/mailbox/mailbox_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestMailboxPilotPreservesFIFOForDistinctReceivers`:11 | convert-to-Rust-corpus | Violation of Mailbox Pilot Preserves FIFOFor Distinct Receivers |
| `TestMailboxPilotCancellationDeadlockSnapshotAndReplay`:21 | convert-to-Rust-corpus | Violation of Mailbox Pilot Cancellation Deadlock Snapshot And Replay |
| `TestMailboxPilotSurfacesWorldCapacityWithoutFallback`:114 | convert-to-Rust-corpus | Violation of Mailbox Pilot Surfaces World Capacity Without Fallback |
| `TestMailboxRestoreRejectsOmittedOwnedWorldRequest`:133 | convert-to-Rust-corpus | Violation of Mailbox Restore Rejects Omitted Owned World Request |

## tools/gomad/internal/model/world/process/config_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestConfigurationRoundTripsAndRejectsMalformedInput`:8 | convert-to-Rust-corpus | Violation of Configuration Round Trips And Rejects Malformed Input |

## tools/gomad/internal/model/world/process/session_terminal_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestSessionNormalizesDetailBeforeOrderedClassification`:49 | convert-to-Rust-corpus | Violation of Session Normalizes Detail Before Ordered Classification |
| `TestSessionRetainsJoinedErrorPrecedenceAndUnknownIdentity`:81 | convert-to-Rust-corpus | Violation of Session Retains Joined Error Precedence And Unknown Identity |
| `TestSessionValidatesBeforeCallingErrors`:114 | convert-to-Rust-corpus | Violation of Session Validates Before Calling Errors |

## tools/gomad/internal/model/world/recording_terminal_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestRecorderRejectsExternalCallbacksWithoutMutation`:32 | convert-to-Rust-corpus | Violation of Recorder Rejects External Callbacks Without Mutation |
| `TestRecorderAndModelIgnoreSentinelRebinding`:71 | convert-to-Rust-corpus | Violation of Recorder And Model Ignore Sentinel Rebinding |
| `TestOwnedContextRetainsTypedErrorFields`:114 | convert-to-Rust-corpus | Violation of Owned Context Retains Typed Error Fields |
| `TestOwnedTerminalPreservesAllCategoriesAndRecordingBytes`:137 | convert-to-Rust-corpus | Violation of Owned Terminal Preserves All Categories And Recording Bytes |
| `TestDetachedTerminalRejectsInvalidDataAndTypedNilWithoutClosing`:188 | convert-to-Rust-corpus | Violation of Detached Terminal Rejects Invalid Data And Typed Nil Without Closing |
| `TestOwnedModelErrorPreservesMessagesAndCauseIdentity`:213 | convert-to-Rust-corpus | Violation of Owned Model Error Preserves Messages And Cause Identity |
| `TestModelGeneratedTransitionContextRemainsRecordable`:244 | convert-to-Rust-corpus | Violation of Model Generated Transition Context Remains Recordable |

## tools/gomad/internal/model/world/recording_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWorldRecordingEnforcesTransitionBytesBeforeMutation`:8 | convert-to-Rust-corpus | Violation of World Recording Enforces Transition Bytes Before Mutation |
| `TestWorldRecordingEnvelopeRoundTripsExecutedTransitions`:24 | convert-to-Rust-corpus | Violation of World Recording Envelope Round Trips Executed Transitions |
| `TestWorldRecordingCarriesStructuredTerminalResults`:57 | convert-to-Rust-corpus | Violation of World Recording Carries Structured Terminal Results |
| `TestWorldRecordingCarriesStructuredCapacityError`:86 | convert-to-Rust-corpus | Violation of World Recording Carries Structured Capacity Error |

## tools/gomad/internal/model/world/replay_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestReplayPlanRawSyntaxUsesStrictDecoder`:9 | convert-to-Rust-corpus | Violation of Replay Plan Raw Syntax Uses Strict Decoder |
| `TestWorldReplayValidatesEveryTransitionAndFinalDigest`:17 | convert-to-Rust-corpus | Violation of World Replay Validates Every Transition And Final Digest |
| `TestWorldReplayDivergenceDoesNotMutateStateOrCursor`:63 | convert-to-Rust-corpus | Violation of World Replay Divergence Does Not Mutate State Or Cursor |
| `TestWorldReplayRejectsInvalidPlanBeforeUse`:89 | convert-to-Rust-corpus | Violation of World Replay Rejects Invalid Plan Before Use |
| `TestWorldReplayRejectsUnexpectedOperationAfterPlan`:97 | convert-to-Rust-corpus | Violation of World Replay Rejects Unexpected Operation After Plan |

## tools/gomad/internal/model/world/snapshot_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWorldSnapshotRestoreContinuesWithIdenticalState`:9 | convert-to-Rust-corpus | Violation of World Snapshot Restore Continues With Identical State |
| `TestRestoreRejectsCorruptionWithoutPublishingWorld`:61 | convert-to-Rust-corpus | Violation of Restore Rejects Corruption Without Publishing World |
| `TestWorldJSONIdentitiesRequireCanonicalDecimalStrings`:98 | convert-to-Rust-corpus | Violation of World JSONIdentities Require Canonical Decimal Strings |
| `TestWorldStateDigestGolden`:121 | convert-to-Rust-corpus | Violation of World State Digest Golden |

## tools/gomad/internal/model/world/world_test.go

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `TestWorldLifecycleCopiesDataAndDeliversWholeInstant`:9 | convert-to-Rust-corpus | Violation of World Lifecycle Copies Data And Delivers Whole Instant |
| `TestWorldOrdinaryOrderingIsLexicographicAndSeedIndependent`:50 | convert-to-Rust-corpus | Violation of World Ordinary Ordering Is Lexicographic And Seed Independent |
| `TestWorldEquivalentOrderingRepeatsAndCanVaryBySeed`:83 | convert-to-Rust-corpus | Violation of World Equivalent Ordering Repeats And Can Vary By Seed |
| `TestWorldCancellationIdleAndDeadlock`:98 | convert-to-Rust-corpus | Violation of World Cancellation Idle And Deadlock |
| `TestWorldInvalidAndCapacityErrorsDoNotMutateState`:153 | convert-to-Rust-corpus | Violation of World Invalid And Capacity Errors Do Not Mutate State |

## tools/gomad/rust/wasmhost/build.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/build_identity.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/abi.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `operation_locality_tables_cover_their_imports`:1471 | convert-to-Rust-corpus | Violation of operation locality tables cover their imports |
| `operation_profile_keys_are_unique_and_index_their_table`:1485 | convert-to-Rust-corpus | Violation of operation profile keys are unique and index their table |

## tools/gomad/rust/wasmhost/src/cache.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `crc_throughput`:441 | convert-to-Rust-corpus | Violation of crc throughput |
| `diagnostic_cache_phases_separate_read_verify_and_deserialize_failures`:471 | convert-to-Rust-corpus | Violation of diagnostic cache phases separate read verify and deserialize failures |
| `corrupt_entry_fails_closed_as_infrastructure_naming_entry`:622 | convert-to-Rust-corpus | Violation of corrupt entry fails closed as infrastructure naming entry |
| `oversized_entry_lengths_fail_closed_without_loading_or_mutation`:643 | convert-to-Rust-corpus | Violation of oversized entry lengths fail closed without loading or mutation |
| `v2_envelope_is_ignored_and_preserved`:718 | convert-to-Rust-corpus | Violation of v2 envelope is ignored and preserved |
| `bad_directory_mode_is_infrastructure`:754 | convert-to-Rust-corpus | Violation of bad directory mode is infrastructure |
| `stale_staging_file_from_reused_pid_is_not_fatal`:764 | convert-to-Rust-corpus | Violation of stale staging file from reused pid is not fatal |
| `storage_full_during_publication_runs_uncached`:785 | convert-to-Rust-corpus | Violation of storage full during publication runs uncached |
| `cache_identity_binds_helper_source_platform_and_engine_settings`:819 | convert-to-Rust-corpus | Violation of cache identity binds helper source platform and engine settings |

## tools/gomad/rust/wasmhost/src/clock.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `clock_advances_every_participant_to_the_earliest_deadline`:773 | convert-to-Rust-corpus | Violation of clock advances every participant to the earliest deadline |
| `clock_restart_hold_blocks_advance_until_removed`:797 | convert-to-Rust-corpus | Violation of clock restart hold blocks advance until removed |

## tools/gomad/rust/wasmhost/src/clock_conformance.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `clock_corpus_headers_match_the_go_pin`:33 | convert-to-Rust-corpus | Missing required named or scripted/generated/recorded cases, or invalid captured corpus header/pin |
| `clock_corpus_replays_exactly`:250 | convert-to-Rust-corpus | Violation of clock corpus replays exactly |
| `generated_clock_runs_reproduce_every_observed_response`:270 | convert-to-Rust-corpus | Violation of generated clock runs reproduce every observed response |

## tools/gomad/rust/wasmhost/src/cluster/execution/manual.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `every_distinct_module_must_hit_cache_for_a_warm_cluster`:30 | convert-to-Rust-corpus | Violation of every distinct module must hit cache for a warm cluster |

## tools/gomad/rust/wasmhost/src/cluster/execution/mod.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/execution/transport.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/execution/transport_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `go_cluster_transport_codec_corpus`:106 | convert-to-Rust-corpus | Violation of go cluster transport codec corpus |
| `codec_literal_stage_guards_reject_target_kind_and_nonrouting_errors`:164 | convert-to-Rust-corpus | Violation of codec literal stage guards reject target kind and nonrouting errors |

## tools/gomad/rust/wasmhost/src/cluster/execution/transport_input_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `cluster_input_sequence_rejects_changed_observations_and_wire`:76 | convert-to-Rust-corpus | Violation of cluster input sequence rejects changed observations and wire |
| `rust_cluster_input_eof_corpus`:191 | convert-to-Rust-corpus | Violation of rust cluster input eof corpus |
| `eof_literals_and_stage_controls_reject_stale_or_completed_input`:207 | convert-to-Rust-corpus | Violation of eof literals and stage controls reject stale or completed input |

## tools/gomad/rust/wasmhost/src/cluster/execution/transport_replay.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `tagged_replay_transactions_preserve_attribution_and_poison_broken_framing`:5 | convert-to-Rust-corpus | Violation of tagged replay transactions preserve attribution and poison broken framing |

## tools/gomad/rust/wasmhost/src/cluster/execution/transport_trace_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `rust_cluster_transport_io_trace_corpus`:114 | convert-to-Rust-corpus | Violation of rust cluster transport io trace corpus |
| `trace_literals_reject_changed_chunks_bytes_and_error_stages`:161 | convert-to-Rust-corpus | Violation of trace literals reject changed chunks bytes and error stages |

## tools/gomad/rust/wasmhost/src/cluster/handoff/baton.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `cluster_baton_serializes_in_instance_order`:162 | convert-to-Rust-corpus | Violation of cluster baton serializes in instance order |
| `cluster_baton_abort_releases_waiters`:186 | convert-to-Rust-corpus | Violation of cluster baton abort releases waiters |

## tools/gomad/rust/wasmhost/src/cluster/handoff/broker.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/handoff/broker/corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `broker_lifecycle_scripts`:77 | convert-to-Rust-corpus | Violation of broker lifecycle scripts |

## tools/gomad/rust/wasmhost/src/cluster/handoff/broker/readiness_probe_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `readiness_probe_keeps_three_transitions_and_observes_queued_controls_at_next_boundary`:12 | convert-to-Rust-corpus | Violation of readiness probe keeps three transitions and observes queued controls at next boundary |

## tools/gomad/rust/wasmhost/src/cluster/handoff/broker/tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `changed_inspection_boundaries_never_materialize_world_history`:41 | convert-to-Rust-corpus | Violation of changed inspection boundaries never materialize world history |
| `finish_preserves_world_transition_capacity`:64 | convert-to-Rust-corpus | Violation of finish preserves world transition capacity |
| `failed_peer_preserves_parked_incarnations_declared_boundary`:86 | convert-to-Rust-corpus | Violation of failed peer preserves parked incarnations declared boundary |
| `failed_cluster_drains_source_then_sorted_peers_under_thread_jitter`:118 | convert-to-Rust-corpus | Violation of failed cluster drains source then sorted peers under thread jitter |
| `event_order_controller_commands_match_parser_fixture`:182 | convert-to-Rust-corpus | Violation of event order controller commands match parser fixture |
| `diagnostic_world_calls_charge_nested_exclusive_owner_wall`:224 | convert-to-Rust-corpus | Violation of diagnostic world calls charge nested exclusive owner wall |
| `network_only_inspection_changes_bind_the_network_digest_without_world_wire`:272 | convert-to-Rust-corpus | Violation of network only inspection changes bind the network digest without world wire |
| `inspection_wire_reuses_transitions_and_refuses_at_the_generated_bound`:314 | convert-to-Rust-corpus | Violation of inspection wire reuses transitions and refuses at the generated bound |
| `oversized_world_discards_stale_wire_and_never_publishes_partial_serialization`:380 | convert-to-Rust-corpus | Violation of oversized world discards stale wire and never publishes partial serialization |
| `inspection_world_cache_matches_every_world_mutation_and_failed_operations`:409 | convert-to-Rust-corpus | Violation of inspection world cache matches every world mutation and failed operations |
| `broker_replays_recorded_go_orders`:699 | convert-to-Rust-corpus | Violation of broker replays recorded go orders |
| `broker_world_wire_is_bounded_while_written`:765 | convert-to-Rust-corpus | Violation of broker world wire is bounded while written |
| `incremental_recording_matches_eager_oracle_without_namespace`:1308 | convert-to-Rust-corpus | Violation of incremental recording matches eager oracle without namespace |
| `incremental_recording_matches_eager_oracle_with_namespace`:1317 | convert-to-Rust-corpus | Violation of incremental recording matches eager oracle with namespace |
| `incremental_recording_matches_eager_oracle_with_storage`:1326 | convert-to-Rust-corpus | Violation of incremental recording matches eager oracle with storage |
| `eager_writes_supersede_deferred_network_commitments_at_handoffs`:1429 | convert-to-Rust-corpus | Violation of eager writes supersede deferred network commitments at handoffs |
| `handoffs_refuse_a_deferred_network_commitment_left_pending`:1437 | convert-to-Rust-corpus | Violation of handoffs refuse a deferred network commitment left pending |
| `incremental_recording_matches_eager_oracle_when_sealed_or_exhausted`:1457 | convert-to-Rust-corpus | Violation of incremental recording matches eager oracle when sealed or exhausted |
| `finished_broker_world_uses_the_shared_binary_owner`:1469 | convert-to-Rust-corpus | Violation of finished broker world uses the shared binary owner |
| `native_world_configuration_requires_every_explicit_capacity`:1501 | convert-to-Rust-corpus | Violation of native world configuration requires every explicit capacity |
| `registration_checks_requests_before_payload_and_refusal_keeps_world`:1537 | convert-to-Rust-corpus | Violation of registration checks requests before payload and refusal keeps world |
| `inspection_failures_keep_their_termination_and_stop_inspection`:1574 | convert-to-Rust-corpus | Violation of inspection failures keep their termination and stop inspection |

## tools/gomad/rust/wasmhost/src/cluster/handoff/mod.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/mod.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/network.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/network/corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `go_network_cluster_corpus`:42 | convert-to-Rust-corpus | Violation of go network cluster corpus |

## tools/gomad/rust/wasmhost/src/cluster/network/evidence.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/network/evidence/decode.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `cluster_network_admission_separates_shared_record_and_local_world_limits`:354 | convert-to-Rust-corpus | Violation of cluster network admission separates shared record and local world limits |
| `cluster_network_admission_rejects_resealed_semantically_invalid_records`:405 | convert-to-Rust-corpus | Violation of cluster network admission rejects resealed semantically invalid records |
| `cluster_network_decoder_refuses_malformed_evidence`:503 | convert-to-Rust-corpus | Violation of cluster network decoder refuses malformed evidence |

## tools/gomad/rust/wasmhost/src/cluster/network/standalone.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/cluster/network/standalone/corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `go_standalone_network_adapter_corpus`:82 | convert-to-Rust-corpus | Violation of go standalone network adapter corpus |
| `standalone_network_world_byte_bound_is_typed_and_preserves_the_rejected_world`:262 | convert-to-Rust-corpus | Violation of standalone network world byte bound is typed and preserves the rejected world |

## tools/gomad/rust/wasmhost/src/cluster/network/tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `inspection_network_cache_follows_own_and_foreign_mutations`:61 | convert-to-Rust-corpus | Violation of inspection network cache follows own and foreign mutations |
| `simulation_network_and_fault_choices_change_real_delivery_and_topology`:111 | convert-to-Rust-corpus | Violation of simulation network and fault choices change real delivery and topology |
| `changed_network_inspections_never_rehash_native_history`:205 | convert-to-Rust-corpus | Violation of changed network inspections never rehash native history |

## tools/gomad/rust/wasmhost/src/cluster/network/tests/readiness_probe.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `readiness_probe_empty_full_and_repeated_do_not_consume_completions`:33 | convert-to-Rust-corpus | Violation of readiness probe empty full and repeated do not consume completions |
| `readiness_probe_rejects_malformed_duplicate_foreign_stale_and_inactive_frames`:77 | convert-to-Rust-corpus | Violation of readiness probe rejects malformed duplicate foreign stale and inactive frames |
| `readiness_probe_advances_deadline_and_preserves_error_until_poll`:146 | convert-to-Rust-corpus | Violation of readiness probe advances deadline and preserves error until poll |
| `readiness_probe_matches_legacy_scans_across_stream_mutations`:168 | convert-to-Rust-corpus | Violation of readiness probe matches legacy scans across stream mutations |

## tools/gomad/rust/wasmhost/src/conformance.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `token_validator_accepts_non_object_json`:1215 | convert-to-Rust-corpus | Violation of token validator accepts non object json |
| `namespace_conformance_corpus`:1230 | convert-to-Rust-corpus | Violation of namespace conformance corpus |
| `namespace_image_conformance_corpus`:1424 | convert-to-Rust-corpus | Violation of namespace image conformance corpus |
| `namespace_wire_conformance_corpus`:1554 | convert-to-Rust-corpus | Violation of namespace wire conformance corpus |
| `volume_conformance_corpus`:1595 | convert-to-Rust-corpus | Violation of volume conformance corpus |

## tools/gomad/rust/wasmhost/src/determinism_fixture.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `fixture_draw_signal_preserves_declared_seed_bits`:122 | convert-to-Rust-corpus | Violation of fixture draw signal preserves declared seed bits |
| `fixture_clock_shift_occurs_once_after_a_retained_prefix`:129 | convert-to-Rust-corpus | Violation of fixture clock shift occurs once after a retained prefix |
| `fixture_handoff_reorders_two_runnable_members_once`:138 | convert-to-Rust-corpus | Violation of fixture handoff reorders two runnable members once |

## tools/gomad/rust/wasmhost/src/determinism_fixture/control.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/determinism_fixture/control_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `explicit_fault_controls_are_scoped_owned_and_never_admit_an_ordinary_role`:5 | convert-to-Rust-corpus | Violation of explicit fault controls are scoped owned and never admit an ordinary role |
| `output_clock_uses_only_its_execution_and_exact_logical_output_owner`:45 | convert-to-Rust-corpus | Violation of output clock uses only its execution and exact logical output owner |
| `every_private_fixture_control_requires_opt_in_and_is_cleared_before_plain_execution`:79 | convert-to-Rust-corpus | Violation of every private fixture control requires opt in and is cleared before plain execution |

## tools/gomad/rust/wasmhost/src/diagnostic.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `diagnostic_cpu_deltas_are_phase_samples_not_process_totals`:1154 | convert-to-Rust-corpus | Violation of diagnostic cpu deltas are phase samples not process totals |
| `diagnostic_abi_buckets_do_not_overlap_and_residual_stays_unattributed`:1175 | convert-to-Rust-corpus | Violation of diagnostic abi buckets do not overlap and residual stays unattributed |
| `diagnostic_owner_attribution_reconciles_exactly`:1242 | convert-to-Rust-corpus | Violation of diagnostic owner attribution reconciles exactly |
| `diagnostic_attribution_errors_are_counted_not_hidden`:1323 | convert-to-Rust-corpus | Violation of diagnostic attribution errors are counted not hidden |
| `diagnostic_missing_op_frame_fails_reconciliation`:1366 | convert-to-Rust-corpus | Violation of diagnostic missing op frame fails reconciliation |
| `diagnostic_excluded_imports_stay_out_of_v1_buckets`:1389 | convert-to-Rust-corpus | Violation of diagnostic excluded imports stay out of v1 buckets |
| `diagnostic_histogram_buckets_bound_their_values`:1415 | convert-to-Rust-corpus | Violation of diagnostic histogram buckets bound their values |
| `diagnostic_thread_profiles_merge_into_the_session`:1461 | convert-to-Rust-corpus | Violation of diagnostic thread profiles merge into the session |
| `diagnostic_completed_profiles_keep_one_histogram_and_bounded_rows`:1503 | convert-to-Rust-corpus | Violation of diagnostic completed profiles keep one histogram and bounded rows |
| `linked_report_capacity_retains_drop_counters`:1528 | convert-to-Rust-corpus | Violation of linked report capacity retains drop counters |
| `diagnostic_report_is_fixed_bounded_and_counters_saturate`:1549 | convert-to-Rust-corpus | Violation of diagnostic report is fixed bounded and counters saturate |

## tools/gomad/rust/wasmhost/src/engine.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `typed_write_abi_gathers_scattered_empty_and_unsigned_inputs_with_state_reuse`:1664 | convert-to-Rust-corpus | Violation of typed write abi gathers scattered empty and unsigned inputs with state reuse |
| `typed_write_abi_pointer_and_capacity_priority_precede_callback_mutation`:1700 | convert-to-Rust-corpus | Violation of typed write abi pointer and capacity priority precede callback mutation |
| `typed_write_abi_max_buffer_and_reservation_failures_restore_ownership`:1763 | convert-to-Rust-corpus | Violation of typed write abi max buffer and reservation failures restore ownership |
| `typed_write_abi_charges_offered_output_including_partial_and_failed_writes`:1795 | convert-to-Rust-corpus | Violation of typed write abi charges offered output including partial and failed writes |
| `typed_write_abi_reply_errors_chunks_interrupts_and_transport_restore_buffers`:1829 | convert-to-Rust-corpus | Violation of typed write abi reply errors chunks interrupts and transport restore buffers |
| `typed_write_byte_and_base64_buffers_reuse_independent_live_states`:1990 | convert-to-Rust-corpus | Violation of typed write byte and base64 buffers reuse independent live states |
| `typed_write_frame_uses_independent_state_scratch_and_reuses_it`:2065 | convert-to-Rust-corpus | Violation of typed write frame uses independent state scratch and reuses it |
| `typed_write_reference_keeps_literal_wire_and_three_field_request_state`:2107 | convert-to-Rust-corpus | Violation of typed write reference keeps literal wire and three field request state |
| `typed_write_serialization_matches_value_counterfactual_and_bounds_retained_capacity`:2157 | convert-to-Rust-corpus | Violation of typed write serialization matches value counterfactual and bounds retained capacity |
| `typed_write_shared_decoder_chunks_interrupts_and_transport_restore_scratch`:2246 | convert-to-Rust-corpus | Violation of typed write shared decoder chunks interrupts and transport restore scratch |
| `typed_write_reference_capacity_and_decoder_fail_before_owner_merge`:2312 | convert-to-Rust-corpus | Violation of typed write reference capacity and decoder fail before owner merge |
| `unchanged_boundaries_never_materialize_world`:2411 | convert-to-Rust-corpus | Violation of unchanged boundaries never materialize world |
| `handoff_and_terminal_cuts_defer_world_until_the_current_boundary`:2435 | convert-to-Rust-corpus | Violation of handoff and terminal cuts defer world until the current boundary |
| `ordinary_model_does_not_capture_state_without_debug_hooks`:2489 | convert-to-Rust-corpus | Violation of ordinary model does not capture state without debug hooks |
| `go_owned_debug_refuses_active_inspection_and_malformed_frames`:2498 | convert-to-Rust-corpus | Violation of go owned debug refuses active inspection and malformed frames |
| `typed_clock_state_capacity_preserves_attempted_and_completed_counts`:2510 | convert-to-Rust-corpus | Violation of typed clock state capacity preserves attempted and completed counts |
| `typed_clock_streams_bounded_chunks_before_final_flush_matching_json_reference`:2531 | convert-to-Rust-corpus | Violation of typed clock streams bounded chunks before final flush matching json reference |
| `untyped_execution_io_failure_is_infrastructure`:2579 | convert-to-Rust-corpus | Violation of untyped execution io failure is infrastructure |
| `broken_callback_transport_is_infrastructure`:2587 | convert-to-Rust-corpus | Violation of broken callback transport is infrastructure |

## tools/gomad/rust/wasmhost/src/event_order.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `poisoned_linked_session_never_emits_complete`:381 | convert-to-Rust-corpus | Violation of poisoned linked session never emits complete |
| `event_order_records_full_values_and_actual_coordinates_in_commit_order`:395 | convert-to-Rust-corpus | Violation of event order records full values and actual coordinates in commit order |
| `event_order_never_marks_a_dropped_or_disabled_stream_complete`:438 | convert-to-Rust-corpus | Violation of event order never marks a dropped or disabled stream complete |

## tools/gomad/rust/wasmhost/src/guest_profile.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/guest_profile/preservation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `profile_configuration_rejects_original_malformed_bounds_before_sampling`:4 | convert-to-Rust-corpus | Violation of profile configuration rejects original malformed bounds before sampling |

## tools/gomad/rust/wasmhost/src/lib.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/linked.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `linked_terminal_retains_actual_phase_and_process_metrics`:871 | convert-to-Rust-corpus | Violation of linked terminal retains actual phase and process metrics |
| `aggregate_is_combined_and_failed_append_preserves_prefix`:916 | convert-to-Rust-corpus | Violation of aggregate is combined and failed append preserves prefix |
| `aggregate_addition_overflow_is_refused`:928 | convert-to-Rust-corpus | Violation of aggregate addition overflow is refused |
| `actual_native_terminal_excludes_postterminal_work_and_retains_separate_cleanup`:962 | convert-to-Rust-corpus | Violation of actual native terminal excludes postterminal work and retains separate cleanup |

## tools/gomad/rust/wasmhost/src/linked/streaming_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `short_markers_are_forwarded_before_guest_waits_for_controller_release`:41 | convert-to-Rust-corpus | Violation of short markers are forwarded before guest waits for controller release |

## tools/gomad/rust/wasmhost/src/linked_cancellation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `cancellation_interrupts_only_its_registered_engine_and_releases_it`:106 | convert-to-Rust-corpus | Violation of cancellation interrupts only its registered engine and releases it |

## tools/gomad/rust/wasmhost/src/linked_cluster.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `second_store_writes_do_not_replace_the_primary_reopen_image`:1839 | convert-to-Rust-corpus | Violation of second store writes do not replace the primary reopen image |

## tools/gomad/rust/wasmhost/src/linked_cluster/lifecycle_predicate_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `failed_wait_preserves_first_terminal_and_never_starts_later_incarnations`:37 | convert-to-Rust-corpus | Violation of failed wait preserves first terminal and never starts later incarnations |
| `requested_stores_are_lazy_and_cancellation_before_execution_has_no_actions`:72 | convert-to-Rust-corpus | Violation of requested stores are lazy and cancellation before execution has no actions |
| `mandatory_cleanup_keeps_only_authored_start_actions`:117 | convert-to-Rust-corpus | Violation of mandatory cleanup keeps only authored start actions |
| `typed_owner_admission_refuses_missing_model_namespace_and_unattributable_handles`:246 | convert-to-Rust-corpus | Violation of typed owner admission refuses missing model namespace and unattributable handles |
| `retired_framed_guest_callbacks_are_refused_by_native_owner_sink`:301 | convert-to-Rust-corpus | Violation of retired framed guest callbacks are refused by native owner sink |
| `observed_failure_drains_prestarted_peers_without_executing_their_waits`:324 | convert-to-Rust-corpus | Violation of observed failure drains prestarted peers without executing their waits |
| `sealed_deadline_is_bounded_before_mutation_and_cannot_be_extended`:353 | convert-to-Rust-corpus | Violation of sealed deadline is bounded before mutation and cannot be extended |
| `cancellation_after_actual_output_keeps_joined_incarnations_and_start_prefix`:379 | convert-to-Rust-corpus | Violation of cancellation after actual output keeps joined incarnations and start prefix |
| `synchronous_observer_cancellation_preserves_before_and_after_refusals`:436 | convert-to-Rust-corpus | Violation of synchronous observer cancellation preserves before and after refusals |
| `refused_restart_preserves_owner_and_allows_retry_on_same_broker`:493 | convert-to-Rust-corpus | Violation of refused restart preserves owner and allows retry on same broker |
| `native_host_termination_preserves_watchdog_and_parent_cancellation_status`:625 | convert-to-Rust-corpus | Violation of native host termination preserves watchdog and parent cancellation status |
| `live_cleanup_reaps_a_noncooperative_store_at_the_original_five_second_grace`:719 | convert-to-Rust-corpus | Violation of live cleanup reaps a noncooperative store at the original five second grace |
| `network_owner_admission_refuses_original_per_node_and_initial_incarnation_cases`:827 | convert-to-Rust-corpus | Violation of network owner admission refuses original per node and initial incarnation cases |
| `storage_volume_order_boot_admission_keeps_the_same_first_refusal`:914 | convert-to-Rust-corpus | Violation of storage volume order boot admission keeps the same first refusal |
| `actual_storage_engine_selection_and_divergence_retain_peak_memory_and_enumeration`:1083 | convert-to-Rust-corpus | Violation of actual storage engine selection and divergence retain peak memory and enumeration |
| `observed_failure_drain_waits_for_held_native_peer_without_post_failure_controls`:1234 | convert-to-Rust-corpus | Violation of observed failure drain waits for held native peer without post failure controls |
| `memory_capacity_keeps_owned_model_metadata_before_and_after_runtime_bootstrap`:1398 | convert-to-Rust-corpus | Violation of memory capacity keeps owned model metadata before and after runtime bootstrap |

## tools/gomad/rust/wasmhost/src/linked_cluster/live_controller_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `conditional_output_wait_retains_world_and_actual_terminal`:99 | convert-to-Rust-corpus | Violation of conditional output wait retains world and actual terminal |
| `unbound_same_kind_and_cancellation_remain_failures_with_armed_marker`:174 | convert-to-Rust-corpus | Violation of unbound same kind and cancellation remain failures with armed marker |
| `conditional_output_boundary_preserves_wait_lifecycle`:248 | convert-to-Rust-corpus | Violation of conditional output boundary preserves wait lifecycle |
| `final_network_uses_finished_shared_owner_instead_of_cached_incarnation`:440 | convert-to-Rust-corpus | Violation of final network uses finished shared owner instead of cached incarnation |
| `joined_storage_crash_preserves_successful_session_world_and_network`:535 | convert-to-Rust-corpus | Violation of joined storage crash preserves successful session world and network |
| `shutdown_reinsertion_preserves_actual_live_guest_world`:638 | convert-to-Rust-corpus | Violation of shutdown reinsertion preserves actual live guest world |
| `actual_linked_world_transition_byte_bound_changes_capacity_and_replays_the_retained_prefix`:694 | convert-to-Rust-corpus | Violation of actual linked world transition byte bound changes capacity and replays the retained prefix |
| `one_and_thirty_two_mib_world_transition_bounds_change_the_same_actual_guest`:747 | convert-to-Rust-corpus | Violation of one and thirty two mib world transition bounds change the same actual guest |
| `measurement_phase_callbacks_guard_actual_store_dispatch_and_join_for_both_schedules`:865 | convert-to-Rust-corpus | Violation of measurement phase callbacks guard actual store dispatch and join for both schedules |

## tools/gomad/rust/wasmhost/src/linked_cluster/network_binding_preservation_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `selected_boot_network_limits_bind_first_later_restart_and_explicit_shared_models`:42 | convert-to-Rust-corpus | Violation of selected boot network limits bind first later restart and explicit shared models |

## tools/gomad/rust/wasmhost/src/linked_conditions.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `shared_workers_commit_to_one_session_and_closed_tokens_refuse`:254 | convert-to-Rust-corpus | Violation of shared workers commit to one session and closed tokens refuse |
| `overlapping_execution_sessions_never_share_recorders`:293 | convert-to-Rust-corpus | Violation of overlapping execution sessions never share recorders |
| `shared_session_event_and_byte_exhaustion_never_complete`:321 | convert-to-Rust-corpus | Violation of shared session event and byte exhaustion never complete |
| `diagnostic_helper_cannot_authorize_an_ordinary_role`:349 | convert-to-Rust-corpus | Violation of diagnostic helper cannot authorize an ordinary role |
| `empty_feature_helper_preserves_ordinary_role`:381 | convert-to-Rust-corpus | Violation of empty feature helper preserves ordinary role |
| `existing_nonordinary_roles_cannot_authorize_ordinary`:396 | convert-to-Rust-corpus | Violation of existing nonordinary roles cannot authorize ordinary |
| `role_cache_configuration_preserves_existing_engine_revisions`:414 | convert-to-Rust-corpus | Violation of role cache configuration preserves existing engine revisions |
| `compiled_cache_configuration_keeps_original_feature_precedence`:450 | convert-to-Rust-corpus | Violation of compiled cache configuration keeps original feature precedence |

## tools/gomad/rust/wasmhost/src/linked_fixture.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/linked_inspection.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/linked_inspection_predicate_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `decision_mapping_skips_observations_and_refuses_missing_or_relabelled_boundaries`:104 | convert-to-Rust-corpus | Violation of decision mapping skips observations and refuses missing or relabelled boundaries |
| `interrupted_select_poll_keeps_its_presented_decision_coordinate`:133 | convert-to-Rust-corpus | Violation of interrupted select poll keeps its presented decision coordinate |
| `creation_records_use_the_first_creation_coordinate_not_the_decision_record`:146 | convert-to-Rust-corpus | Violation of creation records use the first creation coordinate not the decision record |
| `inspection_control_binds_raw_prefix_registry_and_exact_boundary`:173 | convert-to-Rust-corpus | Violation of inspection control binds raw prefix registry and exact boundary |
| `terminal_network_raw_stream_cannot_be_replaced_by_an_unrelated_state_commitment`:265 | convert-to-Rust-corpus | Violation of terminal network raw stream cannot be replaced by an unrelated state commitment |
| `inspection_output_bound_includes_json_and_heap`:364 | convert-to-Rust-corpus | Violation of inspection output bound includes json and heap |
| `debug_library_records_preserve_failures_raw_strings_and_coordinates`:416 | convert-to-Rust-corpus | Violation of debug library records preserve failures raw strings and coordinates |
| `debug_heap_profile_keeps_rate_stack_and_inuse_sample`:477 | convert-to-Rust-corpus | Violation of debug heap profile keeps rate stack and inuse sample |

## tools/gomad/rust/wasmhost/src/linked_network.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/linked_simulation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `scenario_override_selects_actual_ready_start_and_preserves_terminal_order`:538 | convert-to-Rust-corpus | Violation of scenario override selects actual ready start and preserves terminal order |
| `decision_byte_capacity_stops_before_recording_or_consuming_a_choice`:584 | convert-to-Rust-corpus | Violation of decision byte capacity stops before recording or consuming a choice |

## tools/gomad/rust/wasmhost/src/linked_storage.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/linked_storage_inspection.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/linked_storage_inspection/state.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `coherently_resealed_native_storage_sections_reach_typed_semantic_guards`:316 | convert-to-Rust-corpus | Violation of coherently resealed native storage sections reach typed semantic guards |
| `actual_native_storage_state_refuses_forged_context_and_unrequested_context_presence`:406 | convert-to-Rust-corpus | Violation of actual native storage state refuses forged context and unrequested context presence |

## tools/gomad/rust/wasmhost/src/linked_storage_inspection/state/strict.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/local_model.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/main.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/metrics.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `platform_layout_and_units_are_checked`:140 | convert-to-Rust-corpus | Violation of platform layout and units are checked |
| `own_peak_rss_reads_only_wellformed_vmhwm`:186 | convert-to-Rust-corpus | Violation of own peak rss reads only wellformed vmhwm |
| `exec_does_not_inherit_parent_peak`:205 | convert-to-Rust-corpus | Violation of exec does not inherit parent peak |

## tools/gomad/rust/wasmhost/src/model.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `namespace_capacity_refuses_before_mutation_and_preserves_completed_evidence`:324 | convert-to-Rust-corpus | Violation of namespace capacity refuses before mutation and preserves completed evidence |
| `namespace_unsupported_exact_bounds_preserve_message_bytes_and_ids`:343 | convert-to-Rust-corpus | Violation of namespace unsupported exact bounds preserve message bytes and ids |
| `entropy_boundaries_and_literal_transcript_preserve_offsets`:400 | convert-to-Rust-corpus | Violation of entropy boundaries and literal transcript preserve offsets |
| `captured_entropy_matches_shared_golden_vectors`:419 | convert-to-Rust-corpus | Violation of captured entropy matches shared golden vectors |
| `transcript_digest_matches_pinned_independent_sha256`:452 | convert-to-Rust-corpus | Violation of transcript digest matches pinned independent sha256 |
| `pending_clock_inspection_preserves_run_and_digest`:488 | convert-to-Rust-corpus | Violation of pending clock inspection preserves run and digest |
| `runtime_creation_divergence_keeps_an_inspectable_prefix`:524 | convert-to-Rust-corpus | Violation of runtime creation divergence keeps an inspectable prefix |
| `runtime_creation_sites_preserve_alternative_identities_and_exact_replay`:567 | convert-to-Rust-corpus | Violation of runtime creation sites preserve alternative identities and exact replay |
| `runtime_creation_diagnostics_keep_trace_ordinals`:631 | convert-to-Rust-corpus | Violation of runtime creation diagnostics keep trace ordinals |
| `runtime_decision_refuses_malformed_header_v2_layouts_without_recording`:685 | convert-to-Rust-corpus | Violation of runtime decision refuses malformed header v2 layouts without recording |
| `debug_exchange_preserves_open_clock_run_and_canonical_state`:785 | convert-to-Rust-corpus | Violation of debug exchange preserves open clock run and canonical state |
| `debug_seals_validated_pending_decision_without_consuming_tape_suffix`:820 | convert-to-Rust-corpus | Violation of debug seals validated pending decision without consuming tape suffix |
| `diagnostic_model_hooks_measure_actual_hash_encoding_and_flush`:898 | convert-to-Rust-corpus | Violation of diagnostic model hooks measure actual hash encoding and flush |
| `diagnostic_calibration_preserves_model_bytes_without_timing_feedback`:915 | convert-to-Rust-corpus | Violation of diagnostic calibration preserves model bytes without timing feedback |
| `typed_clock_preserves_json_reference_and_literal_canonical_bytes`:979 | convert-to-Rust-corpus | Violation of typed clock preserves json reference and literal canonical bytes |
| `typed_clock_reservation_digest_and_read_step_failures_preserve_state`:1076 | convert-to-Rust-corpus | Violation of typed clock reservation digest and read step failures preserve state |
| `profile_local_import_stages_against_json_reference`:1108 | convert-to-Rust-corpus | Violation of profile local import stages against json reference |
| `bounds_trace_transcript_and_evidence_chunks`:1380 | convert-to-Rust-corpus | Violation of bounds trace transcript and evidence chunks |
| `witness_event_preserves_signed_virtual_time_without_changing_canonical_bytes`:1402 | convert-to-Rust-corpus | Violation of witness event preserves signed virtual time without changing canonical bytes |

## tools/gomad/rust/wasmhost/src/model/choice_wire_generated.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/model/guest_state.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/model/imports.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/model/inspection.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `alternating_snapshot_handles_hash_unchanged_recovery_material_once_each`:1186 | convert-to-Rust-corpus | Violation of alternating snapshot handles hash unchanged recovery material once each |
| `recovery_cache_preserves_uncached_snapshot_and_evidence_bytes`:1222 | convert-to-Rust-corpus | Violation of recovery cache preserves uncached snapshot and evidence bytes |
| `recovery_cache_bounds_keys_material_entry_pressure_and_oversized_bypass`:1300 | convert-to-Rust-corpus | Violation of recovery cache bounds keys material entry pressure and oversized bypass |
| `recovery_cache_keeps_original_snapshot_refusal_order`:1389 | convert-to-Rust-corpus | Violation of recovery cache keeps original snapshot refusal order |
| `repeated_static_mount_and_world_cuts_fit_bounded_metadata`:1447 | convert-to-Rust-corpus | Violation of repeated static mount and world cuts fit bounded metadata |
| `state_commitments_bind_domains_lengths_and_last_bytes`:1495 | convert-to-Rust-corpus | Violation of state commitments bind domains lengths and last bytes |
| `workflow_scale_inventory_fits_without_omitting_boundaries`:1520 | convert-to-Rust-corpus | Violation of workflow scale inventory fits without omitting boundaries |
| `snapshot_with_supplied_and_deferred_commitments_matches_fresh_snapshots`:1639 | convert-to-Rust-corpus | Violation of snapshot with supplied and deferred commitments matches fresh snapshots |
| `snapshot_with_unchanged_storage_refuses_a_different_retained_log`:1720 | convert-to-Rust-corpus | Violation of snapshot with unchanged storage refuses a different retained log |
| `boundary_refuses_a_deferred_network_commitment`:1759 | convert-to-Rust-corpus | Violation of boundary refuses a deferred network commitment |

## tools/gomad/rust/wasmhost/src/model/replay.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `replay_preflight_accepts_empty_and_preserves_following_json`:242 | convert-to-Rust-corpus | Violation of replay preflight accepts empty and preserves following json |
| `replay_preflight_refuses_malformed_blocks_and_expectations`:261 | convert-to-Rust-corpus | Violation of replay preflight refuses malformed blocks and expectations |
| `replay_preflight_binary_lf_and_chunk_boundaries_reach_trace_validation`:403 | convert-to-Rust-corpus | Violation of replay preflight binary lf and chunk boundaries reach trace validation |

## tools/gomad/rust/wasmhost/src/model/replay_plan.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `runnable_creation_metadata_projects_without_forcing_metadata`:614 | convert-to-Rust-corpus | Violation of runnable creation metadata projects without forcing metadata |
| `replay_plan_go_corpus`:649 | convert-to-Rust-corpus | Violation of replay plan go corpus |

## tools/gomad/rust/wasmhost/src/model/reply.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/model/runtime_preservation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `runtime_replay_forces_logical_identity_and_refuses_each_original_boundary_mismatch`:57 | convert-to-Rust-corpus | Violation of runtime replay forces logical identity and refuses each original boundary mismatch |
| `tampered_runtime_tapes_end_in_their_typed_divergence_reasons`:143 | convert-to-Rust-corpus | Violation of tampered runtime tapes end in their typed divergence reasons |

## tools/gomad/rust/wasmhost/src/model/tests/default_cut_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `default_cut_literals_reject_snapshot_and_changed_stage_contracts`:6 | convert-to-Rust-corpus | Violation of default cut literals reject snapshot and changed stage contracts |
| `rust_model_default_cut_corpus`:169 | convert-to-Rust-corpus | Violation of rust model default cut corpus |

## tools/gomad/rust/wasmhost/src/model_preservation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `captured_arguments_environment_and_entropy_preserve_original_values_and_chunking`:107 | convert-to-Rust-corpus | Violation of captured arguments environment and entropy preserve original values and chunking |
| `environment_rejects_original_ambiguous_configuration_variants`:136 | convert-to-Rust-corpus | Violation of environment rejects original ambiguous configuration variants |
| `transcript_clock_run_preserves_original_literal_digest_modes_and_pending_inspection`:154 | convert-to-Rust-corpus | Violation of transcript clock run preserves original literal digest modes and pending inspection |
| `exploratory_clock_and_mixed_poll_preserve_original_virtual_time_and_event_order`:204 | convert-to-Rust-corpus | Violation of exploratory clock and mixed poll preserve original virtual time and event order |
| `namespace_rights_offsets_cookies_containment_and_closure_preserve_original_sequence`:274 | convert-to-Rust-corpus | Violation of namespace rights offsets cookies containment and closure preserve original sequence |
| `captured_namespace_owns_payload_and_infers_absence_without_host_access`:440 | convert-to-Rust-corpus | Violation of captured namespace owns payload and infers absence without host access |
| `network_v2_rejects_original_retired_profile_before_state_construction`:502 | convert-to-Rust-corpus | Violation of network v2 rejects original retired profile before state construction |
| `transcript_clock_run_retains_original_51_52_byte_capacity_boundary`:521 | convert-to-Rust-corpus | Violation of transcript clock run retains original 51 52 byte capacity boundary |
| `namespace_sync_emits_original_versioned_binary_transcript`:541 | convert-to-Rust-corpus | Violation of namespace sync emits original versioned binary transcript |
| `socket_refusal_preserves_original_binary_transcript_and_namespace_identity`:555 | convert-to-Rust-corpus | Violation of socket refusal preserves original binary transcript and namespace identity |

## tools/gomad/rust/wasmhost/src/module_metadata.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/namespace/call.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `namespace_binary_transcript_codec_corpus`:366 | convert-to-Rust-corpus | Violation of namespace binary transcript codec corpus |
| `typed_admission_retains_lower_layer_callback_and_zero_limit_guards`:492 | convert-to-Rust-corpus | Violation of typed admission retains lower layer callback and zero limit guards |
| `read_buffer_refusal_precedes_transcript_refusal`:506 | convert-to-Rust-corpus | Violation of read buffer refusal precedes transcript refusal |

## tools/gomad/rust/wasmhost/src/namespace/image.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `handles_fragmented_and_interrupted_reads_without_large_requests`:314 | convert-to-Rust-corpus | Violation of handles fragmented and interrupted reads without large requests |

## tools/gomad/rust/wasmhost/src/namespace/mod.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `diagnostic_output_ordinals_survive_draining_each_stream`:1850 | convert-to-Rust-corpus | Violation of diagnostic output ordinals survive draining each stream |
| `diagnostic_namespace_state_tracks_mutations_and_caches_by_revision`:1868 | convert-to-Rust-corpus | Violation of diagnostic namespace state tracks mutations and caches by revision |
| `drained_output_keeps_charge_and_cannot_be_taken_twice`:1892 | convert-to-Rust-corpus | Violation of drained output keeps charge and cannot be taken twice |
| `oversized_typed_write_refuses_before_output_accounting`:1934 | convert-to-Rust-corpus | Violation of oversized typed write refuses before output accounting |
| `constructor_checks_preopen_capacity_below_public_minimum`:1953 | convert-to-Rust-corpus | Violation of constructor checks preopen capacity below public minimum |
| `inspection_revision_advances_on_every_mutating_entry_point`:1966 | convert-to-Rust-corpus | Violation of inspection revision advances on every mutating entry point |

## tools/gomad/rust/wasmhost/src/namespace/transport.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `descriptor_validation_does_not_allocate_declared_bytes`:212 | convert-to-Rust-corpus | Violation of descriptor validation does not allocate declared bytes |
| `read_image_rejects_malformed_working_directory_digest`:220 | convert-to-Rust-corpus | Violation of read image rejects malformed working directory digest |

## tools/gomad/rust/wasmhost/src/network/connection.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/network/history.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/network/mod.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/network/stall.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/network/stream_abi.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/network/stream_abi/corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `stream_wire_literals_and_real_consumer_reject_changed_vectors`:302 | convert-to-Rust-corpus | Violation of stream wire literals and real consumer reject changed vectors |
| `rust_network_stream_wire_corpus`:359 | convert-to-Rust-corpus | Violation of rust network stream wire corpus |

## tools/gomad/rust/wasmhost/src/network/tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `record_hashes_each_digest_once`:35 | convert-to-Rust-corpus | Violation of record hashes each digest once |
| `record_preserves_history_and_snapshot_wire_validation`:46 | convert-to-Rust-corpus | Violation of record preserves history and snapshot wire validation |
| `record_check_matches_record_failure_at_every_wire_limit`:77 | convert-to-Rust-corpus | Violation of record check matches record failure at every wire limit |
| `inspection_lane_chains_preserve_canonical_order_and_bind_every_transition`:110 | convert-to-Rust-corpus | Violation of inspection lane chains preserve canonical order and bind every transition |
| `record_validates_original_history_before_canonical_reordering_changes_error_precedence`:161 | convert-to-Rust-corpus | Violation of record validates original history before canonical reordering changes error precedence |
| `implicit_listen_exhaustion_records_zero_port_binary_identity`:182 | convert-to-Rust-corpus | Violation of implicit listen exhaustion records zero port binary identity |

## tools/gomad/rust/wasmhost/src/network_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `network_corpus_classification_rejects_unclassified_fixtures`:176 | convert-to-Rust-corpus | Violation of network corpus classification rejects unclassified fixtures |
| `network_history_binary_identity_matches_every_authentic_step_and_record`:203 | convert-to-Rust-corpus | Violation of network history binary identity matches every authentic step and record |
| `network_full_corpus_requires_all_scripts_every_step_and_final_record`:435 | convert-to-Rust-corpus | Violation of network full corpus requires all scripts every step and final record |

## tools/gomad/rust/wasmhost/src/profile_inventory.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/protocol.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `stable_backend_view_retains_all_protocol_fields_except_volatile_metrics`:656 | convert-to-Rust-corpus | Violation of stable backend view retains all protocol fields except volatile metrics |

## tools/gomad/rust/wasmhost/src/storage_evidence_bounds.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/storage_evidence_bounds_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `partial_header_recognizes_every_original_split_and_rejects_short_or_wrong_headers`:25 | convert-to-Rust-corpus | Violation of partial header recognizes every original split and rejects short or wrong headers |
| `storage_budget_counts_every_volume_and_saturates_original_overflow_cases`:53 | convert-to-Rust-corpus | Violation of storage budget counts every volume and saturates original overflow cases |
| `native_sink_keeps_independent_storage_caps_and_rejected_append_has_no_effect`:97 | convert-to-Rust-corpus | Violation of native sink keeps independent storage caps and rejected append has no effect |
| `partial_state_retains_original_complete_and_partial_limits_with_fragmented_header`:117 | convert-to-Rust-corpus | Violation of partial state retains original complete and partial limits with fragmented header |
| `saturated_storage_allowance_keeps_the_existing_aggregate_limit`:155 | convert-to-Rust-corpus | Violation of saturated storage allowance keeps the existing aggregate limit |
| `native_preparation_retains_guard_before_consuming_model_and_namespace_inputs`:175 | convert-to-Rust-corpus | Violation of native preparation retains guard before consuming model and namespace inputs |
| `storage_guard_is_lazy_and_leaves_other_stream_bytes_unchanged`:208 | convert-to-Rust-corpus | Violation of storage guard is lazy and leaves other stream bytes unchanged |

## tools/gomad/rust/wasmhost/src/storage_preservation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `partial_metadata_scalar_and_escaped_path_bounds_preserve_original_extremes`:5 | convert-to-Rust-corpus | Violation of partial metadata scalar and escaped path bounds preserve original extremes |
| `refused_large_history_retains_original_hundred_thousand_write_prefix`:50 | convert-to-Rust-corpus | Violation of refused large history retains original hundred thousand write prefix |
| `maximum_enumeration_limits_preserve_actual_native_empty_volume_evidence`:159 | convert-to-Rust-corpus | Violation of maximum enumeration limits preserve actual native empty volume evidence |

## tools/gomad/rust/wasmhost/src/storage_transport.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `reference_precrash_image_keeps_actual_pending_effects`:80 | convert-to-Rust-corpus | Violation of reference precrash image keeps actual pending effects |
| `exploration_crash_selection_changes_recovered_bytes`:114 | convert-to-Rust-corpus | Violation of exploration crash selection changes recovered bytes |

## tools/gomad/rust/wasmhost/src/storage_transport/crash_failure_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `crash_failure_literals_reject_changed_original_tuples`:6 | convert-to-Rust-corpus | Violation of crash failure literals reject changed original tuples |
| `rust_storage_crash_failure_corpus`:147 | convert-to-Rust-corpus | Violation of rust storage crash failure corpus |

## tools/gomad/rust/wasmhost/src/storage_transport/preservation_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `original_crash_context_admission_positive_and_negative_predicates`:38 | convert-to-Rust-corpus | Violation of original crash context admission positive and negative predicates |
| `original_crash_context_equality_preserves_every_operation_field_and_null_shape`:102 | convert-to-Rust-corpus | Violation of original crash context equality preserves every operation field and null shape |
| `original_large_context_transport_round_trips_five_thousand_operations`:203 | convert-to-Rust-corpus | Violation of original large context transport round trips five thousand operations |
| `original_selected_states_recover_and_divergence_keeps_enumeration_without_image`:275 | convert-to-Rust-corpus | Violation of original selected states recover and divergence keeps enumeration without image |
| `original_large_observed_selection_uses_fifty_four_thousand_byte_paths`:379 | convert-to-Rust-corpus | Violation of original large observed selection uses fifty four thousand byte paths |
| `original_volume_validation_first_refusal_is_stable_for_sixty_four_orders`:440 | convert-to-Rust-corpus | Violation of original volume validation first refusal is stable for sixty four orders |

## tools/gomad/rust/wasmhost/src/storage_transport/validation_order_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `first_refusal_is_insertion_order_independent_and_matches_go`:19 | convert-to-Rust-corpus | Violation of first refusal is insertion order independent and matches go |

## tools/gomad/rust/wasmhost/src/storage_transport/vector_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `storage_context_sequence_rejects_changed_outcomes_and_inputs`:96 | convert-to-Rust-corpus | Violation of storage context sequence rejects changed outcomes and inputs |
| `storage_vector_guards_reject_changed_original_contracts`:121 | convert-to-Rust-corpus | Violation of storage vector guards reject changed original contracts |
| `rust_storage_transport_vectors_corpus`:349 | convert-to-Rust-corpus | Violation of rust storage transport vectors corpus |

## tools/gomad/rust/wasmhost/src/stress_jitter.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `stress_jitter_delay_is_bounded`:43 | convert-to-Rust-corpus | Violation of stress jitter delay is bounded |
| `stress_jitter_is_a_distinct_engine_identity`:50 | convert-to-Rust-corpus | Violation of stress jitter is a distinct engine identity |
| `stress_jitter_runs_inside_the_linked_role`:58 | convert-to-Rust-corpus | Violation of stress jitter runs inside the linked role |

## tools/gomad/rust/wasmhost/src/transcript.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `poll_input_length_matches_encoding`:263 | convert-to-Rust-corpus | Violation of poll input length matches encoding |
| `clock_run_rejects_folding_count_and_id_overflow`:281 | convert-to-Rust-corpus | Violation of clock run rejects folding count and id overflow |

## tools/gomad/rust/wasmhost/src/volume/allocation_probe.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/volume/config.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/volume/crash.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/volume/faults.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `go_generated_fault_decision_corpus`:350 | convert-to-Rust-corpus | Violation of go generated fault decision corpus |
| `go_generated_errno_mapping_corpus`:407 | convert-to-Rust-corpus | Violation of go generated errno mapping corpus |

## tools/gomad/rust/wasmhost/src/volume/hash.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `file_byte_mutable_borrows_invalidate_before_edit_and_clones_keep_independent_caches`:102 | convert-to-Rust-corpus | Violation of file byte mutable borrows invalidate before edit and clones keep independent caches |
| `raw_writes_preserve_stream_across_nonconsuming_finish`:124 | convert-to-Rust-corpus | Violation of raw writes preserve stream across nonconsuming finish |
| `integers_use_eight_little_endian_bytes`:145 | convert-to-Rust-corpus | Violation of integers use eight little endian bytes |
| `bytes_prefix_the_eight_byte_length`:167 | convert-to-Rust-corpus | Violation of bytes prefix the eight byte length |
| `strings_frame_utf8_byte_length_including_nul`:193 | convert-to-Rust-corpus | Violation of strings frame utf8 byte length including nul |
| `field_boundaries_and_order_change_the_digest`:203 | convert-to-Rust-corpus | Violation of field boundaries and order change the digest |

## tools/gomad/rust/wasmhost/src/volume/image.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `installed_lifecycle_readonly_volume_inventory_uses_actual_decoded_images`:639 | convert-to-Rust-corpus | Violation of installed lifecycle readonly volume inventory uses actual decoded images |
| `go_generated_image_admission_corpus`:734 | convert-to-Rust-corpus | Violation of go generated image admission corpus |
| `go_generated_image_identity_corpus`:851 | convert-to-Rust-corpus | Violation of go generated image identity corpus |
| `decoding_keeps_durable_payloads_without_constructing_discarded_namespace_nodes`:901 | convert-to-Rust-corpus | Violation of decoding keeps durable payloads without constructing discarded namespace nodes |
| `rehashed_images_refuse_persisted_inode_aliases_across_volumes_and_reserved_nodes`:926 | convert-to-Rust-corpus | Violation of rehashed images refuse persisted inode aliases across volumes and reserved nodes |
| `every_go_corpus_image_roundtrips_full_state_and_fault_occurrences`:1000 | convert-to-Rust-corpus | Violation of every go corpus image roundtrips full state and fault occurrences |

## tools/gomad/rust/wasmhost/src/volume/mod.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/volume/state.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `exploration_persistence_survives_a_crash_without_pending_selection`:1932 | convert-to-Rust-corpus | Violation of exploration persistence survives a crash without pending selection |
| `repeated_snapshots_do_not_rehash_unchanged_file_bytes`:1986 | convert-to-Rust-corpus | Violation of repeated snapshots do not rehash unchanged file bytes |
| `detached_open_file_preserves_data_until_the_last_close`:2010 | convert-to-Rust-corpus | Violation of detached open file preserves data until the last close |
| `allocation_write_and_sync_keep_link_separate_from_data`:2034 | convert-to-Rust-corpus | Violation of allocation write and sync keep link separate from data |
| `directory_metadata_and_unlink_sync_match_go_selection`:2043 | convert-to-Rust-corpus | Violation of directory metadata and unlink sync match go selection |
| `pure_core_matches_every_go_metadata_namespace_and_lifecycle_step`:2057 | convert-to-Rust-corpus | Violation of pure core matches every go metadata namespace and lifecycle step |
| `remove_all_checks_busy_working_directory_before_readonly`:2451 | convert-to-Rust-corpus | Violation of remove all checks busy working directory before readonly |
| `exhausted_inode_counter_refuses_create_and_planned_mkdir_without_wrap`:2463 | convert-to-Rust-corpus | Violation of exhausted inode counter refuses create and planned mkdir without wrap |
| `exhausted_lifecycle_inode_headroom_refuses_before_durable_or_volatile_mutation`:2484 | convert-to-Rust-corpus | Violation of exhausted lifecycle inode headroom refuses before durable or volatile mutation |
| `rejected_file_bounds_leave_data_history_and_offsets_unchanged`:2510 | convert-to-Rust-corpus | Violation of rejected file bounds leave data history and offsets unchanged |
| `replacing_open_target_retains_its_inode_until_last_close`:2525 | convert-to-Rust-corpus | Violation of replacing open target retains its inode until last close |
| `aggregate_byte_failure_keeps_go_record_before_growth_order`:2562 | convert-to-Rust-corpus | Violation of aggregate byte failure keeps go record before growth order |

## tools/gomad/rust/wasmhost/src/world/deflate/huffman.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/deflate/matcher.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/deflate/mod.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/deflate/reader.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/deflate/testdata/standalone.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/deflate/tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `default_compression_matches_pinned_go`:31 | convert-to-Rust-corpus | Violation of default compression matches pinned go |
| `decode_all_go_block_types_with_bounded_output`:73 | convert-to-Rust-corpus | Violation of decode all go block types with bounded output |
| `accepts_independent_go_compression_levels`:91 | convert-to-Rust-corpus | Violation of accepts independent go compression levels |
| `actual_world_snapshots_match_raw_gzip`:114 | convert-to-Rust-corpus | Violation of actual world snapshots match raw gzip |
| `mutated_stream_admission_matches_go`:142 | convert-to-Rust-corpus | Violation of mutated stream admission matches go |
| `original_full_profile_bounds`:174 | convert-to-Rust-corpus | Violation of original full profile bounds |
| `changing_distributions_and_lazy_matches_agree_with_go`:213 | convert-to-Rust-corpus | Violation of changing distributions and lazy matches agree with go |
| `fixed_empty_and_malformed_deflate`:244 | convert-to-Rust-corpus | Violation of fixed empty and malformed deflate |
| `dynamic_distance_code_count_bound`:284 | convert-to-Rust-corpus | Violation of dynamic distance code count bound |

## tools/gomad/rust/wasmhost/src/world/deflate/tokens.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/deflate/writer.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/digest.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/gzip.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `canonical_empty_and_small_streams`:94 | convert-to-Rust-corpus | Violation of canonical empty and small streams |
| `optional_headers_single_member_and_header_crc`:120 | convert-to-Rust-corpus | Violation of optional headers single member and header crc |

## tools/gomad/rust/wasmhost/src/world/mod.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/order.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world/owner_preservation_tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `binary_world_owner_rejects_foreign_seed_capacities_and_trailing_bytes`:78 | convert-to-Rust-corpus | Violation of binary world owner rejects foreign seed capacities and trailing bytes |
| `world_owner_codec_measured_cardinality`:113 | convert-to-Rust-corpus | Violation of world owner codec measured cardinality |
| `compressed_world_owner_measured_cardinality`:142 | convert-to-Rust-corpus | Violation of compressed world owner measured cardinality |

## tools/gomad/rust/wasmhost/src/world/retained.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `fixed_owner_tags_remain_valid_below_application_string_capacity`:268 | convert-to-Rust-corpus | Violation of fixed owner tags remain valid below application string capacity |
| `detached_world_replays_before_admitting_header_digest_or_trailing_bytes`:326 | convert-to-Rust-corpus | Violation of detached world replays before admitting header digest or trailing bytes |

## tools/gomad/rust/wasmhost/src/world/tests.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `inspection_chain_tracks_every_mutation_and_exact_expanded_wire_size`:30 | convert-to-Rust-corpus | Violation of inspection chain tracks every mutation and exact expanded wire size |
| `inspection_chain_binds_configuration_order_and_length_framed_payloads`:104 | convert-to-Rust-corpus | Violation of inspection chain binds configuration order and length framed payloads |
| `exhausted_transition_revision_refuses_every_mutation_before_state_changes`:149 | convert-to-Rust-corpus | Violation of exhausted transition revision refuses every mutation before state changes |
| `world_hmac_matches_rfc4231`:184 | convert-to-Rust-corpus | Violation of world hmac matches rfc4231 |
| `world_choice_rank_matches_go_vectors`:208 | convert-to-Rust-corpus | Violation of world choice rank matches go vectors |
| `world_identity_and_transition_exhaustion_do_not_wrap_or_mutate`:251 | convert-to-Rust-corpus | Violation of world identity and transition exhaustion do not wrap or mutate |
| `cached_state_digest_matches_uncached_after_every_mutation`:301 | convert-to-Rust-corpus | Violation of cached state digest matches uncached after every mutation |
| `state_digest_past_the_frame_bound_keeps_the_retained_prefix`:355 | convert-to-Rust-corpus | Violation of state digest past the frame bound keeps the retained prefix |
| `transition_byte_limit_refuses_before_mutation_at_the_exact_native_record_boundary`:393 | convert-to-Rust-corpus | Violation of transition byte limit refuses before mutation at the exact native record boundary |
| `every_world_transition_byte_refusal_preserves_the_accepted_prefix`:425 | convert-to-Rust-corpus | Violation of every world transition byte refusal preserves the accepted prefix |

## tools/gomad/rust/wasmhost/src/world/wire_generated.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/src/world_conformance.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `world_corpus_headers_match_the_go_pin`:35 | convert-to-Rust-corpus | Missing required named or scripted/generated/recorded cases, or invalid captured corpus header/pin |
| `world_corpus_replays_exactly`:203 | convert-to-Rust-corpus | Violation of world corpus replays exactly |
| `final_world_wires_round_trip_through_the_binary_owner`:251 | convert-to-Rust-corpus | Violation of final world wires round trip through the binary owner |

## tools/gomad/rust/wasmhost/tests/allocation_receiving.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `allocation_receiving`:10 | convert-to-Rust-corpus | Violation of allocation receiving |

## tools/gomad/rust/wasmhost/tests/cluster.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `manual_inspection_reaps_waiting_peer_and_preserves_never_started_node`:42 | convert-to-Rust-corpus | Violation of manual inspection reaps waiting peer and preserves never started node |
| `cluster_broker_runs_recorded_go_orders_with_central_faults`:539 | convert-to-Rust-corpus | Violation of cluster broker runs recorded go orders with central faults |

## tools/gomad/rust/wasmhost/tests/common/admission_corpus_literals.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `diagnostic_handoff_refusal_rejects_changed_frame_process_or_evidence`:845 | convert-to-Rust-corpus | Violation of diagnostic handoff refusal rejects changed frame process or evidence |

## tools/gomad/rust/wasmhost/tests/common/closed_cluster_guards.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `manual_restart_raw_result_guards_reject_wrong_kind_and_missing_nodes`:4 | convert-to-Rust-corpus | Violation of manual restart raw result guards reject wrong kind and missing nodes |
| `network_bound_midpoint_rejects_changed_limit_from_committed_streams`:58 | convert-to-Rust-corpus | Violation of network bound midpoint rejects changed limit from committed streams |
| `closed_cluster_cross_case_raw_and_evidence_guards`:650 | convert-to-Rust-corpus | Violation of closed cluster cross case raw and evidence guards |
| `closed_cluster_raw_guard_rejects_changed_literal_and_empty_probe`:703 | convert-to-Rust-corpus | Violation of closed cluster raw guard rejects changed literal and empty probe |

## tools/gomad/rust/wasmhost/tests/common/core_preservation_input.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/tests/common/helper_corpus_inputs.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/tests/common/helper_corpus_projection.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/tests/common/helper_feature_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `feature_sidecar_rejects_schema_source_build_case_and_unknown_field_drift`:91 | convert-to-Rust-corpus | Violation of feature sidecar rejects schema source build case and unknown field drift |
| `fuel_off_stage_comes_from_preserved_baseline_including_failed_instantiation`:202 | convert-to-Rust-corpus | Violation of fuel off stage comes from preserved baseline including failed instantiation |
| `fuel_off_sidecars_require_early_zero_and_terminal_offered_fuel`:230 | convert-to-Rust-corpus | Violation of fuel off sidecars require early zero and terminal offered fuel |
| `feature_hooks_reject_missing_malformed_unbounded_and_zero_raw_profiles`:283 | convert-to-Rust-corpus | Violation of feature hooks reject missing malformed unbounded and zero raw profiles |
| `feature_sidecars_cannot_replace_preserved_frames_evidence_or_profile_counts`:417 | convert-to-Rust-corpus | Violation of feature sidecars cannot replace preserved frames evidence or profile counts |
| `attribution_schema_extension_preserves_legacy_buckets`:446 | convert-to-Rust-corpus | Violation of attribution schema extension preserves legacy buckets |

## tools/gomad/rust/wasmhost/tests/common/inspection.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/tests/common/model_corpus_literals.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/tests/common/namespace_corpus_literals.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `namespace_corpus_literal_guards_reject_input_and_result_drift`:250 | convert-to-Rust-corpus | Violation of namespace corpus literal guards reject input and result drift |

## tools/gomad/rust/wasmhost/tests/common/network_inspection.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/tests/common/profile_stderr.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `llvm_profile_diagnostics_are_rejected`:17 | convert-to-Rust-corpus | Violation of llvm profile diagnostics are rejected |

## tools/gomad/rust/wasmhost/tests/common/residual_helper_literals.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `residual_helper_literals_reject_wrong_stage_and_callback_projection`:107 | convert-to-Rust-corpus | Violation of residual helper literals reject wrong stage and callback projection |

## tools/gomad/rust/wasmhost/tests/common/write_corpus_literals.rs

Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.

## tools/gomad/rust/wasmhost/tests/core_fault_preservation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `the_same_actual_guest_observes_only_the_explicit_execution_fault_bootstrap`:15 | convert-to-Rust-corpus | Violation of the same actual guest observes only the explicit execution fault bootstrap |

## tools/gomad/rust/wasmhost/tests/core_runtime_preservation.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `ordinary_linked_exit_requires_the_configured_runtime_finish`:29 | convert-to-Rust-corpus | Violation of ordinary linked exit requires the configured runtime finish |
| `native_outcomes_keep_real_exit_trap_fuel_and_output_claims_within_admitted_limits`:68 | convert-to-Rust-corpus | Violation of native outcomes keep real exit trap fuel and output claims within admitted limits |
| `native_preparation_refuses_invalid_inputs_before_guest_instantiation`:110 | convert-to-Rust-corpus | Violation of native preparation refuses invalid inputs before guest instantiation |
| `native_model_admission_rejects_unknown_fields_at_each_typed_level`:140 | convert-to-Rust-corpus | Violation of native model admission rejects unknown fields at each typed level |
| `rejected_stock_import_inventory_has_an_actual_ordinary_refusal_for_every_site`:169 | convert-to-Rust-corpus | Violation of rejected stock import inventory has an actual ordinary refusal for every site |
| `malformed_guest_memory_is_a_trap_instead_of_capacity_or_host_io`:229 | convert-to-Rust-corpus | Violation of malformed guest memory is a trap instead of capacity or host io |
| `runtime_replay_refuses_collection_without_finish_and_classifies_malformed_decisions_invalid`:245 | convert-to-Rust-corpus | Violation of runtime replay refuses collection without finish and classifies malformed decisions invalid |
| `guest_transport_classifications_return_after_native_execution_is_completed_and_joined`:320 | convert-to-Rust-corpus | Violation of guest transport classifications return after native execution is completed and joined |

## tools/gomad/rust/wasmhost/tests/determinism_observed.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `native_inspection_preserves_trace_decision_cuts_and_recovery_witness`:89 | convert-to-Rust-corpus | Violation of native inspection preserves trace decision cuts and recovery witness |
| `native_inspection_rejects_unknown_schema_and_tracks_zero_choice_incarnation`:115 | convert-to-Rust-corpus | Violation of native inspection rejects unknown schema and tracks zero choice incarnation |

## tools/gomad/rust/wasmhost/tests/diagnostic_roles.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `typed_native_role_admission_requires_the_exact_compiled_identity`:107 | convert-to-Rust-corpus | Violation of typed native role admission requires the exact compiled identity |
| `release_binary_cannot_authorize_a_diagnostic_or_stress_identity`:130 | convert-to-Rust-corpus | Violation of release binary cannot authorize a diagnostic or stress identity |
| `single_native_role_matrix_refuses_before_guest_output_and_joins_every_execution`:141 | convert-to-Rust-corpus | Violation of single native role matrix refuses before guest output and joins every execution |
| `cluster_native_role_matrix_refuses_before_any_store_and_joins_admitted_stores`:187 | convert-to-Rust-corpus | Violation of cluster native role matrix refuses before any store and joins admitted stores |
| `private_guest_profile_configuration_refuses_original_invalid_bounds_before_execution`:242 | convert-to-Rust-corpus | Violation of private guest profile configuration refuses original invalid bounds before execution |
| `every_fixture_fault_requires_explicit_native_role_before_single_or_cluster_start`:272 | convert-to-Rust-corpus | Violation of every fixture fault requires explicit native role before single or cluster start |
| `malformed_native_callback_memory_publishes_no_output_or_transcript`:347 | convert-to-Rust-corpus | Violation of malformed native callback memory publishes no output or transcript |

## tools/gomad/rust/wasmhost/tests/engine.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `determinism_refuses_unknown_execute_fields_and_kinds_before_started`:123 | convert-to-Rust-corpus | Violation of determinism refuses unknown execute fields and kinds before started |
| `determinism_refuses_unknown_remote_reply_fields_and_kinds`:140 | convert-to-Rust-corpus | Violation of determinism refuses unknown remote reply fields and kinds |
| `determinism_refuses_unknown_split_remote_reply_fields_and_kinds`:161 | convert-to-Rust-corpus | Violation of determinism refuses unknown split remote reply fields and kinds |
| `guest_profile_is_symbolized_bounded_and_preserves_evidence_on_write_failure`:239 | convert-to-Rust-corpus | Violation of guest profile is symbolized bounded and preserves evidence on write failure |
| `guest_profile_drops_epochs_elapsed_inside_host_calls`:329 | convert-to-Rust-corpus | Violation of guest profile drops epochs elapsed inside host calls |
| `reports_compile_cache_execution_and_resource_metrics_separately`:346 | convert-to-Rust-corpus | Violation of reports compile cache execution and resource metrics separately |
| `cache_hit_ignores_missing_or_replaced_module_bytes`:395 | convert-to-Rust-corpus | Violation of cache hit ignores missing or replaced module bytes |
| `cache_miss_verifies_module_before_compilation_or_publication`:439 | convert-to-Rust-corpus | Violation of cache miss verifies module before compilation or publication |
| `cache_rejects_poison_stale_links_and_permissive_storage`:478 | convert-to-Rust-corpus | Violation of cache rejects poison stale links and permissive storage |
| `cache_serializes_parallel_publication_without_preserving_guest_heap`:604 | convert-to-Rust-corpus | Violation of cache serializes parallel publication without preserving guest heap |
| `cache_identity_uses_running_inode_after_atomic_helper_replacement`:663 | convert-to-Rust-corpus | Violation of cache identity uses running inode after atomic helper replacement |

## tools/gomad/rust/wasmhost/tests/helper_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `write_callback_raw_lines_preserve_literal_key_order_and_bytes`:356 | convert-to-Rust-corpus | Violation of write callback raw lines preserve literal key order and bytes |
| `captured_write_callback_requires_literal_lf_without_cr_normalization`:409 | convert-to-Rust-corpus | Violation of captured write callback requires literal lf without cr normalization |
| `raw_wire_frames_preserve_duplicate_keys_without_accepting_extra_frames`:743 | convert-to-Rust-corpus | Violation of raw wire frames preserve duplicate keys without accepting extra frames |
| `residual_regeneration_selection_is_exact_and_generation_only`:928 | convert-to-Rust-corpus | Violation of residual regeneration selection is exact and generation only |
| `runtime_choice_hash_is_attributed_to_transcript`:934 | convert-to-Rust-corpus | Violation of runtime choice hash is attributed to transcript |
| `helper_transcript_corpus_reproduces_expected_outcomes`:946 | convert-to-Rust-corpus | Violation of helper transcript corpus reproduces expected outcomes |
| `namespace_ownership_literal_rejects_wrong_admission_stage`:981 | convert-to-Rust-corpus | Violation of namespace ownership literal rejects wrong admission stage |
| `loader_refuses_output_that_varies_across_runs`:988 | convert-to-Rust-corpus | Violation of loader refuses output that varies across runs |
| `loader_refuses_unscripted_requests_and_unused_replies`:999 | convert-to-Rust-corpus | Violation of loader refuses unscripted requests and unused replies |
| `derived_inputs_follow_their_producer`:1020 | convert-to-Rust-corpus | Violation of derived inputs follow their producer |
| `committed_cases_keep_their_cross_case_relations`:1025 | convert-to-Rust-corpus | Violation of committed cases keep their cross case relations |
| `model_refusal_literals_reject_wrong_stage_messages`:1030 | convert-to-Rust-corpus | Violation of model refusal literals reject wrong stage messages |
| `scatter_tuple_literals_reject_changed_payload_and_callback_ids`:1035 | convert-to-Rust-corpus | Violation of scatter tuple literals reject changed payload and callback ids |
| `inspection_schema_literals_reject_wrong_schema_and_world_encoding`:1040 | convert-to-Rust-corpus | Violation of inspection schema literals reject wrong schema and world encoding |
| `max_buffer_literals_reject_wrong_callback_identity_and_local_state`:1045 | convert-to-Rust-corpus | Violation of max buffer literals reject wrong callback identity and local state |

## tools/gomad/rust/wasmhost/tests/implementation_identity.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `compiled_identity_matches_the_exact_captured_native_source_and_build_context`:61 | convert-to-Rust-corpus | Violation of compiled identity matches the exact captured native source and build context |
| `native_storage_evidence_owners_rotate_identity_and_missing_owners_refuse`:115 | convert-to-Rust-corpus | Violation of native storage evidence owners rotate identity and missing owners refuse |
| `native_inspection_owners_rotate_identity_and_missing_owners_refuse`:133 | convert-to-Rust-corpus | Violation of native inspection owners rotate identity and missing owners refuse |
| `strict_typed_json_owners_rotate_identity_and_missing_owners_refuse`:146 | convert-to-Rust-corpus | Violation of strict typed json owners rotate identity and missing owners refuse |
| `empty_source_closure_refuses_before_producing_an_identity`:154 | convert-to-Rust-corpus | Violation of empty source closure refuses before producing an identity |

## tools/gomad/rust/wasmhost/tests/inventory.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `inventory_covers_every_operation`:418 | convert-to-Rust-corpus | Violation of inventory covers every operation |
| `operation_table_parser_fails_closed`:468 | convert-to-Rust-corpus | Violation of operation table parser fails closed |
| `inventory_loader_rejects_unknown_fields_and_reused_ids`:479 | convert-to-Rust-corpus | Violation of inventory loader rejects unknown fields and reused ids |
| `inventory_review_date_requires_a_calendar_day`:507 | convert-to-Rust-corpus | Violation of inventory review date requires a calendar day |
| `inventory_loader_rejects_every_malformed_row_variant`:537 | convert-to-Rust-corpus | Violation of inventory loader rejects every malformed row variant |
| `inventory_preserves_historical_identities`:613 | convert-to-Rust-corpus | Violation of inventory preserves historical identities |
| `identity_history_preserves_accepted_prefix_and_allows_appends`:640 | convert-to-Rust-corpus | Violation of identity history preserves accepted prefix and allows appends |
| `retired_import_requires_historical_identity_and_refusal_evidence`:689 | convert-to-Rust-corpus | Violation of retired import requires historical identity and refusal evidence |
| `completeness_rejects_unaccepted_retirement`:733 | convert-to-Rust-corpus | Violation of completeness rejects unaccepted retirement |
| `helper_source_scan_sites_are_inventoried`:1203 | convert-to-Rust-corpus | Violation of helper source scan sites are inventoried |
| `retired_helper_allowlist_does_not_cover_new_sites`:1221 | convert-to-Rust-corpus | Violation of retired helper allowlist does not cover new sites |
| `helper_source_scan_skips_only_test_items`:1248 | convert-to-Rust-corpus | Violation of helper source scan skips only test items |
| `source_scan_skips_multiline_raw_and_combined_test_items`:1274 | convert-to-Rust-corpus | Violation of source scan skips multiline raw and combined test items |
| `source_scan_explicit_roots_own_their_sibling_module_directory`:1298 | convert-to-Rust-corpus | Violation of source scan explicit roots own their sibling module directory |
| `source_scan_refuses_inline_modules_that_hide_external_children`:1372 | convert-to-Rust-corpus | Violation of source scan refuses inline modules that hide external children |
| `engine_settings_are_inventoried`:1446 | convert-to-Rust-corpus | Violation of engine settings are inventoried |
| `serde_json_objects_serialize_in_sorted_key_order`:1492 | convert-to-Rust-corpus | Violation of serde json objects serialize in sorted key order |
| `inventory_pins_require_append_only_dated_reviews`:1606 | convert-to-Rust-corpus | Violation of inventory pins require append only dated reviews |
| `review_history_rejects_stale_dates_and_non_utc_timestamps`:1628 | convert-to-Rust-corpus | Violation of review history rejects stale dates and non utc timestamps |
| `inventory_reviewed_versions_match_exact_candidate_pins`:1672 | convert-to-Rust-corpus | Violation of inventory reviewed versions match exact candidate pins |
| `every_escape_resolves_a_readme_heading`:1693 | convert-to-Rust-corpus | Violation of every escape resolves a readme heading |
| `inventory_covers_every_diagnostic_draw_stream`:1805 | convert-to-Rust-corpus | Violation of inventory covers every diagnostic draw stream |
| `appended_retired_identity_cannot_reuse_an_existing_site`:1842 | convert-to-Rust-corpus | Violation of appended retired identity cannot reuse an existing site |
| `inventory_admits_only_retired_go_migration_reference_shapes`:1871 | convert-to-Rust-corpus | Violation of inventory admits only retired go migration reference shapes |
| `inventory_rejects_invalid_stable_ids_and_reference_paths`:1891 | convert-to-Rust-corpus | Violation of inventory rejects invalid stable ids and reference paths |

## tools/gomad/rust/wasmhost/tests/inventory/architecture.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `native_modeled_state_rejects_host_imports_on_both_source_platforms`:142 | convert-to-Rust-corpus | Violation of native modeled state rejects host imports on both source platforms |
| `modeled_state_io_admits_named_types_and_refuses_host_streams`:187 | convert-to-Rust-corpus | Violation of modeled state io admits named types and refuses host streams |

## tools/gomad/rust/wasmhost/tests/inventory/evidence.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `inventory_native_and_retained_model_evidence_resolves`:127 | convert-to-Rust-corpus | Violation of inventory native and retained model evidence resolves |
| `inventory_all_evidence_references_resolve`:165 | convert-to-Rust-corpus | Violation of inventory all evidence references resolve |
| `evidence_requires_an_adjacent_test_attribute_across_closed_multiline_attributes`:185 | convert-to-Rust-corpus | Violation of evidence requires an adjacent test attribute across closed multiline attributes |

## tools/gomad/rust/wasmhost/tests/inventory/evidence_migrations.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `evidence_migrations_bind_complete_reviewed_original_assertions_without_original_pass`:191 | convert-to-Rust-corpus | Violation of evidence migrations bind complete reviewed original assertions without original pass |
| `pending_claims_and_missing_receivers_cannot_resolve_as_complete_migrations`:197 | convert-to-Rust-corpus | Violation of pending claims and missing receivers cannot resolve as complete migrations |
| `retired_go_peer_wire_has_no_installed_native_entrypoint_and_reference_helper_stays_explicit`:243 | convert-to-Rust-corpus | Violation of retired go peer wire has no installed native entrypoint and reference helper stays explicit |

## tools/gomad/rust/wasmhost/tests/inventory/host.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `receiving_host_effect_sites_are_inventoried`:239 | convert-to-Rust-corpus | Violation of receiving host effect sites are inventoried |
| `host_migrations_preserve_exact_identity_and_active_receiving_sites`:278 | convert-to-Rust-corpus | Violation of host migrations preserve exact identity and active receiving sites |
| `unlisted_appended_host_retirement_is_not_a_source_migration`:407 | convert-to-Rust-corpus | Violation of unlisted appended host retirement is not a source migration |

## tools/gomad/rust/wasmhost/tests/inventory/runtime.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `source_inventory_preserves_the_collector_file_patch_prohibition`:258 | convert-to-Rust-corpus | Violation of source inventory preserves the collector file patch prohibition |
| `wasip1_runtime_draw_and_clock_references_are_reviewed`:373 | convert-to-Rust-corpus | Violation of wasip1 runtime draw and clock references are reviewed |
| `runtime_historical_retirement_keeps_appended_sites_active`:432 | convert-to-Rust-corpus | Violation of runtime historical retirement keeps appended sites active |
| `wasip1_seed_finding_and_seeded_draw_decision_premises`:480 | convert-to-Rust-corpus | Violation of wasip1 seed finding and seeded draw decision premises |

## tools/gomad/rust/wasmhost/tests/linked_compile_cache.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `linked_cache_publishes_once_reuses_and_refuses_poison_or_links`:75 | convert-to-Rust-corpus | Violation of linked cache publishes once reuses and refuses poison or links |
| `linked_cache_defaults_disabled_and_rejects_relative_configuration`:123 | convert-to-Rust-corpus | Violation of linked cache defaults disabled and rejects relative configuration |
| `cluster_cache_uses_same_private_store_and_poison_is_host_failure`:198 | convert-to-Rust-corpus | Violation of cluster cache uses same private store and poison is host failure |

## tools/gomad/rust/wasmhost/tests/linked_evidence_bounds.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `standalone_accepts_original_activity_aggregate_without_reserving_it`:41 | convert-to-Rust-corpus | Violation of standalone accepts original activity aggregate without reserving it |
| `cluster_accepts_original_activity_aggregate_without_reserving_it`:50 | convert-to-Rust-corpus | Violation of cluster accepts original activity aggregate without reserving it |
| `aggregate_zero_is_still_refused`:99 | convert-to-Rust-corpus | Violation of aggregate zero is still refused |

## tools/gomad/rust/wasmhost/tests/linked_simulation_corpus.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `linked_cluster_network_query_uses_the_broker_instead_of_standalone_fallback`:88 | convert-to-Rust-corpus | Violation of linked cluster network query uses the broker instead of standalone fallback |
| `scenario_forcing_runs_fresh_stores_and_retains_repeatable_actual_world`:116 | convert-to-Rust-corpus | Violation of scenario forcing runs fresh stores and retains repeatable actual world |
| `go_issued_simulation_plans_force_the_observed_scenario_rank`:163 | convert-to-Rust-corpus | Violation of go issued simulation plans force the observed scenario rank |
| `unconsumed_or_changed_live_alternatives_preserve_typed_divergence`:194 | convert-to-Rust-corpus | Violation of unconsumed or changed live alternatives preserve typed divergence |
| `live_incarnation_watchdog_shutdown_evidence_is_repeatable`:228 | convert-to-Rust-corpus | Violation of live incarnation watchdog shutdown evidence is repeatable |
| `divergence_stops_live_incarnations_in_sorted_order`:349 | convert-to-Rust-corpus | Violation of divergence stops live incarnations in sorted order |

## tools/gomad/rust/wasmhost/tests/linked_streaming.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `short_output_callbacks_apply_backpressure_to_completion`:103 | convert-to-Rust-corpus | Violation of short output callbacks apply backpressure to completion |
| `concurrent_requests_keep_streams_arguments_and_completion_isolated`:135 | convert-to-Rust-corpus | Violation of concurrent requests keep streams arguments and completion isolated |
| `consumer_failure_retains_admitted_bytes_once_and_stops_execution`:187 | convert-to-Rust-corpus | Violation of consumer failure retains admitted bytes once and stops execution |
| `streaming_does_not_forward_output_rejected_by_namespace_limit`:202 | convert-to-Rust-corpus | Violation of streaming does not forward output rejected by namespace limit |
| `streaming_does_not_forward_output_rejected_by_evidence_limit`:217 | convert-to-Rust-corpus | Violation of streaming does not forward output rejected by evidence limit |
| `cancellation_retains_and_forwards_only_available_output`:243 | convert-to-Rust-corpus | Violation of cancellation retains and forwards only available output |
| `streaming_and_capture_only_retain_identical_evidence`:267 | convert-to-Rust-corpus | Violation of streaming and capture only retain identical evidence |

## tools/gomad/rust/wasmhost/tests/replay_preflight.rs

| Test / source line | Disposition | Named failure contract |
| --- | --- | --- |
| `replay_preflight_refuses_before_instantiation`:19 | convert-to-Rust-corpus | Violation of replay preflight refuses before instantiation |
