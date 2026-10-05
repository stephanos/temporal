package lower

// How a Case's reads wait, derived from the API behavior the kit declares (fn-118 R3, R7). The
// fixtures are the checked-in Model IR, whose reads write no interval since fn-118.5: each read waits
// as the hints derive, and each hint the Cases need is shown to be needed by the Case or Query its
// removal refuses.

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const workflowService = "/temporal.api.workflowservice.v1.WorkflowService/"

// derivedModel is a Model's checked-in IR, whose every poll leaves its wait to derive, then edited.
func derivedModel(t *testing.T, name string, edits ...func(*umpirespb.Model)) *umpirespb.Model {
	t.Helper()
	m := loaded(t, name)
	for _, c := range commandsOfModel(m) {
		require.Zero(t, c.GetPoll().GetIntervalMs(), "command %s writes no interval", c.GetId())
	}
	for _, edit := range edits {
		edit(m)
	}
	return m
}

// commandsOfModel is every command of every realization of a Model.
func commandsOfModel(m *umpirespb.Model) []*umpirespb.Command {
	var out []*umpirespb.Command
	for _, r := range m.GetRealizations() {
		for _, s := range r.GetScripts() {
			out = append(out, commandsOf(s)...)
		}
	}
	return out
}

// lowerDerived lowers one Query of a derived Model.
func lowerDerived(t *testing.T, m *umpirespb.Model, query string) (*Lowering, error) {
	t.Helper()
	p, err := NewProducer(m)
	if err != nil {
		return nil, err
	}
	return p.Lower(query, cp.IdentityFor("temporal.case", "derived", query))
}

// wait is how one read waits, as a test spells it: once, or every interval within its hints, each
// "id=ms".
type wait struct {
	once     bool
	interval int64
	hints    []string
}

var (
	readOnce          = wait{once: true}
	deliveredAnswered = wait{interval: 250, hints: []string{"cause.delivery=3000", "cause.activityAnswer=2000"}}
	scheduled         = map[string]wait{"controller/await-scheduled": {interval: 250, hints: []string{"cause.workflowTask=5000"}}}
)

// derivedWaits is every read of every Case the checked-in Models lower to, once its wait is derived.
var derivedWaits = map[string]map[string]map[string]wait{
	"activity": {
		"completion":          {"controller/await-completed": deliveredAnswered},
		"nonRetryableFailure": {"controller/await-failed": deliveredAnswered},
		"pauseResume":         {"controller/await-paused": readOnce, "controller/await-completed": deliveredAnswered},
		"scheduleToStartTimeout": {"controller/await-timed-out": {interval: 250,
			hints: []string{"deadline.scheduleToStart=2000", "cause.timer=3000"}}},
		"terminate":                          {"controller/await-terminated": readOnce},
		"activityProtocol.terminateSettles":  {"controller/await-terminated": readOnce},
		"activityProtocol.cancelIsRequested": {},
		"retry": {"controller/await-completed": {interval: 250, hints: []string{"cause.delivery=3000", "cause.activityAnswer=2000",
			"deadline.backoff=1000", "cause.timer=3000", "cause.delivery=3000", "cause.activityAnswer=2000"}}},
	},
	"activity-race": {
		"heldAdmission.staleDelivery":     {"controller/await-paused": readOnce},
		"admissionResponseLoss.committed": {},
	},
	"nexus-caller": {
		"syncCompletion": scheduled, "asyncCompletion": scheduled, "asyncFailure": scheduled, "handlerError": scheduled,
		"scheduleToStartTimeout": scheduled, "startToCloseTimeout": scheduled,
		"retry": {"controller/await-scheduled": scheduled["controller/await-scheduled"],
			"controller/pending-attempts": {interval: 250, hints: []string{"cause.handlerReply=5000",
				"visibility.handlerReply.describeWorkflowExecution=2000"}}},
	},
	"nexus-control":   {"forgedCompletion": scheduled},
	"nexus-operation": {"nexusOperation.terminateSettles": {"controller/await-terminated": readOnce}, "nexusOperation.cancelIsRequested": {}},
}

