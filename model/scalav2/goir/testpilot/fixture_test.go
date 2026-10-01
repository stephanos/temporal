package testpilot

// The realizations of model/scalav2/lifter/testdata/lifts/Realizations.scala.fixture, lifted into
// expected/realizations.json: a run id one command binds and two branches read, and the activity
// specimen's held race, which declares what Testpilot cannot run yet.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/scalav2/goir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	liftsDir       = "model/scalav2/lifter/testdata/lifts/"
	realizationsAt = liftsDir + "Realizations.scala.fixture"
	admissionAt    = liftsDir + "Admission.scala.fixture"
)

func liftedRealizations(t *testing.T) *modelirspb.Model {
	t.Helper()
	m, err := goir.Load(filepath.Join("..", "..", "lifter", "testdata", "lifts", "expected", "realizations.json"))
	require.NoError(t, err)
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
	p, err := NewProducer(liftedRealizations(t))
	require.NoError(t, err)
	l, err := p.Lower("syncCompletion", cp.IdentityFor("temporal.case", "fixture", "learnedRun"))
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

	scheduled := instruction(t, c, "controller", "await-scheduled")
	closed := instruction(t, c, "controller", "await-close")
	require.Equal(t, []string{"controller/start-workflow"}, after(scheduled))
	require.Equal(t, []string{"controller/start-workflow"}, after(closed))
	require.Equal(t, "workflow-run", slotOf(assigned(t, scheduled.GetInstruction().GetReadEvidence().GetRequestAssignments(), "execution.run_id")))
	require.Equal(t, "workflow-run", slotOf(assigned(t, closed.GetInstruction().GetInvokeRpc().GetRequestAssignments(), "execution.run_id")))
	bound := cp.Present(slot("workflow-run"))
	require.True(t, proto.Equal(bound, scheduled.GetGuard()), "a command reads the learned text only where it is bound")
	require.True(t, proto.Equal(bound, closed.GetGuard()))
	require.Equal(t, []string{"controller/await-scheduled", "controller/await-close"}, after(instruction(t, c, "controller", "history")))

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
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "..", file))
	require.NoError(t, err)
	lines := strings.Split(string(source), "\n")
	require.LessOrEqual(t, n, len(lines))
	return lines[n-1]
}

