package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/model-checkers/veil"
	"go.temporal.io/server/tests/umpire3/observation"
	"go.temporal.io/server/tests/umpire3/protocol"
	releaseassurance "go.temporal.io/server/tests/umpire3/release"
)

func TestResolveSourceDependenciesFollowsLocalLeanImports(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "Feature"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Root.lean"), []byte("import Feature.Model\nimport Lean.Data.Json\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Feature", "Model.lean"), []byte("import Feature.Shared\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Feature", "Shared.lean"), []byte("def shared := true\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Unrelated.lean"), []byte("def unrelated := true\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "selection.json"), []byte("{}\n"), 0o600))

	dependencies, err := resolveSourceDependencies(root, "Root.lean", []string{"selection.json"})
	require.NoError(t, err)
	require.Equal(t, []string{
		"Feature/Model.lean", "Feature/Shared.lean", "Root.lean", "selection.json",
	}, dependencyPaths(dependencies))
	require.Equal(t, []string{"Feature/Shared.lean"}, dependencies[0].Imports)
	require.Empty(t, dependencies[1].Imports)
	require.Equal(t, []string{"Feature/Model.lean"}, dependencies[2].Imports)
	require.Empty(t, dependencies[3].Imports)
}

func TestRunLeanRebuildsChangedImports(t *testing.T) {
	root, err := os.MkdirTemp("../../model", ".export-stale-import-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(root))
	})
	for name, contents := range map[string]string{
		"lean-toolchain": "leanprover/lean4:v4.33.0\n",
		"lakefile.toml":  "name = \"export-stale-import-test\"\ndefaultTargets = [\"Dep\"]\n[[lean_lib]]\nname = \"Dep\"\n",
		"Dep.lean":       "def exportedValue : String := \"before\"\n",
		"Main.lean":      "import Dep\ndef main : IO Unit := IO.println exportedValue\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600))
	}
	command := exec.Command("mise", "exec", "--", "lake", "build")
	command.Dir = root
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "initial Lean build: %s", output)
	require.NoError(t, os.WriteFile(filepath.Join(root, "Dep.lean"),
		[]byte("def exportedValue : String := \"after\"\n"), 0o600))

	output, err = runLean(root, "Main.lean", "semantic-hash", "")
	require.NoError(t, err)
	require.Equal(t, "after\n", string(output))
}

func TestExportCatalogRunsLeanAndProducesValidatedCatalog(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportCatalog("../../model", catalogSpec, &output))

	catalog, err := protocol.DecodeCatalog(bytes.NewReader(output.Bytes()), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, "temporal/umpire3/catalog/v1", catalog.CatalogVersion)
	require.Len(t, catalog.Actions, 33)
	for _, identifier := range []protocol.ActionKind{
		protocol.ActionKindContinueWorkflow,
		protocol.ActionKindResetWorkflow,
		protocol.ActionKindRouteWorkflowTask,
		protocol.ActionKindFenceWorkflowOwner,
	} {
		_, ok := catalog.Action(string(identifier))
		require.True(t, ok, identifier)
	}
}

func TestExportProofManifestRunsLeanAndMatchesExperiment(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportProofManifest("../../model", proofSpecs["nexus"], &output))

	manifest, err := protocol.DecodeProofManifest(bytes.NewReader(output.Bytes()), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, "nexus-tasks-refinement-v1", manifest.Identifier)
	require.Equal(t, "Umpire3.Temporal.System.NexusTasks.nexusTasksRefinesProduct", manifest.Theorem)
	require.Equal(t, protocol.ResultClassRefinementProved, manifest.ResultClass)
	require.Equal(t, protocol.TrustBadgeKernelWithDeclaredAxioms, manifest.TrustBadge)
	require.Equal(t, []string{"propext"}, manifest.Axioms)
	require.NotEmpty(t, manifest.Statement)
	require.NotEmpty(t, manifest.SourceDependencies)
}

func TestExportReleaseCandidateRefreshesSourceBindings(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportReleaseCandidate(
		"../../testdata/umpire3-1.2.json",
		[]string{"../../testdata/nexus-cancellation.json", "../../testdata/update-lifecycle.json"},
		"../../migration/ledger.json",
		&output,
	))

	release, err := protocol.DecodeReleaseManifest(output.Bytes())
	require.NoError(t, err)
	require.NoError(t, releaseassurance.ValidateAgainstCurrent(release))
	require.Equal(t, "umpire3/migration-ledger/v3", release.Migration.FormatVersion)
	require.Equal(t, release.Migration.BehaviorCount,
		release.Migration.ExactCount+release.Migration.SemanticEquivalentCount+
			release.Migration.PartialCount+release.Migration.InventoryOnlyCount)
}

