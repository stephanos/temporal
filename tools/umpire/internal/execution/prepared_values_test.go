package execution

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

func TestPreparedStartRejectsValue(t *testing.T) {
	c, catalog, policy := capabilityFixture(t)
	c.Program.Entrypoints[1].Nodes[0].Outcome.Fields = append(c.Program.Entrypoints[1].Nodes[0].Outcome.Fields, &umpirespb.OutcomeFieldSchema{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)})
	_, err := Prepare(c, catalog, policy)
	require.ErrorContains(t, err, "StartNexusOperation")
}

func TestPreparedOutcomeParity(t *testing.T) {
	c, catalog, policy := capabilityFixture(t)
	for _, entry := range c.Program.Entrypoints {
		for _, node := range entry.Nodes {
			node.Outcome.Fields = append(node.Outcome.Fields, &umpirespb.OutcomeFieldSchema{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_DETAIL, Type: scalar(umpirespb.SCALAR_KIND_TEXT)})
		}
	}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	ctx := context.Background()
	for _, target := range []struct{ entry, node int }{{0, 1}, {1, 0}, {1, 1}} {
		entry := p.Entrypoints()[target.entry]
		plan := entry.Instructions()[target.node]
		store, err := newValueStore(p, "run")
		require.NoError(t, err)
		a, err := store.activate(entry.ID(), "activation")
		require.NoError(t, err)
		coord := Coordinate{RunID: "run", EntrypointID: entry.ID(), ActivationID: "activation", InstructionID: plan.Source().InstructionId, Attempt: 1}
		for name, raw := range map[string]*umpirespb.InstructionOutcome{
			"success":    {Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Detail: "detail"},
			"value":      {Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: textValue("owned")},
			"wrong type": {Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: true}}},
			"malformed":  {Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: &umpirespb.Value{}},
			"oversized":  {Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: textValue(strings.Repeat("x", 5000))},
			"protocol":   {Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS, ProtocolCode: "denied"},
			"sdk":        {Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, SdkFailureCode: "failed"},
			"unknown":    {Status: 999}, "missing": nil,
		} {
			t.Run(entry.ID()+"/"+coord.InstructionID+"/"+name, func(t *testing.T) {
				for _, limit := range []int64{entry.RuntimeWorkLimit(), 1, 20, 40, 80} {
					batch, work, err := a.stage(ctx, coord, EffectResult{Outcome: raw}, limit)
					snapshot, sharedWork, sharedErr := plan.ValidateOutcome(ctx, raw, limit)
					require.Equal(t, err, sharedErr)
					require.Equal(t, work, sharedWork)
					if err != nil {
						require.Nil(t, snapshot)
						continue
					}
					require.True(t, proto.Equal(batch.outcome, snapshot.Outcome))
					require.Equal(t, batch.fields, snapshot.Fields)
					require.True(t, proto.Equal(raw, snapshot.Outcome))
					snapshot.Outcome.Detail = "mutated"
					require.NotEqual(t, "mutated", batch.outcome.Detail)
				}
				if target.entry == 1 && target.node == 1 {
					snapshot, work, err := plan.ValidateOutcome(ctx, raw, entry.RuntimeWorkLimit())
					if name == "value" || name == "sdk" {
						require.NoError(t, err)
						require.NotNil(t, snapshot)
						_, _, err = plan.ValidateOutcome(ctx, raw, work)
						require.NoError(t, err)
						_, _, err = plan.ValidateOutcome(ctx, raw, work-1)
						require.Error(t, err)
					} else {
						require.Error(t, err)
					}
				}
			})
		}
	}
}

func TestPreparedInputActivationIsolation(t *testing.T) {
	c, catalog, policy := capabilityFixture(t)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	entry := p.Entrypoints()[1]
	plan := entry.Instructions()[2]
	before := p.Snapshot()
	for _, text := range []string{"first", "second", "third", "fourth"} {
		t.Run(text, func(t *testing.T) {
			t.Parallel()
			fields := map[int32]*umpirespb.Value{int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS): {Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: int32(umpirespb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE)}}}}
			lookup := func(ref ir.Reference) *umpirespb.Value {
				require.Equal(t, "workflow", ref.Entrypoint)
				require.Equal(t, "await", ref.ID)
				return fields[ref.Field]
			}
			value, enabled, _, err := plan.EvaluateInput(context.Background(), lookup, entry.RuntimeWorkLimit())
			require.NoError(t, err)
			require.False(t, enabled)
			require.Nil(t, value)
			fields[int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS)].GetEnumValue().Number = int32(umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)
			_, _, _, err = plan.EvaluateInput(context.Background(), lookup, entry.RuntimeWorkLimit())
			require.Error(t, err)
			fields[int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE)] = textValue(text)
			value, enabled, work, err := plan.EvaluateInput(context.Background(), lookup, entry.RuntimeWorkLimit())
			require.NoError(t, err)
			require.True(t, enabled)
			require.Equal(t, text, value.GetText())
			value.Value = &umpirespb.Value_Text{Text: "changed"}
			require.Equal(t, text, fields[int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE)].GetText())
			_, _, _, err = plan.EvaluateInput(context.Background(), lookup, work)
			require.NoError(t, err)
			_, _, _, err = plan.EvaluateInput(context.Background(), lookup, work-1)
			require.Error(t, err)
			require.True(t, proto.Equal(before, p.Snapshot()))
		})
	}
}

func TestPreparedTerminalResultsAndDeclaredTypes(t *testing.T) {
	c, catalog, policy := capabilityFixture(t)
	for _, pair := range []struct{ entry, node int }{{1, 2}, {2, 0}} {
		n := c.Program.Entrypoints[pair.entry].Nodes[pair.node]
		n.Outcome.Fields = append(n.Outcome.Fields, &umpirespb.OutcomeFieldSchema{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)})
	}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	for _, pair := range []struct {
		entry, node int
		want        string
	}{{1, 2, "done"}, {2, 0, "accepted"}} {
		entry := p.Entrypoints()[pair.entry]
		n := entry.Instructions()[pair.node]
		typ, ok := n.OutcomeType(umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE)
		require.True(t, ok)
		require.True(t, proto.Equal(scalar(umpirespb.SCALAR_KIND_TEXT), typ))
		typ.Shape = nil
		typ, ok = n.OutcomeType(umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE)
		require.True(t, ok)
		require.NotNil(t, typ.Shape)
		typ, ok = n.OutcomeType(umpirespb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE)
		require.False(t, ok)
		require.Nil(t, typ)
		value, enabled, _, err := n.EvaluateInput(context.Background(), func(ref ir.Reference) *umpirespb.Value {
			if ref.Field == int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS) {
				return &umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: int32(umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)}}}
			}
			return textValue("done")
		}, entry.RuntimeWorkLimit())
		require.NoError(t, err)
		require.True(t, enabled)
		require.Equal(t, pair.want, value.GetText())
		raw := &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: value}
		snapshot, _, err := n.ValidateOutcome(context.Background(), raw, entry.RuntimeWorkLimit())
		require.NoError(t, err)
		raw.Value.Value = &umpirespb.Value_Text{Text: "changed"}
		require.Equal(t, pair.want, snapshot.Outcome.Value.GetText())
		snapshot.Outcome.Value.Value = &umpirespb.Value_Text{Text: "changed again"}
		require.Equal(t, pair.want, snapshot.Fields[umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE].GetText())
	}
}
