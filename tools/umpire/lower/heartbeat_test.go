package lower

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
)

func TestHeartbeatPendingPublicationPrecedesItsFutureTimer(t *testing.T) {
	for _, test := range []struct {
		name          string
		mode          umpirespb.WithholdingMode
		firstStep     int
		secondAttempt int64
		want          string
		complete      bool
	}{
		{"pending after prefix", umpirespb.WITHHOLDING_MODE_SDK_PENDING, 1, 2, "", false},
		{"context publishes after timer", umpirespb.WITHHOLDING_MODE_CONTEXT, 1, 2, "follows later evidence", false},
		{"pending cannot confirm future timer", umpirespb.WITHHOLDING_MODE_SDK_PENDING, 3, 2, "precedes earlier evidence", false},
		{"duplicate attempt record", umpirespb.WITHHOLDING_MODE_SDK_PENDING, 1, 1, "two kinds of evidence", false},
		{"finish after prefix", umpirespb.WITHHOLDING_MODE_CONTEXT, 1, 0, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			class := func(action string) *umpirespb.ActionClass { return &umpirespb.ActionClass{Action: action} }
			performance := func(action string, command *umpirespb.Command) *umpirespb.Item {
				return &umpirespb.Item{Performs: []*umpirespb.Performance{{Step: class(action), Command: command}}}
			}
			script := &umpirespb.Script{Id: "attempts", Activation: &umpirespb.Script_Activity{Activity: &umpirespb.ActivityActivation{Starts: []*umpirespb.ActionClass{class("poll")}}},
				Items: []*umpirespb.Item{
					performance("heartbeat", &umpirespb.Command{Instruction: &umpirespb.Command_AttemptHeartbeat{AttemptHeartbeat: &umpirespb.AttemptHeartbeat{}}}),
					{When: []*umpirespb.ActionClass{class("timer")}, Command: &umpirespb.Command{Instruction: &umpirespb.Command_AttemptWithheld{AttemptWithheld: &umpirespb.AttemptWithheld{Mode: test.mode}}}},
					performance("finish", &umpirespb.Command{Instruction: &umpirespb.Command_Finish{Finish: &umpirespb.Finish{}}}),
				}}
			evidence := func(number int64) *umpirespb.Evidence {
				return &umpirespb.Evidence{Position: &umpirespb.Position{File: "heartbeat.scala", Line: 1}, From: &umpirespb.Evidence_RunEvent{RunEvent: &umpirespb.RunEventSource{Attempt: &umpirespb.AttemptOf{Script: "attempts", Number: number}}}}
			}
			confirmed := func(kind string, steps ...int) cp.Confirmation {
				return cp.Confirmation{Source: &cp.EvidenceSource{KindID: kind}, Steps: steps}
			}
			l := &lowering{a: &asked{r: &umpirespb.Realization{Scripts: []*umpirespb.Script{script}}},
				adapter:       &adapter{classKey: func(c *umpirespb.ActionClass) string { return c.GetAction() }, evidence: map[string]*umpirespb.Evidence{"first": evidence(1), "second": evidence(test.secondAttempt)}},
				keys:          []string{"start", "poll", "heartbeat", "timer", "backoff", "poll", "finish"},
				confirmations: []cp.Confirmation{confirmed("first", test.firstStep), confirmed("receipt", 2), confirmed("second", 3, 4, 5), confirmed("completed", 6)}}
			if test.firstStep == 3 {
				l.confirmations[0], l.confirmations[1] = l.confirmations[1], l.confirmations[0]
			}
			if test.secondAttempt == 1 {
				l.confirmations = []cp.Confirmation{confirmed("first", 1), confirmed("second", 2), confirmed("completed", 6)}
			}
			if test.complete {
				script.Items = append(script.Items[:1], script.Items[2])
				l.keys = []string{"start", "poll", "heartbeat", "finish"}
				l.confirmations = []cp.Confirmation{confirmed("first", 1), confirmed("receipt", 2), confirmed("completed", 3)}
			}
			_, answered := l.attemptClasses(script)
			require.NotContains(t, answered, "heartbeat", "a prefix is not an attempt disposition")
			problems := l.late()
			if test.want == "" {
				require.Empty(t, problems)
				return
			}
			require.Len(t, problems, 1)
			require.Contains(t, problems[0].Construct, test.want)
			require.Equal(t, "heartbeat.scala:1", problems[0].Position)
		})
	}
}
