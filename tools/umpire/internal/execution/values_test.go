package execution

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func dataFixture(t *testing.T) (*PreparedProgram, *activationValues, Coordinate) {
	t.Helper()
	c, catalog, policy := fixture(t)
	c.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "text", Kind: umpirespb.SLOT_KIND_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	c.Program.Observations = []*umpirespb.ObservationSchema{{ObservationId: "text", Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	c.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: field("text"), Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "text"}}, {Sink: &umpirespb.ProjectionSink_ObservationId{ObservationId: "text"}}}}}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(p, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	return p, values, Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}
}
func effectResponse(p *PreparedProgram, text string) EffectResult {
	response := dynamicpb.NewMessage(p.graphs[0].nodes[0].method.Output())
	response.Set(response.Descriptor().Fields().ByName("text"), protoreflect.ValueOfString(text))
	return EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}
}
func TestValuesStageAtomicallyAndOwnSnapshots(t *testing.T) {
	p, values, coord := dataFixture(t)
	ctx := context.Background()
	raw := effectResponse(p, "first")
	batch, work, err := values.stage(ctx, coord, raw, values.workLimit())
	require.NoError(t, err)
	require.Empty(t, values.slots)
	raw.Response.ProtoReflect().Set(raw.Response.ProtoReflect().Descriptor().Fields().ByName("text"), protoreflect.ValueOfString("changed"))
	raw.Outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS
	require.NoError(t, values.commit(ctx, batch))
	require.Equal(t, "first", values.slots["text"].GetText())
	require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, batch.outcome.Status)
	require.Equal(t, "first", batch.facts[0].observations[0].Value.GetText())
	require.Error(t, values.commit(ctx, batch))
	_, other, _ := dataFixture(t)
	require.Error(t, other.commit(ctx, batch))
	require.Empty(t, other.slots)
	_, _, err = other.stage(ctx, coord, effectResponse(p, "first"), work-1)
	require.Error(t, err)
	require.Empty(t, other.slots)
	values.store.seal()
	require.Error(t, values.commit(ctx, batch))
}
func TestValuesRejectWrongOwnershipAndUnprojectedPayload(t *testing.T) {
	p, values, coord := dataFixture(t)
	for _, mutate := range []func(*Coordinate, *EffectResult){
		func(c *Coordinate, _ *EffectResult) { c.RunID = "other" }, func(c *Coordinate, _ *EffectResult) { c.ActivationID = "other" }, func(c *Coordinate, _ *EffectResult) { c.Attempt = 2 },
		func(_ *Coordinate, r *EffectResult) {
			r.Outcome.Value = &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "secret"}}
		}, func(_ *Coordinate, r *EffectResult) { r.Response = &umpirespb.InstructionOutcome{} },
	} {
		c := coord
		r := effectResponse(p, "first")
		mutate(&c, &r)
		batch, _, err := values.stage(context.Background(), c, r, values.workLimit())
		require.Error(t, err)
		require.Nil(t, batch)
		require.Empty(t, values.slots)
	}
}
func TestRequestReadsGuardedSlotsWithoutRebinding(t *testing.T) {
	c, catalog, policy := fixture(t)
	node := c.Program.Entrypoints[0].Nodes[0]
	node.Instruction.GetInvokeRpc().RequestAssignments = []*umpirespb.RequestAssignment{{Target: field("text"), Value: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}}}}}}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(p, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coord := Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}
	request, enabled, work, err := values.request(context.Background(), coord, values.workLimit())
	require.NoError(t, err)
	require.True(t, enabled)
	wire, err := proto.Marshal(request)
	require.NoError(t, err)
	require.Equal(t, []byte{0x0a, 7, 'r', 'e', 'q', 'u', 'e', 's', 't'}, wire)
	_, _, _, err = values.request(context.Background(), coord, work-1)
	require.Error(t, err)
}

