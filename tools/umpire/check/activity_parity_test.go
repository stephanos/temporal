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

// What the product machine disables is what the baseline pins by hand: a canceled answer without a
// cancel request, a worker stop, and a start of a paused activity.
func TestActivityDisabledBehaviorIsTheBaselines(t *testing.T) {
	product := built(t, activityModel(t))["activityProduct"]
	for _, pair := range [][2]string{
		{"started", "respondCanceled"},
		{"paused", "poll"},
		{"scheduled", "stop"},
		{"completed", "timeout"},
	} {
		require.True(t, disabled(product, pair[0], pair[1]), pair)
	}
	require.False(t, disabled(product, "completed", "terminate"), "a control of an activity that is over is answered notFound, not disabled")
	require.Equal(t, []interp.Result{{Outcome: "notFound", State: "completed", Facts: []string{}}},
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

// The activity's claims are declared in its levels' files, product/Product.scala and
// system/System.scala, in the `properties`, `laws` and `queries` objects of its machine objects,
// beside the system contract's that are written there once: the competing timers'. Every
// declaration there is lifted, into the activity root or the system contract's, and every claim the
// activity root lifts is declared there. A capabilities section brings its companions' Properties,
// each with a Scenario and Query named `<machine>.<property>`.
func TestActivityEveryClaimDeclarationIsLifted(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "model", "temporal", "features", "activity", "standalone")
	files := []string{"product/Product.scala", "system/System.scala"}
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
	for _, match := range regexp.MustCompile(`(?m)^[ \t]*val (\w+)\s*=\s*\(?\s*(?:(query)\b|(?:\w+\.)?(property|scenario)\b)`).FindAllStringSubmatch(string(source), -1) {
		declared = append(declared, match[2]+match[3]+" "+match[1])
	}
	capabilityProperties := map[string][]string{
		"terminalStatesAreFinal":    {"Closable"},
		"closedIsRejectedUniformly": {"Closable"},
		"pausedIsNotDispatched":     {"Pausable", "Pollable"},
		"terminateSettles":          {"Terminable"},
		"cancelIsRequested":         {"Cancelable"},
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
		for _, match := range regexp.MustCompile(`val \w+: Capability\s*=\s*(\w+)\(`).FindAllStringSubmatch(body, -1) {
			named[match[1]] = true
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
		// The system contract's claims declared elsewhere, in system/Record.scala,
		// system/WithTaskQueue.scala and the task queue, are not these files'.
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
		// The interpreter's own account of a disabled pair agrees with the rows.
		for _, state := range mm.Table.States {
			for _, class := range mm.Classes {
				require.Equal(t, !hasRow(mm, state+"-"+class.Key), disabled(mm, state, class.Key), "%s-%s", state, class.Key)
			}
		}
		require.Equal(t, got.StatesTimesClass, got.DisabledPairs+len(got.Rows))
	}
}
