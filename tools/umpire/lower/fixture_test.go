package lower

// The realizations of model/irgen/testdata/lifts/Realizations.scala, lifted into
// expected/realizations.json: a run id one command binds and two branches read, and a held race,
// which declares what Testpilot cannot run yet.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	liftsDir       = "model/irgen/testdata/lifts/"
	realizationsAt = liftsDir + "Realizations.scala"
)

func liftedRealizations(t *testing.T) *umpirespb.Model {
	t.Helper()
	m, err := umpiremodel.Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", "realizations.json"))
	require.NoError(t, err)
	return m
}

// preparable gives every realization of m the API behavior a Case needs to be prepared that the
// lifter fixtures leave out, as the Temporal kit declares it (model/temporal/realize/Behavior.scala):
// how attempts are numbered and the limits of an instruction that writes none.
func preparable(m *umpirespb.Model) *umpirespb.Model {
	for _, r := range m.GetRealizations() {
		if r.Behavior == nil {
			r.Behavior = &umpirespb.ApiBehavior{}
		}
		r.Behavior.AttemptNumbering = &umpirespb.AttemptNumbering{First: 1, OneRun: true}
		r.Behavior.InstructionDefaults = &umpirespb.InstructionLimit{TimeoutMs: 10000, Attempts: 1}
	}
	return m
}

func slotOf(e *testpilotspb.Expression) string { return e.GetReference().GetSlotId() }

func assigned(t *testing.T, assignments []*testpilotspb.RequestAssignment, target string) *testpilotspb.Expression {
	t.Helper()
	for _, a := range assignments {
		if a.GetTarget() == target {
			return a.GetValue()
		}
	}
	require.FailNow(t, "no assignment to "+target)
	return nil
}

func after(n *testpilotspb.InstructionNode) []string {
	if n.GetAfter() == nil {
		return nil
	}
	out := []string{}
	for _, ref := range n.GetAfter().GetInstructions() {
		out = append(out, ref.GetEntrypointId()+"/"+ref.GetInstructionId())
	}
	return out
}

// The start call binds the run id once, as a text. Two commands read it, each running after the start
// call alone, so neither waits for the other; the history read runs after both. Ordinary preparation
// admits the Case.
func TestALearnedTextIsBoundOnceAndReadByIndependentBranches(t *testing.T) {
	p, err := NewProducer(preparable(liftedRealizations(t)))
	require.NoError(t, err)
	l, err := p.Lower("run.opens", cp.IdentityFor("temporal.case", "fixture", "learnedRun"))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing)
	c := l.Case

	require.Len(t, c.GetProgram().GetSlots(), 1)
	learned := c.GetProgram().GetSlots()[0]
	require.Equal(t, "workflow-run", learned.GetSlotId())
	require.Equal(t, testpilotspb.SCALAR_KIND_TEXT, learned.GetValue().GetSingular().GetScalar().GetKind())

	start := instruction(t, c, "controller", "start-workflow")
	require.Nil(t, start.GetAfter())
	reads := start.GetInstruction().GetInvokeRpc().GetResponseReads()
	require.Len(t, reads, 1)
	require.Equal(t, "run_id", reads[0].GetPath())
	require.Equal(t, testpilotspb.READ_CARDINALITY_ONE, reads[0].GetCardinality())
	require.Len(t, reads[0].GetTargets(), 1)
	require.Equal(t, "workflow-run", reads[0].GetTargets()[0].GetSlotId())

	started := instruction(t, c, "controller", "await-started")
	closed := instruction(t, c, "controller", "await-close")
	require.Equal(t, []string{"controller/start-workflow"}, after(started))
	require.Equal(t, []string{"controller/start-workflow"}, after(closed))
	require.Equal(t, "workflow-run", slotOf(assigned(t, started.GetInstruction().GetReadEvidence().GetRequestAssignments(), "execution.run_id")))
	require.Equal(t, "workflow-run", slotOf(assigned(t, closed.GetInstruction().GetInvokeRpc().GetRequestAssignments(), "execution.run_id")))
	bound := cp.Present(slot("workflow-run"))
	require.True(t, proto.Equal(bound, started.GetGuard()), "a command reads the learned text only where it is bound")
	require.True(t, proto.Equal(bound, closed.GetGuard()))
	require.Equal(t, []string{"controller/await-started", "controller/await-close"}, after(instruction(t, c, "controller", "history")))

	encoded, err := protojson.Marshal(c)
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "learned-run-profile",
		Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "nexus-endpoint"})
	require.NoError(t, err)
	_, err = testpilot.Prepare(source, profile)
	require.NoError(t, err)
}