func TestValuesSealRejectsFreshBatchAndReads(t *testing.T) {
	p, values, coord := dataFixture(t)
	ctx := context.Background()
	batch, _, err := values.stage(ctx, coord, effectResponse(p, "ready"), values.workLimit())
	require.NoError(t, err)
	values.store.seal()
	require.Error(t, values.commit(ctx, batch))
	require.Empty(t, values.slots)
	_, _, err = values.stage(ctx, coord, effectResponse(p, "ready"), values.workLimit())
	require.Error(t, err)
	_, _, _, err = values.request(ctx, coord, values.workLimit())
	require.Error(t, err)
}

func TestValuesGuardedMissingSlotsAndActivationIsolation(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "text", Kind: umpirespb.SLOT_KIND_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	c.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: field("text"), Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "text"}}}}}
	consumer := rpcNode("consumer")
	consumer.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = present(slot("text"))
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*umpirespb.RequestAssignment{{Target: field("text"), Value: slot("text")}}
	c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, consumer)
	addWorker(c)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(p, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coord := Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "consumer", Attempt: 1}
	request, enabled, _, err := values.request(context.Background(), coord, values.workLimit())
	require.NoError(t, err)
	require.False(t, enabled)
	require.Nil(t, request)
	w, err := values.newWork(context.Background(), values.workLimit())
	require.NoError(t, err)
	_, err = values.evaluate(w, p.graphs[0].nodes[1].assignments[0].value)
	require.Error(t, err)
	producer := coord
	producer.InstructionID = "call"
	batch, _, err := values.stage(context.Background(), producer, effectResponse(p, "shared"), values.workLimit())
	require.NoError(t, err)
	require.NoError(t, values.commit(context.Background(), batch))
	request, enabled, _, err = values.request(context.Background(), coord, values.workLimit())
	require.NoError(t, err)
	require.True(t, enabled)
	require.Equal(t, "shared", request.ProtoReflect().Get(request.ProtoReflect().Descriptor().Fields().ByName("text")).String())
	workerA, err := store.activate("workflow", "worker-a")
	require.NoError(t, err)
	workerB, err := store.activate("workflow", "worker-b")
	require.NoError(t, err)
	require.Empty(t, workerA.slots)
	require.Empty(t, workerB.slots)
	require.Error(t, workerA.commit(context.Background(), batch))
	_, err = store.activate("controller", "second-controller")
	require.Error(t, err)
}

func TestValuesConcurrentRunIsolationAndSeal(t *testing.T) {
	p, _, _ := dataFixture(t)
	results := make(chan error, 8)
	for i := range 8 {
		go func() {
			store, err := newValueStore(p, fmt.Sprintf("run-%d", i))
			if err != nil {
				results <- err
				return
			}
			values, err := store.activate("controller", "activation")
			if err != nil {
				results <- err
				return
			}
			coord := Coordinate{RunID: store.runID, EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}
			batch, _, err := values.stage(context.Background(), coord, effectResponse(p, store.runID), values.workLimit())
			if err != nil {
				results <- err
				return
			}
			ready := make(chan struct{})
			done := make(chan struct{})
			go func() { <-ready; store.seal(); close(done) }()
			close(ready)
			err = values.commit(context.Background(), batch)
			<-done
			if err == nil && values.slots["text"].GetText() != store.runID {
				results <- errors.New("crossed Run value")
				return
			}
			if err != nil && len(values.slots) != 0 {
				results <- errors.New("partial sealed write")
				return
			}
			results <- nil
		}()
	}
	for range 8 {
		require.NoError(t, <-results)
	}
}

