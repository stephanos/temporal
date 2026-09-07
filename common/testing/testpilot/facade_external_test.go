package testpilot_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type proofDriver struct {
	identity  testpilot.DriverIdentity
	openErr   error
	closeErr  error
	program   testpilot.PreparedProgram
	validated int
	opened    int
	closed    int
}

func (d *proofDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}

func (d *proofDriver) Validate(_ context.Context, program testpilot.PreparedProgram) error {
	d.program = program
	d.validated++
	return nil
}

func (d *proofDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	d.opened++
	if d.openErr != nil {
		return nil, d.openErr
	}
	return &proofSession{driver: d}, nil
}

type proofSession struct {
	testpilot.Session
	driver *proofDriver
}

func (s *proofSession) Close(context.Context) error {
	s.driver.closed++
	return s.driver.closeErr
}

func TestExternalDriverCleanupFailurePreservesVerdict(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)

	driver := &proofDriver{identity: prepared.Identity(), closeErr: errors.New("cleanup unavailable")}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_FAILED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Equal(t, 1, driver.closed)
}

var _ testpilot.Driver = (*proofDriver)(nil)

func TestExternalDriverExecutesBoundedCase(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)

	driver := &proofDriver{identity: prepared.Identity()}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Equal(t, 1, driver.validated)
	require.Equal(t, 1, driver.opened)
	require.Equal(t, 1, driver.closed)
}

func TestExternalDriverReceivesCopiedPreparedRoles(t *testing.T) {
	source, profile := proofFixture(t)
	source.Version.Minor = 1
	source.Program.Environment = []*testpilotspb.EnvironmentDefinition{{BindingId: "namespace"}, {BindingId: "queue"}}
	source.Program.Roles = []*testpilotspb.RoleDefinition{
		{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
		{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"},
	}
	profile.Roles = []testpilot.RolePolicy{
		{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
		{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
	}
	profile.EnvironmentBindings = []testpilot.EnvironmentBinding{{ID: "namespace", Value: "namespace-a"}, {ID: "queue", Value: "queue-a"}}
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	driver := &proofDriver{identity: prepared.Identity()}

	_, _, err = prepared.Run(t.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, []testpilot.PreparedRole{
		{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingID: "namespace", Namespace: "namespace-a", ResourceBindingID: "queue", Resource: "queue-a"},
		{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingID: "namespace", Namespace: "namespace-a"},
	}, driver.program.Roles())
	roles := driver.program.Roles()
	roles[0].Namespace = "changed"
	require.Equal(t, "namespace-a", driver.program.Roles()[0].Namespace)
}

func TestExternalDriverFailureDoesNotRequireCleanup(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)

	driverErr := errors.New("driver unavailable")
	driver := &proofDriver{identity: prepared.Identity(), openErr: driverErr}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.ErrorIs(t, err, driverErr)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Equal(t, 1, driver.opened)
	require.Zero(t, driver.closed)
}

func TestPublicPackageDependencyBoundary(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "list", "-tags", "test_dep", "-deps", "go.temporal.io/server/common/testing/testpilot")
	output, err := command.Output()
	require.NoError(t, err)
	for _, dependency := range strings.Fields(string(output)) {
		for _, prefix := range []string{"go.temporal.io/server/tools/umpire", "go.temporal.io/sdk", "go.temporal.io/server/tests/testcore"} {
			require.False(t, dependency == prefix || strings.HasPrefix(dependency, prefix+"/"), "forbidden dependency: %s", dependency)
		}
	}
}

func proofFixture(t testing.TB) (*testpilotspb.Case, testpilot.ProfileSpec) {
	t.Helper()
	catalog, err := testpilot.NewCatalog(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	programLimits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	source := &testpilotspb.Case{
		Version: &testpilotspb.FormatVersion{Major: 1},
		CaseId:  "case",
		Program: &testpilotspb.Program{
			ProgramId: "program",
			Entrypoints: []*testpilotspb.EntrypointDefinition{{
				EntrypointId: "controller",
				Activation:   &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}},
			}},
			Cleanup: &testpilotspb.CleanupDefinition{EntrypointId: "cleanup"},
			Limits:  programLimits,
		},
		Contract: &testpilotspb.Contract{
			ContractId: "contract",
			Limits:     proto.CloneOf(contractLimits),
			Rules: []*testpilotspb.ContractRuleDefinition{{
				RuleId:         "safety",
				Kind:           testpilotspb.CONTRACT_RULE_KIND_SAFETY,
				InitialStateId: "start",
				States: []*testpilotspb.ContractStateDefinition{
					{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_NONTERMINAL},
					{StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
				},
				Transitions: []*testpilotspb.ContractTransitionDefinition{{
					TransitionId:  "complete",
					SourceStateId: "start",
					TargetStateId: "good",
					EventFilter:   &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}},
					Predicate:     &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}},
					SupportKind:   testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
				}},
			}},
		},
	}
	return source, testpilot.ProfileSpec{Identity: "proof", Catalog: catalog, ProgramLimits: proto.CloneOf(programLimits), ContractLimits: contractLimits}
}