// lineOf is the text of a line a position names, read from the repository.
func lineOf(t *testing.T, position string) string {
	t.Helper()
	file, line, ok := strings.Cut(position, ":")
	require.True(t, ok, "%q names no line", position)
	n, err := strconv.Atoi(line)
	require.NoError(t, err)
	source, err := os.ReadFile(filepath.Join("..", "..", "..", file))
	require.NoError(t, err)
	lines := strings.Split(string(source), "\n")
	require.LessOrEqual(t, n, len(lines))
	return lines[n-1]
}

// The held race of the fixture declares what no Driver realizes: a durable commit read back through a
// public listing, and a control that holds the deliveries of a channel, with the commands that use it.
// Lowering names each where it was written, as a limit no task owns, and builds no Case around them.
// The race a Driver does realize holds what a step dispatched to a task queue, reads the commit from
// the release that observed it, and runs on a machine whose authored monitors are no gap
// (ir/activity-race.json; TestTheHeldRaceLowers). Nor is an activity script: Testpilot runs an
// activity's attempts (the errand; TestAnActivityScriptLowersToItsAttemptsInOrder).
func TestWhatTestpilotCannotRunIsNamedWithItsOwner(t *testing.T) {
	p, err := NewProducer(liftedRealizations(t))
	require.NoError(t, err)
	l, err := p.Lower("race.paused", cp.IdentityFor("temporal.case", "fixture", "pauseRace"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, l.Standing)
	require.Nil(t, l.Case)
	require.Empty(t, l.Inventory)

	type gap struct{ construct, id, owner, file, written string }
	want := []gap{
		{"durable-commit observation", "fixture.realizations.race.evidence.dispatchEnqueued", ownerNone, realizationsAt, "Evidence.read("},
		{"hold-delivery control", "hold-dispatch", ownerNone, realizationsAt, "Actuator(holdDispatch"},
		{"hold-delivery command", "controller/hold-dispatch-before-start", ownerNone, realizationsAt, "hold-dispatch-before-start"},
		{"hold-delivery command", "controller/release-dispatch", ownerNone, realizationsAt, "release-dispatch"},
	}
	var got []gap
	for i, u := range l.Unsupported {
		require.NotEmpty(t, u.Why)
		file, _, _ := strings.Cut(u.Position, ":")
		written := ""
		if i < len(want) && strings.Contains(lineOf(t, u.Position), want[i].written) {
			written = want[i].written
		}
		got = append(got, gap{u.Construct, u.ID, u.Owner, file, written})
	}
	require.Equal(t, want, got)
}

// A path with a step an actor takes and no command performs is refused: the Case would wait for a
// step nothing drives.
func TestAStepNoScriptPerformsIsRefused(t *testing.T) {
	m := loaded(t, "nexus-caller")
	for _, s := range m.GetRealizations()[0].GetScripts() {
		if s.GetId() == "handler" {
			performs := s.GetItems()[0].GetPerforms()
			s.GetItems()[0].Performs = performs[:1]
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	_, err = p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.ErrorContains(t, err, "scenario syncReplied takes handlerReply-syncSuccess, a step of handler, and no script of realization asyncNexus performs it")
	require.ErrorContains(t, err, "model/temporal/features/nexuscaller/system/System.scala:")
	_, err = p.Lower("asyncCompletion", nexusIdentity("asyncCompletion"))
	require.NoError(t, err, "a path that takes only performed steps still lowers")
}

// A find Query the search found no witness for is no Case: there is no path to realize.
func TestAQueryWithNoWitnessIsRefused(t *testing.T) {
	m := loaded(t, "nexus-caller")
	for _, q := range m.GetQueries() {
		if q.GetName() == "syncCompletion" {
			q.Property = &umpirespb.ClaimRef{Machine: "nexusProtocol", Name: "completionFails"}
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	_, err = p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.ErrorContains(t, err, "query syncCompletion has no witness to realize: its check is not-found")
}

func raceScript(t *testing.T, m *umpirespb.Model, id string) *umpirespb.Script {
	t.Helper()
	for _, r := range m.GetRealizations() {
		for _, s := range r.GetScripts() {
			if r.GetName() == "pauseRace" && s.GetId() == id {
				return s
			}
		}
	}
	require.FailNow(t, "no script "+id)
	return nil
}

// A gap is reported only for a realization that is otherwise sound. A descriptor it crosses and a step
// of the path nothing performs are errors whatever Testpilot cannot run of it, each is reported, and
// neither is lowered to `unsupported`.
func TestAGapHidesNoError(t *testing.T) {
	misnamed := func(t *testing.T, m *umpirespb.Model) {
		raceScript(t, m, "controller").GetItems()[1].GetCommand().GetRpc().Method =
			"/temporal.api.workflowservice.v1.WorkflowService/BeginActivityExecution"
	}
	unperformed := func(t *testing.T, m *umpirespb.Model) {
		s := raceScript(t, m, "controller")
		s.Items = append(s.GetItems()[:2], s.GetItems()[3:]...)
	}
	const noMethod = "temporal.api.workflowservice.v1.WorkflowService has no method BeginActivityExecution"
	const noPerformer = "scenario heldRace takes push, a step of doorkeeper, and no script of realization pauseRace performs it"
	for _, c := range []struct {
		name   string
		mutate []func(*testing.T, *umpirespb.Model)
		want   []string
	}{
		{"a method the service does not have", []func(*testing.T, *umpirespb.Model){misnamed}, []string{noMethod}},
		{"a step no script performs", []func(*testing.T, *umpirespb.Model){unperformed}, []string{noPerformer}},
		{"both", []func(*testing.T, *umpirespb.Model){misnamed, unperformed}, []string{noMethod, noPerformer}},
		{"a kind of evidence read from one value", []func(*testing.T, *umpirespb.Model){func(_ *testing.T, m *umpirespb.Model) {
			for _, r := range m.GetRealizations() {
				if r.GetName() == "pauseRace" {
					r.GetEvidence()[0].GetRead().Path = "next_page_token"
				}
			}
		}}, []string{"evidence fixture.realizations.race.evidence.doorOpened is read from next_page_token, which is no repeated message"}},
		{"a fact of the path no kind of evidence records", []func(*testing.T, *umpirespb.Model){func(_ *testing.T, m *umpirespb.Model) {
			for _, r := range m.GetRealizations() {
				if r.GetName() == "pauseRace" {
					r.GetEvidence()[0].Records = "pausedElsewhere"
				}
			}
		}}, []string{"query race.paused: doorOpened: evidence.kind-unknown"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := liftedRealizations(t)
			for _, mutate := range c.mutate {
				mutate(t, m)
			}
			p, err := NewProducer(m)
			require.NoError(t, err)
			l, err := p.Lower("race.paused", cp.IdentityFor("temporal.case", "fixture", "pauseRace"))
			require.Nil(t, l)
			for _, want := range c.want {
				require.ErrorContains(t, err, want)
			}
			require.ErrorContains(t, err, liftsDir)
		})
	}
}

// The standing of a Query is decided in one place: an error before a gap, a gap before the standing of
// a Query that has nothing to lower, and a Case only when none of them is there.
func TestAStandingIsAnErrorThenAGapThenWhatTheQueryIs(t *testing.T) {
	problem := errorAt(nil, "a problem")
	gap := []Unsupported{{Construct: "a gap"}}
	for _, c := range []struct {
		name       string
		problems   []error
		gaps       []Unsupported
		realizable Standing
		want       Standing
		failed     bool
	}{
		{"nothing in the way", nil, nil, Lowered, Lowered, false},
		{"a gap", nil, gap, Lowered, NotSupported, false},
		{"an error", []error{problem}, nil, Lowered, "", true},
		{"an error and a gap", []error{problem}, gap, Lowered, "", true},
		{"a verify Query", nil, nil, NothingToRealize, NothingToRealize, false},
		{"no realization", nil, nil, NoRealization, NoRealization, false},
		{"an error of a Query that realizes nothing", []error{problem}, nil, NothingToRealize, "", true},
		{"a gap of a Query that realizes nothing", nil, gap, NoRealization, NotSupported, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			standing, err := standingOf(c.problems, c.gaps, c.realizable)
			require.Equal(t, c.want, standing)
			require.Equal(t, c.failed, err != nil)
			if c.failed {
				require.ErrorIs(t, err, problem)
			}
		})
	}
}

// A Property whose predicate reaches a hole is not lowered as if it rejected the step: the Query has
// no Case, and both what the search could not establish and the predicate that could not be read are
// reported.
func TestAPropertyThatReachesAHoleIsNotLowered(t *testing.T) {
	m := loaded(t, "nexus-caller")
	m.Holes = append(m.Holes, &umpirespb.Hole{Id: "fixture.unknown", Name: "unknown"})
	var holds string
	for _, p := range m.GetProperties() {
		if p.GetName() == "syncSucceeds" {
			holds = p.GetHolds()
		}
	}
	for _, f := range m.GetFunctions() {
		if f.GetName() == holds {
			f.Body = &umpirespb.Expr{Position: f.GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: "fixture.unknown"}}
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.Nil(t, l)
	require.ErrorContains(t, err, "query syncCompletion has no witness to realize: its check is incomplete")
	require.ErrorContains(t, err, "property syncSucceeds: ")
	var hole *umpiremodel.Hole
	require.ErrorAs(t, err, &hole)
	require.Equal(t, "fixture.unknown", hole.ID)
}

// A Property reads the step the checker reads: its explanation with its outcome, state and facts. The
// door's Property holds only of a push that says why the door opened, the search finds it, and the
// lowering fixes the same step's fact. With another explanation in the Property the search finds
// nothing, and nothing is lowered.
func TestAPropertyIsLoweredOverTheStepItIsCheckedOn(t *testing.T) {
	found := func(m *umpirespb.Model) umpiremodel.ReceiptKind {
		for _, r := range umpiremodel.Check(m, umpiremodel.DefaultScope).Receipts {
			if r.Subject == umpiremodel.QuerySubject && r.Key.Name == "door.opens" {
				return r.Kind
			}
		}
		return ""
	}
	identity := cp.IdentityFor("temporal.case", "fixture", "door")
	m := liftedRealizations(t)
	require.Equal(t, umpiremodel.Found, found(m))
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("door.opens", identity)
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing)
	var rules []string
	for _, r := range l.Case.GetProvenance().GetCorrelatedRules() {
		rules = append(rules, r.GetRuleId())
	}
	require.Equal(t, []string{"fixture.realizations.property.opensBecauseTheLatchGives.fact-doorOpened"}, rules)

	m = liftedRealizations(t)
	rewritten := 0
	var rewrite func(x *umpirespb.Expr)
	rewrite = func(x *umpirespb.Expr) {
		if x.GetLiteral().GetText() == "the latch gives" {
			x.GetLiteral().Kind = &umpirespb.Value_Text{Text: "the hinge gives"}
			rewritten++
		}
		for _, operand := range []*umpirespb.Expr{x.GetBinary().GetLeft(), x.GetBinary().GetRight()} {
			if operand != nil {
				rewrite(operand)
			}
		}
	}
	for _, f := range m.GetFunctions() {
		if strings.Contains(f.GetName(), "opensBecauseTheLatchGives") {
			rewrite(f.GetBody())
		}
	}
	require.Equal(t, 1, rewritten)
	require.Equal(t, umpiremodel.NotFound, found(m))
	p, err = NewProducer(m)
	require.NoError(t, err)
	_, err = p.Lower("door.opens", identity)
	require.ErrorContains(t, err, "query door.opens has no witness to realize: its check is not-found")
}
