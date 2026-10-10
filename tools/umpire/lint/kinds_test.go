package lint

import (
	"maps"
	"slices"
	"sync"
	"testing"

	"go.temporal.io/server/common/testing/testpilot/duration"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

// The checked-in IR the kinds are read over, each loaded once and cloned before any mutation.
var (
	activityIR     = loaded("../../../model/ir/activity-standalone.json")
	nexusControlIR = loaded("../../../model/ir/nexus-workflow-control.json")
	capturedIR     = loaded("../../../model/irgen/testdata/lifts/expected/captured.json")
	declarationsIR = loaded("../../../model/irgen/testdata/lifts/expected/declarations.json")
)

func loaded(path string) func() (*umpirespb.Model, error) {
	return sync.OnceValues(func() (*umpirespb.Model, error) { return ir.Load(path) })
}

// unrealizedByMachine is the standing lowering gives a find Query in these tests: unrealized where no
// realization runs its Scenario's machine.
var unrealizedByMachine = Lowering{Unrealized: func(_ *umpirespb.Query, s *umpirespb.Scenario, rs []*umpirespb.Realization) (bool, error) {
	return !slices.ContainsFunc(rs, func(r *umpirespb.Realization) bool { return r.GetMachine() == s.GetMachine() }), nil
}}

// read is the Model of a loaded IR after the mutations, applied to a copy of it.
func read(t *testing.T, load func() (*umpirespb.Model, error), mutations ...func(*umpirespb.Model)) *Model {
	t.Helper()
	ir, err := load()
	require.NoError(t, err)
	ir = proto.CloneOf(ir)
	for _, mutate := range mutations {
		mutate(ir)
	}
	m, err := Of("", ir, unrealizedByMachine, Options{})
	require.NoError(t, err)
	return m
}

// reading is what a kind's tallies say, by owner: its population and its findings' subjects.
type reading struct {
	population map[string]int
	subjects   map[string][]string
}

func run(t *testing.T, m *Model, kind func(*Model) ([]Tally, error)) reading {
	t.Helper()
	tallies, err := kind(m)
	require.NoError(t, err)
	out := reading{population: map[string]int{}, subjects: map[string][]string{}}
	for _, x := range tallies {
		require.NotContains(t, out.population, x.Owner, "one tally per owner")
		require.LessOrEqual(t, len(x.Findings), x.Population)
		out.population[x.Owner] = x.Population
		for _, f := range x.Findings {
			require.Equal(t, x.Kind, f.Kind)
			require.Equal(t, x.Owner, f.Owner)
			require.NotEmpty(t, f.Message)
			require.NotEmpty(t, f.Position)
			out.subjects[x.Owner] = append(out.subjects[x.Owner], f.Subject)
		}
	}
	return out
}

func realization(ir *umpirespb.Model, machine string) *umpirespb.Realization {
	i := slices.IndexFunc(ir.GetRealizations(), func(r *umpirespb.Realization) bool { return r.GetMachine() == machine })
	return ir.GetRealizations()[i]
}

func activityRealizedPopulation(count int) map[string]int {
	return map[string]int{
		"activitySystem": count, "byIDCancellation": count, "byIDCompletion": count, "byIDFailure": count,
		"completion": count, "retryFailures": count, "cancellation": count,
		"pausing": count, "dispatchEligibility": count, "timeouts": count,
		"deferredReset": count, "heartbeatCompletion": count, "heartbeatExhaustion": count,
		"heartbeatRetry": count, "timeoutRetry": count,
	}
}

func activityMachinePopulation(product, system int) map[string]int {
	counts := activityRealizedPopulation(system)
	counts["activityProduct"] = product
	counts["resetKeepingPause"], counts["resetSettlement"] = system, system
	counts["activityWorker"], counts["polling"] = 2, 2
	return counts
}

func TestUnaskedProperties(t *testing.T) {
	r := run(t, read(t, activityIR), unaskedProperties)
	want := map[string]int{
		"activityProduct": 3, "activitySystem": 29, "standaloneActivity": 1,
		"completion": 1, "retryFailures": 3, "cancellation": 2,
		"pausing": 1, "dispatchEligibility": 3, "timeouts": 2,
		"byIDCancellation": 2, "byIDCompletion": 1, "byIDFailure": 1, "deferredReset": 1,
		"heartbeatCompletion": 1, "heartbeatExhaustion": 1, "heartbeatRetry": 1,
		"resetKeepingPause": 1, "resetSettlement": 9, "timeoutRetry": 2,
	}
	require.Equal(t, want, r.population)
	require.Empty(t, r.subjects)

	dropped := read(t, activityIR, func(ir *umpirespb.Model) {
		before := len(ir.Queries)
		ir.Queries = slices.DeleteFunc(ir.Queries, func(q *umpirespb.Query) bool {
			return q.GetName() == "terminate" && q.GetScenario().GetMachine() == "activitySystem"
		})
		require.Equal(t, 1, before-len(ir.Queries))
	})
	r = run(t, dropped, unaskedProperties)
	require.Equal(t, map[string][]string{"activitySystem": {"terminated"}}, r.subjects)
	require.Equal(t, want, r.population)
}

func TestUnfiredVerifies(t *testing.T) {
	r := run(t, read(t, activityIR), unfiredVerifies)
	require.Equal(t, map[string]int{"activityProduct": 3, "activitySystem": 27, "dispatchEligibility": 2, "retryFailures": 1, "standaloneActivity": 1}, r.population)
	require.Empty(t, r.subjects)

	// The product never enables stop, so a Property about its steps alone is read on none.
	unfired := read(t, activityIR, func(ir *umpirespb.Model) {
		changed := 0
		for _, p := range ir.GetProperties() {
			if p.GetName() == "activityProduct.closedIsRejectedUniformly" {
				p.When = &umpirespb.Property_WhenAction{WhenAction: "stop"}
				changed++
			}
		}
		require.Equal(t, 1, changed)
	})
	r = run(t, unfired, unfiredVerifies)
	require.Equal(t, map[string][]string{"activityProduct": {"activityProduct.closedIsRejectedUniformly"}}, r.subjects)
	require.Equal(t, 3, r.population["activityProduct"])
}

func TestUnperformedActions(t *testing.T) {
	r := run(t, read(t, activityIR), unperformedActions)
	// The worker answers and client controls are one action per RPC; the timers are the system's.
	// No performance binds poll: the activity script starts with it.
	want := map[string][]string{
		"activitySystem":      {"reset", "recordHeartbeat", "respondCompletedByID", "respondFailedByID", "respondCanceledByID"},
		"byIDCancellation":    {"pause", "unpause", "terminate", "reset", "recordHeartbeat", "respondCompleted", "respondFailed", "respondCanceled", "respondCompletedByID", "respondFailedByID", "stop"},
		"byIDCompletion":      {"pause", "unpause", "requestCancel", "terminate", "reset", "poll", "recordHeartbeat", "respondCompleted", "respondFailed", "respondCanceled", "respondFailedByID", "respondCanceledByID", "stop"},
		"byIDFailure":         {"pause", "unpause", "requestCancel", "terminate", "reset", "recordHeartbeat", "respondCompleted", "respondFailed", "respondCanceled", "respondCompletedByID", "respondCanceledByID", "stop"},
		"deferredReset":       {"pause", "unpause", "requestCancel", "terminate", "recordHeartbeat", "respondFailed", "respondCanceled", "respondCompletedByID", "respondFailedByID", "respondCanceledByID", "stop"},
		"heartbeatCompletion": {"pause", "unpause", "requestCancel", "terminate", "reset", "respondFailed", "respondCanceled", "respondCompletedByID", "respondFailedByID", "respondCanceledByID", "stop"},
		"heartbeatExhaustion": {"pause", "unpause", "requestCancel", "terminate", "reset", "respondFailed", "respondCanceled", "respondCompletedByID", "respondFailedByID", "respondCanceledByID", "stop"},
		"heartbeatRetry":      {"pause", "unpause", "requestCancel", "terminate", "reset", "respondFailed", "respondCanceled", "respondCompletedByID", "respondFailedByID", "respondCanceledByID", "stop"},
		"timeoutRetry":        {"pause", "unpause", "requestCancel", "terminate", "reset", "recordHeartbeat", "respondCanceled", "respondCompletedByID", "respondFailedByID", "respondCanceledByID", "stop"},
	}
	for _, owner := range []string{"completion", "retryFailures", "cancellation", "pausing", "dispatchEligibility", "timeouts"} {
		want[owner] = slices.Clone(want["activitySystem"])
	}
	require.Equal(t, activityRealizedPopulation(15), r.population)
	require.Equal(t, want, r.subjects)

	unperformed := read(t, activityIR, func(ir *umpirespb.Model) {
		removed := 0
		for _, s := range realization(ir, "activitySystem").GetScripts() {
			s.Items = slices.DeleteFunc(s.Items, func(item *umpirespb.Item) bool {
				match := slices.ContainsFunc(item.GetPerforms(), func(p *umpirespb.Performance) bool {
					return p.GetStep().GetAction() == "temporal.actors.worker.worker.stop"
				})
				if match {
					removed++
				}
				return match
			})
		}
		require.Equal(t, 1, removed)
	})
	r = run(t, unperformed, unperformedActions)
	mutant := maps.Clone(want)
	mutant["activitySystem"] = append(slices.Clone(want["activitySystem"]), "stop")
	require.Equal(t, mutant, r.subjects)
	require.Equal(t, activityRealizedPopulation(15), r.population)
}

func TestUnevidencedFacts(t *testing.T) {
	r := run(t, read(t, activityIR), unevidencedFacts)
	want := map[string][]string{
		"activitySystem":      {"heartbeatReceived", "heartbeatTimedOut"},
		"byIDCancellation":    {"statusPaused", "statusCompleted", "statusFailed", "statusTerminated", "statusTimedOut", "heartbeatReceived", "heartbeatTimedOut"},
		"byIDCompletion":      {"statusStarted", "statusPaused", "statusCancelRequested", "statusFailed", "statusCanceled", "statusTerminated", "statusTimedOut", "attemptCount", "heartbeatReceived", "heartbeatTimedOut"},
		"byIDFailure":         {"statusPaused", "statusCancelRequested", "statusCompleted", "statusCanceled", "statusTerminated", "statusTimedOut", "heartbeatReceived", "heartbeatTimedOut"},
		"deferredReset":       {"statusPaused", "statusCancelRequested", "statusFailed", "statusCanceled", "statusTerminated", "statusTimedOut", "heartbeatReceived"},
		"heartbeatCompletion": {"statusPaused", "statusCancelRequested", "statusFailed", "statusCanceled", "statusTerminated", "statusTimedOut", "heartbeatTimedOut"},
		"heartbeatExhaustion": {"statusPaused", "statusCancelRequested", "statusCompleted", "statusFailed", "statusCanceled", "statusTerminated"},
		"heartbeatRetry":      {"statusPaused", "statusCancelRequested", "statusFailed", "statusCanceled", "statusTerminated", "statusTimedOut"},
		"timeoutRetry":        {"statusPaused", "statusCancelRequested", "statusCanceled", "statusTerminated", "statusTimedOut", "heartbeatReceived", "heartbeatTimedOut"},
	}
	for _, owner := range []string{"completion", "retryFailures", "cancellation", "pausing", "dispatchEligibility", "timeouts"} {
		want[owner] = slices.Clone(want["activitySystem"])
	}
	require.Equal(t, activityRealizedPopulation(12), r.population)
	require.Equal(t, want, r.subjects)

	renamed := read(t, activityIR, func(ir *umpirespb.Model) {
		changed := 0
		for _, e := range realization(ir, "activitySystem").GetEvidence() {
			if e.GetRecords() == "statusPaused" {
				e.Records = "statusPausedElsewhere"
				changed++
			}
		}
		require.Equal(t, 1, changed)
	})
	r = run(t, renamed, unevidencedFacts)
	mutant := maps.Clone(want)
	mutant["activitySystem"] = append([]string{"statusPaused"}, want["activitySystem"]...)
	require.Equal(t, mutant, r.subjects)
	require.Equal(t, activityRealizedPopulation(12), r.population)

	// A machine with no realization is not counted, and a fact its evidence function names but no
	// evidence kind records is.
	r = run(t, read(t, capturedIR), unevidencedFacts)
	require.Equal(t, map[string]int{"store": 3}, r.population)
	require.Equal(t, map[string][]string{"store": {"stored", "staged", "lost"}}, r.subjects)
}

func TestUntakenChoices(t *testing.T) {
	r := run(t, read(t, nexusControlIR), untakenChoices)
	require.Equal(t, map[string]int{"trustingCaller": 2}, r.population)
	require.Empty(t, r.subjects)

	// The forged alternative behind a condition that never holds: its copy is still called, and no
	// reachable state takes it.
	r = run(t, read(t, nexusControlIR, func(ir *umpirespb.Model) {
		for _, f := range ir.GetFunctions() {
			if f.GetName() != "temporal.features.nexus.workflow.system.TrustingCaller$.effects$.forgedComplete" {
				continue
			}
			join := f.GetBody().GetIf().GetThen().GetBinary()
			join.Left = &umpirespb.Expr{Position: join.GetLeft().GetPosition(), Kind: &umpirespb.Expr_If{If: &umpirespb.If{
				Condition: &umpirespb.Expr{Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: false}}}},
				Then:      join.GetLeft(),
				Else:      &umpirespb.Expr{Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{}}},
			}}}
		}
	}), untakenChoices)
	require.Equal(t, map[string]int{"trustingCaller": 2}, r.population)
	require.Equal(t, map[string][]string{"trustingCaller": {"forged"}}, r.subjects)

	r = run(t, read(t, activityIR), untakenChoices)
	require.Equal(t, map[string]int{"activityProduct": 17}, r.population)
	require.Empty(t, r.subjects)
}