func TestExportFirstOrderViewRunsLeanAndPreservesVariant(t *testing.T) {
	for _, variant := range []string{"sound", "mutated"} {
		t.Run(variant, func(t *testing.T) {
			var output bytes.Buffer
			require.NoError(t, exportFirstOrderView("../../model", firstOrderSpecs[variant], &output))

			view, err := protocol.DecodeFirstOrderView(bytes.NewReader(output.Bytes()), protocol.DefaultDecodeLimit)
			require.NoError(t, err)
			require.Equal(t, protocol.TargetID("nexus-cancellation"), view.Target)
			require.Equal(t, protocol.PropertyIDNexusCancellationWonExcludesSuccess, view.Property)
			require.Equal(t, protocol.TrustBadgeKernelWithDeclaredAxioms, view.Relation.TrustBadge)
			require.NotEmpty(t, view.Relation.Declaration)
			require.NotEmpty(t, view.Oracle.States)
		})
	}
}

func TestExportVeilBindingRunsLeanAndBindsCompiledDeclarations(t *testing.T) {
	for _, test := range []struct {
		variant           string
		firstOrderVariant string
		trustMode         veil.SMTTrustMode
	}{
		{variant: "sound", firstOrderVariant: "sound", trustMode: veil.ReconstructedSMT},
		{variant: "mutated", firstOrderVariant: "stale-completion-guard-removed", trustMode: veil.ReconstructedSMT},
		{variant: "trusted", firstOrderVariant: "sound", trustMode: veil.TrustedSMT},
	} {
		t.Run(test.variant, func(t *testing.T) {
			var output bytes.Buffer
			require.NoError(t, exportVeilBinding("../../model", veilBindingSpecs[test.variant],
				firstOrderSpecs[map[string]string{"sound": "sound", "mutated": "mutated", "trusted": "sound"}[test.variant]],
				&output))

			binding, err := veil.DecodeBindingArtifact(bytes.NewReader(output.Bytes()),
				protocol.DefaultDecodeLimit)
			require.NoError(t, err)
			var firstOrderOutput bytes.Buffer
			firstOrderSpec := firstOrderSpecs[map[string]string{
				"sound": "sound", "mutated": "mutated", "trusted": "sound",
			}[test.variant]]
			require.NoError(t, exportFirstOrderView("../../model", firstOrderSpec, &firstOrderOutput))
			view, err := protocol.DecodeFirstOrderView(bytes.NewReader(firstOrderOutput.Bytes()),
				protocol.DefaultDecodeLimit)
			require.NoError(t, err)
			require.Equal(t, test.firstOrderVariant, view.Variant)
			require.NoError(t, binding.ValidateAgainst(view))
			require.Equal(t, test.trustMode, binding.Binding.TrustMode)
			require.Equal(t, protocol.VeilBackendRevision, binding.BackendRevision)
			require.NotEqual(t, "derived", binding.ArtifactDigest)
		})
	}
}

func TestVeilBindingSourceDigestCoversDeclarationsAndSemanticProofs(t *testing.T) {
	dependencies, err := resolveSourceDependencies("../../model",
		veilBindingSpecs["sound"].sourceRoot, veilBindingSpecs["sound"].inputs)
	require.NoError(t, err)
	require.Subset(t, dependencyPaths(dependencies), []string{
		"Temporal/Targets/NexusCancellationFencingFirstOrder.lean",
		"Temporal/Veil/NexusCancellationFencing/Binding.lean",
		"Temporal/Veil/NexusCancellationFencing/Sound.lean",
		"Temporal/Veil/NexusCancellationFencing/SoundConcrete.lean",
		"Temporal/Veil/NexusCancellationFencing/SoundSemantics.lean",
		"Temporal/Veil/NexusCancellationFencing/SoundConcreteSemantics.lean",
		"Umpire3/Veil/Semantics.lean",
		"lake-manifest.json",
		"lean-toolchain",
	})
}