// waitsOf is how each read of a Case waits.
func waitsOf(c *testpilotspb.Case) map[string]wait {
	out := map[string]wait{}
	for _, e := range c.GetProgram().GetEntrypoints() {
		for _, n := range e.GetInstructions() {
			read := n.GetInstruction().GetReadEvidence()
			if read == nil {
				continue
			}
			w := wait{once: read.GetOnce(), interval: read.GetPollIntervalMilliseconds()}
			for _, h := range n.GetWaitHints() {
				w.hints = append(w.hints, fmt.Sprintf("%s=%d", h.GetHintId(), h.GetAtMostMilliseconds()))
			}
			out[e.GetEntrypointId()+"/"+n.GetInstructionId()] = w
		}
	}
	return out
}

// declaredAt is where a Model's realizations declare each hint, and each timer step's deadline.
func declaredAt(t *testing.T, m *umpirespb.Model) map[string]*umpirespb.Position {
	t.Helper()
	out := map[string]*umpirespb.Position{}
	for _, r := range m.GetRealizations() {
		for _, v := range r.GetBehavior().GetVisibility() {
			out[v.GetId()] = v.GetPosition()
		}
		for _, c := range r.GetBehavior().GetCauses() {
			out[c.GetId()] = c.GetPosition()
		}
		for _, s := range r.GetServerSteps() {
			action := s.GetStep().GetAction()
			out["deadline."+action[strings.LastIndex(action, ".")+1:]] = s.GetPosition()
		}
	}
	return out
}

// preparedAsIs prepares a Case under the Profile derived from it, in an environment that offers what
// every Temporal Case asks of it, and requires preparation to carry it unchanged.
func preparedAsIs(t *testing.T, c *testpilotspb.Case) {
	t.Helper()
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	encoded, err := protojson.Marshal(c)
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	settings := map[string]string{}
	for _, s := range c.GetProgram().GetRequiredSettings() {
		settings[s.GetKey()] = s.GetValue()
	}
	profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "derived-profile", Namespace: "namespace",
		TaskQueue: "task-queue", HandlerTaskQueue: "task-queue-handler", NexusEndpoint: "nexus-endpoint", DeliveryControl: true,
		DynamicConfig: settings})
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	require.True(t, proto.Equal(source, prepared.Snapshot()), "preparation carries the Case unchanged")
}

// Each read of an existing Case reads once after a write of its own script that is visible at once,
// and otherwise polls within the bounds of the causes it waits for: a pause or a terminate is read
// back once, an answered attempt within a delivery and an answer, a retried one across both
// attempts and the backoff timer between them, a timed-out one within the deadline and the timer's
// slack, the scheduled event within a workflow task, and a retryable handler error within the reply
// and its eventual visibility. Each hint names the line that declares it, the timeout is their sum,
// and Testpilot prepares every Case unchanged.
func TestReadsWaitAsTheApiBehaviorDerives(t *testing.T) {
	for model, queries := range derivedWaits {
		m := derivedModel(t, model)
		at := declaredAt(t, m)
		for query, want := range queries {
			t.Run(model+"/"+query, func(t *testing.T) {
				l, err := lowerDerived(t, m, query)
				require.NoError(t, err)
				require.Equal(t, Lowered, l.Standing)
				require.Equal(t, want, waitsOf(l.Case))
				for _, e := range l.Case.GetProgram().GetEntrypoints() {
					for _, n := range e.GetInstructions() {
						var sum int64
						for _, h := range n.GetWaitHints() {
							declared := at[h.GetHintId()]
							require.NotNil(t, declared, h.GetHintId())
							require.Equal(t, declared.GetFile(), h.GetSource().GetPath(), h.GetHintId())
							require.Equal(t, declared.GetLine(), h.GetSource().GetLine(), h.GetHintId())
							sum += h.GetAtMostMilliseconds()
						}
						if len(n.GetWaitHints()) > 0 {
							require.Equal(t, sum, n.GetLimits().GetTimeoutMilliseconds())
						} else if n.GetInstruction().GetReadEvidence().GetOnce() {
							require.Nil(t, n.GetLimits(), "a read once keeps the Profile's limit for its one call")
						}
					}
				}
				preparedAsIs(t, l.Case)
			})
		}
	}
}

