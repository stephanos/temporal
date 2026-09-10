package execution

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func carrierFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	source, catalog, policy := capabilityFixture(t)
	policy.Roles[0].ReservationCarriers = []ReservationCarrierPolicy{{
		Method: "/example.Service/Call",
		Shapes: []ReservationCarrierShape{
			{Kind: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 2},
			{Kind: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 4},
		},
	}}
	return source, catalog, policy
}

func TestPrepareCompilesDeterministicReservationCarrierTopology(t *testing.T) {
	source, catalog, policy := carrierFixture(t)
	controller := source.Program.Entrypoints[0]
	controller.Instructions[0].ActivationReservations[0].Count = 2
	controller.Instructions[0].ActivationReservations[1].Count = 4
	workflow := source.Program.Entrypoints[1]
	workflow.Instructions[0].Guard = &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: false}}}}
	secondStart := proto.CloneOf(workflow.Instructions[0])
	secondStart.InstructionId = "start_second"
	secondStart.Guard = nil
	workflow.Instructions = append(workflow.Instructions, secondStart)
	secondController := proto.CloneOf(controller.Instructions[0])
	secondController.InstructionId = "call_second"
	secondController.ActivationReservations[0].Count = 1
	secondController.ActivationReservations[1].Count = 2
	controller.Instructions = append(controller.Instructions, secondController)

	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	isolated, err := Prepare(proto.CloneOf(source), catalog, policy)
	require.NoError(t, err)
	want := ReservationCarrierPlan{
		EndpointRoleID: "endpoint",
		Method:         "/example.Service/Call",
		Reservations: []ReservationTopology{
			{EntrypointID: "workflow", Kind: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, Count: 2},
			{EntrypointID: "handler", Kind: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, Count: 4},
		},
		Routes: []ReservationRoute{
			{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 0, SourceInstructionID: "start", HandlerEntrypointID: "handler", HandlerOrdinal: 0},
			{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 1, SourceInstructionID: "start", HandlerEntrypointID: "handler", HandlerOrdinal: 1},
			{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 0, SourceInstructionID: "start_second", HandlerEntrypointID: "handler", HandlerOrdinal: 2},
			{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 1, SourceInstructionID: "start_second", HandlerEntrypointID: "handler", HandlerOrdinal: 3},
		},
	}
	plan, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, want, plan)
	second, ok := prepared.ReservationCarrier("controller", "call_second")
	require.True(t, ok)
	require.Len(t, second.Routes, 2)
	other, ok := isolated.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, want, other)

	plan.Routes[0].HandlerOrdinal = 99
	plan.Reservations[0].Count = 99
	policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 1
	source.Program.Entrypoints[1].Instructions[0].Instruction.GetStartNexusOperation().Operation = "changed"
	again, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, want, again)
}

func TestPrepareExposesWorkflowOnlyCarrierReservations(t *testing.T) {
	source, catalog, policy := fixture(t)
	addWorker(source, &policy)
	source.Program.Entrypoints[0].Instructions[0].ActivationReservations = []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	plan, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, []ReservationTopology{{EntrypointID: "workflow", Kind: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, Count: 1}}, plan.Reservations)
	require.Empty(t, plan.Routes)
}

func TestCompileCarrierTopologyChargesAdmissionWork(t *testing.T) {
	source, catalog, policy := carrierFixture(t)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	graphIndex := make(map[string]*graph, len(prepared.graphs))
	for _, graph := range prepared.graphs {
		graphIndex[graph.id] = graph
	}
	a := admission{prepared: prepared, graphIndex: graphIndex, work: ir.DefaultLimits().Work - 5}

	_, err = a.compileCarrierTopology(prepared.graphs[0], prepared.graphs[0].nodes[0])
	var admissionError *ir.Error
	require.ErrorAs(t, err, &admissionError)
	require.Equal(t, ir.LimitExceeded, admissionError.Category)
	require.Equal(t, "admission work ceiling exceeded", admissionError.Detail)
}

