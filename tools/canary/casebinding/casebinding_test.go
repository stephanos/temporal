package casebinding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
// system callback, one workflow command, and no Known Gap to close. The Case itself is read, not
// only the hand-authored Profile, so a Case that names another method fails here.
func TestThePinnedCaseStaysOnThePublicSurface(t *testing.T) {
	source := mustSource(t)
	require.Empty(t, source.GetProvenance().GetKnownGaps())
	var document any
	require.NoError(t, json.Unmarshal(Case(), &document))
	methods := casedMethods(document)
	require.NotEmpty(t, methods)
	for _, method := range methods {
		require.Contains(t, []string{startWorkflowExecution, getWorkflowExecutionHistory}, method)
	}
	require.Equal(t, []string{"history", "source.scheduled"}, source.GetContract().GetCorrelated().GetSources())
	profile := ProfileSpec(committed(t), nil, testEnvironment)
	require.Equal(t, []string{startWorkflowExecution, getWorkflowExecutionHistory}, profile.Roles[0].Methods)
	require.NotContains(t, profile.Opcodes, testpilot.InjectFault)
	require.NotContains(t, profile.Opcodes, testpilot.NexusOperationCompletion)
	require.Equal(t, []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}, profile.CommandTypes)
	require.NotEmpty(t, source.GetContract().GetCorrelated().GetRules(), "the Contract decides from correlated public observations")
}

// A Case other than the policy's refuses before anything is prepared: a policy naming another
// identity, or pinned bytes changed by one byte.
func TestBindRefusesAnotherCase(t *testing.T) {
	canary := *committed(t)
	other := canary
	other.CaseIdentity = "0000000000000000000000000000000000000000000000000000000000000000"
	_, err := Bind(&other, testEnvironment)
	require.ErrorContains(t, err, "the policy's is")
	_, err = Bind(nil, testEnvironment)
	require.Error(t, err)

	changed := Case()
	changed = bytes.Replace(changed, []byte(`"producerVersion":"1"`), []byte(`"producerVersion":"2"`), 1)
	require.False(t, bytes.Equal(changed, pinned))
	_, err = bind(changed, &canary, testEnvironment)
	require.ErrorContains(t, err, "the policy's is", "a changed Case is another identity")
}

func TestCaseReturnsACopy(t *testing.T) {
	require.True(t, bytes.Equal(Case(), pinned))
	clone := Case()
	clone[0] = ' '
	require.False(t, bytes.Equal(clone, pinned))
}

// identityDriver answers only Identity; a Run that reaches Validate or Open has already acquired
// the authority it must not.
type identityDriver struct {
	identity          testpilot.DriverIdentity
	validated, opened bool
}

func (d *identityDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}

func (d *identityDriver) Validate(context.Context, testpilot.PreparedProgram) error {
	d.validated = true
	return nil
}

func (d *identityDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	d.opened = true
	return nil, errors.New("not opened")
}

// The prepared Case refuses a Driver that carries a crossed Profile name, another catalog or other
// bindings before it validates or opens anything, so no authority is acquired for a crossed Case.
func TestThePreparedCaseRefusesACrossedDriverBeforeAuthority(t *testing.T) {
	bound, err := Bind(committed(t), testEnvironment)
	require.NoError(t, err)
	prepared := bound.Prepared.Identity()
	for name, identity := range map[string]testpilot.DriverIdentity{
		"a crossed Profile name": {Profile: testEnvironment.Identity, Catalog: prepared.Catalog, Bindings: prepared.Bindings},
		"another catalog":        {Profile: prepared.Profile, Catalog: "another-catalog", Bindings: prepared.Bindings},
		"other bindings":         {Profile: prepared.Profile, Catalog: prepared.Catalog, Bindings: "other-bindings"},
	} {
		t.Run(name, func(t *testing.T) {
			driver := &identityDriver{identity: identity}
			run, verdict, err := bound.Prepared.Run(t.Context(), driver)
			require.Error(t, err)
			require.Nil(t, run)
			require.Nil(t, verdict)
			require.False(t, driver.validated || driver.opened, "nothing is validated or opened for a crossed Driver")
		})
	}
}

// casedMethods is every "method" value anywhere in a decoded Case.
func casedMethods(node any) []string {
	var methods []string
	switch value := node.(type) {
	case map[string]any:
		for key, child := range value {
			if method, ok := child.(string); ok && key == "method" {
				methods = append(methods, method)
				continue
			}
			methods = append(methods, casedMethods(child)...)
		}
	case []any:
		for _, child := range value {
			methods = append(methods, casedMethods(child)...)
		}
	}
	return methods
}

func mustSource(t *testing.T) *testpilotspb.Case {
	t.Helper()
	source, err := testpilot.DecodeCaseProtoJSON(Case())
	require.NoError(t, err)
	return source
}
