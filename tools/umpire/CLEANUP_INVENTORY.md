# Umpire tooling cleanup inventory

This ledger freezes the post-fn-64/fn-62 Umpire tooling tree before fn-66 deletes anything and
records the later fn-69 Testpilot ownership closure. It
extends the accounting in the immutable fn-64 migration ledger; it does not revise that historical
record. Every package and command found by `go list -tags test_dep ./tools/umpire/...` is classified
below. A removal decision requires repository-wide consumer evidence, rather than absence of a Go
import alone.

## Frozen baseline and prerequisites

- Task-start commit: `ff9ea9827157255068a87086630651a43cc01060`.
- Task-start cumulative index tree: `cfd213f2f73071400632cd4b8b5dfae300943e5a`.
- Conductor spec-start commit/tree: `.flow/tmp/fn66-start-head` and `.flow/tmp/fn66-start-tree`.
- fn-62 and fn-64 both report `done` with completion-review status `ship`. The fresh fn-66 plan
  review at `/tmp/plan-review-receipt-fn-66-remove-unused-umpire-tooling-after.json` records
  `codex:gpt-5.6-sol:medium` and `SHIP` for this three-task graph.
- The task baseline command was
  `go test -count=1 -tags test_dep ./tools/umpire/artifact ./tools/umpire/cmd/umpire-artifact ./tools/umpire/internal/artifactv2`.
  The first run failed because the default macOS temporary path traversed the `/var` symlink. The
  same command with `TMPDIR` set to a physical `/private/tmp/fn66-task1.*` directory exited 0:
  all three packages passed. No source changed between those observations.
- The fn-64 ledger remains byte-identical to `HEAD` (`git hash-object`:
  `c467855d75b6ab9cced284c4bc39e24782584f6e`).

## Current package and command ownership

The frozen inventory contains 21 Go packages, including seven `main` packages. After fn-66 and the
fn-69 extraction, `go list -tags test_dep ./tools/umpire/...` reports 10 retained authoring,
Producer, regression, and vocabulary packages, including six `main` packages. The Case protocol and
runtime now contribute four packages under `common/testing/testpilot`, and the functional Driver
contributes four under `tests/testcore/testpilot`. There are no unclassified rows.

| Package | Decision | Concrete consumer or removal evidence |
| --- | --- | --- |
| `tools/umpire` | removed by fn-69 | The public runtime facade moved to `common/testing/testpilot`; Umpire retains only authoring, Producer, regression, and vocabulary packages. |
| `tools/umpire/artifact` | remove in .2 | Its only Go importers are `cmd/umpire-artifact` and its own tests. Repository search outside those roots finds only the Make wrappers and one active compatibility-document row. No script, workflow, generator, manifest, or retained runtime imports it. |
| `tools/umpire/caseartifact` | removed by fn-69 | Canonical decoding and packing are owned by the public `common/testing/testpilot` boundary. |
| `tools/umpire/cmd/umpire-artifact` | remove in .2 | Only `UMPIRE_ARTIFACT_COMMAND`, `umpire-check-artifact`, `umpire-check-artifact-set`, and the matching `.PHONY` entries call it. No workflow or script calls the command directly. It has no replacement CLI. |
| `tools/umpire/cmd/umpire-check-retired-vocabulary` | retained | `make umpire-check-retired-vocabulary`, the aggregate regression target, and `vocabulary/retired_vocabulary_test.go` exercise the real command. |
| `tools/umpire/cmd/umpire-export-proto-descriptors` | retained | `UMPIRE_EXPORT_PROTO_DESCRIPTORS_COMMAND` builds `proto/umpire-public.binpb`; its test builds and invokes the real binary and checks nonzero status and stderr. |
| `tools/umpire/cmd/umpire-gen-case-runtime-conformance` | retained | Make generation/check targets build `temporal-testpilot`, invoke this command, compare the entire managed fixture tree, and run facade conformance. |
| `tools/umpire/cmd/umpire-gen-lean-api` | retained | `make umpire-gen-lean-api`, `model/README.md`, and the checked-in `model/Temporal/API*.lean` outputs depend on it. Its fixture target rewrites only the owned basic fixture. |
| `tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog` | retained | `make umpire-gen-lean-dynamic-config-catalog`, `model/README.md`, and `model/Temporal/DynamicConfig*.lean` depend on it; project tests cover helper-process stdout/stderr failures. |
| `tools/umpire/cmd/umpire-gen-regression-views` | retained | Make generation/check targets, the single-entry production manifest, Switch Go/Markdown outputs, and generator tests consume it. |
| `tools/umpire/internal/artifactv2` | retained after trim in .3 | Its independent retained consumers are `cmd/umpire-gen-regression-views/generated_view.go` and `regression/generated_view.go`. The complete Experiment reader closure below remains; the four orphan source files and clone test authorized below are removed. |
| `tools/umpire/internal/execution` | removed by fn-69 | Its complete owner moved to `common/testing/testpilot/internal/execution`. |
| `tools/umpire/internal/ir` | removed by fn-69 | Its complete owner moved to `common/testing/testpilot/internal/ir`. |
| `tools/umpire/internal/retiredvocabulary` | retained | Implementation of the retained vocabulary command and aggregate regression gate. |
| `tools/umpire/regression` | retained | Owns checked-in generated-view verification and `ci_workflow_test.go`, including Make wiring and fn-64 ledger/generic-promotion assertions. |
| `tools/umpire/temporal` | removed by fn-69 | The functional Driver moved to `tests/testcore/testpilot`, retaining separate server and worker authority. |
| `tools/umpire/temporal/internal/delivery` | removed by fn-69 | Delivery moved with the functional Driver to `tests/testcore/testpilot/internal/delivery`. |
| `tools/umpire/temporal/server` | removed by fn-69 | Server transport moved to `tests/testcore/testpilot/server`. |
| `tools/umpire/temporal/worker` | removed by fn-69 | SDK execution moved to `tests/testcore/testpilot/worker`. |
| `tools/umpire/verification` | removed by fn-69 | Contract preparation and evaluation moved to `common/testing/testpilot/internal/verification`. |
| `tools/umpire/vocabulary` | retained test package | Its external test runs the real vocabulary command and protects the active terminology gate. |

`tools/planindex` and `make umpire-check-plan-index` are adjacent tooling, outside the
`tools/umpire/...` inventory. fn-66 retained them deliberately, and fn-81 revalidated that
retention: the validator is what keeps `.plans/index.json` honest, so it is the gate over this
repository's own documentation reconciliation rather than a candidate for removal.

## Authorized deletion paths

Task .2 removed the complete 49-file public artifact tree plus the two-file CLI tree. The 49 files are
10 implementation files, 12 test files, and 27 fixtures. It also removes only the three direct Make
references (`UMPIRE_ARTIFACT_COMMAND`, both wrapper recipes, and their two `.PHONY` names) and updates
the three active ownership documents named by .2. No compatibility wrapper or replacement command is
authorized.

Public implementation files:

- `tools/umpire/artifact/artifact.go`
- `tools/umpire/artifact/errors.go`
- `tools/umpire/artifact/evidence.go`
- `tools/umpire/artifact/experiment.go`
- `tools/umpire/artifact/json.go`
- `tools/umpire/artifact/limits.go`
- `tools/umpire/artifact/publish.go`
- `tools/umpire/artifact/result.go`
- `tools/umpire/artifact/runtime.go`
- `tools/umpire/artifact/set.go`
- `tools/umpire/cmd/umpire-artifact/main.go`

