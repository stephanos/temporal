package testpilot_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type proofDriver struct {
	identity testpilot.DriverIdentity
	openErr  error
	closeErr error
	opened   int
	closed   int
}

func (d *proofDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
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
	require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_FAILED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
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
	require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Equal(t, 1, driver.opened)
	require.Equal(t, 1, driver.closed)
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

func proofFixture(t testing.TB) (*testpilotpb.Case, testpilot.ProfileSpec) {
	t.Helper()
	catalog, err := testpilot.NewCatalog(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	programLimits := &testpilotpb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	contractLimits := &testpilotpb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	source := &testpilotpb.Case{
		Version: &testpilotpb.FormatVersion{Major: 1},
		CaseId:  "case",
		Program: &testpilotpb.Program{
			ProgramId: "program",
			Entrypoints: []*testpilotpb.EntrypointDefinition{{
				EntrypointId: "controller",
				Activation:   &testpilotpb.EntrypointDefinition_Controller{Controller: &testpilotpb.ControllerActivation{}},
			}},
			Cleanup: &testpilotpb.CleanupDefinition{EntrypointId: "cleanup"},
			Limits:  programLimits,
		},
		Contract: &testpilotpb.Contract{
			ContractId: "contract",
			Limits:     proto.CloneOf(contractLimits),
			Rules: []*testpilotpb.ContractRuleDefinition{{
				RuleId:         "safety",
				Kind:           testpilotpb.CONTRACT_RULE_KIND_SAFETY,
				InitialStateId: "start",
				States: []*testpilotpb.ContractStateDefinition{
					{StateId: "start", Status: testpilotpb.CONTRACT_STATE_STATUS_NONTERMINAL},
					{StateId: "good", Status: testpilotpb.CONTRACT_STATE_STATUS_SATISFIED},
				},
				Transitions: []*testpilotpb.ContractTransitionDefinition{{
					TransitionId:  "complete",
					SourceStateId: "start",
					TargetStateId: "good",
					EventFilter:   &testpilotpb.RunEventFilter{Kinds: []testpilotpb.RunEventKind{testpilotpb.RUN_EVENT_KIND_RUN_CLOSED}},
					Predicate:     &testpilotpb.ContractExpression{Expression: &testpilotpb.ContractExpression_Literal{Literal: &testpilotpb.Value{Value: &testpilotpb.Value_BoolValue{BoolValue: true}}}},
					SupportKind:   testpilotpb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
				}},
			}},
		},
	}
	return source, testpilot.ProfileSpec{Identity: "proof", Catalog: catalog, ProgramLimits: proto.CloneOf(programLimits), ContractLimits: contractLimits}
}