func TestUnreadRefinements(t *testing.T) {
	r := run(t, read(t, activityIR), unreadRefinements)
	population := map[string]int{}
	subjects := map[string][]string{}
	for _, owner := range []string{"activitySystem", "completion", "retryFailures", "cancellation", "pausing", "dispatchEligibility", "timeouts"} {
		population[owner] = 1
		subjects[owner] = []string{owner + " refines activityProduct"}
	}
	require.Equal(t, population, r.population)
	require.Equal(t, subjects, r.subjects)

	r = run(t, read(t, capturedIR), unreadRefinements)
	require.Equal(t, map[string]int{"disk": 1}, r.population)
	require.Empty(t, r.subjects)

	unread := read(t, capturedIR, func(ir *umpirespb.Model) {
		ir.Queries = slices.DeleteFunc(ir.Queries, func(q *umpirespb.Query) bool { return q.GetThrough() })
	})
	r = run(t, unread, unreadRefinements)
	require.Equal(t, map[string][]string{"disk": {"disk refines store"}}, r.subjects)
}

func TestUnreadObservations(t *testing.T) {
	r := run(t, read(t, activityIR), unreadObservations)
	want := activityRealizedPopulation(2)
	want["heartbeatCompletion"], want["heartbeatExhaustion"], want["heartbeatRetry"] = 3, 3, 3
	require.Equal(t, want, r.population)
	require.Empty(t, r.subjects)

	// A history read lifts its evidence into history-event.
	r = run(t, read(t, nexusControlIR), unreadObservations)
	require.Equal(t, map[string]int{"trustingCaller": 2}, r.population)
	require.Empty(t, r.subjects)

	unread := read(t, activityIR, func(ir *umpirespb.Model) {
		r := realization(ir, "activitySystem")
		spare := proto.CloneOf(r.GetObservations()[0])
		spare.Id = "spare-evidence"
		r.Observations = append(r.Observations, spare)
	})
	r = run(t, unread, unreadObservations)
	want["activitySystem"] = 3
	require.Equal(t, want, r.population)
	require.Equal(t, map[string][]string{"activitySystem": {"spare-evidence"}}, r.subjects)
}

