package realization_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/realization"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// fn-133 R17: the carrier of each class a realization performs, derived from its binding.

func realizationOf(t *testing.T, file, name string) *umpirespb.Realization {
	t.Helper()
	m, err := ir.Load("../../../model/ir/" + file)
	require.NoError(t, err)
	i := slices.IndexFunc(m.GetRealizations(), func(r *umpirespb.Realization) bool { return r.GetName() == name })
	require.GreaterOrEqual(t, i, 0, "%s has no realization %s", file, name)
	return proto.CloneOf(m.GetRealizations()[i])
}

// The class as "<action>(<input cases>)", the action by its last segment.
func spelled(c *umpirespb.ActionClass) string {
	parts := strings.Split(c.GetAction(), ".")
	var inputs []string
	for _, in := range c.GetInputs() {
		inputs = append(inputs, spelledValue(in))
	}
	return parts[len(parts)-1] + "(" + strings.Join(inputs, ",") + ")"
}

// A value as its case, with its fields' values in parentheses.
func spelledValue(v *umpirespb.Value) string {
	switch {
	case v.GetEnum() != nil:
		var fields []string
		for _, f := range v.GetEnum().GetFields() {
			fields = append(fields, spelledValue(f))
		}
		if len(fields) == 0 {
			return v.GetEnum().GetCase()
		}
		return v.GetEnum().GetCase() + "(" + strings.Join(fields, ",") + ")"
	case v.GetKind() != nil:
		if b, ok := v.GetKind().(*umpirespb.Value_Bool); ok {
			if b.Bool {
				return "true"
			}
			return "false"
		}
	}
	return v.String()
}

func carried(t *testing.T, r *umpirespb.Realization) map[string][]string {
	t.Helper()
	mappings, err := realization.Carriers(r, protoregistry.GlobalFiles)
	require.NoError(t, err)
	out := map[string][]string{}
	for _, m := range mappings {
		var messages []string
		for _, c := range m.Carriers {
			messages = append(messages, c.Kind+" "+c.Message)
		}
		out[spelled(m.Class)] = messages
	}
	return out
}

// Each standalone activity RPC carries its own request; the worker's answers and its delivery carry
// what the worker sends and receives; a stop the realization performs with a fault carries nothing;
// a class no binding performs is unmapped.
func TestCarriersOfTheStandaloneActivity(t *testing.T) {
	c := carried(t, realizationOf(t, "activity-standalone.json", "standalone"))
	request := func(m string) []string { return []string{"rpc temporal.api.workflowservice.v1." + m} }
	require.Equal(t, request("PauseActivityExecutionRequest"), c["pause()"])
	require.Equal(t, request("UnpauseActivityExecutionRequest"), c["unpause()"])
	require.Equal(t, request("RequestCancelActivityExecutionRequest"), c["requestCancel()"])
	require.Equal(t, request("TerminateActivityExecutionRequest"), c["terminate()"])
	// A partial class pattern binds one exact class; every bound class of the start carries the start.
	for _, scheduleToStart := range []string{"unset", "expires"} {
		for _, startToClose := range []string{"unset", "expires"} {
			for _, heartbeat := range []string{"unset", "expires"} {
				for _, startDelay := range []string{"unset", "expires"} {
					for _, maxAttempts := range []string{"unlimited", "one", "two"} {
						class := "start(" + strings.Join([]string{"unset", scheduleToStart, startToClose, heartbeat, startDelay, maxAttempts}, ",") + ")"
						require.Equal(t, request("StartActivityExecutionRequest"), c[class], class)
					}
				}
			}
		}
	}
	require.NotContains(t, c, "start(expires,unset,unset,unset,unset,unlimited)", "a class no binding performs has no carrier")
	require.Equal(t, []string{"activity-answer temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest"}, c["respondCompleted()"])
	require.Equal(t, []string{"activity-answer temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest"}, c["respondFailed(fatal)"])
	require.Equal(t, []string{"activity-answer temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest"}, c["respondFailed(retryable)"])
	require.Equal(t, []string{"activity-answer temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest"}, c["respondCanceled()"])
	require.Equal(t, []string{"activity-delivery temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"}, c["poll()"])
	stop, ok := c["stop()"]
	require.True(t, ok, "the stop is performed")
	require.Empty(t, stop, "a fault carries nothing the action owns")
	// An await an onPath carries reads; it is no carrier of the classes it waits on.
	require.NotContains(t, c, "scheduleToStart()")
}

