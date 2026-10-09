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

type activityClaimKey struct {
	Family, Owner, Kind, Name string
}

func activityClaimDifference(declared, lifted []activityClaimKey) (missing, unexpected []activityClaimKey) {
	counts := map[activityClaimKey]int{}
	for _, key := range declared {
		counts[key]++
	}
	for _, key := range lifted {
		if counts[key] == 0 {
			unexpected = append(unexpected, key)
		} else {
			counts[key]--
		}
	}
	for key, count := range counts {
		for range count {
			missing = append(missing, key)
		}
	}
	return missing, unexpected
}

func activitySourceClaims(t *testing.T, source string) ([]activityClaimKey, int) {
	t.Helper()
	packages := regexp.MustCompile(`(?m)^package ([\w.]+)\s*$`).FindAllStringSubmatch(source, -1)
	var parts []string
	for _, match := range packages {
		parts = append(parts, match[1])
	}
	family := strings.Join(parts, ".")
	require.NotEmpty(t, family)
	objects := regexp.MustCompile(`(?m)^object (\w+)\s+extends\b`).FindAllStringSubmatchIndex(source, -1)
	declarations := regexp.MustCompile(`(?m)^[ \t]*val (\w+)\s*=\s*\(?\s*(query|(?:\w+\.)?property|(?:\w+\.)?scenario)\b`)
	constructors := regexp.MustCompile(`(?:\.|\b)(property|scenario|query)\(\s*"([^"\n]+)"`)
	explicit := regexp.MustCompile(`^\s*\(\s*"([^"\n]+)"`)
	var declared []activityClaimKey
	sections := 0
	for i, object := range objects {
		name := source[object[2]:object[3]]
		owner := strings.ToLower(name[:1]) + name[1:]
		end := len(source)
		if i+1 < len(objects) {
			end = objects[i+1][0]
		}
		body := source[object[1]:end]
		for _, match := range constructors.FindAllStringSubmatch(body, -1) {
			declared = append(declared, activityClaimKey{family, owner, match[1], match[2]})
		}
		for _, match := range declarations.FindAllStringSubmatchIndex(body, -1) {
			if explicit.MatchString(body[match[1]:]) {
				continue
			}
			name := body[match[2]:match[3]]
			kind := body[match[4]:match[5]]
			if _, suffix, qualified := strings.Cut(kind, "."); qualified {
				kind = suffix
			}
			declared = append(declared, activityClaimKey{family, owner, kind, name})
		}
		expanded, count := activityCapabilityClaims(t, body, family, owner)
		declared = append(declared, expanded...)
		sections += count
	}
	require.NotEmpty(t, objects)
	return declared, sections
}

func activityCapabilityClaims(t *testing.T, source, family, machine string) ([]activityClaimKey, int) {
	t.Helper()
	var declared []activityClaimKey
	capabilityProperties := map[string][]string{
		"terminalStatesAreFinal":    {"Closable"},
		"closedIsRejectedUniformly": {"Closable"},
		"pausedIsNotDispatched":     {"Pausable", "Pollable"},
		"terminateSettles":          {"Terminable"},
	}
	// Each capabilities section brings its companions' Properties; the two-capability Property
	// needs both declared kinds. The Scala test holds this list to the companions' definitions.
	sections := regexp.MustCompile(`(?ms)^[ \t]*object capabilities extends Capabilities:\n(.*?)(?:^[ \t]*object queries:|\z)`).FindAllStringSubmatchIndex(source, -1)
	for _, section := range sections {
		named := map[string]bool{}
		body := source[section[2]:section[3]]
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
					declared = append(declared, activityClaimKey{family, machine, claim, machine + "." + name + "." + property})
				}
			}
		}
		require.NotEmpty(t, named, machine)
		for property, required := range capabilityProperties {
			if !slices.ContainsFunc(required, func(kind string) bool { return !named[kind] }) {
				for _, kind := range []string{"property", "scenario", "query"} {
					declared = append(declared, activityClaimKey{family, machine, kind, machine + "." + property})
				}
			}
		}
	}
	return declared, len(sections)
}

