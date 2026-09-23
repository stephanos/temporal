package casebinding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/protobuf/proto"
)

var testEnvironment = testpilotdriver.Environment{
	Identity: "ignored", Namespace: "canary-test", TaskQueue: "canary-test-queue",
	HandlerTaskQueue: "canary-test-queue-handler", NexusEndpoint: "canary-test-endpoint",
}

func committed(t *testing.T) *policy.Policy {
	t.Helper()
	canary, err := policy.Embedded()
	require.NoError(t, err)
	return canary
}

// The pinned Case is the one the policy names, and it prepares under the canary's Profile name
// with the environment's coordinates and the tree's catalog, with no connection.
func TestBindPreparesThePinnedCase(t *testing.T) {
	canary := committed(t)
	identity, err := Identity()
	require.NoError(t, err)
	require.Equal(t, canary.CaseIdentity, identity)

	bound, err := Bind(canary, testEnvironment)
	require.NoError(t, err)
	require.Equal(t, "temporal.case.nexusCallerCanary.syncCompletion", bound.Source.GetCaseId())
	prepared := bound.Prepared.Identity()
	require.Equal(t, canary.CaseProfile, prepared.Profile, "the Profile name is the policy's, whatever the environment names")
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	require.Equal(t, catalog.Identity(), prepared.Catalog)
	require.NotEmpty(t, prepared.Bindings)

	other := testEnvironment
	other.Namespace = "another-namespace"
	elsewhere, err := Bind(canary, other)
	require.NoError(t, err)
	require.NotEqual(t, prepared.Bindings, elsewhere.Prepared.Identity().Bindings, "the coordinates are bound into the identity")
}

// The hand-authored Profile equals what Testpilot would derive for the pinned Case, so drift on
// either side -- a Case that needs more, or a derivation or ceiling that changed -- fails here and
// is reviewed, rather than changing what the production credential may do.
func TestTheHandAuthoredProfileIsTheDerivedOne(t *testing.T) {
	canary := committed(t)
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	authored := ProfileSpec(canary, catalog, testEnvironment)
	environment := testEnvironment
	environment.Identity = canary.CaseProfile
	derived, err := testpilotdriver.DeriveProfile(mustSource(t), catalog, environment)
	require.NoError(t, err)

	require.Equal(t, derived.Identity, authored.Identity)
	require.Equal(t, derived.Roles, authored.Roles)
	require.Equal(t, derived.Opcodes, authored.Opcodes)
	require.Equal(t, derived.CommandTypes, authored.CommandTypes)
	require.Equal(t, derived.EnvironmentBindings, authored.EnvironmentBindings)
	require.Empty(t, derived.Configuration)
	require.Empty(t, authored.Configuration)
	require.True(t, proto.Equal(derived.ProgramLimits, authored.ProgramLimits))
	require.True(t, proto.Equal(derived.ContractLimits, authored.ContractLimits))
	require.True(t, proto.Equal(derived.CorrelatedLimits, authored.CorrelatedLimits))
	require.Equal(t, derived.InstructionDefaults, authored.InstructionDefaults)
}

// The Case asks only for the public surface: two public WorkflowService methods, no fault, no
// system callback, one workflow command, and no Known Gap to close.
func TestThePinnedCaseStaysOnThePublicSurface(t *testing.T) {
	source := mustSource(t)
	require.Empty(t, source.GetProvenance().GetKnownGaps())
	profile := ProfileSpec(committed(t), nil, testEnvironment)
	require.Equal(t, []string{startWorkflowExecution, getWorkflowExecutionHistory}, profile.Roles[0].Methods)
	require.NotContains(t, profile.Opcodes, testpilot.InjectFault)
	require.NotContains(t, profile.Opcodes, testpilot.NexusOperationCompletion)
	require.Equal(t, []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}, profile.CommandTypes)
	require.NotEmpty(t, source.GetContract().GetCorrelated().GetRules(), "the Contract decides from correlated public observations")
}

// A Case other than the policy's refuses before anything is prepared.
func TestBindRefusesAnotherCase(t *testing.T) {
	canary := *committed(t)
	canary.CaseIdentity = "0000000000000000000000000000000000000000000000000000000000000000"
	_, err := Bind(&canary, testEnvironment)
	require.ErrorContains(t, err, "the policy's is")
	_, err = Bind(nil, testEnvironment)
	require.Error(t, err)
	require.True(t, bytes.Equal(Case(), pinned))
	clone := Case()
	clone[0] = ' '
	require.False(t, bytes.Equal(clone, pinned), "Case returns a copy")
}

func mustSource(t *testing.T) *testpilotspb.Case {
	t.Helper()
	source, err := testpilot.DecodeCaseProtoJSON(Case())
	require.NoError(t, err)
	return source
}
