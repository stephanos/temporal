package lower

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func TestHeartbeatDetailsLowerOrderedTypedMessages(t *testing.T) {
	payload := func(text string) *umpirespb.Proto {
		return &umpirespb.Proto{Message: "temporal.api.common.v1.Payload", Fields: []*umpirespb.ProtoField{{Name: "data", Value: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Utf8{Utf8: text}}}}}
	}
	values := []*umpirespb.Proto{payload("first"), payload("second")}
	written := &umpirespb.Proto{Position: &umpirespb.Position{File: "heartbeat.scala", Line: 7}, Message: "temporal.api.common.v1.Payloads", Fields: []*umpirespb.ProtoField{{Name: "payloads", Value: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Messages{Messages: &umpirespb.ProtoMessages{Values: values}}}}}}
	var details commonpb.Payloads
	require.NoError(t, (&writer{}).into(written, &details))
	require.Len(t, details.GetPayloads(), 2)
	require.Equal(t, []byte("first"), details.GetPayloads()[0].GetData())
	require.Equal(t, []byte("second"), details.GetPayloads()[1].GetData())
	values[0].Fields[0].Value.GetKind().(*umpirespb.ProtoValue_Utf8).Utf8 = "changed"
	require.Equal(t, []byte("first"), details.GetPayloads()[0].GetData(), "the lowered details own their bytes")
}

func TestHeartbeatDetailsRefuseCrossedMessageLists(t *testing.T) {
	for _, test := range []struct {
		name, message, field, want string
		values                     []*umpirespb.Proto
	}{
		{"scalar", "temporal.api.common.v1.Payload", "data", "is no repeated message", nil},
		{"map", "temporal.api.common.v1.Payload", "metadata", "is no repeated message", nil},
		{"wrong message", "temporal.api.common.v1.Payloads", "payloads", "is written where a temporal.api.common.v1.Payload belongs", []*umpirespb.Proto{{Message: "temporal.api.common.v1.Payloads"}}},
		{"nil message", "temporal.api.common.v1.Payloads", "payloads", "has no message", []*umpirespb.Proto{nil}},
		{"malformed message", "temporal.api.common.v1.Payloads", "payloads", "no protobuf message", []*umpirespb.Proto{{}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			written := &umpirespb.Proto{Position: &umpirespb.Position{File: "heartbeat.scala", Line: 7}, Message: test.message, Fields: []*umpirespb.ProtoField{{Name: test.field, Value: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Messages{Messages: &umpirespb.ProtoMessages{Values: test.values}}}}}}
			_, err := (&writer{}).message(written, nil)
			require.ErrorContains(t, err, test.want)
			require.ErrorContains(t, err, "heartbeat.scala:7")
		})
	}
}

func TestHeartbeatDetailsRefuseANilMessageList(t *testing.T) {
	written := &umpirespb.Proto{Position: &umpirespb.Position{File: "heartbeat.scala", Line: 7}, Message: "temporal.api.common.v1.Payloads", Fields: []*umpirespb.ProtoField{{Name: "payloads", Value: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Messages{}}}}}
	_, err := (&writer{}).message(written, nil)
	require.ErrorContains(t, err, "has no message list")
	require.ErrorContains(t, err, "heartbeat.scala:7")
}
