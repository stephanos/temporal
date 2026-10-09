package check

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
)

// The product rejects a canceled answer without a cancel request, and keeps disabled the worker
// poll of a paused activity, worker stop and terminal timeout.
func TestActivityDisabledBehaviorIsTheBaselines(t *testing.T) {
	product := built(t, activityModel(t))["activityProduct"]
	for _, pair := range [][2]string{
		{"paused", "poll"},
		{"scheduled", "stop"},
		{"completed", "timeout"},
	} {
		require.True(t, disabled(product, pair[0], pair[1]), pair)
	}
	require.Equal(t, []interp.Result{{Outcome: "rejected-invalidArgument", State: "started", Facts: []string{},
		Because: "cancellation was not requested (chasm/lib/activity/model/model.go:171)"}},
		sideOf(product.Table).Rows[rowIndex(t, product.Table, "started-respondCanceled")].Results)
	require.False(t, disabled(product, "completed", "terminate"), "a control of an activity that is over is answered notFound, not disabled")
	require.Equal(t, []interp.Result{{Outcome: "rejected-notFound", State: "completed", Facts: []string{}}},
		sideOf(product.Table).Rows[rowIndex(t, product.Table, "completed-terminate")].Results)
}

// Each machine's evidence lines are its fact type's cases in the order the IR declares them, each
// confirmed by the recorded kind of its own name.
func TestActivityEvidenceIsInCatalogOrder(t *testing.T) {
	m := activityModel(t)
	machines := built(t, m)
	for _, name := range []string{"activitySystem", "activityProduct"} {
		var catalog [][2]string
		for _, c := range admType(m, machines[name].Decl.GetFactType()).GetEnum().GetCases() {
			catalog = append(catalog, [2]string{c.GetName(), c.GetName()})
		}
		require.NotEmpty(t, catalog, name)
		require.Equal(t, catalog, machines[name].Table.Evidence, name)
	}
}