// Every find Query of the checked-in Models that lowers to a Case is in derivedWaits, so the test
// above leaves none out.
func TestDerivedWaitsCoverEveryLoweredQuery(t *testing.T) {
	for model, queries := range derivedWaits {
		p, err := NewProducer(loaded(t, model))
		require.NoError(t, err)
		var lowered []string
		for _, q := range loaded(t, model).GetQueries() {
			if q.GetForm() != umpirespb.Query_FORM_FIND {
				continue
			}
			l, err := p.Lower(q.GetName(), cp.IdentityFor("temporal.case", "derived", q.GetName()))
			require.NoError(t, err)
			if l.Standing == Lowered {
				lowered = append(lowered, q.GetName())
			}
		}
		var listed []string
		for query := range queries {
			listed = append(listed, query)
		}
		require.ElementsMatch(t, lowered, listed, model)
	}
}

// visibilityOf edits the visibility of one pair.
func visibilityOf(id string, edit func(*umpirespb.Visibility)) func(*umpirespb.Model) {
	return func(m *umpirespb.Model) {
		for _, r := range m.GetRealizations() {
			for _, v := range r.GetBehavior().GetVisibility() {
				if v.GetId() == id {
					edit(v)
				}
			}
		}
	}
}

// A write of the read's own script that is visible only eventually makes the read poll, within the
// visibility's own bound and every its interval; at once, the same read reads once.
func TestAReadAfterAnEventuallyVisibleWritePollsWithinItsBound(t *testing.T) {
	const terminated = "visibility.terminateActivityExecution.describeActivityExecution"
	l, err := lowerDerived(t, derivedModel(t, "activity"), "terminate")
	require.NoError(t, err)
	require.Equal(t, map[string]wait{"controller/await-terminated": readOnce}, waitsOf(l.Case))

	eventually := derivedModel(t, "activity", visibilityOf(terminated, func(v *umpirespb.Visibility) {
		v.EventuallyWithin = &umpirespb.WaitBound{Position: v.GetPosition(), IntervalMs: 100, AtMostMs: 1500}
	}))
	l, err = lowerDerived(t, eventually, "terminate")
	require.NoError(t, err)
	require.Equal(t, map[string]wait{"controller/await-terminated": {interval: 100, hints: []string{terminated + "=1500"}}}, waitsOf(l.Case))
	node := instruction(t, l.Case, "controller", "await-terminated")
	require.Equal(t, int64(1500), node.GetLimits().GetTimeoutMilliseconds())
	require.Equal(t, declaredAt(t, eventually)[terminated].GetLine(), node.GetWaitHints()[0].GetSource().GetLine())
	preparedAsIs(t, l.Case)
}

// without removes one hint, by id, from every realization of a Model.
func without(id string) func(*umpirespb.Model) {
	return func(m *umpirespb.Model) {
		for _, r := range m.GetRealizations() {
			b := r.GetBehavior()
			b.Visibility = slices.DeleteFunc(b.GetVisibility(), func(v *umpirespb.Visibility) bool { return v.GetId() == id })
			b.Causes = slices.DeleteFunc(b.GetCauses(), func(c *umpirespb.CauseBound) bool { return c.GetId() == id })
		}
	}
}

// withoutStep removes the server step of one class, by its action's name, from every realization.
func withoutStep(action string) func(*umpirespb.Model) {
	return func(m *umpirespb.Model) {
		for _, r := range m.GetRealizations() {
			r.ServerSteps = slices.DeleteFunc(r.GetServerSteps(), func(s *umpirespb.ServerStep) bool {
				return strings.HasSuffix(s.GetStep().GetAction(), "."+action)
			})
		}
	}
}

