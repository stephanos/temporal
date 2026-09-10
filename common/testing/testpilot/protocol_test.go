package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestProtocolEncodesExpressionAndStateScopes(t *testing.T) {
	t.Parallel()
	programExpression := messageDescriptor(t, "ProgramExpression")
	require.NotNil(t, programExpression.Oneofs().ByName("expression"))
	require.Nil(t, programExpression.Fields().ByName("observation"))
	require.Nil(t, programExpression.Fields().ByName("capture"))
	require.Nil(t, programExpression.Fields().ByName("run_event"))
	contractExpression := messageDescriptor(t, "ContractExpression")
	require.NotNil(t, contractExpression.Oneofs().ByName("expression"))
	require.Nil(t, contractExpression.Fields().ByName("slot"))
	require.Nil(t, contractExpression.Fields().ByName("outcome"))
	slot := messageDescriptor(t, "SlotDefinition")
	require.NotNil(t, slot.Oneofs().ByName("content"))
	require.Nil(t, slot.Fields().ByName("kind"))
	entrypoint := messageDescriptor(t, "EntrypointDefinition")
	require.NotNil(t, entrypoint.Oneofs().ByName("activation"))
	require.Nil(t, entrypoint.Fields().ByName("context"))
	diagnostic := messageDescriptor(t, "RunDiagnostic")
	require.True(t, diagnostic.Fields().ByName("supporting_event_sequence").HasPresence())
	run := messageDescriptor(t, "Run")
	require.True(t, run.Fields().ByName("evaluation_failure_sequence").HasPresence())
}

func TestProtocolUsesCohesivePublicVocabulary(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"temporal/server/api/testpilot/v1/case.proto",
		"temporal/server/api/testpilot/v1/value.proto",
		"temporal/server/api/testpilot/v1/expression.proto",
		"temporal/server/api/testpilot/v1/instruction.proto",
		"temporal/server/api/testpilot/v1/program.proto",
		"temporal/server/api/testpilot/v1/run.proto",
		"temporal/server/api/testpilot/v1/contract.proto",
	} {
		_, err := protoregistry.GlobalFiles.FindFileByPath(path)
		require.NoError(t, err, path)
	}
	for _, retired := range []string{
		"ValueExpression", "Case" + "Metadata", "CaseDefinition" + "Binding", "CaseDefinition" + "Kind",
		"CaseKnown" + "Gap", "CaseKnownGapK" + "ind", "OptionalString", "SlotSchema", "SlotKind",
		"ActivationBinding", "Entrypoint" + "Context", "RunEventKinds", "RunEventSequence",
		"RunDisposition", "RuleVerdictKind", "VerdictKind",
		"ContractHorizon" + "Definition",
		"Scoped" + "Binding", "Scoped" + "CaptureDeclaration", "Scoped" + "CaptureRef", "Scoped" + "Clause",
		"Scoped" + "Clock", "Scoped" + "Comparison", "Scoped" + "ComparisonOperator", "Scoped" + "Contract",
		"Scoped" + "Correlation", "Scoped" + "CorrelationGroup", "Scoped" + "Endpoint", "Scoped" + "Evidence",
		"Scoped" + "EvidenceBinding", "Scoped" + "EvidenceField", "Scoped" + "EvidenceMeaning",
		"Scoped" + "EvidenceProjection", "Scoped" + "EvidenceRule", "Scoped" + "FieldDisposition",
		"Scoped" + "FieldPolicy", "Scoped" + "Identity", "Scoped" + "Limits", "Scoped" + "Operand",
		"Scoped" + "Predicate", "Scoped" + "PredicateField", "Scoped" + "ProjectionRule", "Scoped" + "Transition",
		"Scoped" + "Value",
	} {
		_, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName("temporal.server.api.testpilot.v1." + retired))
		require.Error(t, err, retired)
	}
}

func messageDescriptor(t *testing.T, name protoreflect.Name) protoreflect.MessageDescriptor {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("temporal.server.api.testpilot.v1." + protoreflect.FullName(name))
	require.NoError(t, err)
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	require.True(t, ok)
	return message
}
