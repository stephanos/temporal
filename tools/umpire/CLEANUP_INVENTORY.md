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
| `tools/umpire/cmd/umpire-gen-case-runtime-conformance` | retained | Make generation/check targets build `temporal-case-runtime`, invoke this command, compare the entire managed fixture tree, and run facade conformance. |
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
`tools/umpire/...` inventory, and explicitly retained outside fn-66 scope.

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
| Switch Experiment `switch.query.exact-action`, artifact checksum `sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd` | source `model/Umpire/Examples/testdata/switch-experiment-spec.json`: `55f0961e02761ed6ec3718ef6d22fa4284e70e729dba7f21fabb0a3e8798bac0` |
| Switch generated Go view | `8a23cdc22e53a2a9d2860522d3f1353898336f5758830ffb32d42948125300a5` |
| Switch generated Markdown view | `33608f42fccedfe34309a429506778f0272eeec8acd2a136b8310c8936e46747` |
| Case Runtime conformance tree, six named classes / 12 files | SHA-256 of its sorted `sha256sum` manifest: `7809b6829822c097dccb76a07e4abdba233f3a552cb553c5ab019ca1888dddb9` |
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
  work. A 10x increase in retained reader, generator, Case Runtime, or live-test load therefore
  follows the unchanged implementations and cost bounds exercised by the retained gates.

## Reconciliation

- Realized removal: .2 removed 51 files, 92 Tests, 1 Fuzz target, and 27 fixtures; .3 owns 5 files
  and 5 Tests. The total is 56 files, 97 Tests, 1 Fuzz target, and 27 fixtures.
- Every current package and command has a concrete retained consumer or an evidence-backed removal
  decision. Every fixture origin/reference and every candidate top-level test is accounted for.
- The post-.2 symbol and import closure found no retained consumer of the four internal codecs or
  clone helpers. Repository searches found no additional unused package, command, helper, fixture,
  or active reference in scope. There was no scope conflict requiring revision before .3.
