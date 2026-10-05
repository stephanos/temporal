package model

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// What the product machine disables is what the baseline pins by hand: a canceled answer without a
// cancel request, a worker stop, and a start of a paused activity.
func TestActivityDisabledBehaviorIsTheBaselines(t *testing.T) {
	product := built(t, activityModel(t))["activityProduct"]
	for _, pair := range [][2]string{
		{"started", "attemptResult-canceled"},
		{"paused", "attemptStart"},
		{"scheduled", "workerStop"},
		{"completed", "timeout"},
	} {
		require.True(t, product.Disabled(pair[0], pair[1]), pair)
	}
	require.False(t, product.Disabled("completed", "control-terminate"), "a control of an activity that is over is answered notFound, not disabled")
	require.Equal(t, []Result{{Outcome: "notFound", State: "completed", Facts: []string{}}},
		sideOf(product.Table).Rows[rowIndex(t, product.Table, "completed-control-terminate")].Results)
}

// Each machine's evidence lines are its fact type's cases in the order the IR declares them, each
// confirmed by the recorded kind of its own name.
func TestActivityEvidenceIsInCatalogOrder(t *testing.T) {
	m := activityModel(t)
	machines := built(t, m)
	for _, name := range []string{"activityProtocol", "activityProduct"} {
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
// beside the system contract's that are written there once: the competing timers'. Every declaration there is lifted, into the activity
// root or the system contract's, and every claim the activity root lifts is declared there. A capability declaration declares, for each law of the catalog whose capabilities
// it names, the law's Property, Scenario and Query, each named `<machine>.<law>`.
func TestActivityEveryClaimDeclarationIsLifted(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "model", "temporal", "features", "standaloneactivity")
	files := []string{"product/Product.scala", "system/System.scala"}
	var source []byte
	at := map[string]bool{}
	for _, file := range files {
		text, err := os.ReadFile(filepath.Join(dir, file))
		require.NoError(t, err)
		source = append(append(source, text...), '\n')
		at["model/temporal/features/standaloneactivity/"+file] = true
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
	// A law's instance is named after the val that declares its call: `val terminalIsFinal =
	// terminalStatesAreFinal(activityProduct)(…)`.
	laws := []string{}
	for _, file := range []string{"Close.scala", "Pause.scala", "Terminate.scala", "Cancel.scala"} {
		text, err := os.ReadFile(filepath.Join("..", "..", "..", "model", "temporal", "capabilities", file))
		require.NoError(t, err)
		for _, match := range regexp.MustCompile(`(?m)^object (\w+)\b`).FindAllStringSubmatch(string(text), -1) {
			laws = append(laws, match[1])
		}
	}
	require.NotEmpty(t, laws)
	law := regexp.MustCompile(`(?m)^[ \t]*val (\w+)\s*=\s*(?:\w+\.)*(` + strings.Join(laws, "|") + `)\(`)
	for _, match := range law.FindAllStringSubmatch(string(source), -1) {
		declared = append(declared, "property "+match[1])
	}
	// The catalog says which capabilities each law reads; a declaration's own capabilities are the
	// constructors it calls at the top of its argument list.
	var catalog struct {
		Catalog []struct {
			Law          string   `json:"law"`
			Capabilities []string `json:"capabilities"`
		} `json:"catalog"`
	}
	encoded, err := os.ReadFile(filepath.Join("..", "..", "..", "model", "ir", "activity.laws.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &catalog))
	require.NotEmpty(t, catalog.Catalog)
	// A machine object declares its own capabilities in its `laws`, `capabilities(limits = ...)(...)`,
	// named after the object it sits in.
	capabilityDeclarations := regexp.MustCompile(`(?m)^[ \t]*val \w+\s*=\s*capabilities\([^)]*\)\(`).FindAllStringSubmatchIndex(string(source), -1)
	require.Len(t, capabilityDeclarations, 2, "the product's and the protocol's")
	objects := regexp.MustCompile(`(?m)^object (\w+) extends Machine\b`).FindAllSubmatchIndex(source, -1)
	for _, at := range capabilityDeclarations {
		machine := ""
		for _, o := range objects {
			if o[0] < at[0] {
				name := string(source[o[2]:o[3]])
				machine = strings.ToLower(name[:1]) + name[1:]
			}
		}
		require.NotEmpty(t, machine)
		named := map[string]bool{}
		argument, depth := at[1], 1
		for i := argument; depth > 0; i++ {
			switch source[i] {
			case '(':
				if depth == 1 {
					constructor := regexp.MustCompile(`(\w+)\s*$`).FindSubmatch(source[argument:i])
					require.NotNil(t, constructor, machine)
					named[string(constructor[1])] = true
				}
				depth++
			case ')':
				depth--
			case ',':
				if depth == 1 {
					argument = i + 1
				}
			default:
			}
		}
		require.NotEmpty(t, named, machine)
		for _, entry := range catalog.Catalog {
			if !slices.ContainsFunc(entry.Capabilities, func(c string) bool { return !named[c] }) {
				for _, kind := range []string{"property", "scenario", "query"} {
					declared = append(declared, kind+" "+machine+"."+entry.Law)
				}
			}
		}
	}
	system, err := Load(activitySystemIR)
	require.NoError(t, err)
	lifted := map[string]bool{}
	for _, m := range []*umpirespb.Model{activityModel(t), system} {
		// The system contract's claims declared elsewhere, in record/, withTaskQueue/ and the task
		// queue, are not this file's.
		here := func(p *umpirespb.Position) bool { return m != system || at[p.GetFile()] }
		for _, p := range m.GetProperties() {
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
				require.Equal(t, !hasRow(mm, state+"-"+class.Key), mm.Disabled(state, class.Key), "%s-%s", state, class.Key)
			}
		}
		require.Equal(t, got.StatesTimesClass, got.DisabledPairs+len(got.Rows))
	}
}