// R7: each adopted hint is needed by an existing Case. With it removed, the Case is refused at its
// read: a missing visibility names the write, a method or a kind of cause, and the read's method; a
// missing bound names the kind of cause and the read. A delivery's and a timer's bounds also bound
// the server steps that declare those kinds, so the reader refuses the realization at the step
// first; the step's own removal then refuses the read.
func TestEachAdoptedHintIsNeededByTheCaseItsRemovalRefuses(t *testing.T) {
	describeActivity, history := workflowService+"DescribeActivityExecution", workflowService+"GetWorkflowExecutionHistory"
	describeWorkflow := workflowService + "DescribeWorkflowExecution"
	describeOperation := workflowService + "DescribeNexusOperationExecution"
	unseen := func(write, read string) string {
		return fmt.Sprintf("and the realization declares no visibility of %s to %s", write, read)
	}
	for _, tc := range []struct {
		model, hint, query string
		refusal            []string
	}{
		{"activity", "visibility.startActivityExecution.describeActivityExecution", "completion",
			[]string{"command controller/await-completed reads " + describeActivity + " after command controller/start-activity",
				unseen(workflowService+"StartActivityExecution", describeActivity)}},
		{"activity", "visibility.pauseActivityExecution.describeActivityExecution", "pauseResume",
			[]string{"command controller/await-paused reads " + describeActivity, unseen(workflowService+"PauseActivityExecution", describeActivity)}},
		{"activity", "visibility.unpauseActivityExecution.describeActivityExecution", "pauseResume",
			[]string{"command controller/await-completed reads " + describeActivity, unseen(workflowService+"UnpauseActivityExecution", describeActivity)}},
		{"activity", "visibility.terminateActivityExecution.describeActivityExecution", "terminate",
			[]string{"command controller/await-terminated reads " + describeActivity, unseen(workflowService+"TerminateActivityExecution", describeActivity)}},
		{"activity", "visibility.requestCancelActivityExecution.describeActivityExecution", "cancel",
			[]string{"command controller/await-canceled reads " + describeActivity + " after command controller/request-cancel-activity",
				unseen(workflowService+"RequestCancelActivityExecution", describeActivity)}},
		{"activity", "visibility.activityAnswer.describeActivityExecution", "completion",
			[]string{"command controller/await-completed reads " + describeActivity + " after command activity/complete-attempt",
				"which is an activity answer, " + unseen("an activity answer", describeActivity)}},
		{"nexus-operation", "visibility.startNexusOperationExecution.describeNexusOperationExecution", "nexusOperation.terminateSettles",
			[]string{"command controller/await-terminated reads " + describeOperation, unseen(workflowService+"StartNexusOperationExecution", describeOperation)}},
		{"nexus-operation", "visibility.terminateNexusOperationExecution.describeNexusOperationExecution", "nexusOperation.terminateSettles",
			[]string{"command controller/await-terminated reads " + describeOperation, unseen(workflowService+"TerminateNexusOperationExecution", describeOperation)}},
		{"nexus-caller", "visibility.startWorkflowExecution.getWorkflowExecutionHistory", "syncCompletion",
			[]string{"command controller/await-scheduled reads " + history + " after command controller/start-workflow",
				unseen(workflowService+"StartWorkflowExecution", history)}},
		{"nexus-caller", "visibility.workflowTask.getWorkflowExecutionHistory", "syncCompletion",
			[]string{"command controller/await-scheduled reads " + history + " after command workflow/start-nexus-operation",
				unseen("a workflow task", history)}},
		{"nexus-caller", "visibility.handlerReply.describeWorkflowExecution", "retry",
			[]string{"command controller/pending-attempts reads " + describeWorkflow + " after command handler/respond-error-retryable",
				unseen("a handler reply", describeWorkflow)}},
		{"activity", "cause.activityAnswer", "completion",
			[]string{"command controller/await-completed waits for command activity/complete-attempt",
				"which is an activity answer, and the realization declares no bound of an activity answer"}},
		{"nexus-caller", "cause.workflowTask", "syncCompletion",
			[]string{"command controller/await-scheduled waits for command workflow/start-nexus-operation",
				"which is a workflow task, and the realization declares no bound of a workflow task"}},
		{"nexus-caller", "cause.handlerReply", "retry",
			[]string{"command controller/pending-attempts waits for command handler/respond-error-retryable",
				"which is a handler reply, and the realization declares no bound of a handler reply"}},
		{"activity", "cause.delivery", "completion", []string{"server step attemptStart is a delivery, and the realization bounds no delivery"}},
		{"activity", "cause.timer", "scheduleToStartTimeout", []string{"server step scheduleToStart is a timer, and the realization bounds no timer"}},
	} {
		t.Run(tc.hint, func(t *testing.T) {
			kept, err := lowerDerived(t, derivedModel(t, tc.model), tc.query)
			require.NoError(t, err)
			require.Contains(t, []Standing{Lowered, NotSupported}, kept.Standing)
			_, err = lowerDerived(t, derivedModel(t, tc.model, without(tc.hint)), tc.query)
			require.Error(t, err)
			for _, says := range tc.refusal {
				require.ErrorContains(t, err, says)
			}
		})
	}
	for _, tc := range []struct{ step, query, refused string }{
		{"attemptStart", "completion", "command controller/await-completed waits for step attemptStart, which no command performs"},
		{"scheduleToStart", "scheduleToStartTimeout", "command controller/await-timed-out waits for step scheduleToStart, which no command performs"},
		{"backoff", "retry", "command controller/await-completed waits for step backoff, which no command performs"},
	} {
		t.Run("server step "+tc.step, func(t *testing.T) {
			_, err := lowerDerived(t, derivedModel(t, "activity", withoutStep(tc.step)), tc.query)
			require.ErrorContains(t, err, tc.refused)
		})
	}
}

