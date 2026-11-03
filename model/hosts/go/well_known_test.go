package temporal_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// These current wire values are also checked by the Rust native-type controls.
func TestNativeWellKnownPresence(t *testing.T) {
	t.Run("Empty oneof", func(t *testing.T) {
		present := &workflowpb.PendingActivityInfo{
			AssignedBuildId: &workflowpb.PendingActivityInfo_UseWorkflowBuildId{
				UseWorkflowBuildId: &emptypb.Empty{},
			},
		}
		encoded, err := proto.Marshal(present)
		require.NoError(t, err)
		require.Equal(t, []byte{0x6a, 0}, encoded)
		var decoded workflowpb.PendingActivityInfo
		require.NoError(t, proto.Unmarshal(encoded, &decoded))
		require.True(t, proto.Equal(present, &decoded))
		require.False(t, proto.Equal(&workflowpb.PendingActivityInfo{}, &decoded))
	})
	t.Run("DoubleValue wrapper", func(t *testing.T) {
		zero := &taskqueuepb.TaskQueueMetadata{MaxTasksPerSecond: wrapperspb.Double(0)}
		encoded, err := proto.Marshal(zero)
		require.NoError(t, err)
		require.Equal(t, []byte{0x0a, 0}, encoded)
		for _, value := range []*taskqueuepb.TaskQueueMetadata{
			zero, {MaxTasksPerSecond: wrapperspb.Double(12.5)},
		} {
			encoded, err := proto.Marshal(value)
			require.NoError(t, err)
			var decoded taskqueuepb.TaskQueueMetadata
			require.NoError(t, proto.Unmarshal(encoded, &decoded))
			require.True(t, proto.Equal(value, &decoded))
			require.False(t, proto.Equal(&taskqueuepb.TaskQueueMetadata{}, &decoded))
		}
	})
}