// Each explicitly listed source belongs to its declared IR export. Timeouts.scala owns two
// subjects, with only CompetingTimeouts rooted in the record export.
func TestActivityEveryClaimDeclarationIsLifted(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "model", "temporal", "features", "activity", "standalone")
	for _, exported := range []struct {
		ir           string
		files        map[string][]string
		capabilities int
	}{
		{
			ir: "activity-standalone.json",
			files: map[string][]string{
				"product/Product.scala":            nil,
				"system/System.scala":              nil,
				"system/Completion.scala":          nil,
				"system/RetryFailures.scala":       nil,
				"system/Cancellation.scala":        nil,
				"system/Pausing.scala":             nil,
				"system/DispatchEligibility.scala": nil,
				"system/Timeouts.scala":            {"timeouts"},
				"system/RetryTimeouts.scala":       nil,
				"system/Heartbeat.scala":           nil,
				"system/ResponseByID.scala":        nil,
				"system/Reset.scala":               nil,
				"system/DispatchWithWorker.scala":  nil,
			},
			capabilities: 2,
		},
		{
			ir:    "activity-standalone-record.json",
			files: map[string][]string{"system/Timeouts.scala": {"competingTimeouts"}},
		},
	} {
		t.Run(exported.ir, func(t *testing.T) {
			var declared []activityClaimKey
			at := map[string]bool{}
			sections := 0
			for _, file := range slices.Sorted(maps.Keys(exported.files)) {
				text, err := os.ReadFile(filepath.Join(dir, file))
				require.NoError(t, err)
				claims, count := activitySourceClaims(t, string(text))
				sections += count
				for _, key := range claims {
					owners := exported.files[file]
					if len(owners) == 0 || slices.Contains(owners, key.Owner) {
						declared = append(declared, key)
					}
				}
				at["model/temporal/features/activity/standalone/"+file] = true
			}
			require.Equal(t, exported.capabilities, sections)
			m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", exported.ir))
			require.NoError(t, err)
			families := map[string]string{}
			for _, machine := range m.GetMachines() {
				families[machine.GetName()] = machine.GetFamily()
			}
			for _, composition := range m.GetCompositions() {
				families[composition.GetName()] = composition.GetFamily()
			}
			var lifted []activityClaimKey
			claim := func(position *umpirespb.Position, owner, kind, name string) {
				if exported.ir == "activity-standalone.json" || at[position.GetFile()] {
					require.NotEmpty(t, families[owner], owner)
					lifted = append(lifted, activityClaimKey{families[owner], owner, kind, name})
				}
			}
			for _, p := range m.GetProperties() {
				if strings.HasPrefix(p.GetName(), p.GetMachine()+".") {
					require.NotNil(t, p.GetOrigin(), p.GetName())
					require.True(t, strings.HasPrefix(p.GetOrigin().GetName(), "temporal.capabilities."), p.GetName())
					require.NotEmpty(t, p.GetOrigin().GetPosition().GetFile(), p.GetName())
				}
				claim(p.GetPosition(), p.GetMachine(), "property", p.GetName())
			}
			for _, s := range m.GetScenarios() {
				claim(s.GetPosition(), s.GetMachine(), "scenario", s.GetName())
			}
			for _, q := range m.GetQueries() {
				claim(q.GetPosition(), q.GetScenario().GetMachine(), "query", q.GetName())
			}
			require.NotEmpty(t, declared)
			missing, unexpected := activityClaimDifference(declared, lifted)
			require.Empty(t, missing, "source declarations missing from %s", exported.ir)
			require.Empty(t, unexpected, "claims absent from the source inventory for %s", exported.ir)
		})
	}
}

func TestActivityClaimCoverageRejectsMissingOwnerCrossedOwnerAndDroppedName(t *testing.T) {
	const source = `package temporal
package features.activity.standalone.system
object Completion extends Derived(ActivitySystem.rebind()):
  object properties:
    val completes = property when worker.respondCompleted holds (_.records(Fact.statusCompleted))
object Pausing extends Derived(ActivitySystem.rebind()):
  object properties:
    val completes = property when worker.respondCompleted holds (_.records(Fact.statusCompleted))
object DispatchEligibility
    extends Derived(ActivitySystem.rebind()):
  object properties:
    val completes = property when worker.respondCompleted holds (_.records(Fact.statusCompleted))
object ByIDCancellation extends Derived(ActivitySystem.unmonitored):
  object properties:
    val cancelIsRequested = property("activitySystem.cancelIsRequested") holds (_.records(Fact.statusCancelRequested))
  object queries:
    val heldCancellation = scenario.actions(client.requestCancel)
    val cancelIsRequested =
      (query(
        "activitySystem.cancelIsRequested"
      ) find properties.cancelIsRequested in heldCancellation limits four)
`
	declared, sections := activitySourceClaims(t, source)
	require.Zero(t, sections)
	family := "temporal.features.activity.standalone.system"
	require.ElementsMatch(t, []activityClaimKey{
		{family, "completion", "property", "completes"},
		{family, "pausing", "property", "completes"},
		{family, "dispatchEligibility", "property", "completes"},
		{family, "byIDCancellation", "property", "activitySystem.cancelIsRequested"},
		{family, "byIDCancellation", "scenario", "heldCancellation"},
		{family, "byIDCancellation", "query", "activitySystem.cancelIsRequested"},
	}, declared)
	for _, test := range []struct {
		name                string
		change              func([]activityClaimKey) []activityClaimKey
		missing, unexpected activityClaimKey
	}{
		{
			name: "one completes owner is missing",
			change: func(keys []activityClaimKey) []activityClaimKey {
				return slices.DeleteFunc(keys, func(key activityClaimKey) bool {
					return key.Owner == "pausing" && key.Kind == "property" && key.Name == "completes"
				})
			},
			missing: activityClaimKey{family, "pausing", "property", "completes"},
		},
		{
			name: "a declaration crosses an owner",
			change: func(keys []activityClaimKey) []activityClaimKey {
				for i := range keys {
					if keys[i].Owner == "pausing" {
						keys[i].Owner = "completion"
					}
				}
				return keys
			},
			missing:    activityClaimKey{family, "pausing", "property", "completes"},
			unexpected: activityClaimKey{family, "completion", "property", "completes"},
		},
		{
			name: "an explicit constructor name is lost",
			change: func(keys []activityClaimKey) []activityClaimKey {
				for i := range keys {
					if keys[i].Owner == "byIDCancellation" && keys[i].Kind == "query" {
						keys[i].Name = "cancelIsRequested"
					}
				}
				return keys
			},
			missing:    activityClaimKey{family, "byIDCancellation", "query", "activitySystem.cancelIsRequested"},
			unexpected: activityClaimKey{family, "byIDCancellation", "query", "cancelIsRequested"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			missing, unexpected := activityClaimDifference(declared, test.change(slices.Clone(declared)))
			require.Equal(t, []activityClaimKey{test.missing}, missing)
			if test.unexpected == (activityClaimKey{}) {
				require.Empty(t, unexpected)
			} else {
				require.Equal(t, []activityClaimKey{test.unexpected}, unexpected)
			}
		})
	}
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