// A read of a realization that declares an API behavior writes no interval: its wait is derived.
// One that writes its own is a finding, kept only where an acceptance records why; a realization that
// declares no behavior derives nothing, so its polls are not counted.
func TestExplicitWaits(t *testing.T) {
	r := run(t, read(t, activityIR), explicitWaits)
	want := activityRealizedPopulation(2)
	want["activitySystem"], want["byIDCancellation"], want["byIDCompletion"] = 6, 3, 1
	for _, owner := range []string{"completion", "retryFailures", "cancellation", "pausing", "dispatchEligibility", "timeouts"} {
		want[owner] = 6
	}
	require.Equal(t, want, r.population)
	require.Empty(t, r.subjects)

	explicit := func(ir *umpirespb.Model) {
		for _, c := range commandsNamed(realization(ir, "activitySystem"), "await-completed") {
			c.GetPoll().Interval = duration.FromMilliseconds(250)
		}
	}
	r = run(t, read(t, activityIR, explicit), explicitWaits)
	require.Equal(t, map[string][]string{"activitySystem": {"controller/await-completed"}}, r.subjects)

	r = run(t, read(t, activityIR, explicit, func(ir *umpirespb.Model) {
		r := realization(ir, "activitySystem")
		r.Behavior, r.ServerSteps = nil, nil
	}), explicitWaits)
	delete(want, "activitySystem")
	require.Equal(t, want, r.population)
}

