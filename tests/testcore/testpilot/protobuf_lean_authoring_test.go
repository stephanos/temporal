package testpilot

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

func TestLeanAuthoringProtoJSONStrictDecode(t *testing.T) {
	programEnvironment := (&testpilotspb.Program{}).ProtoReflect().Descriptor().Fields().ByName("environment")
	require.NotNil(t, programEnvironment)
	require.EqualValues(t, 8, programEnvironment.Number())
	roleFields := (&testpilotspb.RoleDefinition{}).ProtoReflect().Descriptor().Fields()
	require.EqualValues(t, 3, roleFields.ByName("namespace_binding_id").Number())
	require.EqualValues(t, 4, roleFields.ByName("resource_binding_id").Number())
	programExpressionEnvironment := (&testpilotspb.ProgramExpression{}).ProtoReflect().Descriptor().
		Fields().ByName("environment")
	require.NotNil(t, programExpressionEnvironment)
	require.EqualValues(t, 12, programExpressionEnvironment.Number())
	require.Nil(t, (&testpilotspb.ContractExpression{}).ProtoReflect().Descriptor().
		Fields().ByName("environment"))

	path := os.Getenv("TESTPILOT_LEAN_AUTHORING_CASE")
	if path == "" {
		t.Skip("set TESTPILOT_LEAN_AUTHORING_CASE to the rendered Lean fixture")
	}

	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	require.Equal(t, "binding-case", decoded.GetCaseId())
	require.Equal(t, int32(1), decoded.GetVersion().GetMinor())
	require.Equal(t, []byte{0, 255, 128}, decoded.GetProvenance().GetProducerData())
	require.Equal(t, int64(9223372036854775807), decoded.GetProgram().GetLimits().GetMaxAttempts())

	program := decoded.GetProgram()
	require.Len(t, program.GetEnvironment(), 3)
	require.Equal(t, []string{"namespace", "task.queue", "nexus.endpoint"}, []string{
		program.GetEnvironment()[0].GetBindingId(),
		program.GetEnvironment()[1].GetBindingId(),
		program.GetEnvironment()[2].GetBindingId(),
	})
	require.Equal(t, "nexus.endpoint", program.GetRoles()[0].GetResourceBindingId())
	require.Equal(t, "namespace", program.GetRoles()[1].GetNamespaceBindingId())
	require.Equal(t, "task.queue", program.GetRoles()[2].GetResourceBindingId())
	environment := program.GetEntrypoints()[0].GetInstructions()[0].GetInstruction().
		GetFinish().GetResult().GetEnvironment()
	require.NotNil(t, environment)
	require.Empty(t, environment.GetBindingId())

	rule := decoded.GetContract().GetRules()[0]
	require.Equal(t, int64(9223372036854775807), rule.GetHorizon().GetElapsedMilliseconds())
	contractAny := rule.GetTransitions()[0].GetPredicate().GetAny()
	require.NotNil(t, contractAny)
	require.Equal(t, "run", contractAny.GetOperands()[1].GetEquals().GetRight().GetLiteral().GetText())
	require.Equal(t,
		"type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion",
		contractAny.GetOperands()[2].GetEquals().GetRight().GetLiteral().GetMessageValue().GetTypeUrl(),
	)
}