Task .3 removed these paths after a fresh post-.2 reference check confirmed the same closure:

- `tools/umpire/internal/artifactv2/runtime.go`
- `tools/umpire/internal/artifactv2/evidence.go`
- `tools/umpire/internal/artifactv2/result.go`
- `tools/umpire/internal/artifactv2/clone.go`
- `tools/umpire/internal/artifactv2/clone_test.go`

It must retain `tools/umpire/internal/artifactv2/artifact.go`,
`tools/umpire/internal/artifactv2/natural.go`, and `artifact_test.go`. Any additional unused item
requires a reviewed task adjustment before deletion.

## Candidate Test and Fuzz accounting

Every entry below is a top-level Go test function. The owning file is the removal owner. All 92
public artifact/CLI Tests and the one Fuzz target listed individually below were removed with .2;
clone tests remain for .3. “Replaced” means a named
retained test covers the surviving contract. “Intentionally retired” means the asserted public
runtime/evidence/result/set/CLI contract itself is obsolete and has no replacement surface.

| Deleted owner | Test/Fuzz entry points | Disposition and surviving owner |
| --- | --- | --- |
| `tools/umpire/artifact/errors_test.go` | `TestStrictJSONErrorCodesAreStableThroughWrapping` | intentionally retired with public artifact error API |
| `tools/umpire/artifact/evidence_test.go` | `TestRawEvidenceV2CanonicalFixtureRoundTrip`<br>`TestRawEvidenceV2ChecksumsUseExactPrettyPreimages`<br>`TestRawEvidenceV2AcceptsClosedStatusAndValueGrammar`<br>`TestRawEvidenceV2RejectsClosedGrammarMutations`<br>`TestRawEvidenceV2RejectsSemanticAndCanonicalMutations`<br>`TestRawEvidenceV2ClosesBindingsSourcesAndControlReceipts`<br>`TestRawEvidenceV2EvidenceCeilings`<br>`TestRawEvidenceV2WrongContainersAreMalformedValues` | intentionally retired; Case Run Events and Contract evaluation replaced legacy evidence artifacts |
| `tools/umpire/artifact/experiment_test.go` | `TestExperimentV2CanonicalFixturesRoundTrip`<br>`TestExperimentV2ChecksumsUseExactPrettyPreimages`<br>`TestExperimentV2RejectsOneAtATimeMutations`<br>`TestExperimentV2StringBounds`<br>`TestExperimentV2RejectsMalformedDefinitionIDSets`<br>`TestExperimentV2EncodeRejectsInvalidValues` | replaced for the retained planning Experiment by `internal/artifactv2/artifact_test.go` plus generated-view tests |
| `tools/umpire/artifact/golden_test.go` | `TestCrossLanguageGoldensExactCanonicalFixtures`<br>`TestCrossLanguageGoldensRejectAlternateWhitespace`<br>`TestCrossLanguageGoldensExactFieldSequencesAndChecksums`<br>`TestCrossLanguageGoldensNestedProjectionsAndReceiptLink`<br>`TestCrossLanguageGoldensRejectIdentityAndClosureMutations` | Experiment coverage replaced by the retained reader/generator tests; runtime/evidence/result portions intentionally retired |
| `tools/umpire/artifact/json_test.go` | `TestStrictJSONCanonicalPretty`<br>`TestStrictJSONAcceptsOnlyCanonicalPrettyBytes`<br>`TestStrictJSONRejectsNoncanonicalNumberSpellings`<br>`TestStrictJSONRejectsEveryStructuralClass`<br>`TestStrictJSONUsesCanonicalAndValidationHooks`<br>`TestStrictJSONCountsPunctuationScalarsAndRootDepth`<br>`TestStrictJSONScannerUsesBoundedBookkeeping`<br>`TestStrictJSONStopsObjectBookkeepingAtNPlusOne`<br>`TestStrictJSONPreservesHigherPrecedencePastCollectionLimit`<br>`TestStrictJSONCountsDecodedStringBytesWithoutMaterializingStrings`<br>`TestStrictJSONAppliesStableErrorPrecedence`<br>`TestStrictJSONBoundOverridesCanOnlyTighten`<br>`FuzzStrictJSONNoPanicOrPermissiveSuccess` | intentionally retired with the public generic strict-JSON engine; retained Experiment rejection coverage is in `internal/artifactv2/artifact_test.go` and generated-view tests |
| `tools/umpire/artifact/limits_test.go` | `TestStrictJSONEveryDeclaredCeilingHasItsExactValue`<br>`TestStrictJSONStructuralCeilingsUseEncodedBoundaries`<br>`TestStrictJSONPerFamilyCeilingsUseEncodedNNPlusOneBoundaries` | intentionally retired with obsolete public artifact-family ceilings |
| `tools/umpire/artifact/publish_test.go` | `TestPublishSetLoadsOneCompleteImmutableDirectory`<br>`TestAdmitSetFilesOwnsExactInputSnapshot`<br>`TestLoadSetRejectsUnsafeOrConflictingDestinations`<br>`TestPublishSetDoesNotRepairConflictingDestination`<br>`TestLoadSetRequiresExactDigestDirectory`<br>`TestPublishSetReadersObserveAbsenceOrOneCompleteSet` | intentionally retired; no retained artifact-set publication workflow |
| `tools/umpire/artifact/result_test.go` | `TestResultV2AcceptedEvidenceAndResolvedResultRoundTrip`<br>`TestResultV2AdmitsKindMajorMultistepCoordinateOrder`<br>`TestResultV2CanonicalLeanFixtureParity`<br>`TestResultV2EvidenceClosedStatusAndNullabilityMatrix`<br>`TestResultV2ExhaustiveClosedDiagnosticClassifications`<br>`TestResultV2SpecializedDiagnosticStringBounds`<br>`TestResultV2EvidenceRejectsIncompleteLinksStaleReferencesAndRawLeakage`<br>`TestResultV2EvidenceAdmitsOpaqueDigestTokenWithoutRawValue`<br>`TestResultV2AdmitsOpaqueObservationTraceIdentity`<br>`TestResultV2EvidenceRequiresRejectedDispositionOnlyForAcceptedEvidence`<br>`TestResultV2ImplementationPropertySemanticAndChecksumMatrices`<br>`TestResultV2RejectsImplementationLinkDiagnosticIdentityDrift`<br>`TestResultV2ImplementationLinkDiagnosticIdentityUsesExactPrettyPreimage`<br>`TestResultV2RejectsCanonicalChecksumAndClosureMutations`<br>`TestResultV2RejectsStaleQuerySummaryPartition`<br>`TestResultV2EvaluationChecksumUsesExactPrettyPreimage`<br>`TestResultV2EvaluationChecksumChangesForPlanOnlyMutation`<br>`TestResultV2AdmitsOperationalFailureIndependentlyFromResolvedSemantics`<br>`TestResultV2RejectsEveryNestedProjectionFieldOrderMutation` | intentionally retired; Case `Run` and `Verdict` are the current result contract |
| `tools/umpire/artifact/runtime_test.go` | `TestRuntimeConfigurationV2CanonicalFixtureRoundTrip`<br>`TestRuntimeV2ExperimentRunCanonicalFixtureRoundTrip`<br>`TestRuntimeV2ArtifactBindingsCloseAgainstExperiment`<br>`TestRuntimeV2ChecksumsUseExactPrettyPreimages`<br>`TestRuntimeV2ExperimentRunClosedStatusMatrices`<br>`TestRuntimeV2ExperimentRunRejectsOperationalStatusAndPhaseProgression`<br>`TestRuntimeV2RejectsCrossBoundaryInconsistency`<br>`TestRuntimeV2StringBounds`<br>`TestRuntimeConfigurationV2RejectsOneAtATimeMutations`<br>`TestRuntimeV2ExperimentRunRejectsOneAtATimeMutations` | intentionally retired with the legacy runtime configuration and ExperimentRun artifact contracts |
| `tools/umpire/artifact/set_execution_test.go` | `TestExecutableSetAdmitExecutionReusesExactInputBytes`<br>`TestExecutionSetAdmitEvaluationReusesExactInputBytes`<br>`TestExecutionSetAdmitEvaluationRequiresPlanTargetAtOneLinkEndpoint` | intentionally retired; admitted Case preparation replaces executable/evaluation artifact sets |
| `tools/umpire/artifact/set_test.go` | `TestArtifactSetEvaluationClosureCanonicalManifest`<br>`TestArtifactSetManifestAdmissionRequiresExactCanonicalBytes`<br>`TestArtifactSetAdmitsOnlyThreeExactClosures`<br>`TestArtifactSetExecutableProjectionIsExactAndImmutable`<br>`TestArtifactSetRejectsMemberPathAndOrderMutations`<br>`TestArtifactSetRejectsNoncanonicalAndStaleMembersAtomically`<br>`TestUnsupportedFormatMixedArtifactSetsPrecedeChecksumsAndClosure`<br>`TestUnsupportedFormatMixedArtifactSetsPreserveStructuralPrecedence`<br>`TestArtifactSetAdmittedValueOwnsManifestBytes` | intentionally retired; no retained artifact-set workflow |
| `tools/umpire/artifact/version_test.go` | `TestUnsupportedFormatArtifactFamiliesPrecedeFieldValidation`<br>`TestUnsupportedFormatArtifactSetManifestPrecedesMemberValidation` | Experiment-version rejection replaced by retained reader tests; other families intentionally retired |
| `tools/umpire/cmd/umpire-artifact/main_test.go` | `TestCheckAcceptsEveryRetainedArtifactFamilyWithoutMutatingInput`<br>`TestCheckRejectsNoncanonicalBytesWithoutMutatingInput`<br>`TestCheckSetAcceptsCompleteSetWithoutMutationOrPublication`<br>`TestCheckSetRejectsCompactMemberWithoutMutatingInput`<br>`TestCheckSetClassifiesUnsupportedMemberBeforeStaleManifestChecksum`<br>`TestCheckSetRejectsUnexpectedOversizedFileBeforeReading`<br>`TestCheckSetRejectsUnexpectedFilesAndSymlinks`<br>`TestCommandUsageErrorsUseExitTwoAndStderrOnly` | intentionally retired with the CLI; no compatibility command |
| `tools/umpire/internal/artifactv2/clone_test.go` | `TestCopyArtifactDocumentsPreservesZeroAndEmptyCollections`<br>`TestCopyExperimentIsolatesEveryMutableDescendant`<br>`TestCopyRuntimeConfigurationIsolatesEveryMutableDescendant`<br>`TestCopyExperimentRunIsolatesEveryMutableDescendant`<br>`TestCopyRawEvidenceIsolatesEveryMutableDescendantAndPreservesScalarValues` | intentionally retired after .2 leaves the clone helpers unreferenced; retained readers return fresh decoded values and do not expose these copy APIs |

