package lint

import (
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
)

// The checked-in IR the kinds are read over, each loaded once and cloned before any mutation.
var (
	activityIR     = loaded("../../../model/ir/activity.json")
	nexusControlIR = loaded("../../../model/ir/nexus-control.json")
	capturedIR     = loaded("../../../model/irgen/testdata/lifts/expected/captured.json")
)

func loaded(path string) func() (*umpirespb.Model, error) {
	return sync.OnceValues(func() (*umpirespb.Model, error) { return model.Load(path) })
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

func TestUnaskedProperties(t *testing.T) {
	r := run(t, read(t, activityIR), unaskedProperties)
	require.Equal(t, map[string]int{"activityProduct": 3, "activityProtocol": 10, "standaloneActivity": 1}, r.population)
	require.Empty(t, r.subjects)

	dropped := read(t, activityIR, func(ir *umpirespb.Model) {
		ir.Queries = slices.DeleteFunc(ir.Queries, func(q *umpirespb.Query) bool { return q.GetName() == "terminate" })
	})
	r = run(t, dropped, unaskedProperties)
	require.Equal(t, map[string][]string{"activityProtocol": {"terminated"}}, r.subjects)
	require.Equal(t, 10, r.population["activityProtocol"])
}

func TestUnfiredVerifies(t *testing.T) {
	r := run(t, read(t, activityIR), unfiredVerifies)
	require.Equal(t, map[string]int{"activityProduct": 3, "standaloneActivity": 1}, r.population)
	require.Empty(t, r.subjects)

	// The product never enables workerStop, so a Property about its steps alone is read on none.
	unfired := read(t, activityIR, func(ir *umpirespb.Model) {
		for _, p := range ir.GetProperties() {
			if p.GetName() == "activityProduct.closedIsRejectedUniformly" {
				p.When = &umpirespb.Property_WhenAction{WhenAction: "workerStop"}
			}
		}
	})
	r = run(t, unfired, unfiredVerifies)
	require.Equal(t, map[string][]string{"activityProduct": {"activityProduct.closedIsRejectedUniformly"}}, r.subjects)
	require.Equal(t, 3, r.population["activityProduct"])
}

func TestUnperformedActions(t *testing.T) {
	r := run(t, read(t, activityIR), unperformedActions)
	// start, attemptStart, attemptResult, control and workerStop; the timers are the system's. No
	// performance binds attemptStart: the activity script starts with it.
	require.Equal(t, map[string]int{"activityProtocol": 5}, r.population)
	require.Empty(t, r.subjects)

	unperformed := read(t, activityIR, func(ir *umpirespb.Model) {
		for _, s := range realization(ir, "activityProtocol").GetScripts() {
			s.Items = slices.DeleteFunc(s.Items, func(item *umpirespb.Item) bool {
				return slices.ContainsFunc(item.GetPerforms(), func(p *umpirespb.Performance) bool {
					return p.GetStep().GetAction() == "temporal.worker.Worker$package$.workerStop"
				})
			})
		}
	})
	r = run(t, unperformed, unperformedActions)
	require.Equal(t, map[string][]string{"activityProtocol": {"workerStop"}}, r.subjects)
	require.Equal(t, 5, r.population["activityProtocol"])
}

func TestUnevidencedFacts(t *testing.T) {
	r := run(t, read(t, activityIR), unevidencedFacts)
	require.Equal(t, map[string]int{"activityProtocol": 10}, r.population)
	require.Empty(t, r.subjects)

	renamed := read(t, activityIR, func(ir *umpirespb.Model) {
		for _, e := range realization(ir, "activityProtocol").GetEvidence() {
			if e.GetRecords() == "statusPaused" {
				e.Records = "statusPausedElsewhere"
			}
		}
	})
	r = run(t, renamed, unevidencedFacts)
	require.Equal(t, map[string][]string{"activityProtocol": {"statusPaused"}}, r.subjects)
	require.Equal(t, 10, r.population["activityProtocol"])

	// A machine with no realization is not counted, and a fact its evidence function names but no
	// evidence kind records is.
	r = run(t, read(t, capturedIR), unevidencedFacts)
	require.Equal(t, map[string]int{"store": 3}, r.population)
	require.Equal(t, map[string][]string{"store": {"stored", "staged", "lost"}}, r.subjects)
}

func TestUntakenChoices(t *testing.T) {
	r := run(t, read(t, nexusControlIR), untakenChoices)
	require.Equal(t, map[string]int{"forgedCompletion": 2}, r.population)
	require.Empty(t, r.subjects)

	// The forged alternative behind a condition that never holds: its copy is still called, and no
	// reachable state takes it.
	r = run(t, read(t, nexusControlIR, func(ir *umpirespb.Model) {
		for _, f := range ir.GetFunctions() {
			if f.GetName() != "temporal.features.nexuscaller.ForgedCompletion$.effects$.forgedComplete" {
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
	require.Equal(t, map[string]int{"forgedCompletion": 2}, r.population)
	require.Equal(t, map[string][]string{"forgedCompletion": {"forged"}}, r.subjects)

	r = run(t, read(t, activityIR), untakenChoices)
	require.Empty(t, r.population)
}

func TestUnreadRefinements(t *testing.T) {
	r := run(t, read(t, activityIR), unreadRefinements)
	require.Equal(t, map[string]int{"activityProtocol": 1}, r.population)
	require.Equal(t, map[string][]string{"activityProtocol": {"activityProtocol refines activityProduct"}}, r.subjects)

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
	require.Equal(t, map[string]int{"activityProtocol": 1}, r.population)
	require.Empty(t, r.subjects)

	// A history read lifts its evidence into history-event.
	r = run(t, read(t, nexusControlIR), unreadObservations)
	require.Equal(t, map[string]int{"forgedCompletion": 2}, r.population)
	require.Empty(t, r.subjects)

	unread := read(t, activityIR, func(ir *umpirespb.Model) {
		r := realization(ir, "activityProtocol")
		spare := proto.CloneOf(r.GetObservations()[0])
		spare.Id = "spare-evidence"
		r.Observations = append(r.Observations, spare)
	})
	r = run(t, unread, unreadObservations)
	require.Equal(t, map[string]int{"activityProtocol": 2}, r.population)
	require.Equal(t, map[string][]string{"activityProtocol": {"spare-evidence"}}, r.subjects)
}

// A read of a realization that declares an API behavior writes no interval: its wait is derived.
// One that writes its own is a finding, kept only where an acceptance records why; a realization that
// declares no behavior derives nothing, so its polls are not counted.
func TestExplicitWaits(t *testing.T) {
	r := run(t, read(t, activityIR), explicitWaits)
	require.Equal(t, map[string]int{"activityProtocol": 6}, r.population)
	require.Empty(t, r.subjects)

	explicit := func(ir *umpirespb.Model) {
		for _, c := range commandsNamed(realization(ir, "activityProtocol"), "await-completed") {
			c.GetPoll().IntervalMs = 250
		}
	}
	r = run(t, read(t, activityIR, explicit), explicitWaits)
	require.Equal(t, map[string][]string{"activityProtocol": {"controller/await-completed"}}, r.subjects)

	r = run(t, read(t, activityIR, explicit, func(ir *umpirespb.Model) {
		r := realization(ir, "activityProtocol")
		r.Behavior, r.ServerSteps = nil, nil
	}), explicitWaits)
	require.Empty(t, r.population)
}

// commandsNamed is every command of a realization with an id.
func commandsNamed(r *umpirespb.Realization, id string) []*umpirespb.Command {
	return slices.DeleteFunc(commands(r), func(c *umpirespb.Command) bool { return c.GetId() != id })
}

func TestUnreachableValues(t *testing.T) {
	r := run(t, read(t, activityIR), unreachableValues)
	require.Equal(t, map[string]int{"activityProduct": 9, "activityProtocol": 21, "activityWorker": 2, "polling": 2}, r.population)
	require.Empty(t, r.subjects)

	r = run(t, read(t, capturedIR), unreachableValues)
	// A channel's contents are a combination of deliveries, which no finding names.
	require.Equal(t, map[string][]string{"putOnly": {"stage=durable"}}, r.subjects)
	require.Equal(t, 3, r.population["putOnly"])
}

func TestNeverEnabled(t *testing.T) {
	r := run(t, read(t, activityIR), neverEnabled)
	require.Equal(t, map[string]int{"activityProduct": 11, "activityProtocol": 22, "activityWorker": 2, "polling": 3}, r.population)
	// The product's worker stop is disabled in every state: its stop is the protocol's alone.
	require.Equal(t, map[string][]string{"activityProduct": {"workerStop"}}, r.subjects)
}

func TestUnproduced(t *testing.T) {
	r := run(t, read(t, activityIR), unproduced)
	require.Equal(t, map[string]int{"activityProduct": 11, "activityProtocol": 14, "activityWorker": 1, "polling": 1}, r.population)
	require.Empty(t, r.subjects)

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
	require.Equal(t, map[string]int{"activityProtocol": 11}, r.population)
	require.Empty(t, r.subjects)

	r = run(t, read(t, capturedIR), unrealizedFinds)
	require.Equal(t, map[string]int{"detailedPair": 1, "disk": 1, "store": 1}, r.population)
	require.Equal(t, map[string][]string{"detailedPair": {"bothPutHeld"}, "disk": {"disk.any.everPut"}}, r.subjects)

	m := read(t, activityIR)
	m.lowering = Lowering{}
	_, err := unrealizedFinds(m)
	require.ErrorContains(t, err, "lowering's standing")
}
