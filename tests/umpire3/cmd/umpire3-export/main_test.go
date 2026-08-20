package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
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

func TestExportGoIdentifiersUsesCatalogVocabulary(t *testing.T) {
	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)

	var output bytes.Buffer
	require.NoError(t, exportGoIdentifiers(catalog, &output))
	require.Contains(t, output.String(), "ActionKindRequestCancellation")
	require.Contains(t, output.String(), `ActionKind = "request-cancellation"`)
	require.Contains(t, output.String(), `PropertyIDNexusCancellationWonExcludesSuccess`)
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
	require.Equal(t, "false", jsonPath(t, output.Bytes(), "additionalProperties"))
}

func TestExportMonitorCatalogRunsLeanAndMatchesSemanticCatalog(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportMonitorCatalog("../../model", monitorSpec, &output))

	catalog, err := protocol.DecodeMonitorCatalog(output.Bytes())
	require.NoError(t, err)
	require.Len(t, catalog.Programs, 15)
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

func TestExportCompositionRunsLeanAndReportsObligations(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportComposition("../../model", compositionSpec, &output))

	composition, err := protocol.DecodeComposition(output.Bytes())
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassMetadataValidated, composition.ResultClass)
	require.Len(t, composition.Targets, 15)
	require.Empty(t, composition.MissingMetadata())
}

func TestExportParityLedgerRunsLeanAndCoversInventory(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, exportParityLedger("../../model", paritySpec, &output))

	ledger, err := protocol.DecodeParityLedger(output.Bytes())
	require.NoError(t, err)
	require.Len(t, ledger.Entries, 20)
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
	require.Equal(t, 16, complete)
	require.Equal(t, 4, incomplete)
}

func TestExportCoverageDenominatorRunsLeanAndCoversNexusLifecycle(t *testing.T) {
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
	defined := 0
	for _, target := range denominator.Targets {
		if target.Status == protocol.CoverageDenominatorDefined {
			defined++
		}
	}
	require.Equal(t, 1, defined)
	require.Len(t, denominator.Targets[0].Edges, 17)
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
