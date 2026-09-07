package execution

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestProjectionStagesOrderedElementsAndRejectsLimitsAtomically(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Limits.MaxPathFanout = 2
	c.Program.Observations = []*testpilotspb.ObservationDefinition{{ObservationId: "item", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}}
	n := c.Program.Entrypoints[0].Instructions[0]
	n.Limits.MaxEmittedEvents = 2
	path := field("items")
	path.Segments[0].Selector = &testpilotspb.FieldPathSegment_Repeated{Repeated: &testpilotspb.RepeatedWildcard{}}
	n.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotspb.ResponseProjection{{Source: path, Kind: testpilotspb.PROJECTION_KIND_EMIT_EACH, Targets: []*testpilotspb.ProjectionTarget{{Target: &testpilotspb.ProjectionTarget_ObservationId{ObservationId: "item"}}}}}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(p, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coord := Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}
	response := dynamicpb.NewMessage(p.graphs[0].nodes[0].method.Output())
	list := response.Mutable(response.Descriptor().Fields().ByName("items")).List()
	list.Append(protoreflect.ValueOfString("b"))
	list.Append(protoreflect.ValueOfString("a"))
	raw := EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}
	batch, work, err := values.stage(context.Background(), coord, raw, values.workLimit())
	require.NoError(t, err)
	require.Len(t, batch.facts, 2)
	require.Equal(t, int64(0), batch.facts[0].index)
	require.Equal(t, int64(1), batch.facts[1].index)
	require.Equal(t, "b", batch.facts[0].observations[0].Value.GetText())
	require.Equal(t, "a", batch.facts[1].observations[0].Value.GetText())
	_, _, err = values.stage(context.Background(), coord, raw, work)
	require.NoError(t, err)
	for _, mutate := range []func(){func() { list.Append(protoreflect.ValueOfString("overflow")) }, func() { list.Truncate(2); p.graphs[0].nodes[0].source.Limits.MaxEmittedEvents = 1 }, func() {
		p.graphs[0].nodes[0].source.Limits.MaxEmittedEvents = 2
		list.Set(0, protoreflect.ValueOfString(strings.Repeat("x", 4096)))
	}} {
		mutate()
		failed, _, err := values.stage(context.Background(), coord, raw, values.workLimit())
		require.Error(t, err)
		require.Nil(t, failed)
		require.Empty(t, values.slots)
		require.Empty(t, values.outcomes)
	}
}
