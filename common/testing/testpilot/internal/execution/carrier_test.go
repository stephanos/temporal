package execution

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func carrierFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	source, catalog, policy := handleFixture(t)
	policy.Roles[0].ReservationCarriers = []contract.ReservationCarrierPolicy{{
		Method: "/example.Service/Call",
		Shapes: []contract.ReservationCarrierShape{
			{Kind: contract.WorkflowEntrypoint, MaximumCount: 1},
			{Kind: contract.NexusHandlerEntrypoint, MaximumCount: 2},
		},
	}}
	return source, catalog, policy
}

func TestPrepareCompilesDeterministicReservationCarrierTopology(t *testing.T) {
	source, catalog, policy := carrierFixture(t)
	workflow := source.Program.Entrypoints[1]
	workflow.Instructions[0].Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: false}}}}
	secondStart := proto.CloneOf(workflow.Instructions[0])
	secondStart.InstructionId = "start_second"
	secondStart.Guard = nil
	secondStart.After = runsAfter("workflow")
	secondStart.Instruction.GetStartNexusOperation().Operation = "operation_second"
	workflow.Instructions = append(workflow.Instructions, secondStart)
	secondHandler := proto.CloneOf(source.Program.Entrypoints[2])
	secondHandler.EntrypointId = "handler_second"
	secondHandler.GetNexusHandler().Operation = "operation_second"
	secondHandler.Instructions[0].Instruction.GetRespondNexus().Kind = testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS
	secondHandler.Instructions[0].Instruction.GetRespondNexus().HandleSlotId = ""
	source.Program.Entrypoints = append(source.Program.Entrypoints, secondHandler)

	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	isolated, err := Prepare(proto.CloneOf(source), catalog, policy)
	require.NoError(t, err)
	want := contract.ReservationCarrierPlan{
		EndpointRoleID: "endpoint",
		Method:         "/example.Service/Call",
		Reservations: []contract.ReservationTopology{
			{EntrypointID: "workflow", Kind: contract.WorkflowEntrypoint, Count: 1},
			{EntrypointID: "handler", Kind: contract.NexusHandlerEntrypoint, Count: 1},
			{EntrypointID: "handler_second", Kind: contract.NexusHandlerEntrypoint, Count: 1},
		},
		Routes: []contract.ReservationRoute{
			{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 0, SourceInstructionID: "start", HandlerEntrypointID: "handler", HandlerOrdinal: 0},
			{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 0, SourceInstructionID: "start_second", HandlerEntrypointID: "handler_second", HandlerOrdinal: 0},
		},
	}
	plan, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, want, plan)
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
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	plan, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, []contract.ReservationTopology{{EntrypointID: "workflow", Kind: contract.WorkflowEntrypoint, Count: 1}}, plan.Reservations)
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
			policy.Roles[0].ReservationCarriers[0].Shapes = []contract.ReservationCarrierShape{shape, shape}
		},
		"unsupported context": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes[0].Kind = contract.ActivityEntrypoint
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
			policy.Roles[0].ReservationCarriers = make([]contract.ReservationCarrierPolicy, 10001)
		},
		"carrier on non-endpoint": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[1].ReservationCarriers = policy.Roles[0].ReservationCarriers
			policy.Roles[0].ReservationCarriers = nil
		},
		"unsupported reservation shape": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes = policy.Roles[0].ReservationCarriers[0].Shapes[:1]
		},
		"reservation cardinality": func(source *testpilotspb.Case, _ *Profile) {
			workflow := proto.CloneOf(source.Program.Entrypoints[1])
			workflow.EntrypointId = "workflow_second"
			source.Program.Entrypoints = append(source.Program.Entrypoints, workflow)
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
	for name, mutate := range map[string]func(*testpilotspb.Case, *Profile){
		"missing handler reservation": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes = policy.Roles[0].ReservationCarriers[0].Shapes[:1]
		},
		"ambiguous handler": func(source *testpilotspb.Case, _ *Profile) {
			handler := proto.CloneOf(source.Program.Entrypoints[2])
			handler.EntrypointId = "handler_second"
			source.Program.Entrypoints = append(source.Program.Entrypoints, handler)
		},
		"crossed handler": func(source *testpilotspb.Case, _ *Profile) {
			source.Program.Entrypoints[2].GetNexusHandler().Operation = "other"
		},
		"handler count mismatch": func(source *testpilotspb.Case, _ *Profile) {
			workflow := source.Program.Entrypoints[1]
			second := proto.CloneOf(workflow.Instructions[0])
			second.InstructionId = "start_second"
			second.After = runsAfter("workflow")
			workflow.Instructions = append(workflow.Instructions, second)
		},
		"handler without workflow": func(_ *testpilotspb.Case, policy *Profile) {
			policy.Roles[0].ReservationCarriers[0].Shapes = policy.Roles[0].ReservationCarriers[0].Shapes[1:]
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

func TestReservationCarrierAuthorityDoesNotRequireReservations(t *testing.T) {
	source, catalog, policy := fixture(t)
	policy.Roles[0].ReservationCarriers = []contract.ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []contract.ReservationCarrierShape{{Kind: contract.WorkflowEntrypoint, MaximumCount: 1}}}}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	_, ok := prepared.ReservationCarrier("controller", "call")
	require.False(t, ok)
}