// commandsNamed is every command of a realization with an id.
func commandsNamed(r *umpirespb.Realization, id string) []*umpirespb.Command {
	return slices.DeleteFunc(commands(r), func(c *umpirespb.Command) bool { return c.GetId() != id })
}

func TestUnreachableValues(t *testing.T) {
	r := run(t, read(t, activityIR), unreachableValues)
	require.Equal(t, activityMachinePopulation(9, 30), r.population)
	require.Empty(t, r.subjects)

	r = run(t, read(t, capturedIR), unreachableValues)
	// A channel's contents are a combination of deliveries, which no finding names.
	require.Equal(t, map[string][]string{"putOnly": {"stage=durable"}}, r.subjects)
	require.Equal(t, 3, r.population["putOnly"])
}

func TestNeverEnabled(t *testing.T) {
	r := run(t, read(t, activityIR), neverEnabled)
	want := activityMachinePopulation(115, 119)
	want["polling"] = 3
	require.Equal(t, want, r.population)
	// The product's worker stop is disabled in every state: its stop is the protocol's alone.
	require.Equal(t, map[string][]string{"activityProduct": {"stop"}}, r.subjects)
}

func TestStuckState(t *testing.T) {
	// The passing fixture: every reachable state of the activity's machines is an end or takes a step.
	r := run(t, read(t, activityIR), stuckStates)
	want := activityMachinePopulation(9, 1670)
	want["standaloneActivity"] = 3340
	require.Equal(t, want, r.population)
	require.Empty(t, r.subjects)

	// The finding fixture: putOnly is the disk without its flush, an internal step, so the staged disk
	// the put leaves is no end and nothing can happen in it. The disk itself flushes it.
	m := read(t, capturedIR)
	tallies, err := stuckStates(m)
	require.NoError(t, err)
	var findings []Finding
	for _, x := range tallies {
		findings = append(findings, x.Findings...)
	}
	require.Len(t, findings, 1)
	f := findings[0]
	require.Equal(t, []string{"putOnly", "staged"}, []string{f.Owner, f.Subject})
	require.Equal(t, where(m.machine("putOnly").GetPosition()), f.Position)
	require.Equal(t, "staged is reachable, is no end and enables no action class: no action can happen in it, so a timer or "+
		"an internal step may be missing a rule; if it is meant to be final, declare it in the machine's ends; "+
		"reached by empty -put/accepted-> staged", f.Message)
	table := m.Machines["putOnly"].Table
	require.NoError(t, table.Replay(table.PathTo("staged")), "the path is a witness of the table")

	// A state whose only pair is a hole is not stuck: the hole declares that unmodeled behavior may
	// happen there, as the progress check reads it.
	holed := read(t, declarationsIR, func(ir *umpirespb.Model) {
		ir.Queries, ir.Scenarios, ir.Progress = nil, nil, nil
		for _, d := range ir.GetMachines() {
			if d.GetName() == "disk" {
				d.Steps = slices.DeleteFunc(d.Steps, func(b *umpirespb.StepBinding) bool {
					return b.GetAction() == "fixture.declarations.flush"
				})
			}
		}
	})
	disk := holed.Machines["disk"]
	require.Empty(t, disk.Table.RowsFrom("staged"), "staged has no row")
	require.Len(t, disk.Holes, 1)
	require.Equal(t, []string{"staged", "crash"}, []string{disk.Holes[0].Source, disk.Holes[0].Class}, "its one pair is a hole")
	require.Contains(t, disk.Table.Reachable, "staged")
	require.NotContains(t, disk.Table.Ends, "staged")
	r = run(t, holed, stuckStates)
	require.Empty(t, r.subjects)
	require.Equal(t, 2, r.population["disk"])
}