// The held race declares the four primitives the reviewed specimens need and Testpilot does not
// have: authored monitors, a durable-commit observation, a hold-delivery control with the commands
// that use it, and an activity's worker script. Lowering names each where it was written, with the
// task that owns it, and builds no Case around them.
func TestWhatTestpilotCannotRunIsNamedWithItsOwner(t *testing.T) {
	p, err := NewProducer(liftedRealizations(t))
	require.NoError(t, err)
	l, err := p.Lower("staleAdmission.pauseRace", cp.IdentityFor("temporal.case", "fixture", "pauseRace"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, l.Standing)
	require.Nil(t, l.Case)
	require.Empty(t, l.Inventory)

	type gap struct{ construct, id, owner, file, written string }
	want := []gap{
		{"authored monitor", "atMostOneActiveAttempt", "fn-107.12", admissionAt, "val atMostOneActiveAttempt"},
		{"authored monitor", "terminalFinality", "fn-107.12", admissionAt, "val terminalFinality"},
		{"durable-commit observation", "fixture.realizations.race.evidence.dispatchEnqueued", "fn-107.10", realizationsAt, "Evidence("},
		{"durable-commit observation", "fixture.realizations.race.evidence.attemptAdmitted", "fn-107.10", realizationsAt, "Evidence("},
		{"hold-delivery control", "hold-dispatch", "fn-107.10", realizationsAt, "Control(holdDispatch"},
		{"hold-delivery command", "controller/hold-dispatch-before-start", "fn-107.10", realizationsAt, "hold-dispatch-before-start"},
		{"hold-delivery command", "controller/release-dispatch", "fn-107.10", realizationsAt, "release-dispatch"},
		{"activity activation", "activity", "fn-107.13", realizationsAt, "Script("},
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

// A path with a step a party takes and no command performs is refused: the Case would wait for a
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
	require.ErrorContains(t, err, "model/scalav2/scala/temporal/nexuscaller/Claims.scala:")
	_, err = p.Lower("asyncCompletion", nexusIdentity("asyncCompletion"))
	require.NoError(t, err, "a path that takes only performed steps still lowers")
}

// A find Query the search found no witness for is no Case: there is no path to realize.
func TestAQueryWithNoWitnessIsRefused(t *testing.T) {
	m := loaded(t, "nexus-caller")
	for _, q := range m.GetQueries() {
		if q.GetName() == "syncCompletion" {
			q.Property = &modelirspb.ClaimRef{Machine: "nexusProtocol", Name: "completionFails"}
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	_, err = p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.ErrorContains(t, err, "query syncCompletion has no witness to realize: its check is not-found")
}

func raceScript(t *testing.T, m *modelirspb.Model, id string) *modelirspb.Script {
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
	misnamed := func(t *testing.T, m *modelirspb.Model) {
		raceScript(t, m, "controller").GetItems()[1].GetCommand().GetRpc().Method =
			"/temporal.api.workflowservice.v1.WorkflowService/BeginActivityExecution"
	}
	unperformed := func(t *testing.T, m *modelirspb.Model) {
		s := raceScript(t, m, "controller")
		s.Items = append(s.GetItems()[:2], s.GetItems()[3:]...)
	}
	const noMethod = "temporal.api.workflowservice.v1.WorkflowService has no method BeginActivityExecution"
	const noPerformer = "scenario heldRace takes control-pause, a step of caller, and no script of realization pauseRace performs it"
	for _, c := range []struct {
		name   string
		mutate []func(*testing.T, *modelirspb.Model)
		want   []string
	}{
		{"a method the service does not have", []func(*testing.T, *modelirspb.Model){misnamed}, []string{noMethod}},
		{"a step no script performs", []func(*testing.T, *modelirspb.Model){unperformed}, []string{noPerformer}},
		{"both", []func(*testing.T, *modelirspb.Model){misnamed, unperformed}, []string{noMethod, noPerformer}},
		{"a kind of evidence read from one value", []func(*testing.T, *modelirspb.Model){func(_ *testing.T, m *modelirspb.Model) {
			for _, r := range m.GetRealizations() {
				if r.GetName() == "pauseRace" {
					r.GetEvidence()[0].GetRead().Path = "next_page_token"
				}
			}
		}}, []string{"evidence fixture.realizations.race.evidence.statusPaused is read from next_page_token, which is no repeated message"}},
		{"a fact of the path no kind of evidence records", []func(*testing.T, *modelirspb.Model){func(_ *testing.T, m *modelirspb.Model) {
			for _, r := range m.GetRealizations() {
				if r.GetName() == "pauseRace" {
					r.GetEvidence()[0].Records = "pausedElsewhere"
				}
			}
		}}, []string{"query staleAdmission.pauseRace: statusPaused: evidence.kind-unknown"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := liftedRealizations(t)
			for _, mutate := range c.mutate {
				mutate(t, m)
			}
			p, err := NewProducer(m)
			require.NoError(t, err)
			l, err := p.Lower("staleAdmission.pauseRace", cp.IdentityFor("temporal.case", "fixture", "pauseRace"))
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
	m.Holes = append(m.Holes, &modelirspb.Hole{Id: "fixture.unknown", Name: "unknown"})
	var holds string
	for _, p := range m.GetProperties() {
		if p.GetName() == "syncSucceeds" {
			holds = p.GetHolds()
		}
	}
	for _, f := range m.GetFunctions() {
		if f.GetName() == holds {
			f.Body = &modelirspb.Expr{Position: f.GetPosition(), Kind: &modelirspb.Expr_Hole{Hole: "fixture.unknown"}}
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.Nil(t, l)
	require.ErrorContains(t, err, "query syncCompletion has no witness to realize: its check is incomplete")
	require.ErrorContains(t, err, "property syncSucceeds: ")
	var hole *goir.Hole
	require.ErrorAs(t, err, &hole)
	require.Equal(t, "fixture.unknown", hole.ID)
}

// A Property reads the step the checker reads: its explanation with its outcome, state and facts. The
// door's Property holds only of a push that says why the door opened, the search finds it, and the
// lowering fixes the same step's fact. With another explanation in the Property the search finds
// nothing, and nothing is lowered.
func TestAPropertyIsLoweredOverTheStepItIsCheckedOn(t *testing.T) {
	found := func(m *modelirspb.Model) goir.ReceiptKind {
		for _, r := range goir.Check(m, goir.DefaultScope).Receipts {
			if r.Subject == goir.QuerySubject && r.Key.Name == "door.opens" {
				return r.Kind
			}
		}
		return ""
	}
	identity := cp.IdentityFor("temporal.case", "fixture", "door")
	m := liftedRealizations(t)
	require.Equal(t, goir.Found, found(m))
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("door.opens", identity)
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing)
	var rules []string
	for _, r := range l.Case.GetProvenance().GetCorrelatedRules() {
		rules = append(rules, r.GetRuleId())
	}
	require.Equal(t, []string{"fixture.realizations.door.property.opensBecauseTheLatchGives.fact-doorOpened"}, rules)

	m = liftedRealizations(t)
	rewritten := 0
	var rewrite func(x *modelirspb.Expr)
	rewrite = func(x *modelirspb.Expr) {
		if x.GetLiteral().GetText() == "the latch gives" {
			x.GetLiteral().Kind = &modelirspb.Value_Text{Text: "the hinge gives"}
			rewritten++
		}
		for _, operand := range []*modelirspb.Expr{x.GetBinary().GetLeft(), x.GetBinary().GetRight()} {
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
	require.Equal(t, goir.NotFound, found(m))
	p, err = NewProducer(m)
	require.NoError(t, err)
	_, err = p.Lower("door.opens", identity)
	require.ErrorContains(t, err, "query door.opens has no witness to realize: its check is not-found")
}
