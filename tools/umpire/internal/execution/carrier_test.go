package execution

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

func carrierFixture(t *testing.T) (*umpirespb.Case, *ir.Catalog, Policy) {
	t.Helper()
	source, catalog, policy := capabilityFixture(t)
	policy.Roles[0].ReservationCarriers = []ReservationCarrierPolicy{{
		Method: "/example.Service/Call",
		Shapes: []ReservationCarrierShape{
			{Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, MaximumCount: 2},
			{Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, MaximumCount: 4},
		},
	}}
	return source, catalog, policy
}

func TestPrepareCompilesDeterministicReservationCarrierTopology(t *testing.T) {
	source, catalog, policy := carrierFixture(t)
	controller := source.Program.Entrypoints[0]
	controller.Nodes[0].ActivationReservations[0].Count = 2
	controller.Nodes[0].ActivationReservations[1].Count = 4
	workflow := source.Program.Entrypoints[1]
	workflow.Nodes[0].Guard = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: false}}}}
	secondStart := proto.CloneOf(workflow.Nodes[0])
	secondStart.InstructionId = "start_second"
	secondStart.Guard = nil
	workflow.Nodes = append(workflow.Nodes, secondStart)
	secondController := proto.CloneOf(controller.Nodes[0])
	secondController.InstructionId = "call_second"
	secondController.ActivationReservations[0].Count = 1
	secondController.ActivationReservations[1].Count = 2
	controller.Nodes = append(controller.Nodes, secondController)

	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	isolated, err := Prepare(proto.CloneOf(source), catalog, policy)
	require.NoError(t, err)
	want := ReservationCarrierPlan{
		EndpointRoleID: "endpoint",
		Method:         "/example.Service/Call",
		Reservations: []ReservationTopology{
			{EntrypointID: "workflow", Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, Count: 2},
			{EntrypointID: "handler", Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, Count: 4},
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
	source.Program.Entrypoints[1].Nodes[0].Instruction.GetStartNexusOperation().Operation = "changed"
	again, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, want, again)
}

func TestPrepareExposesWorkflowOnlyCarrierReservations(t *testing.T) {
	source, catalog, policy := fixture(t)
	addWorker(source)
	source.Program.Entrypoints[0].Nodes[0].ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 1}}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	plan, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, []ReservationTopology{{EntrypointID: "workflow", Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, Count: 1}}, plan.Reservations)
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
	for name, mutate := range map[string]func(*umpirespb.Case, *Policy){
		"method outside ordinary authorization": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers[0].Method = "/example.Service/Other"
		},
		"streaming method": func(source *umpirespb.Case, policy *Policy) {
			policy.Roles[0].Methods = append(policy.Roles[0].Methods, "/example.Service/Stream")
			policy.Roles[0].ReservationCarriers[0].Method = "/example.Service/Stream"
			source.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().Method = "/example.Service/Stream"
		},
		"duplicate method": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers = append(policy.Roles[0].ReservationCarriers, policy.Roles[0].ReservationCarriers[0])
		},
		"duplicate context": func(_ *umpirespb.Case, policy *Policy) {
			shape := policy.Roles[0].ReservationCarriers[0].Shapes[0]
			policy.Roles[0].ReservationCarriers[0].Shapes = []ReservationCarrierShape{shape, shape}
		},
		"unsupported context": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers[0].Shapes[0].Context = umpirespb.ENTRYPOINT_CONTEXT_ACTIVITY
		},
		"zero cardinality": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 0
		},
		"cardinality overflow": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = math.MaxInt64
		},
		"aggregate cardinality": func(_ *umpirespb.Case, policy *Policy) {
			for i := range policy.Roles[0].ReservationCarriers[0].Shapes {
				policy.Roles[0].ReservationCarriers[0].Shapes[i].MaximumCount = policy.Limits.MaxActivations
			}
		},
		"oversized carrier policy": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers = make([]ReservationCarrierPolicy, 10001)
		},
		"carrier on non-endpoint": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[1].ReservationCarriers = policy.Roles[0].ReservationCarriers
			policy.Roles[0].ReservationCarriers = nil
		},
		"missing carrier authority": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers = nil
		},
		"unsupported reservation shape": func(_ *umpirespb.Case, policy *Policy) {
			policy.Roles[0].ReservationCarriers[0].Shapes = policy.Roles[0].ReservationCarriers[0].Shapes[:1]
		},
		"reservation cardinality": func(source *umpirespb.Case, _ *Policy) {
			source.Program.Entrypoints[0].Nodes[0].ActivationReservations[0].Count = 3
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
	for name, mutate := range map[string]func(*umpirespb.Case){
		"missing handler reservation": func(source *umpirespb.Case) {
			source.Program.Entrypoints[0].Nodes[0].ActivationReservations = source.Program.Entrypoints[0].Nodes[0].ActivationReservations[:1]
		},
		"ambiguous handler": func(source *umpirespb.Case) {
			handler := proto.CloneOf(source.Program.Entrypoints[2])
			handler.EntrypointId = "handler_second"
			source.Program.Entrypoints = append(source.Program.Entrypoints, handler)
			source.Program.Entrypoints[0].Nodes[0].ActivationReservations = append(source.Program.Entrypoints[0].Nodes[0].ActivationReservations, &umpirespb.ActivationReservation{EntrypointId: "handler_second", Count: 1})
		},
		"crossed handler": func(source *umpirespb.Case) {
			source.Program.Entrypoints[2].Activation.GetNexusHandler().Operation = "other"
		},
		"handler count mismatch": func(source *umpirespb.Case) {
			source.Program.Entrypoints[0].Nodes[0].ActivationReservations[1].Count = 2
		},
		"handler without workflow": func(source *umpirespb.Case) {
			source.Program.Entrypoints[0].Nodes[0].ActivationReservations = source.Program.Entrypoints[0].Nodes[0].ActivationReservations[1:]
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
	policy.Roles[0].ReservationCarriers = []ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []ReservationCarrierShape{{Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, MaximumCount: 1}}}}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	_, ok := prepared.ReservationCarrier("controller", "call")
	require.False(t, ok)
}
