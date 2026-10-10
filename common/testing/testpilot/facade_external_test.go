package testpilot_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestExternalDriverCleanupFailurePreservesVerdict(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)

	driver := &facadetest.Driver{DriverIdentity: prepared.Identity(), OnOpen: func(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
		return &testsupport.Session{OnClose: func(context.Context) error { return errors.New("cleanup unavailable") }}, nil
	}}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_FAILED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Equal(t, 1, driver.Closed())
}

func TestExternalDriverExecutesBoundedCase(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)

	driver := &facadetest.Driver{DriverIdentity: prepared.Identity()}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Equal(t, 1, driver.Validated())
	require.Len(t, driver.RunIDs(), 1)
	require.Equal(t, 1, driver.Closed())
}

func TestExternalDriverReceivesCopiedPreparedRoles(t *testing.T) {
	source, profile := proofFixture(t)
	source.Program.Roles = []*testpilotspb.Role{
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
	driver := &facadetest.Driver{DriverIdentity: prepared.Identity()}

	_, _, err = prepared.Run(t.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, []testpilot.PreparedRole{
		{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingID: "namespace", Namespace: "namespace-a", ResourceBindingID: "queue", Resource: "queue-a"},
		{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingID: "namespace", Namespace: "namespace-a"},
	}, driver.Program().Roles())
	roles := driver.Program().Roles()
	roles[0].Namespace = "changed"
	require.Equal(t, "namespace-a", driver.Program().Roles()[0].Namespace)
}

func TestExternalDriverFailureDoesNotRequireCleanup(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)

	driverErr := errors.New("driver unavailable")
	driver := &facadetest.Driver{DriverIdentity: prepared.Identity(), OnOpen: func(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
		return nil, driverErr
	}}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.ErrorIs(t, err, driverErr)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Len(t, driver.RunIDs(), 1)
	require.Zero(t, driver.Closed())
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
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	source := &testpilotspb.Case{
		Version: &testpilotspb.FormatVersion{Major: casefile.CurrentMajor, Minor: casefile.CurrentMinor},
		CaseId:  "case",
		Program: &testpilotspb.Program{
			ProgramId: "program",
			Entrypoints: []*testpilotspb.Entrypoint{{
				EntrypointId: "controller",
				Activation:   &testpilotspb.Entrypoint_Controller{Controller: &emptypb.Empty{}},
			}},
			Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"},
		},
		Contract: &testpilotspb.Contract{
			ContractId: "contract",
			Rules:      []*testpilotspb.ContractRule{{RuleId: "safety", InitialStateId: "start", States: []*testpilotspb.ContractState{{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING}, {StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}}, Transitions: []*testpilotspb.ContractTransition{{TransitionId: "complete", SourceStateId: "start", TargetStateId: "good", EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}}, Predicate: cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}}), SupportsEvent: proto.Bool(true)}}}},
		},
	}
	return source, testpilot.ProfileSpec{Identity: "proof", Catalog: catalog, ProgramLimits: testsupport.ProgramLimits(), ContractLimits: contractLimits}
}
