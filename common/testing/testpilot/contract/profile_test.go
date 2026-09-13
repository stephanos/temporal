package contract

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

func TestEntrypointKindOfClassifiesEachActivation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		entrypoint *testpilotspb.Entrypoint
		want       EntrypointKind
	}{
		{name: "controller", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}, want: ControllerEntrypoint},
		{name: "workflow", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{}}}, want: WorkflowEntrypoint},
		{name: "activity", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{}}}, want: ActivityEntrypoint},
		{name: "nexus handler", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{}}}, want: NexusHandlerEntrypoint},
		{name: "no activation", entrypoint: &testpilotspb.Entrypoint{}},
		{name: "nil entrypoint"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, EntrypointKindOf(tc.entrypoint))
		})
	}
	require.Equal(t, NexusHandlerEntrypoint, MaxEntrypointKind)
}