func TestCompositionStuckState(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		t.Run(map[bool]string{false: "deadlock", true: "terminal"}[terminal], func(t *testing.T) {
			m := read(t, declarationsIR, func(ir *umpirespb.Model) {
				ir.Queries, ir.Scenarios, ir.Progress, ir.Properties = nil, nil, nil, nil
				ir.Compositions = slices.DeleteFunc(ir.Compositions, func(c *umpirespb.Composition) bool { return c.GetName() != "pair" })
				if !terminal {
					ir.Compositions[0].Ends.GetLambda().Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Literal{
						Literal: &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: false}}}}
				}
			})
			r := run(t, m, stuckStates)
			require.Equal(t, 2, r.population["pair"])
			require.NotContains(t, r.subjects, "store", "each member's held state is terminal")
			if terminal {
				require.Empty(t, r.subjects)
				return
			}
			require.Equal(t, map[string][]string{"pair": {"held_held"}}, r.subjects)
			tallies, err := stuckStates(m)
			require.NoError(t, err)
			i := slices.IndexFunc(tallies, func(x Tally) bool { return x.Owner == "pair" })
			require.Len(t, tallies[i].Findings, 1)
			f := tallies[i].Findings[0]
			require.Equal(t, where(m.IR.Compositions[0].GetPosition()), f.Position)
			require.Contains(t, f.Message, "reached by nothing_nothing -putBoth/front_accepted-> held_held")
			table, err := m.realizer.TransitionTable("pair")
			require.NoError(t, err)
			witness := table.PathTo("held_held")
			require.Len(t, witness.Steps, 1, "one synchronized step is the shortest path from the distinct start")
			require.NoError(t, table.Replay(witness))
		})
	}
}

