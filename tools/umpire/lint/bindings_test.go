package lint

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// fn-133 R6: an `onPath` of a timer class no realizable path reaches is an unreachable binding. No
// start of the activity's realization sets a schedule-to-close deadline, so no realizable path
// enables the schedule-to-close timer.
func TestUnreachableBindings(t *testing.T) {
	m := read(t, activityIR, func(ir *umpirespb.Model) {
		i := slices.IndexFunc(ir.GetActions(), func(a *umpirespb.Action) bool {
			return strings.HasSuffix(a.GetId(), "deadline.scheduleToClose")
		})
		s := realization(ir, "activitySystem").GetScripts()[0]
		probe := proto.CloneOf(s.GetItems()[0].GetPerforms()[0].GetCommand())
		probe.Id = "probe"
		s.Items = append(s.Items, &umpirespb.Item{
			Position: &umpirespb.Position{File: "fixture.scala", Line: 1},
			Command:  probe,
			When:     []*umpirespb.ActionClass{{Action: ir.GetActions()[i].GetId()}},
		})
	})
	r := run(t, m, unreachableBindings)
	require.Contains(t, r.subjects["standalone"], "scheduleToClose")
	require.NotContains(t, r.subjects["standalone"], "start-unset-unset-unset", "a start every path can take is reachable")
}

// fn-133 R6: a class of a performed action that a realizable path can take and no binding performs
// is uncovered: dropping the binding of the start that expires schedule-to-start uncovers it, and a
// class of no performed action is not counted.
func TestUncoveredClasses(t *testing.T) {
	uncovered := "start-unset-expires-unset"
	before := run(t, read(t, activityIR), uncoveredClasses)
	require.NotContains(t, before.subjects["standalone"], uncovered)

	m := read(t, activityIR, func(ir *umpirespb.Model) {
		for _, s := range realization(ir, "activitySystem").GetScripts() {
			for _, item := range s.GetItems() {
				item.Performs = slices.DeleteFunc(item.Performs, func(p *umpirespb.Performance) bool {
					in := p.GetStep().GetInputs()
					return p.GetStep().GetAction() == "temporal.features.activity.standalone.client.start" && len(in) == 3 &&
						in[1].GetEnum().GetCase() == "expires" && in[0].GetEnum().GetCase() == "unset" && in[2].GetEnum().GetCase() == "unset"
				})
			}
		}
	})
	after := run(t, m, uncoveredClasses)
	require.Contains(t, after.subjects["standalone"], uncovered)
	for _, s := range after.subjects["standalone"] {
		require.NotContains(t, []string{"scheduleToClose", "scheduleToStart", "startToClose"}, s, "a timer is the system's")
	}
}