func TestOutcomeValidationAndIndependentAttemptSnapshots(t *testing.T) {
	p, values, coord := dataFixture(t)
	ctx := context.Background()
	for _, mutate := range []func(*umpirespb.InstructionOutcome){
		func(o *umpirespb.InstructionOutcome) { o.Status = 999 }, func(o *umpirespb.InstructionOutcome) { o.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_UNSPECIFIED },
		func(o *umpirespb.InstructionOutcome) { o.ProtoReflect().SetUnknown([]byte{0x80, 6, 1}) }, func(o *umpirespb.InstructionOutcome) { o.Detail = strings.Repeat("x", 5000) },
		func(o *umpirespb.InstructionOutcome) { o.SdkFailureCode = "worker" },
	} {
		raw := effectResponse(p, "ready")
		mutate(raw.Outcome)
		batch, _, err := values.stage(ctx, coord, raw, values.workLimit())
		require.Error(t, err)
		require.Nil(t, batch)
	}
	p.graphs[0].nodes[0].source.Bounds.MaxAttempts = 2
	raw := EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS, ProtocolCode: "first"}}
	first, _, err := values.stage(ctx, coord, raw, values.workLimit())
	require.NoError(t, err)
	require.NoError(t, values.commit(ctx, first))
	coord.Attempt = 2
	raw.Outcome.ProtocolCode = "second"
	second, _, err := values.stage(ctx, coord, raw, values.workLimit())
	require.NoError(t, err)
	require.NoError(t, values.commit(ctx, second))
	raw.Outcome.ProtocolCode = "mutated"
	require.Equal(t, "first", first.outcome.ProtocolCode)
	require.Equal(t, "second", second.outcome.ProtocolCode)
	require.Len(t, values.outcomes, 2)
	require.Empty(t, values.slots)
}

func TestWorkerOutcomeValuesRemainActivationLocal(t *testing.T) {
	c, catalog, policy := fixture(t)
	addWorker(c)
	node := rpcNode("finish")
	node.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_Finish{Finish: &umpirespb.Finish{Result: textLiteral("done")}}}
	node.Outcome.Fields = append(node.Outcome.Fields, &umpirespb.OutcomeFieldSchema{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)})
	c.Program.Entrypoints[1].Nodes = []*umpirespb.InstructionNode{node}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(p, "run")
	require.NoError(t, err)
	a, err := store.activate("workflow", "a")
	require.NoError(t, err)
	b, err := store.activate("workflow", "b")
	require.NoError(t, err)
	coord := Coordinate{RunID: "run", EntrypointID: "workflow", ActivationID: "a", InstructionID: "finish", Attempt: 1}
	raw := EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: textValue("owned")}}
	batch, _, err := a.stage(context.Background(), coord, raw, a.workLimit())
	require.NoError(t, err)
	require.NoError(t, a.commit(context.Background(), batch))
	raw.Outcome.Value.Value = &umpirespb.Value_Text{Text: "changed"}
	require.Equal(t, "owned", a.latest["finish"].fields[umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE].GetText())
	require.Empty(t, b.latest)
	for _, value := range []*umpirespb.Value{nil, {Value: &umpirespb.Value_BoolValue{BoolValue: true}}} {
		coord.ActivationID = "b"
		raw.Outcome.Value = value
		_, _, err = b.stage(context.Background(), coord, raw, b.workLimit())
		require.Error(t, err)
		require.Empty(t, b.latest)
	}
}

func TestWideExpressionBudgetAndCeilingOverflow(t *testing.T) {
	c, catalog, policy := fixture(t)
	operands := make([]*umpirespb.ValueExpression, 128)
	for i := range operands {
		operands[i] = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: true}}}}
	}
	c.Program.Entrypoints[0].Nodes[0].Guard = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_All{All: &umpirespb.AllExpression{Operands: operands}}}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(p, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coord := Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}
	_, enabled, work, err := values.request(context.Background(), coord, values.workLimit())
	require.NoError(t, err)
	require.True(t, enabled)
	_, _, _, err = values.request(context.Background(), coord, work)
	require.NoError(t, err)
	_, _, _, err = values.request(context.Background(), coord, work-1)
	require.Error(t, err)
	limits := proto.CloneOf(p.source.Limits)
	limits.MaxRequestBytes = math.MaxInt64
	require.Equal(t, int64(math.MaxInt64), runtimeWorkLimit(p.graphs[0], limits))
}