Counts from source declarations reconcile to
**92 Test**, **1 Fuzz**, and the later **5 Test** entries: **97 Test + 1 Fuzz** total.

## Fixture accounting

All 27 paths listed individually below were removed with the removal-owned public artifact package
and CLI tests. The
six top-level fixtures are read by both test packages, directly and through the CLI's
`writeEvaluationSet`; the remaining 21 are artifact-package-test-only. Repository-wide search found
no surviving test, documentation, manifest, generator, or command reference after both removal
roots are excluded. “Mirrored” fixtures were checked-in Go-side copies of Lean-owned canonical
fixtures; “handwritten mutation” fixtures were introduced together to test unsupported majors.

| Fixture path | Origin and current owner | Disposition |
| --- | --- | --- |
| `tools/umpire/artifact/testdata/switch-experiment-v2.json` | mirrored planning Experiment; artifact and CLI tests | replaced by retained `model/Umpire/Examples/testdata/switch-experiment-spec.json` and reader/generator checks |
| `tools/umpire/artifact/testdata/runtime-configuration-v2.json` | mirrored legacy runtime artifact; artifact and CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/experiment-run-v2.json` | mirrored legacy run artifact; artifact and CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/raw-evidence-v2.json` | mirrored legacy evidence input; artifact and CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/evidence-v2.json` | mirrored legacy evidence artifact; artifact and CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/result-v2.json` | mirrored legacy result artifact; artifact and CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/artifact-set-v1.json` | handwritten unsupported-major mutation; version/set tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/artifact-set-v3.json` | handwritten unsupported-major mutation; version/set tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/evidence-v1.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/evidence-v3.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/experiment-run-v1.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/experiment-run-v3.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/experiment-v1.jsonfixture` | handwritten unsupported-major mutation; version test uses the deliberate suffix to avoid active JSON scans | replaced by retained Experiment reader rejection tests |
| `tools/umpire/artifact/testdata/unsupported/experiment-v3.json` | handwritten unsupported-major mutation; version tests | replaced by retained Experiment reader rejection tests |
| `tools/umpire/artifact/testdata/unsupported/raw-evidence-v1.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/raw-evidence-v3.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/result-v1.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/result-v3.json` | handwritten unsupported-major mutation; version/set tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/runtime-configuration-v1.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/unsupported/runtime-configuration-v3.json` | handwritten unsupported-major mutation; version tests | intentionally retired |
| `tools/umpire/artifact/testdata/valid-run-evaluation-set/artifacts/experiment.json` | copied planning member of handwritten set tree; set/CLI tests | planning input replaced by retained model fixture; set copy retired |
| `tools/umpire/artifact/testdata/valid-run-evaluation-set/artifacts/runtime-configuration.json` | copied runtime member of handwritten set tree; set/CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/valid-run-evaluation-set/artifacts/experiment-run.json` | copied run member of handwritten set tree; set/CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/valid-run-evaluation-set/artifacts/raw-evidence.json` | copied raw-evidence member of handwritten set tree; set/CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/valid-run-evaluation-set/artifacts/evidence.json` | copied evidence member of handwritten set tree; set/CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/valid-run-evaluation-set/artifacts/result.json` | copied result member of handwritten set tree; set/CLI tests | intentionally retired |
| `tools/umpire/artifact/testdata/valid-run-evaluation-set/manifest.json` | handwritten canonical set manifest; set/CLI tests | intentionally retired |

## Retained Experiment reader closure

Both generated-view consumers require the same strict Experiment reader. The closure is the whole
declaration set in `internal/artifactv2/artifact.go` and `natural.go`, not merely the exported names
observed at the import boundary.

- Direct consumer symbols: `ExperimentFormat`, `DrivePlanFormat`, `Experiment`, `DrivePlan`,
  `Property`, `Provenance`, `SourceLocation`, `Natural`, `DecodeExperiment`,
  `CanonicalExperimentBytes`, `SealExperiment`, and `ValidDigest`.
- Experiment data types reached by `DrivePlan`: `ModelValue`, `Binding`, `Role`, `Precondition`,
  `Operand`, `Occurrence`, `Limits`, `BehaviorLimits`, `Limit`, `Checkpoint`, `ExploredCounts`, and
  `KnownGap`.