func TestCheckerCoverageDerivesSupportedAndUnsupportedStatusFromEvidence(t *testing.T) {
	inputs := checkerCoverageInputs{
		nativeCertificate: "../../model-checkers/native/results/nexus-cancellation-scale.certificate.json",
		nativeReceipt:     "../../model-checkers/native/results/nexus-cancellation-scale.receipt.json",
		nativeBenchmark:   "../../model-checkers/native/results/nexus-cancellation-scale.benchmark.json",
		veilBinding:       "../../model-checkers/veil/bindings/nexus-cancellation-sound.json",
		veilResults: []string{
			"../../model-checkers/veil/results/nexus-cancellation-sound-concrete.json",
			"../../model-checkers/veil/results/nexus-cancellation-sound-symbolic.json",
			"../../model-checkers/veil/results/nexus-cancellation-sound-invariant.json",
		},
	}
	manifest, err := buildCheckerCoverage(inputs)
	require.NoError(t, err)

	checked := make(map[protocol.CheckerKind]int)
	for _, entry := range manifest.Entries {
		if entry.Status == protocol.CheckerCoverageChecked {
			checked[entry.Checker]++
		}
	}
	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)
	exactTargets := 0
	for _, target := range catalog.Targets {
		exactTargets += len(target.Properties)
	}
	require.Equal(t, exactTargets, checked[protocol.CheckerExact])
	require.Equal(t, 1, checked[protocol.CheckerNative])
	require.Equal(t, 1, checked[protocol.CheckerVeil])
	nativeEntry := findCheckerCoverageEntry(t, manifest, protocol.TargetIDNexusCancellation,
		protocol.PropertyIDNexusCancellationWonExcludesSuccess, protocol.CheckerNative)
	require.Equal(t, []string{"native-certificate", "native-lean-receipt", "native-scale-benchmark"},
		checkerEvidenceKinds(nativeEntry.Evidence))

	inputs.veilBinding = "../../model-checkers/veil/bindings/nexus-cancellation-mutated.json"
	_, err = buildCheckerCoverage(inputs)
	require.ErrorContains(t, err, "does not match the first-order view")
}

func findCheckerCoverageEntry(
	t *testing.T,
	manifest protocol.CheckerCoverageManifest,
	target protocol.TargetID,
	property protocol.PropertyID,
	checker protocol.CheckerKind,
) protocol.CheckerCoverageEntry {
	t.Helper()
	for _, entry := range manifest.Entries {
		if entry.Target == target && entry.Property == property && entry.Checker == checker {
			return entry
		}
	}
	require.FailNow(t, "checker coverage entry not found")
	return protocol.CheckerCoverageEntry{}
}

func checkerEvidenceKinds(evidence []protocol.CheckerEvidence) []string {
	kinds := make([]string, len(evidence))
	for index, item := range evidence {
		kinds[index] = item.Kind
	}
	return kinds
}

func TestExportAttemptViewRunsLeanAndBindsFirstOrderView(t *testing.T) {
	for _, test := range []struct {
		variant           string
		firstOrderVariant string
	}{
		{variant: "sound", firstOrderVariant: "sound"},
		{variant: "mutated", firstOrderVariant: "stale-completion-guard-removed"},
	} {
		t.Run(test.variant, func(t *testing.T) {
			var output bytes.Buffer
			require.NoError(t, exportAttemptView("../../model", attemptSpecs[test.variant],
				firstOrderSpecs[test.variant], test.variant, &output))

			view, err := protocol.DecodeAttemptView(bytes.NewReader(output.Bytes()),
				protocol.DefaultDecodeLimit)
			require.NoError(t, err)
			firstOrder, found, err := protocol.DefaultFirstOrderView(
				protocol.TargetIDNexusCancellation, test.firstOrderVariant)
			require.NoError(t, err)
			require.True(t, found)
			require.NoError(t, view.ValidateAgainst(firstOrder))
			require.Equal(t, firstOrder.SemanticHash, view.FirstOrderSemanticHash)
		})
	}
}

func TestExportGoIdentifiersUsesCatalogVocabulary(t *testing.T) {
	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)

	var output bytes.Buffer
	require.NoError(t, exportGoIdentifiers(catalog, &output))
	require.Contains(t, output.String(), "ActionKindRequestCancellation")
	require.Contains(t, output.String(), `ActionKind = "request-cancellation"`)
	require.Contains(t, output.String(), `PropertyIDNexusCancellationWonExcludesSuccess`)
	require.Contains(t, output.String(), `LeanVersion = "4.28.0"`)
}

func TestExportAuthorFacadeUsesCatalogDescriptionsAndTypedVocabulary(t *testing.T) {
	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)

	var first bytes.Buffer
	require.NoError(t, exportAuthorFacade(catalog, &first))
	var second bytes.Buffer
	require.NoError(t, exportAuthorFacade(catalog, &second))
	require.Equal(t, first.Bytes(), second.Bytes())
	require.Contains(t, first.String(), "func ScheduleOperation(")
	require.Contains(t, first.String(), "schedule a Nexus operation")
	require.Contains(t, first.String(), "func NexusOperation(")
	require.Contains(t, first.String(), "func RequireNexusCancellationWonExcludesSuccess(")
	require.Contains(t, first.String(), "CapabilityNexus")
}

func TestExportExperimentSchemaIsVersionedAndClosed(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportExperimentSchema(&output))
	require.Equal(t, `"umpire3/v2"`, jsonPath(t, output.Bytes(), "properties", "formatVersion", "const"))
	require.Equal(t, "[\"identifier\",\"kind\",\"allowedOutcomes\",\"requiredCapabilities\"]",
		jsonPath(t, output.Bytes(), "properties", "actions", "items", "required"))
	require.Equal(t, "false", jsonPath(t, output.Bytes(), "additionalProperties"))
}