func TestCompositionStuckStateWithUnknownReplacement(t *testing.T) {
	m := read(t, declarationsIR, func(ir *umpirespb.Model) {
		ir.Queries, ir.Scenarios, ir.Progress, ir.Properties = nil, nil, nil, nil
		ir.Compositions = slices.DeleteFunc(ir.Compositions, func(c *umpirespb.Composition) bool { return c.GetName() != "detailedPair" })
		ir.Compositions[0].Ends = nil
		for _, d := range ir.GetMachines() {
			if d.GetName() == "disk" {
				d.Steps = slices.DeleteFunc(d.Steps, func(b *umpirespb.StepBinding) bool { return b.GetAction() == "fixture.declarations.flush" })
			}
		}
	})
	_, err := m.realizer.Composition("detailedPair")
	require.Error(t, err, "the replacement cannot be established through its reachable hole")
	r := run(t, m, stuckStates)
	require.Equal(t, 2, r.population["detailedPair"], "the constructible composition is still inspected")
	require.Empty(t, r.subjects, "the member's hole is an unknown composed pair, not a deadlock")
}

func TestCompositionStuckStateWithRejectedReplacement(t *testing.T) {
	m := read(t, declarationsIR, func(ir *umpirespb.Model) {
		ir.Queries, ir.Scenarios, ir.Progress, ir.Properties = nil, nil, nil, nil
		ir.Compositions = slices.DeleteFunc(ir.Compositions, func(c *umpirespb.Composition) bool { return c.GetName() != "detailedPair" })
		ir.Compositions[0].Ends = nil
		for _, d := range ir.GetMachines() {
			if d.GetName() == "store" {
				d.Starts[0].GetConstruct().Args[0].GetLiteral().GetEnum().Case = "held"
			}
		}
	})
	_, err := m.realizer.Composition("detailedPair")
	var rejected *check.RefinementError
	require.ErrorAs(t, err, &rejected)
	r := run(t, m, stuckStates)
	require.Equal(t, 1, r.population["detailedPair"])
	require.Equal(t, map[string][]string{"detailedPair": {"held_empty"}}, r.subjects)
	require.True(t, slices.ContainsFunc(m.Verified.Receipts, func(r check.Receipt) bool {
		return r.Subject == check.CompositionSubject && r.Key.Owner == "detailedPair" && r.Kind == check.RefinementRejected
	}), "the refinement diagnostic is retained")
}