- Decode/canonical validation closure: `canonicalKeys`, `preflightFormat`, `validateJSONStructure`,
  `validateJSONValue`, `requireEOF`, `ValidateExperiment`, `validateExperimentCollections`,
  `validateDrivePlan`, `validateKnownGaps`, `validateBindings`, `validateRoles`,
  `validatePreconditions`, `validateOperand`, `validateOccurrencesAndCheckpoints`, `validateLimits`,
  `validLimitUnit`, `validateModelValues`, `validateModelValue`, `validateProvenance`,
  `compareBinding`, `validDefinitionID`, `isASCIIAlphanumeric`, `validateDefinitionIDSet`,
  `validateStringSet`, `compareKnownGap`, `knownGapKindRank`, `compareInt`,
  `compareSourceLocation`, and `pointerValue`.
- Seal/checksum closure: `behaviorFingerprintDomain`, `drivePlanChecksumDomain`,
  `experimentChecksumDomain`, `ExpectedDrivePlanChecksum`, `ExpectedExperimentChecksum`,
  `BehaviorFingerprint`, `VerifyExperimentChecksums`, `ValidateExperimentClosure`,
  `encodeJSONLine`, `encodeJSONLineWithIndent`, and `derive`.
- Natural-number closure: `NaturalFromUint64`, `Natural.String`, `Natural.IsZero`,
  `Natural.MarshalJSON`, `Natural.UnmarshalJSON`, `validateNaturalBytes`, and `compareNatural`.

Task .3 must compile and run the generator/regression reader tests after removing the other
artifactv2 files. The two retained lint headers in `artifact.go` are also preserved.

## Retained generated identities

These SHA-256 values freeze output bytes at task-start. They are observations, not an instruction to
regenerate in this documentation-only task.

| Managed identity | Source/output SHA-256 |
| --- | --- |
| Switch Experiment `switch.query.exact-action`, artifact checksum `sha256:fa701806df655fa9cebc9b7d94f36b74176890c96bb535c7a3f6629afe64ff41` | source `model/Umpire/Examples/testdata/switch-experiment-spec.json`: `806e3f1b35e1665717b1ef6a05226d9fcf38cdee3d87ffae84334df24c49e7c2` |
| Switch generated Go view | `8a23cdc22e53a2a9d2860522d3f1353898336f5758830ffb32d42948125300a5` |
| Switch generated Markdown view | `33608f42fccedfe34309a429506778f0272eeec8acd2a136b8310c8936e46747` |
| Testpilot conformance tree, six named classes / 12 files | SHA-256 of its sorted `sha256sum` manifest: `7809b6829822c097dccb76a07e4abdba233f3a552cb553c5ab019ca1888dddb9` |
| Lean API output set (`API.lean`, `API/Proto.lean`, `API/Types.lean`) | SHA-256 of sorted manifest: `613a888529b061c095c9da5ae9301154297ecd1a6459108d8c5f5b0c06551d4e` |
| Dynamic-config output set (`DynamicConfig.lean`, `Settings.lean`, `Types.lean`) | SHA-256 of sorted manifest: `9f114f949e39464e61fe5993e5fe7599a9e96b78f0701c37a4cc02f05b11e970` |
| Semantic inventory `model/SEMANTIC_INVENTORY.md` | `e534439582339a330d32562f5c796a2a9736a7db200786653902d0c97c204d25` |

The conformance classes are `satisfied`, `violated`, `inconclusive`,
`static-preparation-rejection`, `cleanup-failure-after-proved-violation`, and
`cross-run-isolation`; each owns `case.json` and `expected.json`.

## Existing process and byte checks

- `make umpire-check-regression-views` builds the real Lean inspector, invokes the real generator,
  diffs both managed Switch outputs, asserts the retired aggregate output is absent, and runs the
  generator and regression packages.
- `make umpire-check-case-runtime-conformance` builds the real Lean Case renderer, invokes the real
  Go generator, recursively diffs all 12 managed files, then runs generator and public-facade tests.
- `make umpire-check-semantic-inventory` invokes the real Lean renderer and diffs its output.
- `cmd/umpire-export-proto-descriptors/main_test.go` builds and invokes the actual binary and checks
  exit status and stderr; the Make descriptor rule feeds its output to the retained Lean API
  generator.
- `cmd/umpire-gen-regression-views/generate_test.go` preserves inspector stdout/stderr/exit
  contradictions, and `regression/ci_workflow_test.go` executes `make -n
  umpire-check-regression` to protect aggregate wiring.
- `vocabulary/retired_vocabulary_test.go` invokes the actual vocabulary command. The aggregate regression
  target preserves the complete tagged package selector and exact inherited live-test identities.
- The retiring CLI's process-like `run` checks are the eight named `main_test.go` entries above;
  those exit/stdout/stderr contracts intentionally retire with the command and are not replaced by
  mocks.

## Exact lint accounting