func TestExportMonitorCatalogRunsLeanAndMatchesSemanticCatalog(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportMonitorCatalog("../../model", monitorSpec, &output))

	catalog, err := protocol.DecodeMonitorCatalog(output.Bytes())
	require.NoError(t, err)
	require.Len(t, catalog.Programs, 16)
	for _, identifier := range []protocol.PropertyID{
		protocol.PropertyIDWorkflowRunContinuationLineage,
		protocol.PropertyIDWorkflowRunResetLineage,
		protocol.PropertyIDWorkflowTaskRoutingIsolation,
		protocol.PropertyIDWorkflowTaskOwnershipFencing,
	} {
		_, ok := catalog.Program(identifier)
		require.True(t, ok, identifier)
	}
}

func TestExportObservationCatalogRunsLeanAndIncludesCheckedFixtures(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportObservationCatalog("../../model", observationSpec, &output))

	catalog, err := observation.DecodeCatalog(output.Bytes())
	require.NoError(t, err)
	require.Len(t, catalog.Programs, 19)
	require.Len(t, catalog.Fixtures, 24)
	_, ok := catalog.Program(protocol.ObservationIDStaleSuccessAbsent)
	require.True(t, ok)
	_, ok = catalog.Program(protocol.ObservationIDNexusOperationProgressed)
	require.True(t, ok)
	_, ok = catalog.Program(protocol.ObservationIDWorkflowOwnershipFenced)
	require.True(t, ok)
}

func TestExportCompositionRunsLeanAndReportsObligations(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportComposition("../../model", compositionSpec, &output))

	composition, err := protocol.DecodeComposition(output.Bytes())
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassCompositionProved, composition.ResultClass)
	require.Len(t, composition.Targets, 16)
	require.Empty(t, composition.MissingMetadata())
}

func TestExportParityLedgerRunsLeanAndCoversInventory(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportParityLedger("../../model", paritySpec, &output))

	ledger, err := protocol.DecodeParityLedger(output.Bytes())
	require.NoError(t, err)
	require.Len(t, ledger.Entries, 22)
	complete := 0
	incomplete := 0
	for _, entry := range ledger.Entries {
		switch entry.EvidenceStatus {
		case protocol.MetadataPresent:
			require.Equal(t, protocol.FidelityExact, entry.Fidelity)
			require.Equal(t, protocol.EvidenceLocalIntegration, entry.EvidenceLevel)
			complete++
		case protocol.MetadataMissing:
			require.Equal(t, protocol.FidelityPartial, entry.Fidelity)
			require.Equal(t, protocol.EvidenceModelProof, entry.EvidenceLevel)
			incomplete++
		default:
			require.FailNow(t, "unexpected evidence metadata status", entry.EvidenceStatus)
		}
	}
	require.Equal(t, 22, complete)
	require.Zero(t, incomplete)
}

func TestExportCoverageDenominatorRunsLeanAndDefinesEveryTarget(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportCoverageDenominator("../../model", coverageSpec, &output))

	denominator, err := protocol.DecodeCoverageDenominator(output.Bytes())
	require.NoError(t, err)
	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)
	targetProperties := 0
	for _, target := range catalog.Targets {
		targetProperties += len(target.Properties)
	}
	require.Len(t, denominator.Targets, targetProperties)
	for _, target := range denominator.Targets {
		require.Equal(t, protocol.CoverageDenominatorDefined, target.Status)
		require.NotEmpty(t, target.Points)
	}
	require.Len(t, denominator.Targets[0].Edges, 17)
}

func TestExportFamilyDependenciesSelectsOwnedCheckersAndLeanTests(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportFamilyDependencies("../../model", &output))
	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)
	graph, err := protocol.DecodeFamilyDependencyGraph(output.Bytes(), catalog)
	require.NoError(t, err)

	nexus, found := graph.Family(protocol.TargetIDNexusCancellation)
	require.True(t, found)
	require.Equal(t, []string{"exact", "native", "veil"}, nexus.Checkers)
	require.NotEmpty(t, nexus.BuildModules)
	require.NotEmpty(t, nexus.LeanTests)
}

func jsonPath(t *testing.T, encoded []byte, path ...string) string {
	t.Helper()
	var value any
	require.NoError(t, json.Unmarshal(encoded, &value))
	for _, field := range path {
		object, ok := value.(map[string]any)
		require.True(t, ok)
		value = object[field]
	}
	result, err := json.Marshal(value)
	require.NoError(t, err)
	return string(result)
}

func dependencyPaths(dependencies []protocol.SourceDependency) []string {
	paths := make([]string, len(dependencies))
	for index, dependency := range dependencies {
		paths[index] = dependency.Path
	}
	return paths
}
