package experiment

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeNexusExperiment(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)

	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, "nexus-cancellation-stale-completion-v1", experiment.ExperimentID)
	require.Len(t, experiment.Actions, 8)
	require.Len(t, experiment.Checkpoints, 3)
	require.NoError(t, experiment.Validate())
}

func TestExperimentCanonicalEncodingIsStable(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)

	first, err := experiment.CanonicalJSON()
	require.NoError(t, err)
	second, err := experiment.CanonicalJSON()
	require.NoError(t, err)
	require.Equal(t, first, second)

	firstDigest, err := experiment.Digest()
	require.NoError(t, err)
	experiment.Scope.Bounds.MaxDepth++
	secondDigest, err := experiment.Digest()
	require.NoError(t, err)
	require.NotEqual(t, firstDigest, secondDigest)
	experiment.Scope.Assumptions = append(experiment.Scope.Assumptions, Assumption{
		Identifier:    "changed-assumption",
		StatementHash: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	})
	assumptionDigest, err := experiment.Digest()
	require.NoError(t, err)
	require.NotEqual(t, secondDigest, assumptionDigest)
}

func TestDecodeExperimentRejectsUnknownField(t *testing.T) {
	_, err := DecodeExperiment(bytes.NewBufferString(`{
  "formatVersion":"umpire3/v1",
  "experimentID":"unknown-field",
  "unexpected":true
}`), DefaultDecodeLimit)
	require.ErrorContains(t, err, "unknown field")
}

func TestDecodeExperimentRejectsOversizedInput(t *testing.T) {
	_, err := DecodeExperiment(bytes.NewBufferString(`{"formatVersion":"umpire3/v1"}`), 8)
	require.ErrorContains(t, err, "exceeds")
}

func TestExperimentRejectsIncompleteTrace(t *testing.T) {
	experiment := Experiment{FormatVersion: FormatVersion, ExperimentID: "incomplete"}
	require.Error(t, experiment.Validate())
}

func TestExperimentRejectsSensitiveActionData(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	secret := "secret"
	experiment.Actions[0].Arguments = []NamedValue{{
		Name:  "authorizationHeader",
		Value: Value{Type: ValueString, Text: &secret},
	}}
	require.ErrorContains(t, experiment.Validate(), "sensitive")
}

func TestExperimentRejectsUnknownActionVocabulary(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	experiment.Actions[0].Kind = "unknown-action"
	require.ErrorContains(t, experiment.Validate(), "unknown action")
}

func TestExperimentTypedValuesAndBindingsRoundTrip(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)

	reason := "operator requested"
	experiment.Actions[2].Arguments = []NamedValue{{
		Name: "reason",
		Value: Value{
			Type: ValueString,
			Text: &reason,
		},
	}}
	experiment.Actions[0].Bindings = []Binding{{
		Symbol:     "operation",
		Type:       "identity",
		Projection: "operation-id",
	}}
	experiment.Order = []OrderConstraint{{Before: "a1", After: "a2", Relation: OrderSemantic}}

	canonical, err := experiment.CanonicalJSON()
	require.NoError(t, err)
	roundTripped, err := DecodeExperiment(bytes.NewReader(canonical), DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, experiment.Actions[2].Arguments, roundTripped.Actions[2].Arguments)
	require.Equal(t, experiment.Actions[0].Bindings, roundTripped.Actions[0].Bindings)
	require.Equal(t, experiment.Order, roundTripped.Order)
}

func TestExperimentRejectsInvalidTypedValue(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)

	text := "not a boolean"
	experiment.Actions[2].Arguments = []NamedValue{{
		Name:  "invalid",
		Value: Value{Type: ValueBoolean, Text: &text},
	}}
	require.ErrorContains(t, experiment.Validate(), "boolean value")
}

func TestExperimentRejectsOrderCycle(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)

	experiment.Order = []OrderConstraint{
		{Before: "a1", After: "a2", Relation: OrderSemantic},
		{Before: "a2", After: "a1", Relation: OrderSemantic},
	}
	require.ErrorContains(t, experiment.Validate(), "order cycle")
}

func TestExperimentRequiresCurrentCatalogDigest(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)

	experiment.Model.CatalogHash = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	require.ErrorContains(t, experiment.Validate(), "catalog hash")
}

func TestExperimentRejectsUnknownAndDuplicateResources(t *testing.T) {
	experiment := loadNexusExperiment(t)
	experiment.Resources[0].Kind = "unknown-resource"
	require.ErrorContains(t, experiment.Validate(), "unknown resource kind")

	experiment = loadNexusExperiment(t)
	experiment.Resources[1].Identifier = experiment.Resources[0].Identifier
	require.ErrorContains(t, experiment.Validate(), "duplicate resource")
}

func TestExperimentRejectsUnboundAndDuplicateSymbols(t *testing.T) {
	experiment := loadNexusExperiment(t)
	unbound := "missing-operation"
	experiment.Policies[0].Arguments = []NamedValue{{
		Name: "operation", Value: Value{Type: ValueSymbol, Text: &unbound},
	}}
	require.ErrorContains(t, experiment.Validate(), "unbound symbol")

	experiment = loadNexusExperiment(t)
	experiment.Actions[0].Bindings = []Binding{
		{Symbol: "operation", Type: "identity", Projection: "operation-id"},
		{Symbol: "operation", Type: "identity", Projection: "operation-id"},
	}
	require.ErrorContains(t, experiment.Validate(), "duplicate binding symbol")
}

func TestExperimentRejectsUnknownAndMistypedProjection(t *testing.T) {
	experiment := loadNexusExperiment(t)
	experiment.Actions[0].Bindings = []Binding{{Symbol: "operation", Type: "identity", Projection: "missing"}}
	require.ErrorContains(t, experiment.Validate(), "unknown projection")

	experiment = loadNexusExperiment(t)
	experiment.Actions[0].Bindings = []Binding{{Symbol: "operation", Type: "string", Projection: "operation-id"}}
	require.ErrorContains(t, experiment.Validate(), "expected \"identity\"")
}

func TestExperimentRejectsCapabilitiesBeyondCatalogDeclaration(t *testing.T) {
	experiment := loadNexusExperiment(t)
	experiment.Actions[0].RequiredCapabilities = append(
		experiment.Actions[0].RequiredCapabilities,
		"history-observation",
	)
	require.ErrorContains(t, experiment.Validate(), "undeclared capability")
}

func loadNexusExperiment(t *testing.T) Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
