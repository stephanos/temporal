package testpilot

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
)

func TestLeanAuthoringProtoJSONStrictDecode(t *testing.T) {
	path := os.Getenv("TESTPILOT_LEAN_AUTHORING_CASE")
	if path == "" {
		t.Skip("set TESTPILOT_LEAN_AUTHORING_CASE to the rendered Lean fixture")
	}

	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	require.Equal(t, "case", decoded.GetCaseId())
	require.Equal(t, []byte{0, 255, 128}, decoded.GetProvenance().GetProducerData())
	require.Equal(t, int64(9223372036854775807), decoded.GetProgram().GetLimits().GetMaxAttempts())

	programAll := decoded.GetProgram().GetEntrypoints()[0].GetInstructions()[0].GetInstruction().
		GetFinish().GetResult().GetAll()
	require.NotNil(t, programAll)
	require.NotNil(t, programAll.GetOperands()[0].GetPresent().GetOperand().GetPath().GetSource().GetRun())

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