// The activity's claims are declared in product/Product.scala and the lifecycle subject files beside
// system/System.scala, in the `properties`, `capabilities` and `queries` objects of its machine objects,
// beside the system contract's that are written there once: the competing timers'. Every
// declaration there is lifted, into the activity root or the system contract's, and every claim the
// activity root lifts is declared there. A capabilities section brings its companions' Properties,
// each with a Scenario and Query named `<machine>.<property>`.
func TestActivityEveryClaimDeclarationIsLifted(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "model", "temporal", "features", "activity", "standalone")
	files := []string{
		"product/Product.scala", "system/System.scala", "system/RetryTimeouts.scala", "system/Heartbeat.scala",
		"system/ResponseByID.scala", "system/Reset.scala", "system/DispatchWithWorker.scala",
	}
	var source []byte
	at := map[string]bool{}
	for _, file := range files {
		text, err := os.ReadFile(filepath.Join(dir, file))
		require.NoError(t, err)
		source = append(append(source, text...), '\n')
		at["model/temporal/features/activity/standalone/"+file] = true
	}
	declared := []string{}
	for _, match := range regexp.MustCompile(`(?:\.|\b)(property|scenario|query)\(\s*"([^"\n]+)"`).FindAllStringSubmatch(string(source), -1) {
		declared = append(declared, match[1]+" "+match[2])
	}
	// A declaration that states no name is named after its val, which may be a member of an object.
	// Inside a machine object, `property` and `scenario` name the object they are inherited by.
	unnamed := regexp.MustCompile(`(?m)^[ \t]*val (\w+)\s*=\s*\(?\s*(?:(query)\b|(?:\w+\.)?(property|scenario)\b)`)
	for _, match := range unnamed.FindAllStringSubmatchIndex(string(source), -1) {
		if regexp.MustCompile(`^\s*\(\s*"`).Match(source[match[1]:]) {
			continue // Explicit constructor names, including overrides, were counted above.
		}
		kind := ""
		for _, index := range []int{4, 6} {
			if match[index] != -1 {
				kind = string(source[match[index]:match[index+1]])
			}
		}
		declared = append(declared, kind+" "+string(source[match[2]:match[3]]))
	}
	capabilityProperties := map[string][]string{
		"terminalStatesAreFinal":    {"Closable"},
		"closedIsRejectedUniformly": {"Closable"},
		"pausedIsNotDispatched":     {"Pausable", "Pollable"},
		"terminateSettles":          {"Terminable"},
	}
	// Each capabilities section brings its companions' Properties; the two-capability Property
	// needs both declared kinds. The Scala test holds this list to the companions' definitions.
	sections := regexp.MustCompile(`(?ms)^[ \t]*object capabilities extends Capabilities:\n(.*?)(?:^[ \t]*object queries:|\z)`).FindAllStringSubmatchIndex(string(source), -1)
	require.Len(t, sections, 2, "the product's and the protocol's")
	objects := regexp.MustCompile(`(?m)^object (\w+) extends Machine\b`).FindAllSubmatchIndex(source, -1)
	for _, section := range sections {
		machine := ""
		for _, object := range objects {
			if object[0] < section[0] {
				name := string(source[object[2]:object[3]])
				machine = strings.ToLower(name[:1]) + name[1:]
			}
		}
		require.NotEmpty(t, machine)
		named := map[string]bool{}
		body := string(source[section[2]:section[3]])
		bindings := regexp.MustCompile(`(?m)^[ \t]*val (\w+): Capability\s*=\s*(\w+)(?:\[[^\]]+\])?\(`).FindAllStringSubmatchIndex(body, -1)
		for i, match := range bindings {
			end := len(body)
			if i+1 < len(bindings) {
				end = bindings[i+1][0]
			}
			name, kind, parameters := body[match[2]:match[3]], body[match[4]:match[5]], body[match[1]:end]
			named[kind] = true
			var properties []string
			switch kind {
			case "Closable", "Pausable", "Pollable", "Terminable", "Describable":
				// These companions are accounted for by the capability sets below.
			case "Retries":
				properties = []string{"failureReturnsToWaiting", "failureEndsFailed", "failurePauses", "failureCancels", "attemptCountIsWithinPolicy"}
			case "Deadline":
				properties = []string{"firesInWindow", "deadlineTimesOut"}
				if strings.Contains(parameters, "retryable = true") {
					properties = append(properties, "deadlineReturnsToWaiting", "deadlinePauses")
				}
			default:
				require.FailNowf(t, "unknown capability binding", "%s: %s", machine, kind)
			}
			for _, property := range properties {
				for _, claim := range []string{"property", "scenario", "query"} {
					declared = append(declared, claim+" "+machine+"."+name+"."+property)
				}
			}
		}
		require.NotEmpty(t, named, machine)
		for property, required := range capabilityProperties {
			if !slices.ContainsFunc(required, func(kind string) bool { return !named[kind] }) {
				for _, kind := range []string{"property", "scenario", "query"} {
					declared = append(declared, kind+" "+machine+"."+property)
				}
			}
		}
	}
	system, err := ir.Load(activitySystemIR)
	require.NoError(t, err)
	lifted := map[string]bool{}
	for _, m := range []*umpirespb.Model{activityModel(t), system} {
		// The system contract's claims declared elsewhere, in system/Dispatch.scala,
		// system/DispatchRaces.scala, system/DispatchWithTaskQueue.scala and the task queue, are not these files'.
		here := func(p *umpirespb.Position) bool { return m != system || at[p.GetFile()] }
		for _, p := range m.GetProperties() {
			if strings.HasPrefix(p.GetName(), p.GetMachine()+".") {
				require.NotNil(t, p.GetOrigin(), p.GetName())
				require.True(t, strings.HasPrefix(p.GetOrigin().GetName(), "temporal.capabilities."), p.GetName())
				require.NotEmpty(t, p.GetOrigin().GetPosition().GetFile(), p.GetName())
			}
			if here(p.GetPosition()) {
				lifted["property "+p.GetName()] = true
			}
		}
		for _, s := range m.GetScenarios() {
			if here(s.GetPosition()) {
				lifted["scenario "+s.GetName()] = true
			}
		}
		for _, q := range m.GetQueries() {
			if here(q.GetPosition()) {
				lifted["query "+q.GetName()] = true
			}
		}
	}
	require.NotEmpty(t, declared)
	require.ElementsMatch(t, declared, slices.Collect(maps.Keys(lifted)))
}

func TestActivityTablesAccountForEveryPair(t *testing.T) {
	b := bind(activityModel(t), DefaultScope)
	for name, mm := range built(t, activityModel(t)) {
		require.Empty(t, mm.Holes, name)
		if mm.Decl.GetRefines() != nil {
			require.NoError(t, b.refinement(b.subject(name)).err, name)
		}
		got := sideOf(mm.Table)
		rows := make(map[string]bool, len(mm.Table.Rows))
		for _, row := range mm.Table.Rows {
			require.False(t, rows[row.Key], "%s has duplicate row %s", name, row.Key)
			rows[row.Key] = true
		}
		require.Len(t, rows, len(mm.Table.Rows), name)
		// The interpreter's own account of a disabled pair agrees with the rows.
		for _, state := range mm.Table.States {
			for _, class := range mm.Classes {
				require.Equal(t, !rows[state+"-"+class.Key], disabled(mm, state, class.Key), "%s-%s", state, class.Key)
			}
		}
		require.Equal(t, got.StatesTimesClass, got.DisabledPairs+len(got.Rows))
	}
}