func TestCarriersOfActivityHeartbeatUseTheTokenMethod(t *testing.T) {
	r := &umpirespb.Realization{Name: "heartbeat", Scripts: []*umpirespb.Script{{Id: "attempts",
		Activation: &umpirespb.Script_Activity{Activity: &umpirespb.ActivityActivation{}},
		Items: []*umpirespb.Item{{Performs: []*umpirespb.Performance{{
			Step:    &umpirespb.ActionClass{Action: "worker.heartbeat"},
			Command: &umpirespb.Command{Instruction: &umpirespb.Command_AttemptHeartbeat{AttemptHeartbeat: &umpirespb.AttemptHeartbeat{}}},
		}}}},
	}}}
	mappings, err := realization.Carriers(r, protoregistry.GlobalFiles)
	require.NoError(t, err)
	require.Len(t, mappings, 1)
	require.Equal(t, []realization.Carrier{{Kind: "activity-heartbeat", Method: "/temporal.api.workflowservice.v1.WorkflowService/RecordActivityTaskHeartbeat", Message: "temporal.api.workflowservice.v1.RecordActivityTaskHeartbeatRequest"}}, mappings[0].Carriers)
}

// A workflow command carries the attributes it sets, and a handler's answer its message; a derived
// realization's added binding carries only in that realization.
func TestCarriersOfTheNexusCaller(t *testing.T) {
	async := carried(t, realizationOf(t, "nexus-workflow.json", "asyncNexus"))
	require.Equal(t, []string{"workflow-command temporal.api.command.v1.ScheduleNexusOperationCommandAttributes"}, async["schedule(unset,unset,unset)"])
	require.Equal(t, []string{"workflow-command temporal.api.command.v1.ScheduleNexusOperationCommandAttributes"}, async["schedule(unset,expires,unset)"])
	require.Equal(t, []string{"nexus-reply temporal.api.nexus.v1.StartOperationResponse"}, async["reply(async)"])
	require.Equal(t, []string{"nexus-reply temporal.api.nexus.v1.HandlerError"}, async["reply(handlerError(true))"])
	require.Equal(t, []string{"nexus-completion temporal.api.common.v1.Payload"}, async["complete(succeeded)"])
	require.NotContains(t, async, "inspect()")

	controlFile := "nexus-workflow-control.json"
	m, err := ir.Load("../../../model/ir/" + controlFile)
	require.NoError(t, err)
	control := carried(t, m.GetRealizations()[0])
	require.Equal(t, []string{"rpc temporal.api.workflowservice.v1.DescribeWorkflowExecutionRequest"}, control["inspect()"])
}

// One class bound to two different carriers in one script is ambiguous; a call of a method the
// registry does not hold has no carrier to derive.
func TestCarriersRefuseAmbiguousAndUnknownBindings(t *testing.T) {
	r := realizationOf(t, "activity-standalone.json", "standalone")
	controller := r.GetScripts()[0]
	var pause, unpause *umpirespb.Performance
	for _, item := range controller.GetItems() {
		for _, p := range item.GetPerforms() {
			switch spelled(p.GetStep()) {
			case "pause()":
				pause = p
			case "unpause()":
				unpause = p
			}
		}
	}
	twice := proto.CloneOf(r)
	twice.GetScripts()[0].Items = append(twice.GetScripts()[0].Items, &umpirespb.Item{
		Performs: []*umpirespb.Performance{{Step: pause.GetStep(), Command: unpause.GetCommand()}},
	})
	_, err := realization.Carriers(twice, protoregistry.GlobalFiles)
	require.ErrorContains(t, err, "ambiguous")

	unknown := proto.CloneOf(r)
	for _, item := range unknown.GetScripts()[0].GetItems() {
		for _, p := range item.GetPerforms() {
			if spelled(p.GetStep()) == "pause()" {
				p.GetCommand().GetRpc().Method = "/temporal.api.workflowservice.v1.WorkflowService/NoSuchMethod"
			}
		}
	}
	_, err = realization.Carriers(unknown, protoregistry.GlobalFiles)
	require.ErrorContains(t, err, "has no method NoSuchMethod")
}
