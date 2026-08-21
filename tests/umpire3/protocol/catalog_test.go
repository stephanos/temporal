package protocol

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultCatalogOwnsExperimentVocabulary(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	require.Equal(t, CatalogFormatVersion, catalog.FormatVersion)
	require.Equal(t, "4.28.0", catalog.LeanVersion)

	action, ok := catalog.Action("schedule-operation")
	require.True(t, ok)
	require.Equal(t, []CapabilityID{"nexus"}, action.RequiredCapabilities)
	require.True(t, catalog.HasCapability("nexus"))

	firstDigest, err := catalog.Digest()
	require.NoError(t, err)
	catalog.Actions[0].Identifier = "mutated"

	pristine, err := DefaultCatalog()
	require.NoError(t, err)
	secondDigest, err := pristine.Digest()
	require.NoError(t, err)
	require.Equal(t, firstDigest, secondDigest)
}

func TestDefaultCatalogOwnsCloseNexusRuntimeFootprint(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	action, ok := catalog.Action("close-nexus-operation")
	require.True(t, ok)
	require.Contains(t, action.Footprint, FootprintDeclaration{
		Protocol: "http", Service: "nexus", Route: "/service/operation",
	})
	require.Contains(t, action.Footprint, FootprintDeclaration{
		Protocol: "grpc", Service: "history", Route: "UpdateWorkflowExecution",
	})
}

func TestDecodeCatalogRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	encoded, err := json.Marshal(catalog)
	require.NoError(t, err)

	var object map[string]any
	require.NoError(t, json.Unmarshal(encoded, &object))
	object["unexpected"] = true
	encoded, err = json.Marshal(object)
	require.NoError(t, err)

	_, err = DecodeCatalog(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.ErrorContains(t, err, "unknown field")

	canonical, err := catalog.CanonicalJSON()
	require.NoError(t, err)
	_, err = DecodeCatalog(bytes.NewReader(append(canonical, []byte(` {}`)...)), DefaultDecodeLimit)
	require.ErrorContains(t, err, "multiple JSON values")
}

func TestCatalogRejectsDuplicateAndDanglingDeclarations(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	catalog.Actions = append(catalog.Actions, catalog.Actions[0])
	require.ErrorContains(t, catalog.Validate(), "duplicate action")

	catalog, err = DefaultCatalog()
	require.NoError(t, err)
	catalog.Actions[0].RequiredCapabilities = append(
		catalog.Actions[0].RequiredCapabilities,
		CapabilityID("missing-capability"),
	)
	require.ErrorContains(t, catalog.Validate(), "unknown capability")
}

func TestCatalogCanonicalEncodingAndDigestAreStable(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	first, err := catalog.CanonicalJSON()
	require.NoError(t, err)
	second, err := catalog.CanonicalJSON()
	require.NoError(t, err)
	require.Equal(t, first, second)

	firstDigest, err := catalog.Digest()
	require.NoError(t, err)
	catalog.Properties[0].Description += " changed"
	secondDigest, err := catalog.Digest()
	require.NoError(t, err)
	require.NotEqual(t, firstDigest, secondDigest)
}

func TestCatalogPropertiesAreBoundToResolvedLeanTheorems(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	for _, property := range catalog.Properties {
		require.NotEmpty(t, property.Theorem, property.Identifier)
		require.NotEmpty(t, property.Statement, property.Identifier)
		require.NotEqual(t, "derived", property.StatementHash, property.Identifier)
		require.NotContains(t, property.Axioms, "sorryAx", property.Identifier)
	}

	mutated := catalog
	mutated.Properties = append([]PropertyDeclaration(nil), catalog.Properties...)
	mutated.Properties[0].Statement += " changed"
	require.ErrorContains(t, mutated.Validate(), "derived statement hash")
}

func TestCatalogRejectsForbiddenPropertyAxiom(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	catalog.Properties = append([]PropertyDeclaration(nil), catalog.Properties...)
	catalog.Properties[0].Axioms = []string{"Lean.ofReduceBool"}
	catalog.Properties[0].TrustBadge = TrustBadgeKernelWithDeclaredAxioms

	require.ErrorContains(t, catalog.Validate(), "invalid axiom")
}

func TestTaskAcknowledgementTargetDeclaresIndependentMechanismAndRefinement(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	var target TargetDeclaration
	for _, candidate := range catalog.Targets {
		if candidate.Identifier == "foundation-backlog-ack" {
			target = candidate
			break
		}
	}

	require.Equal(t, []string{
		"Temporal.Product.TaskAck",
		"Temporal.System.TaskAck",
		"Temporal.Refinement.TaskAck",
	}, target.Modules)
}

func TestEveryCatalogTargetDeclaresIndependentSystemAndRefinement(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	var missing []string
	for _, target := range catalog.Targets {
		hasSystem := false
		hasRefinement := false
		for _, module := range target.Modules {
			hasSystem = hasSystem || strings.HasPrefix(module, "Temporal.System.")
			hasRefinement = hasRefinement || strings.HasPrefix(module, "Temporal.Refinement.")
		}
		if !hasSystem || !hasRefinement {
			missing = append(missing, target.Identifier)
		}
	}
	require.Empty(t, missing)
}

func TestDefaultCatalogOwnsNexusProgressFamily(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	var target TargetDeclaration
	for _, candidate := range catalog.Targets {
		if candidate.Identifier == "feature-nexus-progress" {
			target = candidate
			break
		}
	}
	require.Equal(t, []string{
		"Temporal.Product.NexusProgress",
		"Temporal.System.NexusProgress",
		"Temporal.Refinement.NexusProgress",
	}, target.Modules)
	require.Equal(t, []string{"nexus-operation.progress"}, target.Properties)
}
