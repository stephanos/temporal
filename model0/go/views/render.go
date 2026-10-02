package views

import (
	"go.temporal.io/server/model/go/nexuscaller"
	"go.temporal.io/server/model/go/standaloneactivity"
	"go.temporal.io/server/model/go/umpire"
)

// All renders every checked-in view, by file name.
func All() (map[string]string, error) {
	out := map[string]string{}
	tables := map[string]umpire.Model{
		"nexusProduct": nexuscaller.NexusProduct, "nexusProtocol": nexuscaller.NexusProtocol,
		"activityProduct": standaloneactivity.ActivityProduct, "activityProtocol": standaloneactivity.ActivityProtocol,
	}
	for name, m := range tables {
		t, err := m.Table()
		if err != nil {
			return nil, err
		}
		out[name+"-table.md"] = Table(t)
		out[name+"-diagram.md"] = Diagram(t)
	}
	nexus, err := Summary(Declarations{Title: "Nexus caller",
		Machines: []umpire.Model{nexuscaller.NexusProduct, nexuscaller.NexusProtocol, nexuscaller.NexusCaller},
		Queries: append(append([]*umpire.Query{}, nexuscaller.FunctionalQueries...), nexuscaller.TerminalHolds,
			nexuscaller.StoppedWorkerRepliesNothing),
		Sets: []*umpire.Set{nexuscaller.NexusCallerTests, nexuscaller.NexusCallerCanary, nexuscaller.NexusCallerExploration}})
	if err != nil {
		return nil, err
	}
	out["nexusCaller-summary.md"] = nexus
	activity, err := Summary(Declarations{Title: "Standalone activity",
		Machines: []umpire.Model{standaloneactivity.ActivityProduct, standaloneactivity.ActivityProtocol,
			standaloneactivity.StandaloneActivity},
		Queries: append(append([]*umpire.Query{}, standaloneactivity.FunctionalQueries...), standaloneactivity.TerminalHolds,
			standaloneactivity.PauseHolds, standaloneactivity.StoppedWorkerStartsNothing),
		Sets: []*umpire.Set{standaloneactivity.StandaloneActivityTests, standaloneactivity.StandaloneActivityCanary,
			standaloneactivity.StandaloneActivityExploration}})
	if err != nil {
		return nil, err
	}
	out["standaloneActivity-summary.md"] = activity
	before, err := standaloneactivity.ProductWithoutControls().Table()
	if err != nil {
		return nil, err
	}
	after, err := standaloneactivity.ActivityProduct.Table()
	if err != nil {
		return nil, err
	}
	out["activityProduct-controls-diff.md"] = Diff("activityProduct: adding the caller's controls", before, after)
	return out, nil
}