func TestCompositionStuckStateConstructionFailure(t *testing.T) {
	m := read(t, declarationsIR, func(ir *umpirespb.Model) {
		ir.Queries, ir.Scenarios, ir.Progress, ir.Properties = nil, nil, nil, nil
		ir.Compositions = slices.DeleteFunc(ir.Compositions, func(c *umpirespb.Composition) bool { return c.GetName() != "pair" })
		ir.Compositions[0].Ends.GetLambda().Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Literal{
			Literal: &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: 1}}}}
	})
	_, err := stuckStates(m)
	require.ErrorContains(t, err, "pair: ends is 1 at held_held, not a Boolean")
}

func TestStuckStateWithRefinementMapHole(t *testing.T) {
	m := read(t, declarationsIR, func(ir *umpirespb.Model) {
		ir.Queries, ir.Scenarios, ir.Progress, ir.Properties = nil, nil, nil, nil
		for _, d := range ir.GetMachines() {
			if d.GetName() != "disk" {
				continue
			}
			for _, f := range ir.GetFunctions() {
				if f.GetName() == d.GetRefines().GetMap() {
					f.Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Hole{Hole: "fixture.declarations.crashUnmodeled"}}
				}
			}
		}
	})
	for _, owner := range []string{"disk", "detailedPair"} {
		t.Run(owner, func(t *testing.T) {
			table, err := m.realizer.TransitionTable(owner)
			require.NoError(t, err, "a refinement-map hole leaves transitions available")
			require.Len(t, table.Reachable, 3)
		})
	}
	r := run(t, m, stuckStates)
	require.Equal(t, map[string]int{"disk": 3, "store": 2, "detailedPair": 3, "pair": 2}, r.population)
	require.Empty(t, r.subjects)
	require.True(t, slices.ContainsFunc(m.Verified.Receipts, func(r check.Receipt) bool {
		return r.Subject == check.RefinementSubject && r.Key.Owner == "disk" && r.Kind == check.Incomplete && len(r.Holes) > 0
	}), "the refinement-map hole remains an explicit diagnostic")
}

func TestUnproduced(t *testing.T) {
	r := run(t, read(t, activityIR), unproduced)
	want := activityMachinePopulation(16, 20)
	want["activityWorker"], want["polling"] = 5, 5
	require.Equal(t, want, r.population)
	subjects := map[string][]string{}
	for owner := range want {
		subjects[owner] = []string{"outcome rejected-alreadyExists"}
	}
	for _, owner := range []string{"activityWorker", "polling"} {
		subjects[owner] = []string{"outcome rejected-notFound", "outcome rejected-alreadyExists", "outcome rejected-failedPrecondition", "outcome rejected-invalidArgument"}
	}
	require.Equal(t, subjects, r.subjects)

	r = run(t, read(t, capturedIR), unproduced)
	require.Equal(t, map[string][]string{
		"disk":    {"fact lost-false", "fact lost-true"},
		"putOnly": {"outcome deferred", "fact staged", "fact lost-false", "fact lost-true"},
		"store":   {"fact staged", "fact lost-true"},
	}, r.subjects)
	require.Equal(t, 6, r.population["putOnly"])
}

func TestUnrealizedFinds(t *testing.T) {
	r := run(t, read(t, activityIR), unrealizedFinds)
	want := activityRealizedPopulation(1)
	want["activitySystem"], want["byIDCancellation"], want["timeoutRetry"] = 2, 2, 2
	want["retryFailures"], want["cancellation"], want["timeouts"] = 2, 2, 2
	want["resetKeepingPause"], want["resetSettlement"] = 1, 9
	require.Equal(t, want, r.population)
	require.Equal(t, map[string][]string{
		"resetKeepingPause": {"keepPausedReset"},
		"resetSettlement":   {"resetCancellation", "resetCompletion", "resetExhaustion", "resetFatality", "resetKeptPause", "resetOutranksPause", "resetRepeated", "resetScheduleToClose", "resetTimeout"},
	}, r.subjects)

	r = run(t, read(t, capturedIR), unrealizedFinds)
	require.Equal(t, map[string]int{"detailedPair": 1, "disk": 1, "store": 1}, r.population)
	require.Equal(t, map[string][]string{"detailedPair": {"bothPutHeld"}, "disk": {"disk.any.everPut"}}, r.subjects)

	m := read(t, activityIR)
	m.lowering = Lowering{}
	_, err := unrealizedFinds(m)
	require.ErrorContains(t, err, "lowering's standing")
}