// A refusal is located at the read it refuses, where the realization declares the command.
func TestARefusalIsLocatedAtTheRead(t *testing.T) {
	m := derivedModel(t, "activity", without("visibility.pauseActivityExecution.describeActivityExecution"))
	var at *umpirespb.Position
	for _, c := range commandsOfModel(m) {
		if c.GetId() == "await-paused" {
			at = c.GetPosition()
		}
	}
	require.NotNil(t, at)
	_, err := lowerDerived(t, m, "pauseResume")
	require.ErrorContains(t, err, fmt.Sprintf("%s:%d: command controller/await-paused", at.GetFile(), at.GetLine()))
}

// explicitly is a Model whose polls each write an interval of their own, as before fn-118.5.
func explicitly(m *umpirespb.Model) {
	for _, c := range commandsOfModel(m) {
		if poll := c.GetPoll(); poll != nil {
			poll.IntervalMs = 250
		}
	}
}

// The inventory says which hints a Case's waits read: each in the instructions whose wait it shapes,
// and the rest unread. A Case whose reads write their own intervals reads none.
func TestTheInventoryAccountsForTheHintsAWaitReads(t *testing.T) {
	hints := func(l *Lowering) map[string][]string {
		out := map[string][]string{}
		for _, e := range l.Inventory {
			if e.Kind == "behavior" || e.Kind == "server_steps" {
				out[e.ID] = append([]string{string(e.Disposition)}, e.As...)
			}
		}
		return out
	}
	const paused, completed = "program.entrypoints[controller].instructions[await-paused]",
		"program.entrypoints[controller].instructions[await-completed]"
	l, err := lowerDerived(t, derivedModel(t, "activity"), "pauseResume")
	require.NoError(t, err)
	got := hints(l)
	require.Equal(t, []string{string(InCase), paused}, got["visibility.startActivityExecution.describeActivityExecution"])
	require.Equal(t, []string{string(InCase), paused}, got["visibility.pauseActivityExecution.describeActivityExecution"])
	require.Equal(t, []string{string(InCase), completed}, got["visibility.unpauseActivityExecution.describeActivityExecution"])
	require.Equal(t, []string{string(InCase), completed}, got["cause.delivery"])
	require.Equal(t, []string{string(InCase), completed}, got["attemptStart"])
	require.Equal(t, []string{string(Unread)}, got["cause.timer"])
	require.Equal(t, []string{string(Unread)}, got["scheduleToStart"])

	p, err := NewProducer(derivedModel(t, "activity", explicitly))
	require.NoError(t, err)
	l, err = p.Lower("pauseResume", activityIdentity("pauseResume"))
	require.NoError(t, err)
	for id, disposition := range hints(l) {
		require.Equal(t, []string{string(Unread)}, disposition, id)
	}
}

