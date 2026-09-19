package execution

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// An instruction's outcome fields follow from its instruction: every one has a status and a detail, a
// controller protocol effect a protocol code, a workflow or Nexus-handler instruction an SDK failure
// code, and only an awaited Nexus operation a value, its text result.
func TestPrepareDerivesOutcomeFieldsFromTheInstruction(t *testing.T) {
	c, catalog, policy := handleFixture(t)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	status := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
	text := scalar(testpilotspb.SCALAR_KIND_TEXT)
	for _, test := range []struct {
		entry, node int
		want        map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType
	}{
		{0, 0, map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: status, testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE: text, testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL: text}},
		{0, 1, map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: status, testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL: text}},
		{0, 2, map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: status, testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE: text, testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL: text}},
		{1, 0, map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: status, testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE: text, testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL: text}},
		{1, 1, map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: status, testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE: text, testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL: text, testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE: text}},
		{1, 2, map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: status, testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE: text, testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL: text}},
		{2, 0, map[testpilotspb.InstructionOutcomeField]*testpilotspb.ValueType{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: status, testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE: text, testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL: text}},
	} {
		plan := p.Entrypoints()[test.entry].Instructions()[test.node]
		for field := testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS; field <= testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE; field++ {
			typ, produced := plan.OutcomeType(field)
			want, wanted := test.want[field]
			require.Equal(t, wanted, produced, "%s %s", plan.Source().GetInstructionId(), field)
			require.True(t, proto.Equal(want, typ), "%s %s", plan.Source().GetInstructionId(), field)
		}
	}
}

func TestPreparedOutcomeParity(t *testing.T) {
	c, catalog, policy := handleFixture(t)
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
		coord := contract.Coordinate{RunID: "run", EntrypointID: entry.ID(), ActivationID: "activation", InstructionID: plan.Source().InstructionId, Attempt: 1}
		for name, raw := range map[string]*testpilotspb.InstructionOutcome{
			"success":    {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Detail: "detail"},
			"value":      {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: textValue("owned")},
			"wrong type": {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}},
			"malformed":  {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: &testpilotspb.Value{}},
			"oversized":  {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: textValue(strings.Repeat("x", 5000))},
			"protocol":   {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "denied"},
			"sdk":        {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, SdkFailureCode: "failed"},
			"unknown":    {Status: 999}, "missing": nil,
		} {
			t.Run(entry.ID()+"/"+coord.InstructionID+"/"+name, func(t *testing.T) {
				for _, limit := range []int64{entry.RuntimeWorkLimit(), 1, 20, 40, 80} {
					batch, work, err := a.stage(ctx, coord, contract.EffectResult{Outcome: raw}, limit)
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
	c, catalog, policy := handleFixture(t)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	entry := p.Entrypoints()[1]
	plan := entry.Instructions()[2]
	before := p.Snapshot()
	for _, text := range []string{"first", "second", "third", "fourth"} {
		t.Run(text, func(t *testing.T) {
			t.Parallel()
			fields := map[int32]*testpilotspb.Value{int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS): {Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: "INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE"}}}}
			lookup := func(ref ir.Reference) *testpilotspb.Value {
				require.Equal(t, "workflow", ref.Entrypoint)
				require.Equal(t, "await", ref.ID)
				return fields[ref.Field]
			}
			value, enabled, _, err := plan.EvaluateInput(context.Background(), lookup, entry.RuntimeWorkLimit())
			require.NoError(t, err)
			require.False(t, enabled)
			require.Nil(t, value)
			fields[int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS)].GetEnumValue().Name = "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"
			_, _, _, err = plan.EvaluateInput(context.Background(), lookup, entry.RuntimeWorkLimit())
			require.Error(t, err)
			fields[int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)] = textValue(text)
			value, enabled, work, err := plan.EvaluateInput(context.Background(), lookup, entry.RuntimeWorkLimit())
			require.NoError(t, err)
			require.True(t, enabled)
			require.Equal(t, text, value.GetTextValue())
			value.Value = &testpilotspb.Value_TextValue{TextValue: "changed"}
			require.Equal(t, text, fields[int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)].GetTextValue())
			_, _, _, err = plan.EvaluateInput(context.Background(), lookup, work)
			require.NoError(t, err)
			_, _, _, err = plan.EvaluateInput(context.Background(), lookup, work-1)
			require.Error(t, err)
			require.True(t, proto.Equal(before, p.Snapshot()))
		})
	}
}

// A Finish or RespondNexus result ends its activation rather than becoming an outcome value, so its
// outcome carries none; an awaited operation's value type is read through a copy.
func TestPreparedTerminalResultsAndOutcomeTypes(t *testing.T) {
	c, catalog, policy := handleFixture(t)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	await := p.Entrypoints()[1].Instructions()[1]
	typ, ok := await.OutcomeType(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)
	require.True(t, ok)
	require.True(t, proto.Equal(scalar(testpilotspb.SCALAR_KIND_TEXT), typ))
	typ.Shape = nil
	typ, ok = await.OutcomeType(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)
	require.True(t, ok)
	require.NotNil(t, typ.Shape)
	typ, ok = await.OutcomeType(testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE)
	require.False(t, ok)
	require.Nil(t, typ)
	for _, pair := range []struct {
		entry, node int
		want        string
	}{{1, 2, "done"}, {2, 0, "accepted"}} {
		entry := p.Entrypoints()[pair.entry]
		n := entry.Instructions()[pair.node]
		_, ok := n.OutcomeType(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)
		require.False(t, ok)
		value, enabled, _, err := n.EvaluateInput(context.Background(), func(ref ir.Reference) *testpilotspb.Value {
			if ref.Field == int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS) {
				return &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"}}}
			}
			return textValue("done")
		}, entry.RuntimeWorkLimit())
		require.NoError(t, err)
		require.True(t, enabled)
		require.Equal(t, pair.want, value.GetTextValue())
		_, _, err = n.ValidateOutcome(context.Background(), &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: value}, entry.RuntimeWorkLimit())
		require.ErrorContains(t, err, "undeclared payload")
		snapshot, _, err := n.ValidateOutcome(context.Background(), &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, entry.RuntimeWorkLimit())
		require.NoError(t, err)
		require.NotContains(t, snapshot.Fields, testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)
	}
}
