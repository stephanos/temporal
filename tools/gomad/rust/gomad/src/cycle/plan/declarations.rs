use crate::cycle::model::{
    Adapter, Bounds, Flag, Isolation, RuntimeBounds, Semantic, StageContract,
};

type ResourceDeclaration = (&'static str, &'static str, [f64; 3], u32);
type SemanticDeclaration = (&'static str, &'static [&'static str], u32);

pub(super) fn contracts() -> Vec<StageContract> {
    let mut stages: Vec<_> = BEFORE_RUNTIME.iter().map(resource).collect();
    for index in 0..32 {
        let mut row = stage(
            &format!("runtime-{index}"),
            "ordinary-helper-publication",
            [2.0, 1.0, 8.0],
            2,
        );
        row.bounds.runtime = Some(RuntimeBounds {
            instances: 25,
            replays: 1,
            seed: index * 1_000_003,
            shard_wall_seconds: 600,
        });
        stages.push(row);
    }
    stages.extend(AFTER_RUNTIME.iter().map(resource));
    stages
}

fn resource(&(label, dependencies, resources, tier): &ResourceDeclaration) -> StageContract {
    stage(label, dependencies, resources, tier)
}

fn stage(label: &str, dependencies: &str, resources: [f64; 3], tier: u32) -> StageContract {
    let dependencies = if label == "runtime-aggregate" {
        (0..32).map(|index| format!("runtime-{index}")).collect()
    } else {
        names(dependencies)
    };
    let [cores, mem_gib, estimated_seconds] = resources;
    let adapter = adapter(label);
    StageContract {
        label: label.into(),
        dependencies,
        isolation_class: isolation(label),
        cores,
        mem_gib,
        estimated_seconds,
        locks: locks(label),
        scratch: scratch(label),
        flags: flags(label),
        tier,
        adapter,
        bounds: Bounds {
            command_default_timeout_s: None,
            primary_timeout_s: primary_timeout(label),
            static_timeout_s: (adapter == Adapter::CheckedCommand).then_some(3600),
            runtime: None,
        },
        required_subgates: vec![],
        repetitions: 0,
    }
}

fn isolation(label: &str) -> Isolation {
    match label {
        "helper-corpus-regen-1" | "helper-corpus-regen-2" => Isolation::Solo,
        "busy-boundaries" | "busy-grouped" | "terminal-host-admission" | "terminal-exit" => {
            Isolation::Exclusive
        }
        "workload-fixture-controlled-success"
        | "workload-fixture-controlled-canary"
        | "workload-fixture-cooperative-success"
        | "workload-fixture-cooperative-canary"
        | "workload-sqlite-cooperative-success"
        | "workload-sqlite-cooperative-canary"
        | "workload-onebox-cooperative-success"
        | "workload-onebox-cooperative-canary"
        | "workload-multi-instance-cooperative-success"
        | "workload-multi-instance-cooperative-canary"
        | "sqlite-selection-0"
        | "sqlite-selection-1"
        | "sqlite-selection-2"
        | "sqlite-selection-3" => Isolation::Budgeted,
        _ => Isolation::Normal,
    }
}

fn adapter(label: &str) -> Adapter {
    match label {
        "validate"
        | "portable"
        | "default-build"
        | "default-vet"
        | "compile-only"
        | "lint-router"
        | "lint-router-vet"
        | "actionlint-build"
        | "actionlint-version"
        | "workflow-lint"
        | "lint-module"
        | "architecture"
        | "lint-root"
        | "test-harness"
        | "world-race"
        | "rust-lint"
        | "rust-default-test"
        | "rust-conformance-unit"
        | "rust-diagnostic-test"
        | "rust-fuel-off-test"
        | "rust-guest-profile-test"
        | "rust-combined-test"
        | "helper-corpus-regen-1"
        | "helper-corpus-regen-2" => Adapter::CheckedCommand,
        "runtime-aggregate" => Adapter::RuntimeAggregate,
        "workload-corpus" | "workload-corpus-after-relocation" => Adapter::Corpus,
        "sqlite-catalogue" => Adapter::SqliteCatalogue,
        value if value.starts_with("runtime-") => Adapter::Runtime,
        value if value.starts_with("workload-") => Adapter::Qualify,
        value if value.starts_with("sqlite-selection-") => Adapter::SqliteSelection,
        _ => Adapter::Primary,
    }
}

fn flags(label: &str) -> Vec<Flag> {
    match label {
        value if value.starts_with("runtime-") && value != "runtime-aggregate" => {
            vec![Flag::Memguard]
        }
        "rust-lint"
        | "rust-default-test"
        | "rust-conformance-unit"
        | "rust-diagnostic-test"
        | "rust-fuel-off-test"
        | "rust-guest-profile-test"
        | "rust-combined-test"
        | "helper-build"
        | "conformance-helper-build"
        | "ordinary"
        | "terminal-module"
        | "storage"
        | "network"
        | "clusters" => vec![Flag::Heavy],
        "terminal-host-admission" | "terminal-exit" => vec![Flag::Cleantree],
        "temporal-inspection" => vec![Flag::Bg, Flag::Heavy],
        _ => vec![],
    }
}

fn locks(label: &str) -> Vec<String> {
    match label {
        "lint-module" | "lint-root" => names("golangci"),
        "rust-lint"
        | "rust-default-test"
        | "rust-conformance-unit"
        | "rust-diagnostic-test"
        | "rust-fuel-off-test"
        | "rust-guest-profile-test"
        | "rust-combined-test"
        | "helper-corpus-regen-1"
        | "helper-corpus-regen-2" => names("cargo-check-target"),
        _ => vec![],
    }
}

fn scratch(label: &str) -> Vec<String> {
    match label {
        "ordinary" => names("common/gomaddebug/testdata/inspection-*"),
        "temporal-inspection" => names("service/history/.gomad-inspection-*"),
        _ => vec![],
    }
}

fn primary_timeout(label: &str) -> Option<u64> {
    match label {
        "ordinary"
        | "storage"
        | "network"
        | "temporal-inspection"
        | "conformance-helper-build"
        | "wasmmeasure-build"
        | "helper-build"
        | "terminal-module"
        | "terminal-exit" => Some(3600),
        "clusters" => Some(5400),
        "retained-durable-restart"
        | "terminal-controls"
        | "ordinary-helper-publication"
        | "terminal-host-admission" => Some(900),
        "busy-grouped" | "busy-boundaries" | "network-streams" | "operator" => Some(1800),
        "artifact" => Some(2700),
        "cluster-provider" | "cluster-functional" | "retained-cluster" => Some(7200),
        _ => None,
    }
}

fn names(value: &str) -> Vec<String> {
    value
        .split(',')
        .filter(|name| !name.is_empty())
        .map(str::to_owned)
        .collect()
}

pub(super) fn semantics() -> Vec<Semantic> {
    let mut rows: Vec<_> = SEMANTIC_BEFORE_RUNTIME.iter().map(semantic).collect();
    for index in 0..32 {
        rows.push(Semantic {
            label: format!("runtime-{index}"),
            required_subgates: vec!["TestCooperativeRuntimeQualification".into()],
            repetitions: 1,
        });
    }
    rows.extend(SEMANTIC_AFTER_RUNTIME.iter().map(semantic));
    rows
}

fn semantic(&(label, subgates, repetitions): &SemanticDeclaration) -> Semantic {
    Semantic {
        label: label.into(),
        required_subgates: subgates.iter().map(|name| (*name).into()).collect(),
        repetitions,
    }
}

const BEFORE_RUNTIME: &[ResourceDeclaration] = &[
    ("validate", "", [2.0, 2.0, 19.0], 0),
    ("portable", "validate", [2.0, 2.0, 7.0], 0),
    ("default-build", "validate", [2.0, 2.0, 2.0], 0),
    ("default-vet", "validate", [2.0, 2.0, 2.0], 0),
    ("compile-only", "validate", [2.0, 2.0, 13.0], 0),
    ("lint-router", "validate", [2.0, 2.0, 34.0], 0),
    ("lint-router-vet", "validate", [1.0, 1.0, 3.0], 0),
    ("actionlint-build", "validate", [1.0, 1.0, 2.0], 0),
    ("actionlint-version", "actionlint-build", [1.0, 1.0, 1.0], 0),
    ("workflow-lint", "actionlint-version", [1.0, 1.0, 1.0], 0),
    ("lint-module", "validate", [4.0, 4.0, 11.0], 0),
    ("architecture", "validate", [2.0, 2.0, 115.0], 0),
    ("lint-root", "validate", [3.0, 4.0, 120.0], 0),
    ("test-harness", "compile-only", [2.0, 2.0, 72.0], 1),
    ("world-race", "compile-only", [2.0, 2.0, 19.0], 1),
    ("rust-lint", "validate", [1.0, 2.0, 94.0], 1),
    ("rust-default-test", "rust-lint", [1.0, 2.0, 236.0], 1),
    (
        "rust-conformance-unit",
        "rust-default-test",
        [1.0, 2.0, 21.0],
        1,
    ),
    (
        "rust-diagnostic-test",
        "rust-conformance-unit",
        [1.0, 2.0, 80.0],
        1,
    ),
    (
        "rust-fuel-off-test",
        "rust-diagnostic-test",
        [1.0, 2.0, 88.0],
        1,
    ),
    (
        "rust-guest-profile-test",
        "rust-fuel-off-test",
        [1.0, 2.0, 168.0],
        1,
    ),
    (
        "rust-combined-test",
        "rust-guest-profile-test",
        [1.0, 2.0, 89.0],
        1,
    ),
    (
        "helper-corpus-regen-1",
        "rust-combined-test",
        [1.0, 2.0, 9.0],
        1,
    ),
    (
        "helper-corpus-regen-2",
        "helper-corpus-regen-1",
        [1.0, 2.0, 9.0],
        1,
    ),
    (
        "wasmmeasure-build",
        "validate,portable,default-build,default-vet,compile-only,lint-router,lint-router-vet,workflow-lint,lint-module",
        [2.0, 2.0, 7.0],
        1,
    ),
    ("terminal-controls", "compile-only", [2.0, 3.0, 7.0], 1),
    (
        "helper-build",
        "wasmmeasure-build,validate,portable,default-build,default-vet,compile-only,lint-router,lint-router-vet,workflow-lint,lint-module",
        [4.0, 4.0, 247.0],
        1,
    ),
    (
        "conformance-helper-build",
        "wasmmeasure-build,validate,portable,default-build,default-vet,compile-only,lint-router,lint-router-vet,workflow-lint,lint-module",
        [4.0, 4.0, 190.0],
        1,
    ),
    (
        "ordinary-helper-publication",
        "helper-build,conformance-helper-build",
        [1.0, 1.0, 1.0],
        1,
    ),
    (
        "ordinary",
        "ordinary-helper-publication",
        [4.0, 5.0, 1151.0],
        2,
    ),
    (
        "retained-durable-restart",
        "ordinary-helper-publication",
        [2.0, 1.0, 16.0],
        2,
    ),
    (
        "cluster-provider",
        "ordinary-helper-publication",
        [4.0, 1.0, 138.0],
        2,
    ),
    (
        "cluster-functional",
        "ordinary-helper-publication",
        [2.0, 1.0, 12.0],
        2,
    ),
    (
        "busy-boundaries",
        "ordinary-helper-publication",
        [12.0, 1.0, 30.0],
        2,
    ),
    (
        "busy-grouped",
        "ordinary-helper-publication",
        [12.0, 1.0, 191.0],
        2,
    ),
    (
        "network-streams",
        "ordinary-helper-publication",
        [2.0, 1.0, 7.0],
        2,
    ),
    (
        "retained-cluster",
        "ordinary-helper-publication",
        [2.0, 1.0, 114.0],
        2,
    ),
    (
        "terminal-module",
        "ordinary-helper-publication",
        [4.0, 5.0, 191.0],
        2,
    ),
    (
        "terminal-host-admission",
        "terminal-module",
        [1.0, 1.0, 3.0],
        2,
    ),
    (
        "terminal-exit",
        "terminal-host-admission",
        [4.0, 6.0, 251.0],
        2,
    ),
];

const AFTER_RUNTIME: &[ResourceDeclaration] = &[
    ("runtime-aggregate", "", [1.0, 1.0, 2.0], 2),
    (
        "artifact",
        "ordinary-helper-publication",
        [2.0, 1.0, 81.0],
        2,
    ),
    (
        "operator",
        "ordinary-helper-publication",
        [2.0, 1.0, 43.0],
        2,
    ),
    (
        "storage",
        "ordinary-helper-publication",
        [4.0, 4.0, 404.0],
        2,
    ),
    ("temporal-inspection", "helper-build", [6.0, 9.0, 1296.0], 2),
    (
        "network",
        "ordinary-helper-publication",
        [4.0, 5.0, 208.0],
        2,
    ),
    (
        "clusters",
        "ordinary-helper-publication",
        [4.0, 4.0, 298.0],
        2,
    ),
    (
        "workload-fixture-controlled-success",
        "clusters",
        [2.0, 1.0, 8.0],
        2,
    ),
    (
        "workload-fixture-controlled-canary",
        "workload-fixture-controlled-success",
        [2.0, 1.0, 9.0],
        2,
    ),
    (
        "workload-fixture-cooperative-success",
        "workload-fixture-controlled-canary",
        [2.0, 1.0, 11.0],
        2,
    ),
    (
        "workload-fixture-cooperative-canary",
        "workload-fixture-cooperative-success",
        [2.0, 1.0, 11.0],
        2,
    ),
    (
        "workload-sqlite-cooperative-success",
        "clusters",
        [2.0, 2.0, 57.0],
        2,
    ),
    (
        "workload-sqlite-cooperative-canary",
        "workload-sqlite-cooperative-success",
        [2.0, 2.0, 33.0],
        2,
    ),
    (
        "workload-onebox-cooperative-success",
        "clusters",
        [2.0, 5.0, 247.0],
        2,
    ),
    (
        "workload-onebox-cooperative-canary",
        "workload-onebox-cooperative-success",
        [2.0, 5.0, 102.0],
        2,
    ),
    (
        "workload-multi-instance-cooperative-success",
        "clusters",
        [2.0, 5.0, 729.0],
        2,
    ),
    (
        "workload-multi-instance-cooperative-canary",
        "workload-multi-instance-cooperative-success",
        [2.0, 5.0, 596.0],
        2,
    ),
    (
        "workload-corpus",
        "workload-fixture-cooperative-canary,workload-sqlite-cooperative-canary,workload-onebox-cooperative-canary,workload-multi-instance-cooperative-canary",
        [1.0, 1.0, 1.0],
        2,
    ),
    ("sqlite-selection-0", "workload-corpus", [2.0, 2.0, 29.0], 2),
    (
        "sqlite-selection-1",
        "sqlite-selection-0",
        [2.0, 2.0, 34.0],
        2,
    ),
    (
        "sqlite-selection-2",
        "sqlite-selection-1",
        [2.0, 2.0, 41.0],
        2,
    ),
    (
        "sqlite-selection-3",
        "sqlite-selection-2",
        [2.0, 2.0, 40.0],
        2,
    ),
    (
        "workload-corpus-after-relocation",
        "sqlite-selection-3",
        [1.0, 1.0, 1.0],
        2,
    ),
    (
        "sqlite-catalogue",
        "workload-corpus-after-relocation",
        [1.0, 1.0, 2.0],
        2,
    ),
];

const SEMANTIC_BEFORE_RUNTIME: &[SemanticDeclaration] = &[
    ("validate", &["validate"], 1),
    ("portable", &["portable"], 1),
    ("default-build", &["default-build"], 1),
    ("default-vet", &["default-vet"], 1),
    ("compile-only", &["compile-only"], 1),
    ("lint-router", &["lint-router"], 1),
    ("lint-router-vet", &["lint-router-vet"], 1),
    ("actionlint-build", &["actionlint-build"], 1),
    ("actionlint-version", &["actionlint-version"], 1),
    ("workflow-lint", &["workflow-lint"], 1),
    ("lint-module", &["lint-module"], 1),
    ("architecture", &["architecture"], 1),
    ("lint-root", &["lint-root"], 1),
    ("test-harness", &["test-harness"], 1),
    ("world-race", &["world-race"], 1),
    ("rust-lint", &["rust-lint"], 1),
    ("rust-default-test", &["rust-default-test"], 1),
    ("rust-conformance-unit", &["rust-conformance-unit"], 1),
    ("rust-diagnostic-test", &["rust-diagnostic-test"], 1),
    ("rust-fuel-off-test", &["rust-fuel-off-test"], 1),
    ("rust-guest-profile-test", &["rust-guest-profile-test"], 1),
    ("rust-combined-test", &["rust-combined-test"], 1),
    ("helper-corpus-regen-1", &["helper-corpus-regen-1"], 1),
    ("helper-corpus-regen-2", &["helper-corpus-regen-2"], 1),
    ("wasmmeasure-build", &["wasmmeasure-build"], 1),
    (
        "terminal-controls",
        &[
            "TestTerminalExitAuthorizationMismatchAfterStarted",
            "TestTerminalExitExpiredContextDoesNotLaunchHelper",
            "TestTerminalExitMeasurementControls",
            "TestTerminalExitMeasurementSendsCapturedInputsBeforeStarted",
            "TestTerminalExitObserverReceivesCompleteAcceptedFrameBeforeValidation",
        ],
        1,
    ),
    ("helper-build", &["helper-build"], 1),
    ("conformance-helper-build", &["conformance-helper-build"], 1),
    (
        "ordinary-helper-publication",
        &["ordinary-helper-publication"],
        1,
    ),
    ("ordinary", &["TestNamespaceConformanceCorpus"], 1),
    (
        "retained-durable-restart",
        &["TestProviderActualCrashDurableRestart"],
        1,
    ),
    (
        "cluster-provider",
        &[
            "TestBinaryWorldEvidenceUsesSharedOwnerDecoder",
            "TestBoundRetainedReplayRejectsBeforeWritingInputs",
            "TestBrokerWorldUsesBinaryOwnerCodec",
            "TestClockConformanceCorpusIsCurrent",
            "TestClockConformanceDriftPinIsCurrent",
            "TestClockConformanceGenerationIsDeterministic",
            "TestClusterShutdownBoundsFinishedClusterByExecutionDeadline",
            "TestCompletedResultsDetachMutableObservationAndCostSurfaces",
            "TestCompletedSummariesDetachCountersWithoutEvidencePayloads",
            "TestCompressedWorldOwnerMeasuredCardinality",
            "TestExplicitWorldProfileRequiresExplicitCapacities",
            "TestHistoryObservationRejectsAmbiguousInputs",
            "TestImplementationIdentityBindsInspectionOwners",
            "TestInspectionCapacityRetainsTemporalScaleBoundariesAndEffects",
            "TestInspectionChoiceMappingUsesDecisionsRatherThanObservations",
            "TestInspectionChoiceMappingUsesPredecisionCoordinateForCreationSites",
            "TestInspectionControlAuthenticatesRawCutAndRegistry",
            "TestInspectionControlRefusesUnregisteredAndIncompleteBeforeReplay",
            "TestInspectionInterruptedSelectPrefixKeepsDecisionMapping",
            "TestInspectionNetworkRawStreamBindsCanonicalIdentity",
            "TestInspectionSearchBisectDeclaresConditionalityAndRejectsObservedReversal",
            "TestInspectionSearchBudgetsAndPredicateErrorsStayInconclusive",
            "TestInspectionSearchRejectsRequestsBeforeEvaluation",
            "TestInspectionSearchScanDoesNotSkipRecoveredFailures",
            "TestInspectionSearchWallBoundIncludesEvaluatorCleanup",
            "TestMarkerBoundaryPreservesSelectedCapacityAndPriorErrors",
            "TestOpenClusterRecordsHelperExecutionDeadline",
            "TestOperatorAcceptsDeclaredStorageCrash",
            "TestOperatorArgumentsSurviveNormalization",
            "TestOperatorStopsActionsAfterFailedWait",
            "TestOperatorStrictPreflight",
            "TestProviderActualCrashDurableRestart",
            "TestProviderBindsEverySelectedBootNetworkLimit",
            "TestProviderCanceledSplitRevocationCancelsAndReaps",
            "TestProviderClusterStartsOnlyRequestedIncarnations",
            "TestProviderCompressedIndependentGuestsLifecycleReplay",
            "TestProviderDetachedOracleCanarySourceLessReplay",
            "TestProviderGroupedAcknowledgmentCrashSourceLessReplay",
            "TestProviderIndependentGuestsLifecycleReplay",
            "TestProviderMandatoryCleanupReapsWithoutLifecycleActions",
            "TestProviderOutputCrashKeepsItsExistingBoundary",
            "TestProviderStopReapsGuestAndPeerContinues",
            "TestProviderStorageCrashRequiresActualDurableBoundary",
            "TestProviderUnboundRevocationOfSplitStorageNodeRestarts",
            "TestRejectedRestartLeavesNoUnownedBrokerSession",
            "TestRetainedAdmissionConsumesPrivateInputsOnlyOnce",
            "TestRetainedCanonicalDecodeAvoidsWholeBundleEncodingAllocation",
            "TestRetainedCanonicalWriterChecksEveryWriteAndBase64Close",
            "TestRetainedCanonicalWriterMatchesAllDeclaredFields",
            "TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput",
            "TestRetainedDecoderAcceptsSharedCampaignPayloadPath",
            "TestRetainedLifecycleRejectsHostCallbacksBeforeReplay",
            "TestRetainedOrdinarySpellingKeepsRawAuthenticationAndPresence",
            "TestRetainedPrimaryBindingRejectsMissingAndForeignModules",
            "TestRetainedReplayRejectsTamperedOrMismatchedTargetModule",
            "TestRetainedReplayRequiresTheInstalledHelper",
            "TestRetainedSchemaV1IsRefusedWithVersionedError",
            "TestRetainedStorageSelectionProjectionIsDetached",
            "TestRetentionCapacityPreflightUsesEncodedLowerBound",
            "TestSQLiteObservationUsesFreshFixedLines",
            "TestSQLiteObservationsForwardToNativeOracleOwner",
            "TestSQLiteOraclePlanRequiresSealedIntent",
            "TestSoftAssertionActualClusterGuestAndRetainedReplay",
            "TestSoftAssertionActualClusterOutputCapacityRemainsInfrastructureFailure",
            "TestSoftAssertionCapacityPrecedesFinalizationErrors",
            "TestSoftAssertionClusterCapacityCountersNeverProveCompleteness",
            "TestSoftAssertionClusterOptOutRequiresRecordedTestWorkload",
            "TestSoftAssertionIncompleteClusterEvidenceIsInconclusive",
            "TestSoftAssertionOutputCapacityAnnotationDoesNotBroadenOtherFailures",
            "TestSoftAssertionRetainedEvidenceRequiresActualOutput",
            "TestSoftAssertionRetainedLiteralBracesRederiveCompleteMessage",
            "TestSoftAssertionRetainedTruncatedOutputIsInconclusive",
            "TestStorageCrashBoundOnlyByCrashBefore",
            "TestStressRetainedReopenPreservesBoundInputsAndRestoresHostOptIn",
            "TestWorldConformanceCorpusIsCurrent",
            "TestWorldConformanceDriftPinIsCurrent",
            "TestWorldConformanceFinalWireRoundTrips",
            "TestWorldOwnerCodecMeasuredCardinality",
        ],
        1,
    ),
    (
        "cluster-functional",
        &[
            "TestBrokerOrdersAreCurrent",
            "TestBrokerOrdersRustNetworkEvidenceRebuildsGoDocuments",
            "TestClusterBootAdmitsStorageCrashSelectionBeforeSealing",
            "TestClusterBrokerDurableRestartParity",
            "TestClusterEventOrderAccessorIsDetached",
            "TestClusterFinishPreservesSessionWithLiveIncarnations",
            "TestClusterFinishWaitsForInspectionCleanup",
            "TestClusterHostTerminationStatus",
            "TestClusterInspectedPrefixRequiresRequestedSealedInstance",
            "TestClusterInspectionAbortsNeighborsWithoutCompletingTheirTapes",
            "TestClusterInspectionMetadataDoesNotReplaceWorldEvidence",
            "TestClusterInternalDeadlineIsWatchdog",
            "TestClusterNetworkDecoderRefusesMalformedEvidence",
            "TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses",
            "TestClusterNetworkRustEvidenceRebuildsTheGoBrokerDocument",
            "TestClusterOutputBoundaryParity",
            "TestClusterPreservesWholeRunInfrastructureFailure",
            "TestClusterProtocolFailureIsNotCancellation",
            "TestClusterRefusedRestartLeavesIncarnationRetryable",
            "TestClusterRejectsGuestCallbacks",
            "TestClusterRejectsResultWithoutModelEvidence",
            "TestClusterRejectsUnattributableFrames",
            "TestClusterRetentionRefusesGoServedCalls",
            "TestClusterShutdownControlOrder",
            "TestClusterShutdownLiveNodesEvidenceOrder",
            "TestClusterShutdownWaitsForObservedFailureDrain",
            "TestClusterStartAndObserveLeaveControllerPaused",
            "TestClusterTransportCodecCorpus",
            "TestClusterTransportCodecCorpusIsCurrent",
            "TestClusterTransportCodecLiteralStageGuards",
            "TestClusterTransportStageCorpus",
            "TestClusterTransportStageCorpusIsCurrent",
            "TestClusterTransportStageLiteralGuards",
            "TestClusterWaitSerializesTurnsWithoutBlockingLiveControls",
            "TestImplementationIdentityBindsCompiledModelDigest",
            "TestImplementationIdentityBindsStorageEvidenceSources",
            "TestImplementationIdentityBindsStrictJSONOwner",
            "TestImplementationIdentityFailsClosedOnMissingSource",
            "TestStorageCrashContextAdmission",
            "TestStorageCrashContextEqualIsExact",
            "TestStorageCrashContextEvidenceRefusal",
            "TestStorageCrashContextLargeChunkTransport",
            "TestStorageCrashContextLargeObservedSelection",
            "TestStorageCrashContextSelectedOwnersAndDivergence",
            "TestStorageCrashValidationOrderCorpus",
            "TestStorageCrashValidationVolumeOrder",
        ],
        1,
    ),
    (
        "busy-boundaries",
        &[
            "TestClusterBrokerDurableRestartParity",
            "TestClusterOutputBoundaryParity",
        ],
        1,
    ),
    (
        "busy-grouped",
        &["TestProviderGroupedAcknowledgmentCrashSourceLessReplay"],
        10,
    ),
    (
        "network-streams",
        &["TestBrokerIndependentGuestStreamsReplay"],
        1,
    ),
    (
        "retained-cluster",
        &[
            "TestBoundRetainedReplayRejectsBeforeWritingInputs",
            "TestInspectionControlRefusesUnregisteredAndIncompleteBeforeReplay",
            "TestProviderCompressedIndependentGuestsLifecycleReplay",
            "TestProviderDetachedOracleCanarySourceLessReplay",
            "TestProviderGroupedAcknowledgmentCrashSourceLessReplay",
            "TestProviderIndependentGuestsLifecycleReplay",
            "TestRetainedAdmissionConsumesPrivateInputsOnlyOnce",
            "TestRetainedCanonicalDecodeAvoidsWholeBundleEncodingAllocation",
            "TestRetainedCanonicalWriterChecksEveryWriteAndBase64Close",
            "TestRetainedCanonicalWriterMatchesAllDeclaredFields",
            "TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput",
            "TestRetainedDecoderAcceptsSharedCampaignPayloadPath",
            "TestRetainedLifecycleRejectsHostCallbacksBeforeReplay",
            "TestRetainedOrdinarySpellingKeepsRawAuthenticationAndPresence",
            "TestRetainedPrimaryBindingRejectsMissingAndForeignModules",
            "TestRetainedReplayRejectsTamperedOrMismatchedTargetModule",
            "TestRetainedReplayRequiresTheInstalledHelper",
            "TestRetainedSchemaV1IsRefusedWithVersionedError",
            "TestRetainedStorageSelectionProjectionIsDetached",
            "TestSoftAssertionActualClusterGuestAndRetainedReplay",
            "TestSoftAssertionRetainedEvidenceRequiresActualOutput",
            "TestSoftAssertionRetainedLiteralBracesRederiveCompleteMessage",
            "TestSoftAssertionRetainedTruncatedOutputIsInconclusive",
            "TestStressRetainedReopenPreservesBoundInputsAndRestoresHostOptIn",
        ],
        1,
    ),
    (
        "terminal-module",
        &["TestPrepareTemporalTerminalExitModule"],
        1,
    ),
    (
        "terminal-host-admission",
        &["TestDiagnosticPerformanceCurrentHostAdmission"],
        1,
    ),
    ("terminal-exit", &["terminal-exit"], 1),
];

const SEMANTIC_AFTER_RUNTIME: &[SemanticDeclaration] = &[
    (
        "runtime-aggregate",
        &["TestCooperativeRuntimeQualificationAggregate"],
        1,
    ),
    (
        "artifact",
        &[
            "TestIntegratedFailureArtifactFreshObservedReplays",
            "TestInstalledOperatorTypedScenarioDetachedReplay",
            "TestInstalledOperatorKillResume",
            "TestInstalledOperatorTypedScenarioKillResume",
        ],
        1,
    ),
    (
        "operator",
        &[
            "TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs",
            "TestInstalledOperatorClusterFailureEndsPromptly",
            "TestInstalledOperatorAsymmetricGuestFailureEndsPromptly",
        ],
        1,
    ),
    (
        "storage",
        &["TestTemporalCommittedWorkflowFreshGuestCrashReopen"],
        1,
    ),
    (
        "temporal-inspection",
        &[
            "TestTemporalInspectionRetainedFailure",
            "TestTemporalInspectionRetainedFailure/shard",
            "TestTemporalInspectionRetainedFailure/workflow",
            "TestTemporalInspectionRetainedFailure/onebox",
        ],
        1,
    ),
    (
        "network",
        &["TestTemporalNetworkNamedResetRetriesAndExactReplay"],
        1,
    ),
    (
        "clusters",
        &["TestTemporalIndependentClustersPartitionHealWholeClusterCrash"],
        1,
    ),
    (
        "workload-fixture-controlled-success",
        &["TestActualMatchedFixtureQualification"],
        1,
    ),
    (
        "workload-fixture-controlled-canary",
        &["TestActualMatchedFixtureQualification"],
        1,
    ),
    (
        "workload-fixture-cooperative-success",
        &["TestActualMatchedFixtureQualification"],
        1,
    ),
    (
        "workload-fixture-cooperative-canary",
        &["TestActualMatchedFixtureQualification"],
        1,
    ),
    (
        "workload-sqlite-cooperative-success",
        &["TestActualSQLiteDurableQualification"],
        1,
    ),
    (
        "workload-sqlite-cooperative-canary",
        &["TestActualSQLiteDurableQualification"],
        1,
    ),
    (
        "workload-onebox-cooperative-success",
        &["TestActualOneboxQualification"],
        1,
    ),
    (
        "workload-onebox-cooperative-canary",
        &["TestActualOneboxQualification"],
        1,
    ),
    (
        "workload-multi-instance-cooperative-success",
        &["TestActualTemporalClusterQualification"],
        1,
    ),
    (
        "workload-multi-instance-cooperative-canary",
        &["TestActualTemporalClusterQualification"],
        1,
    ),
    ("workload-corpus", &["TestActualMandatoryWorkloadCorpus"], 1),
    (
        "sqlite-selection-0",
        &["TestActualSQLiteSelectedCrashQualification"],
        1,
    ),
    (
        "sqlite-selection-1",
        &["TestActualSQLiteSelectedCrashQualification"],
        1,
    ),
    (
        "sqlite-selection-2",
        &["TestActualSQLiteSelectedCrashQualification"],
        1,
    ),
    (
        "sqlite-selection-3",
        &["TestActualSQLiteSelectedCrashQualification"],
        1,
    ),
    (
        "workload-corpus-after-relocation",
        &["TestActualMandatoryWorkloadCorpus"],
        1,
    ),
    (
        "sqlite-catalogue",
        &["TestActualSQLiteSelectionCatalogue"],
        1,
    ),
];