// What only descriptors tell is checked where waits are derived: a visibility's write is a method the
// API binds to POST and its read one it binds to GET, each refused at the hint; and a command calls
// a method bound to one of the two, or what it does cannot be told.
func TestBindingsTellWritesFromReads(t *testing.T) {
	const pause = "visibility.pauseActivityExecution.describeActivityExecution"
	m := derivedModel(t, "activity", visibilityOf(pause, func(v *umpirespb.Visibility) {
		v.Write = &umpirespb.Visibility_Method{Method: workflowService + "DescribeActivityExecution"}
		v.Read = workflowService + "PauseActivityExecution"
	}))
	at := declaredAt(t, m)[pause]
	_, err := lowerDerived(t, m, "terminate")
	require.ErrorContains(t, err, fmt.Sprintf("%s:%d: visibility %s names %s as its write, which the API binds to no HTTP POST", at.GetFile(), at.GetLine(),
		pause, workflowService+"DescribeActivityExecution"))
	require.ErrorContains(t, err, fmt.Sprintf("visibility %s names %s as its read, which the API binds to no HTTP GET", pause,
		workflowService+"PauseActivityExecution"))

	unbound := derivedModel(t, "activity", func(m *umpirespb.Model) {
		for _, c := range commandsOfModel(m) {
			if c.GetId() == "terminate-activity" {
				c.GetRpc().Method = workflowService + "RespondWorkflowTaskCompleted"
			}
		}
	})
	_, err = lowerDerived(t, unbound, "terminate")
	require.ErrorContains(t, err, "command terminate-activity calls "+workflowService+
		"RespondWorkflowTaskCompleted, which the API binds to neither HTTP GET nor POST, so it is told neither a read nor a write")
}

// A call that reads its response is a read too, checked as a poll is but never waiting: after a write
// no visibility names it is refused, and after a write visible to it only eventually it is refused as
// a read that should poll. The fixture lets the forged control's inspections read what they describe:
// the first runs before the handler replies, and only the second reads after the reply.
func TestACallThatReadsIsCheckedAndNeverWaits(t *testing.T) {
	describeWorkflow := workflowService + "DescribeWorkflowExecution"
	reading := func(m *umpirespb.Model) {
		for _, r := range m.GetRealizations() {
			r.Observations = append(r.Observations, &umpirespb.Observed{Id: "described", Position: r.GetPosition(),
				Message: "temporal.api.workflow.v1.WorkflowExecutionInfo"})
		}
		for _, c := range commandsOfModel(m) {
			if c.GetId() == "inspect-workflow" {
				c.GetRpc().Reads = []*umpirespb.ResponseRead{{Path: "workflow_execution_info", Cardinality: umpirespb.ResponseRead_CARDINALITY_ONE,
					Targets: []*umpirespb.Target{{Target: &umpirespb.Target_Observe{Observe: "described"}}}}}
			}
		}
	}
	_, err := lowerDerived(t, derivedModel(t, "nexus-control", reading, without("visibility.handlerReply.describeWorkflowExecution")), "forgedCompletion")
	require.ErrorContains(t, err, "command controller/inspect-workflow-2 reads "+describeWorkflow+" after command handler/respond-async")
	require.ErrorContains(t, err, "which is a handler reply, and the realization declares no visibility of a handler reply to "+describeWorkflow)

	_, err = lowerDerived(t, derivedModel(t, "nexus-control", reading), "forgedCompletion")
	require.ErrorContains(t, err, "command controller/inspect-workflow-2 reads "+describeWorkflow+" once, after command handler/respond-async")
	require.ErrorContains(t, err, "which is visible to it only eventually (visibility.handlerReply.describeWorkflowExecution): a read that waits is a poll")
}