The frozen sorted diagnostic-header file `/tmp/fn62-task7-quality-diagnostic-headers.txt` has 1,316
rows and SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`.
The subtraction was re-derived from exact `../tools/...` path prefixes:

| Stage | Approved removed headers | Required result |
| --- | --- | --- |
| after .2 | 5 rows from `tools/umpire/artifact/result_test.go` | 1,311 rows; `5afaccdacfc74c7940a6f6d059065481113b32406b8bc0ecb5004d5be93c325a` |
| after .3 | the prior 5, plus 1 from `internal/artifactv2/evidence.go` and 38 from `internal/artifactv2/result.go` | 1,272 rows; `06c1dfdf2e88baf387145e49dd835c044a0e7f33bf75d75aea566f17bfc6cce3` |

The two headers in retained `internal/artifactv2/artifact.go` remain. Any additional new, changed,
or missing header fails the final gate even if the total is smaller.

## Final task evidence

- The post-.2 symbol and import scan found zero live consumers of declarations in `runtime.go`,
  `evidence.go`, `result.go`, or `clone.go`, and zero live consumers of the clone test. The two live
  consumers of the retained package use only the complete Experiment reader closure recorded above.
- The focused baseline and post-trim command both exited 0:
  `go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2
  ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression`.
- The direct complete Go selector exited 1 before reaching all packages because the host C
  toolchain could not find `stddef.h`. The same complete tagged selector executed by the physical
  `TMPDIR` aggregate regression target passed all 19 retained packages, including the retained
  artifact reader and both consumers.
- `make umpire-build-model` exited 0. Physical-canonical-`TMPDIR`
  `make umpire-check-regression` exited 0; its generated-view, Case conformance, semantic inventory,
  vocabulary, complete tagged package, and live-test checks all ran, and the exact inherited live
  failure-identity set matched.
- The first `make lint-model` observation exited 2 after concurrent external draft edits introduced
  noncanonical model sources. After their owner preserved those drafts as noncompiled design
  artifacts, the final-tree rerun exited 0 without weakening the owned-source selector.
- The final-tree `make lint-code GOLANGCI_LINT_FIX=false` exited 2 with 1,272 diagnostics and raw
  sorted-header SHA-256 `f53910724a8830c9fc2a58e67a3d3569e5710b8bd4242ecfc54307d9b09e439f`.
  Exactly 1,270 raw headers match the approved final file byte-for-byte. The other two are the same
  `fmt.Errorf` calls and replacements at `catalog.go:33` and `catalog.go:39`, where Revive selected
  its overlapping `use-errors-new` identity instead of the frozen `unnecessary-format` identity.
  Canonicalizing only those two proven label/message aliases produces 1,272 byte-identical expected
  and actual headers with SHA-256
  `6af8054b95719a1cfbaeab137e49dfdc0b654272867910850f5f7c7f65f6b136`. A diagnostic experiment that
  globally disabled `use-errors-new` changed unrelated identities and was discarded; the final
  source and lint configuration preserve the frozen rules.
- The retained Switch source, generated Go view, generated Markdown view, Case conformance tree,
  Lean API set, dynamic-config set, and semantic inventory remained byte-identical through the
  aggregate checks. Retained source and comments in `artifact.go`, `natural.go`, and
  `artifact_test.go` were not edited.
- Completion review found and removed two remaining prospective claims in
  `.plans/UMPIRE4_SPEC_COMPS.md`: exact Artifact/set checking as an intended command and the public
  `tools/umpire/artifact/` entry in the recommended Go tree. This documentation-only correction
  aligns both sections with the already recorded retired surface and does not change executable
  code or prior gate evidence.
- Removing unreachable codecs introduces no allocation, concurrency, crash, security, or runtime
  work. A 10x increase in retained reader, generator, Testpilot, or live-test load therefore
  follows the unchanged implementations and cost bounds exercised by the retained gates.

## Reconciliation

- Realized removal: .2 removed 51 files, 92 Tests, 1 Fuzz target, and 27 fixtures; .3 owns 5 files
  and 5 Tests. The total is 56 files, 97 Tests, 1 Fuzz target, and 27 fixtures.
- Every current package and command has a concrete retained consumer or an evidence-backed removal
  decision. Every fixture origin/reference and every candidate top-level test is accounted for.
- The post-.2 symbol and import closure found no retained consumer of the four internal codecs or
  clone helpers. Repository searches found no additional unused package, command, helper, fixture,
  or active reference in scope. There was no scope conflict requiring revision before .3.

## fn-81 pre-Testpilot generation sweep

This section extends the ledger for
[fn-81](../../.flow/specs/fn-81-delete-the-pre-testpilot-go-generations.md), which deletes the
pre-Testpilot Go generations. It follows the same evidence rule as the fn-66 accounting above: a
removal decision requires repository-wide consumer evidence, and the absence of a Go import alone is
never sufficient. Every deletion-set root and every retained neighbour named by the spec appears
below with the consumers found per search surface and a disposition. It does not revise the fn-66 or
fn-64 records.

### Frozen baseline and prerequisites

- Spec-run base commit: `023cb7d861b6cc0e139564b2faaf10c106a7f37d`.
- Task-start commit: `5b9cc0dc9df7b03d231dc685c1fa99bffdf9c3b3`.
- Prior ledger contents at task start (`git hash-object tools/umpire/CLEANUP_INVENTORY.md`):
  `3090af45f85dd33cdf4dc38386c0e4a5ac832e84`.
- Dependency-closure baseline: `go list -deps -test -tags 'test_dep integration' ./... | sort -u >
  .flow/tmp/fn81/deps-before.txt` — 2,889 packages. The `-test` form is the authoritative closure
  for the tidy comparison, because `go mod tidy` also resolves test-only imports of main-module
  packages.
- Package-set baseline: `go list ./... | sort > .flow/tmp/fn81/pkgs-before.txt` — 546 packages.
- Tracked-file baseline: `git ls-files | wc -l` — 7,785.
- Disk baseline: `du -sh tools/umpire3 model/.lake` — `5.1M` and `2.8G`. `tools/umpire3/model/.lake`
  no longer exists at task start; it is gitignored regenerable Lake output that was removed from the
  working tree before this task, so the 677 MB the spec measured is already reclaimed and is not
  counted again in the fn-81 receipt.
- Gate baselines observed before any edit: `go build -tags 'test_dep integration' ./...` exited 0.
  `go vet -tags test_dep ./...` exited 1 with 15 inherited diagnostics in
  `common/persistence/sql/sqlplugin/tests`, `common/testing/stamp`, `service/frontend`, and
  `tools/tdbg`. None of the 15 is in a deletion-set root or in a file this spec edits, so the vet
  baseline is inherited-red and the fn-81 requirement is that the same 15 remain and no new
  diagnostic appears. `make lint-code GOLANGCI_LINT_FIX=false` reported 1,284 issues at task start.
- `go run ./tools/planindex` exited 1 at task start with 44 inherited findings: eleven `.plans`
  documents that are unregistered or registered-but-absent, and thirty-three flow-record
  discrepancies (unregistered specs, dependency-set mismatches, and status drift). None of them is
  caused by fn-81. Exactly one is in fn-81's declared surface — the stale
  `.plans/UMPIRE_DSL_EVOLUTION_SPEC.md` entry that R7 removes. Registering the unregistered specs,
  fn-81's own included, is the same wider `.plans` reconciliation as the rest and is outside this
  spec's boundaries, which permit notes on `.plans` but no broader rewrite. The fn-81 requirement is
  therefore that the finding count does not rise above the baseline and that every finding this
  sweep itself creates is resolved before it closes. `make umpire-check-regression` does not depend
  on this gate.
- `.flow/tmp` holds a duplicate tree with its own nested modules under `.flow/tmp/fn20.4-base-*`.
  `git ls-files .flow/tmp` returns zero paths, so the whole directory is **untracked** scratch. It
  is not part of the deletion set, is invisible to `go list ./...` from the repository root, and
  needs no classification beyond this row.

### Deletion-set roots

| Root | Decision | Consumers per search surface, and disposition |
| --- | --- | --- |
| `tools/gomad` (23 files, 1,847 lines; module `go.temporal.io/server/tools/gomad`) | delete in .3 | Go: no importer anywhere — `git grep -E '"go\.temporal\.io/server/tools/gomad[/"]' -- '*.go'` outside the tree returns nothing for the root package or any subpackage, and its own module is never required by the root `go.mod`. Makefile: the `gomad-prototype` block at `:303-317`, which is **not** gomad-only and is therefore itemized by line: `:308` runs the deleted `tools/agentworkflow` suite, `:309` runs the **retained** `tools/common/formal` suite (see the retained-neighbour row below), `:310` runs gomad's own suite, `:313` runs `cd model && $(LEAN_LAKE) build Shared` against the **retained** primary Lean workspace, `:314` builds `tools/gomad/formal`, and `:317` builds `tools/gomad/formal/veil`. Module: `tools/gomad/go.mod:5,7` require and replace the nested module `go.temporal.io/server/tools/common/formal`, imported at `conformance/replay.go:7-8` and `trace/corpus.go:4` — gomad is that module's only importer anywhere. Ignore files: `.gitignore:71-72` (`/tools/gomad/formal/.lake/`, `/tools/gomad/formal/veil/.lake/`). Lake: `tools/gomad/formal/lake-manifest.json` and `tools/gomad/formal/veil/lake-manifest.json`; no retained workspace imports either, but `tools/gomad/formal/lakefile.toml:4-6` declares a `Shared` library with `srcDir = "../../../model"`, so it reads the retained primary Lean workspace rather than being fully self-contained. That library is a duplicate view: `model/lakefile.lean:59` already declares `@[default_target] lean_lib Shared`, so `make umpire-build-model` (`cd model && lake build`) builds `model/Shared` and `model/Shared/**` without the gomad lakefile, and deleting gomad loses no Lean build coverage. Workflows, CODEOWNERS, scripts, proto, Lean, mise: none. Disposition: delete with its Makefile block (.4) and ignore rows (.4). |
| `tools/gomad1` (163 files, 55,725 lines; no `go.mod`, compiled inside the root module) | delete in .3 | Go: `git grep 'tools/gomad1' -- '*.go'` outside the tree returns nothing; `ctrl/dropin.go` `RunInSim` has zero callers. Ignore files: `.gitignore:5` (`tests/.gomad-run/`) exists only because `tools/gomad1/ctrl/dropin.go:22,32` defaults its output workspace to `<srcDir>/.gomad-run`; it is a gomad1 artifact path, not a gomad3 one. Makefile, workflows, CODEOWNERS, scripts, proto, Lean, mise: none. Disposition: delete; `.gitignore:5` goes with it in .4. Because it sits in the root module, its removal is the one gomad deletion the root build observes. |
| `tools/gomad2` (193 files, 42,184 lines; module `github.com/temporalio/gomad`) | delete in .3 | Go: no source file in the repository imports `github.com/temporalio/gomad` — the only references are `go.mod:62` (`require github.com/temporalio/gomad v0.0.0`) and `go.mod:263` (`replace github.com/temporalio/gomad => ./tools/gomad2`). Generated manifest: `tools/gomad3/simulation/parity/manifest.go:315` requires every parity source path to carry the `tools/gomad2/` prefix, and `tools/gomad3integration/simulation_contract_test.go:17` reads `simulation/parity/manifest.json`; that manifest is the sole live consumer and is retired in the same commit. Prose: `tools/gomad3/README.md:601` names the manifest. Scripts: `tools/gomad2/test.sh` is self-contained. Proto: `tools/gomad2/internal/tests/testpb.proto:3` declares its own `go_package`, and no retained proto imports it. Docs: `docs/superpowers/specs/2026-08-09-gomad2-entrypoint-design.md:11,23`. Disposition: delete with the root `go.mod` require and replace, the parity manifest package, and the parity assertions; the design record gets a historical banner in .5. |
| `tools/umpire1` (78 files, 14,740 lines) | delete in .2 | Go: `service/history/workflow/cache/cache.go:32` imports `tools/umpire1/model` for `model.Namespace(...).Workflow(...).Execution(...)` entity tags — the production seam — and `tests/testcore/monitor/monitor_test.go:5`. Docs: `docs/superpowers/specs/2026-08-12-umpire-follow-up-observability-design.md:16`. Makefile, workflows, CODEOWNERS, ignore files, scripts, proto, Lean, mise: none. Disposition: the seam removal in .2 precedes deletion; both importers are removed in the same task, the monitor package first (commit 1's `tests/lost_task_test.go` and commit 3's `tests/testcore/monitor`). |
| `tools/umpire2` (309 files, 130,489 lines) | delete in .2 | Go: `tests/testcore/functional_test_base.go:51`, `tests/testcore/test_env_test.go:13`, `tests/testcore/monitor/monitor_test.go:6`, `tests/probe/probe.go:29`, `tests/umpire2_probe_test.go:23-24`, `tests/umpire2_regress_test.go:13-15`, `tests/umpire2_test.go:22`, `cmd/umpire-genmodels/main.go:18`. CODEOWNERS: `:99-102` (four rows over `tools/umpire2/internal/protocol` and `tools/umpire2/testdata/genmodels`). Ignore files: `.gitignore:16-18` (testdata un-ignore rules) and `:46-49` (verification tool output). Workflows: `umpire-model-verification.yml:8,40,106-107,122` — path trigger, `source tools/umpire2/testdata/genmodels/tools.env`, two test selectors, and the results artifact path. Lean: `model/Temporal/Feature/Nexus/Experimental/AutoClose.lean:173`, a doc comment stating that Layer 1 mirrors `tools/umpire2/internal/model/nexus_operation.go` — the only Lean hit for any deletion-set root, and a prose reference rather than a build dependency. Docs: `docs/superpowers/specs/2026-08-12-umpire-follow-up-observability-design.md:22,34,154,258`. Disposition: delete in .2 after every Go importer is gone; the Lean comment is rewritten as a historical provenance note in .5 under R7; the workflow, CODEOWNERS rows, and ignore rows go in .4; the design record gets a banner in .5. |
| `tools/umpire3` (522 files, 92,858 lines, 5.1 MB tracked) | delete in .2 | Go: `tests/umpire3_mechanisms_test.go:38-43`, `tests/umpire3_participant_process_test.go:20`, `tests/umpire3_probe_test.go:22-31`, `tests/umpire3_regress_test.go:6-7`, `tests/umpire3_sdk_test.go:31-39`, `tests/umpire3_test.go:12-14` — all six are deletion-set legacy tests. Makefile: the `UMPIRE3_*` variable blocks at `:85-119` and `:147-178` and roughly seventy `umpire3-*` targets at `:612-999` and `:1282-1392`. Workflows: `.github/workflows/umpire3.yml` in full (its own path trigger, `working_directory: tools/umpire3/model`, `make umpire3-check`, `make umpire3-integration`). Ignore files: `.gitignore:27-30` (testdata un-ignore) and `:68-69` (its Lake build outputs). Lean: a second Lake project under `tools/umpire3/model` (193 tracked files, mathlib-dependent, its own `lean-toolchain`); no retained Lean workspace imports it — the primary `model/` workspace never references `Umpire3.*`. Test fixture: `tools/umpire/vocabulary/retired_vocabulary_test.go:91` writes a fixture at the literal path `tools/umpire3/history.go` inside a `t.TempDir()`, exercising the excluded-history rule. Disposition: delete in .2; the fixture path is repointed to a neutral name in the same task, and the Makefile blocks, workflow, and ignore rows go in .4. |
| `tools/agentworkflow` (43 files, 11,695 lines; module `go.temporal.io/server/tools/agentworkflow`) | delete in .3 | Go: no importer outside the tree; `docs/superpowers/specs/2026-08-24-agentworkflow-configuration-cli-design.md:78` records the same finding independently. Makefile: `:290-301` (`agentworkflow-test`, `-race`, `-vet`, `-check` and their `.PHONY` line) and `:308`, where `gomad-test` also runs the agentworkflow suite. Docs: `docs/research/agentworkflow-oss-landscape.md:19,71` carry sixteen `../../tools/agentworkflow/...` links, plus three design records. Workflows, CODEOWNERS, ignore files, scripts, proto, Lean, mise: none. Disposition: delete; Makefile rows in .4; documentation banners and de-linking in .5. |
| `cmd/umpire-genmodels` (3 files, 1,960 lines) | delete in .2 | Go: imports `common/testing/umpire/verify`, `.../verify/toolchain`, and `tools/umpire2` (`main.go:16-18`, `main_test.go:15-16`, `tool_environment.go:9`); nothing imports it. Makefile: `:84` (`UMPIRE_GENMODELS`) and `:594-610` (`umpire-genmodels`, `umpire-check-genmodels`, `umpire-verify-smoke`, `umpire-verify-nightly`, `.PHONY`). mise: `mise.toml:6-13` (`umpire:install-tools`, `umpire:verify-smoke`). Scripts: `develop/umpire/install-tools.sh:19` is its only other caller and has no other purpose. Workflows: `umpire-model-verification.yml:6,101,108,116`. Disposition: delete in .2 together with the mise tasks and the install script; Makefile and workflow rows in .4. |
| `common/testing/umpire` (96 files, 26,721 lines) | delete in .2 | Go, under `service`: exactly one importer, `service/history/workflow/cache/cache.go:27` (`umpireotel "…/common/testing/umpire/oteladapter"`), verified with `git grep -l 'umpireotel\|common/testing/umpire' -- service`. Other Go: `tests/testcore/monitor/monitor.go:10`, `tests/probe/{probe,coverage,report}.go`, `tests/umpire2_probe_test.go:20`, `tests/umpire2_regress_test.go:9-10`, `tests/umpire2_test.go:20`, `cmd/umpire-genmodels`. CODEOWNERS: `:98` (`/common/testing/umpire/verify/`). Workflows: `umpire-model-verification.yml:7,105`. Docs: `docs/superpowers/specs/2026-08-12-umpire-follow-up-observability-design.md:36,118,257`. Disposition: delete in .2 in the same commit as the three umpire trees, because the seam and the trees form one import cycle across the root module. |
| `service/history` observer comments (6 files) | comment-only edit in .2 | No import, no call: `respondworkflowtaskcompleted/workflow_task_completed_handler.go:847,1171-1174`, `startworkflow/api.go:270-273`, `updateworkflow/api.go:346-349`, `ndc/workflow_resetter.go:282-283`, `workflow/retry.go:324-325`, `workflow/update/util.go:24,122,139,153`. Each points the reader at the umpire test observer, and four of them cite `.plans/UMPIRE.md`, a file that does not exist in this repository. Disposition: rewrite or drop the comment lines only; the OTEL span events they describe are retained production behaviour and no code changes. |
| `tests/testcore/monitor` (2 files, 46 lines) | delete in .2 | Go: `monitor.go` imports `common/testing/umpire`; `monitor_test.go` imports `tools/umpire1` and `tools/umpire2`. Its only consumers are `tests/testcore/functional_test_base.go` (factory, accessor, purge, interceptor) and `tests/testcore/test_env.go` (factory option). Disposition: delete with the monitor API in commit 3 of .2. |
| `tests/umpire2_*.go`, `tests/umpire3_*.go` (9 files) and `tests/lost_task_test.go` (1 file, 5,760 lines together) | delete in .2 | These are the only readers of the deleted trees inside package `tests`. `tests/lost_task_test.go` is branch-only rather than upstream and is the sole retained reader of the monitor API (`AllowMonitorViolations` at `:45,132,200`; `GetMonitor().CheckNamespace` at `:102,155,239`). Disposition: delete in commit 1 of .2, because `go test ./tests` compiles the whole package and would otherwise fail on the removed monitor API. The lost-task property it asserted is recorded below as a future Testpilot regression candidate. |
| `tests/nexus_workflow_test.go` (retained) | restore upstream bodies in .2 | **Consumer found during .2, not during .1 research, and recorded here under R1's error clause.** Three retained functional tests in this file had their bodies replaced on this branch by delegations into the umpire2 sparse-regression engine: `TestNexusOperationStartsStandaloneActivityBidirectionalLinks:629`, `TestNexusCallbackAfterCallerComplete:2385`, and `TestNexusOperationStartToCloseTimeout:2847` each call a `runUmpire2SparseRegression*` helper defined in `tests/umpire2_regress_test.go`. `git diff origin/main -- tests/nexus_workflow_test.go` shows the branch traded 773 upstream lines for 110. Deleting `tests/umpire2_regress_test.go` therefore leaves three retained tests without bodies — an unclassified live consumer of a deletion root inside retained code. Disposition: restore the three method bodies from `origin/main` verbatim in commit 1 of .2, preserving the one branch-added guard (`TestNexusCallbackAfterCallerComplete`'s CHASM skip) that upstream does not carry. This deletes no coverage; it returns the tests to the assertions they made before the generation replaced them, and `go vet -tags 'test_dep integration' ./tests/...` type-checks them against the current branch unchanged. |
| `tests/probe` (3 files, 700 lines) | delete in .2 | Go: imports `common/testing/umpire` and `tools/umpire2`; imported only by `tests/umpire2_probe_test.go`. Disposition: delete in commit 1 of .2. |
| `umpire-check-live-tests` pinned failure list (`Makefile:1150-1160`) | retire in .4 | Nine expected-failure identities, all defined by files this spec deletes. Disposition: R4 replaces the pinned list with an empty baseline plus a passing-identity floor. |

The Lean, proto, and Lake search surfaces were run over every deletion-set root. Apart from the
`AutoClose.lean:173` comment recorded in the `tools/umpire2` row and each tree's own self-contained
files (`tools/gomad2/internal/tests/testpb.proto`, the two `tools/gomad/formal` Lake manifests, and
the `tools/umpire3/model` Lake project), those surfaces returned nothing for any root — the silence
is a checked result, not an unrun search.

Deletion-set totals: 1,435 tracked files and 378,965 tracked lines in the eleven roots above, plus
10 test files and 5,760 lines in `tests`, for 1,445 files and 384,725 lines. Of that, 235,684 lines
are Go; the remainder is Lean under `tools/umpire3/model`, testdata, and fixtures.

### Retained neighbours

| Neighbour | Decision | Reason and evidence |
| --- | --- | --- |
| `tools/fairsim`, `cmd/tools/fairsim` | retained — not a choice | Upstream Temporal code, not a pre-Testpilot generation of this project. `Makefile:561,573-575` and `.gitignore:55` stay untouched. Excluding it is upstream ownership, not a scope decision. |
| `tools/planindex`, `.plans/index.json` | retained — fn-66 carve-out revalidated | fn-66 retained `tools/planindex` and `make umpire-check-plan-index` deliberately as adjacent tooling outside its inventory. fn-81's first draft proposed deleting it; that is reversed here. Its validator is the only thing that keeps `.plans/index.json` honest once the deleted-tree documents are relabelled historical in .5, so deleting it would remove the gate that checks this spec's own documentation reconciliation. |
| `tools/gomad3`, `tools/gomad3sim`, `tools/gomad3integration`, `tests/gomadfunctional` | retained per `.plans/GOMAD_MILESTONES.md` F0 | The 2026-09-08 assessment found gomad3 is the only tree that can run an unchanged Temporal functional test under a deterministic runtime. Retained wiring: `gomad3.yml`, `Makefile:201,240,242,319-352,1396,1477`, `.gitattributes:3-4`, `.github/.yamlfmt:13`, `.gitignore:4` (`.gomad/` is the gomad3 artifact store, distinct from gomad1's `tests/.gomad-run/`), and `tools/gomad3/qualification/corpus/go.mod`. Their only fn-81 edit is the parity-manifest retirement: `tools/gomad3/simulation/parity` is deleted, its importer `internal/gomadtool/validation/script_policy.go:12,29-30` loses the check, `tools/gomad3integration/simulation_contract_test.go` loses the parity assertions, and `tools/gomad3/README.md:601` loses the manifest sentence. `tools/gomad3/.toolchain` (770 MB, generated GOROOTs) is untracked and out of scope. |
| `tools/umpire`, `common/testing/testpilot`, `tests/testcore/testpilot` | retained, untouched apart from reference edits | No deletion-set root imports them and they import no deletion-set root. `tools/umpire/regression/ci_workflow_test.go` is edited in .4 to pin the new gate, and `tools/umpire/vocabulary/retired_vocabulary_test.go:91` is repointed in .2. |
| `tools/common/formal` (6 files; module `go.temporal.io/server/tools/common/formal`) | retained per the spec's Retention set — **but orphaned by this sweep, and .4 owns the replacement** | This is the one retained neighbour that a deletion-set root actually consumes, so it is recorded separately rather than folded into the row above. `tools/gomad/go.mod:5,7` require and replace it, and `tools/gomad/conformance/replay.go:7-8` and `tools/gomad/trace/corpus.go:4` import its `model`, `trace`, and `conformance` packages. Nothing else in the repository imports the module — it is invisible to the root build, so neither `deps-before.txt` nor `pkgs-before.txt` contains it. Its only test invocation is `Makefile:309` (`cd tools/common/formal && GOWORK=off go test -tags test_dep ./...`), which sits inside the `gomad-prototype` block .4 deletes. Deleting gomad therefore leaves it with zero importers and, unless .4 acts, zero build or test coverage. The spec's Retention set keeps it and its Boundaries forbid refactoring retained code, so the disposition is: **keep the module, and in .4 preserve line `:309` as its own named target rather than deleting it with the block.** The `gomad-prototype` block is thus not a block whose only purpose is a deleted tree, and R3's removal rule applies to it line by line. Follow-up recorded, not actioned here: `.gitignore:70` (`/tools/common/formal/.lake/`) is already stale — the module's Lean half migrated to `model/Shared.lean` and `model/Shared/**` and no lakefile remains under `tools/common/formal` — so that row is dead weight independent of fn-81 and is left alone rather than swept in. |
| `.plans`, `docs`, `.turbo` | retained, notes only | Historical reasoning that led to Umpire 4. Records that link into deleted trees gain a historical banner in .5; nothing is deleted. |
| Lean previous generations (Nexus v1, Nexus2, Umpire Artifact, Space, Exploration under `model/`) | out of scope | Imported by live modules or reserved by open specs fn-22, fn-33, fn-79, and fn-80. Their removal needs a roadmap decision, not this sweep. |
| `.flow/tmp/fn20.4-base-*` duplicate tree | untracked scratch | `git ls-files .flow/tmp` returns zero paths. Not tracked, not built, not classified further. |

### Tidy verification (task .3)

`go mod tidy` after the gomad deletions dropped twelve modules from `go.mod` and `go.sum`. R2's
check is that tidy drops **only modules absent from the retained dependency closure**, so the
comparison basis is `.flow/tmp/fn81/deps-after.txt` (2,569 packages), the closure of the retained
tree, captured with the same `go list -deps -test -tags 'test_dep integration' ./...` command as the
before baseline. `deps-before.txt` (2,889) was taken over the whole pre-deletion tree and therefore
still contains the deleted roots' own dependencies; a nonzero count there is the evidence that the
module belonged to a deleted root, not a violation. Every dropped module has zero hits in the
retained closure and zero import sites in any retained `.go` file:

| Dropped module | `deps-before` | `deps-after` | Owner in the deleted set |
| --- | --- | --- | --- |
| `github.com/go-cmd/cmd` | 1 | 0 | `tools/gomad1/ctrl` |
| `github.com/looplab/fsm` | 1 | 0 | `common/testing/umpire/lifecycle.go` (deleted in .2) |
| `github.com/petermattis/goid` | 1 | 0 | `tools/gomad1/runtime` |
| `github.com/pingcap/failpoint` | 1 | 0 | `tools/gomad1/ctrl` |
| `github.com/rivo/uniseg` | 1 | 0 | `tools/gomad1/runtime`, `tools/gomad1/transformer` |
| `github.com/spf13/afero` | 3 | 0 | `tools/gomad1/api/lib`, `tools/gomad1/transformer` |
| `gitlab.com/stone.code/assert` | 1 | 0 | `tools/gomad1/transformer` |
| `github.com/dave/dst` | 0 | 0 | `tools/gomad2/internal/translate`, `.../internal/tests/script` — reached through the root `replace`, never through a root import |
| `github.com/dave/jennifer` | 0 | 0 | indirect requirement of the above |
| `github.com/go-test/deep` | 0 | 0 | indirect requirement of the above |
| `github.com/sergi/go-diff` | 0 | 0 | indirect requirement of `tools/gomad2` |
| `gopkg.in/check.v1` | 0 | 0 | indirect requirement of `github.com/spf13/afero` |

The research prediction recorded in the task ("zero third-party removals beyond the gomad module
itself") was wrong by twelve modules; the four that carried gomad1's transformer and runtime were
compiled inside the root module, so they were real root-module requirements until gomad1 went.

Package-set subset: `comm -13 pkgs-before.txt pkgs-after.txt` is empty — 546 packages before, 422
after, and the 124 removed are exactly the deleted roots. `go mod tidy` run a second time is a no-op
on both `go.mod` and `go.sum`.

### Retired live-test failure identities

R4 retires the pinned expected-failure list. It is pinned **twice**, and .4 retires both copies
together: `Makefile:1150-1160` defines the gate, and `tools/umpire/regression/ci_workflow_test.go:25-43`
pins the same nine identities plus the current live command as the CI assertion over it. All nine
identities are defined by files this spec deletes, so an empty baseline is the correct successor
rather than a weakened gate:

| Pinned identity | Defining file |
| --- | --- |
| `TestUmpire2TestSuite` | `tests/umpire2_test.go` |
| `TestUmpire2TestSuite/TestPlanAndDriveKitchenSinkNexusOperation` | `tests/umpire2_test.go` |
| `TestUmpire2TestSuite/TestPlanAndDriveNexusOperationCHASM` | `tests/umpire2_test.go` |
| `TestUmpire2TestSuite/TestProbeNexusDegraded` | `tests/umpire2_probe_test.go` |
| `TestUmpire2TestSuite/TestProbeNexusExploration` | `tests/umpire2_probe_test.go` |
| `TestUmpire2TestSuite/TestProbeNexusFlagged` | `tests/umpire2_probe_test.go` |
| `TestUmpire2TestSuite/TestProbeNexusRandomized` | `tests/umpire2_probe_test.go` |
| `TestUmpire2TestSuite/TestProbeNexusResilience` | `tests/umpire2_probe_test.go` |
| `TestUmpire3ParticipantProcessCrashAndRestartResumesRealSDKProgram` | `tests/umpire3_participant_process_test.go` |

Because an empty expected set cannot by itself distinguish "everything passed" from "the selector
matched nothing", R4 pairs the empty baseline with a floor requiring at least one `--- PASS`
identity in the verbose output.

### Future Testpilot regression candidates

`tests/lost_task_test.go` is deleted rather than ported, because every fact it reads comes from the
monitor seam. Its three properties are recorded here so a future Testpilot Case can reclaim them
from declared Observations rather than from a white-box adapter:

- **Lost task.** A workflow task written to persistence and then removed by
  `CompleteTasksLessThan` is never delivered to a poller, and the discrepancy between stored and
  polled tasks is observable.
- **Stuck workflow.** A workflow that is started but never receives a
  `RespondWorkflowTaskCompleted` remains in the started state past a bounded horizon.
- **Stuck workflow via the SDK.** A workflow blocked in `workflow.Await` on a condition that never
  becomes true, whose worker is stopped before the first task completes, is detectable as stuck.

### Authorized deletion paths

Deletion is authorized only for the roots in the deletion-set table above, plus:

- `tools/gomad3/simulation/parity` (the parity manifest package retired with `tools/gomad2`).
- `.github/workflows/umpire3.yml` and `.github/workflows/umpire-model-verification.yml`.
- `develop/umpire/install-tools.sh` and the two `mise.toml` tasks that call it.
- The `go.mod` require and replace for `github.com/temporalio/gomad`, and the `go.sum` rows
  `go mod tidy` drops as a consequence.
- The Makefile variables, targets, and `.PHONY` names that serve only a deleted root, the
  CODEOWNERS rows over deleted paths, and the `.gitignore` rows over deleted paths.

No other path is authorized. No compatibility shim, archive branch, or replacement monitor is
authorized: the functional harness loses the seam rather than gaining a no-op.
