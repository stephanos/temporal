package testpilot

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

func TestLeanAuthoringProtoJSONStrictDecode(t *testing.T) {
	// A Program declares no environment bindings: preparation derives them from its roles and
	// references.
	require.Nil(t, (&testpilotspb.Program{}).ProtoReflect().Descriptor().Fields().ByName("environment"))
	roleFields := (&testpilotspb.Role{}).ProtoReflect().Descriptor().Fields()
	require.EqualValues(t, 3, roleFields.ByName("namespace_binding_id").Number())
	require.EqualValues(t, 4, roleFields.ByName("resource_binding_id").Number())
	referenceEnvironment := (&testpilotspb.Reference{}).ProtoReflect().Descriptor().
		Fields().ByName("environment_binding_id")
	require.NotNil(t, referenceEnvironment)
	require.EqualValues(t, 4, referenceEnvironment.Number())

	path := os.Getenv("TESTPILOT_LEAN_AUTHORING_CASE")
	if path == "" {
		t.Skip("set TESTPILOT_LEAN_AUTHORING_CASE to the rendered Lean fixture")
	}

	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	require.Equal(t, "binding-case", decoded.GetCaseId())
	require.Equal(t, int32(0), decoded.GetVersion().GetMinor())
	location := &testpilotspb.SourceLocation{Path: "Testpilot/Tests/ProtoJSON.lean", Line: 2147483647, Provenance: "authored"}
	require.True(t, proto.Equal(&testpilotspb.CaseProvenance{
		ProducerId: "testpilot-tests", ProducerVersion: "1",
		Definitions: []*testpilotspb.DefinitionBinding{{DefinitionId: "test.property", BehaviorFingerprint: "test-property/v1", Kind: testpilotspb.DEFINITION_KIND_PROPERTY}},
		Sources:     []*testpilotspb.SourceLocation{location},
		KnownGaps: []*testpilotspb.KnownGap{
			{Kind: testpilotspb.KNOWN_GAP_KIND_INPUT, Code: "test.gap.input", DetailPresence: &testpilotspb.KnownGap_Detail{Detail: "needs input"}},
			{Kind: testpilotspb.KNOWN_GAP_KIND_CLAIM, Code: "test.gap.claim", SubjectPresence: &testpilotspb.KnownGap_Subject{Subject: "test.property"}, DetailPresence: &testpilotspb.KnownGap_Detail{}},
		},
		CorrelatedRules: []*testpilotspb.CorrelatedRuleBinding{{
			RuleId: "test.rule", PropertyId: "test.property", PropertyFingerprint: "test-property/v1",
			ProjectionId: "test.projection", ProjectionFingerprint: "test-projection/v1", Source: location,
		}},
	}, decoded.GetProvenance()), "%v", decoded.GetProvenance())
	require.Equal(t, int64(9223372036854775807), decoded.GetProgram().GetEntrypoints()[0].GetInstructions()[0].GetLimits().GetMaxAttempts())

	program := decoded.GetProgram()
	require.Equal(t, []string{"nexus.endpoint", "namespace", "task.queue", ""}, testpilot.EnvironmentBindingIDs(program))
	require.Equal(t, "nexus.endpoint", program.GetRoles()[0].GetResourceBindingId())
	require.Equal(t, "namespace", program.GetRoles()[1].GetNamespaceBindingId())
	require.Equal(t, "task.queue", program.GetRoles()[2].GetResourceBindingId())
	environment := program.GetEntrypoints()[0].GetInstructions()[0].GetInstruction().
		GetFinish().GetResult().GetReference().GetReference()
	require.IsType(t, &testpilotspb.Reference_EnvironmentBindingId{}, environment)
	require.Empty(t, environment.(*testpilotspb.Reference_EnvironmentBindingId).EnvironmentBindingId)

	rule := decoded.GetContract().GetRules()[0]
	require.Equal(t, int64(9223372036854775807), rule.GetDeadline().GetElapsedMilliseconds())
	contractAny := rule.GetTransitions()[0].GetPredicate().GetAny()
	require.NotNil(t, contractAny)
	require.Equal(t, "run", contractAny.GetOperands()[1].GetCompare().GetRight().GetLiteral().GetTextValue())
	require.Equal(t,
		"type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion",
		contractAny.GetOperands()[2].GetCompare().GetRight().GetLiteral().GetMessageValue().GetTypeUrl(),
	)
}
