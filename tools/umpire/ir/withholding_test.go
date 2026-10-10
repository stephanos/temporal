package ir

import (
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/common/testing/testpilot/duration"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func TestWithholdingRequiresOneActivityScriptServerTimer(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*umpirespb.Realization, *umpirespb.Script, *umpirespb.Item)
		want   string
	}{
		{"armed timer", func(*umpirespb.Realization, *umpirespb.Script, *umpirespb.Item) {}, ""},
		{"missing timeout basis", func(r *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			for _, step := range withholdingTimers(r, item) {
				step.TimeoutBasis = umpirespb.TIMEOUT_BASIS_UNSPECIFIED
			}
		}, "timeout basis"},
		{"heartbeat-only context basis", func(r *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			for _, step := range withholdingTimers(r, item) {
				step.TimeoutBasis = umpirespb.TIMEOUT_BASIS_HEARTBEAT
			}
		}, "timeout basis"},
		{"unknown timeout basis", func(r *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			for _, step := range withholdingTimers(r, item) {
				step.TimeoutBasis = umpirespb.TimeoutBasis(99)
			}
		}, "timeout basis"},
		{"unknown withholding mode", func(_ *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			item.GetCommand().GetAttemptWithheld().Mode = umpirespb.WithholdingMode(99)
		}, "withholding mode"},
		{"unconditional", func(_ *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			item.When = nil
		}, "without exactly one armed bounded server timer"},
		{"multiple timers", func(r *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			item.When = append(item.When, proto.CloneOf(r.GetServerSteps()[1].GetStep()))
		}, "without exactly one armed bounded server timer"},
		{"repeated timer condition", func(_ *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			item.When = append(item.When, proto.CloneOf(item.GetWhen()[0]))
		}, "without exactly one armed bounded server timer"},
		{"repeated timer command", func(_ *umpirespb.Realization, script *umpirespb.Script, item *umpirespb.Item) {
			other := proto.CloneOf(item)
			other.GetCommand().Id = "withhold-again"
			script.Items = append(script.Items, other)
		}, "withholds"},
		{"wrong owner", func(r *umpirespb.Realization, script *umpirespb.Script, item *umpirespb.Item) {
			script.Items = script.Items[:len(script.Items)-1]
			admScript(t, r, "controller").Items = append(admScript(t, r, "controller").Items, item)
		}, "script controller is no activity's"},
		{"perform bound", func(_ *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			item.Performs = []*umpirespb.Performance{{Position: item.GetPosition(), Step: item.GetWhen()[0], Command: item.GetCommand()}}
			item.Command, item.When = nil, nil
		}, "withholds an attempt under a performance"},
		{"unarmed timer", func(r *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			r.ServerSteps = slices.DeleteFunc(r.ServerSteps, func(step *umpirespb.ServerStep) bool {
				return proto.Equal(step.GetStep(), item.GetWhen()[0])
			})
		}, "without an armed bounded server timer"},
		{"unbounded timer", func(r *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			for _, step := range withholdingTimers(r, item) {
				step.Deadline = duration.FromMilliseconds(0)
			}
		}, "withholds an attempt without an armed bounded server timer"},
		{"non timer cause", func(_ *umpirespb.Realization, _ *umpirespb.Script, item *umpirespb.Item) {
			item.When[0] = &umpirespb.ActionClass{Action: "temporal.features.activity.worker.poll"}
		}, "without an armed bounded server timer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, err := Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone.json"))
			require.NoError(t, err)
			index := slices.IndexFunc(m.GetRealizations(), func(r *umpirespb.Realization) bool { return r.GetName() == "standalone" })
			require.NotEqual(t, -1, index)
			r := m.GetRealizations()[index]
			script := admScript(t, r, "attempts")
			item := &umpirespb.Item{Position: script.GetPosition(), When: []*umpirespb.ActionClass{{Action: "temporal.features.activity.deadline.startToClose"}},
				Command: &umpirespb.Command{Id: "withhold-attempt", Position: script.GetPosition(), Instruction: &umpirespb.Command_AttemptWithheld{AttemptWithheld: &umpirespb.AttemptWithheld{}}}}
			for _, step := range r.GetServerSteps() {
				if step.GetStep().GetAction() == item.When[0].GetAction() {
					step.TimeoutBasis = umpirespb.TIMEOUT_BASIS_START_TO_CLOSE
				}
			}
			script.Items = append(script.Items, item)
			require.NotEmpty(t, withholdingTimers(r, item), "the specimen names its actual bound timer")
			test.change(r, script, item)
			err = Validate(m)
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.want)
			requireLocated(t, err, "model/temporal/features/activity/standalone/system/Realization.scala:")
		})
	}
}

func withholdingTimers(r *umpirespb.Realization, item *umpirespb.Item) []*umpirespb.ServerStep {
	var timers []*umpirespb.ServerStep
	for _, step := range r.GetServerSteps() {
		if proto.Equal(step.GetStep(), item.GetWhen()[0]) {
			timers = append(timers, step)
		}
	}
	return timers
}
