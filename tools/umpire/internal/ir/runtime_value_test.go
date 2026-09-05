package ir

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRuntimeSnapshotDeterministicFailureOrder(t *testing.T) {
	for _, source := range []proto.Message{
		&umpirespb.InstructionOutcome{Status: 999, Detail: "detail", Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "value"}}},
		&structpb.Struct{Fields: map[string]*structpb.Value{"z": {Kind: &structpb.Value_NullValue{NullValue: 999}}, "a": {Kind: &structpb.Value_StringValue{StringValue: "value"}}}},
	} {
		for _, limit := range []int64{1, 5, 20, 40, 100, 1000} {
			limits := DefaultLimits()
			limits.Work = limit
			_, work, want := SnapshotMessage(context.Background(), source, source.ProtoReflect().Descriptor(), limits)
			require.Error(t, want)
			for range 20 {
				snapshot, gotWork, err := SnapshotMessage(context.Background(), source, source.ProtoReflect().Descriptor(), limits)
				require.Nil(t, snapshot)
				require.Equal(t, want, err)
				require.Equal(t, work, gotWork)
			}
		}
	}
}