// A poll that writes its own interval keeps it, its timeout and no hint, as Cases lowered before
// fn-118.5 did; lint's explicit-wait finding asks why it is explicit. A poll left to derive its wait
// writes no deadline, which the reader refuses.
func TestAnExplicitPollKeepsItsInterval(t *testing.T) {
	p, err := NewProducer(derivedModel(t, "activity", explicitly))
	require.NoError(t, err)
	l, err := p.Lower("completion", activityIdentity("completion"))
	require.NoError(t, err)
	require.Equal(t, map[string]wait{"controller/await-completed": {interval: 250}}, waitsOf(l.Case))
	require.Nil(t, instruction(t, l.Case, "controller", "await-completed").GetLimits())

	_, err = NewProducer(derivedModel(t, "activity", func(m *umpirespb.Model) {
		for _, c := range commandsOfModel(m) {
			if c.GetId() == "await-completed" {
				c.TimeoutMs = 1000
			}
		}
	}))
	require.ErrorContains(t, err, "command await-completed of script controller waits within the bound the API behavior derives, and writes a deadline of 1000 milliseconds besides")
}

// A realization that declares no behavior has nothing to derive a wait from: a poll left to derive
// its wait is refused at the first write it reads after, while its calls that read are taken as
// written, as before hints existed.
func TestNoBehaviorDerivesNoWait(t *testing.T) {
	m := derivedModel(t, "activity", func(m *umpirespb.Model) {
		for _, r := range m.GetRealizations() {
			r.Behavior, r.ServerSteps = nil, nil
		}
	})
	_, err := lowerDerived(t, m, "terminate")
	require.ErrorContains(t, err, "command controller/await-terminated reads "+workflowService+"DescribeActivityExecution after command controller/start-activity")
	require.ErrorContains(t, err, "and the realization declares no visibility of "+workflowService+"StartActivityExecution")

	// The forged control's second inspection, made to read, would be refused under the kit's behavior
	// (TestACallThatReadsIsCheckedAndNeverWaits); with none declared it lowers as written.
	unhinted := derivedModel(t, "nexus-control", func(m *umpirespb.Model) {
		for _, r := range m.GetRealizations() {
			r.Behavior, r.ServerSteps = nil, nil
			r.Observations = append(r.Observations, &umpirespb.Observed{Id: "described", Position: r.GetPosition(),
				Message: "temporal.api.workflow.v1.WorkflowExecutionInfo"})
		}
		for _, c := range commandsOfModel(m) {
			if c.GetId() == "inspect-workflow" {
				c.GetRpc().Reads = []*umpirespb.ResponseRead{{Path: "workflow_execution_info", Cardinality: umpirespb.ResponseRead_CARDINALITY_ONE,
					Targets: []*umpirespb.Target{{Target: &umpirespb.Target_Observe{Observe: "described"}}}}}
			}
		}
		explicitly(m)
	})
	l, err := lowerDerived(t, unhinted, "forgedCompletion")
	require.NoError(t, err)
	require.Len(t, instruction(t, l.Case, "controller", "inspect-workflow-2").GetInstruction().GetInvokeRpc().GetResponseReads(), 1)
}

// A closing poll left to derive its wait reads once: the realization declares it is made after its
// sources report nothing more, so it waits for nothing and checks no write.
func TestAClosingPollReadsOnce(t *testing.T) {
	m := derivedModel(t, "nexus-caller", func(m *umpirespb.Model) {
		for _, c := range commandsOfModel(m) {
			if c.GetId() == "await-scheduled" {
				c.Closes = []string{c.GetPoll().GetEvidence()}
			}
		}
		for _, r := range m.GetRealizations() {
			for _, e := range r.GetEvidence() {
				if strings.HasSuffix(e.GetId(), ".scheduled") {
					e.Exhaustive = true
				}
			}
		}
	})
	l, err := lowerDerived(t, m, "retry")
	require.NoError(t, err)
	require.Equal(t, readOnce, waitsOf(l.Case)["controller/await-scheduled"])
}