func TestPrepareRejectsReservationCarrierPolicyErrors(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case, *Profile){
		"method outside ordinary authorization": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Method = "/example.Service/Other"
		},
		"streaming method": func(source *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].Methods = append(policy.Roles[0].Methods, "/example.Service/Stream")
			policy.Roles[0].ReservationCarriers[0].Method = "/example.Service/Stream"
			source.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method = "/example.Service/Stream"
		},
		"duplicate method": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers = append(policy.Roles[0].ReservationCarriers, policy.Roles[0].ReservationCarriers[0])
		},
		"duplicate context": func(_ *testpilotspb.Case, policy *Profile) {
			shape := policy.Roles[0].ReservationCarriers[0].Shapes[0]
			policy.Roles[0].ReservationCarriers[0].Shapes = []ReservationCarrierShape{shape, shape}
		},
		"unsupported context": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes[0].Kind = testpilotspb.ENTRYPOINT_KIND_ACTIVITY
		},
		"zero cardinality": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 0
		},
		"cardinality overflow": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = math.MaxInt64
		},
		"aggregate cardinality": func(_ *testpilotspb.Case, policy *Profile) {
			for i := range policy.Roles[0].ReservationCarriers[0].Shapes {
				policy.Roles[0].ReservationCarriers[0].Shapes[i].MaximumCount = policy.Limits.MaxActivations
			}
		},
		"oversized carrier policy": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers = make([]ReservationCarrierPolicy, 10001)
		},
		"carrier on non-endpoint": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[1].ReservationCarriers = policy.Roles[0].ReservationCarriers
			policy.Roles[0].ReservationCarriers = nil
		},
		"missing carrier authority": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers = nil
		},
		"unsupported reservation shape": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes = policy.Roles[0].ReservationCarriers[0].Shapes[:1]
		},
		"reservation cardinality": func(source *testpilotspb.Case, _ *Profile) {
			source.Program.Entrypoints[0].Instructions[0].ActivationReservations[0].Count = 3
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := carrierFixture(t)
			mutate(source, &policy)
			_, err := Prepare(source, catalog, policy)
			require.Error(t, err)
		})
	}
}

func TestPrepareRejectsInvalidReservationCarrierTopology(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case){
		"missing handler reservation": func(source *testpilotspb.Case) {
			source.Program.Entrypoints[0].Instructions[0].ActivationReservations = source.Program.Entrypoints[0].Instructions[0].ActivationReservations[:1]
		},
		"ambiguous handler": func(source *testpilotspb.Case) {
			handler := proto.CloneOf(source.Program.Entrypoints[2])
			handler.EntrypointId = "handler_second"
			source.Program.Entrypoints = append(source.Program.Entrypoints, handler)
			source.Program.Entrypoints[0].Instructions[0].ActivationReservations = append(source.Program.Entrypoints[0].Instructions[0].ActivationReservations, &testpilotspb.ActivationReservationDefinition{EntrypointId: "handler_second", Count: 1})
		},
		"crossed handler": func(source *testpilotspb.Case) {
			source.Program.Entrypoints[2].GetNexusHandler().Operation = "other"
		},
		"handler count mismatch": func(source *testpilotspb.Case) {
			source.Program.Entrypoints[0].Instructions[0].ActivationReservations[1].Count = 2
		},
		"handler without workflow": func(source *testpilotspb.Case) {
			source.Program.Entrypoints[0].Instructions[0].ActivationReservations = source.Program.Entrypoints[0].Instructions[0].ActivationReservations[1:]
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := carrierFixture(t)
			mutate(source)
			_, err := Prepare(source, catalog, policy)
			require.Error(t, err)
		})
	}
}

func TestReservationCarrierAuthorityDoesNotRequireReservations(t *testing.T) {
	source, catalog, policy := fixture(t)
	policy.Roles[0].ReservationCarriers = []ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []ReservationCarrierShape{{Kind: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 1}}}}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	_, ok := prepared.ReservationCarrier("controller", "call")
	require.False(t, ok)
}
