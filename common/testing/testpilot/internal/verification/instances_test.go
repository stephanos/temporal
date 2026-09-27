package verification

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// instanceWorld is what a Contract is prepared against.
type instanceWorld struct {
	catalog    *ir.Catalog
	view       execution.ProgramView
	limits     *testpilotspb.ContractLimits
	correlated *testpilotspb.CorrelatedLimits
}

// instanceCase is an instanced Contract and the Runs its expansion must evaluate alike.
type instanceCase struct {
	name     string
	world    instanceWorld
	contract *testpilotspb.Contract
	runs     []namedRun
}

type namedRun struct {
	name string
	run  *testpilotspb.Run
	// want is the Verdict status the Run reaches, so the matrix covers each outcome it names.
	want testpilotspb.VerdictStatus
}

// instanceOutcome is everything a Run's evaluation reports by rule ID.
type instanceOutcome struct {
	online, offline []byte
	violations      []Violation
	trace           []transitionTrace
	stop            int64
}

// A Contract with Rule instances and its expansion, built by the test's own expand, are admitted
// alike, and every Run yields byte-identical Verdicts through both, online and offline, with the same
// violations, transition traces and Executor stop.
func TestRuleInstancesEvaluateAsTheirExpansion(t *testing.T) {
	for _, tc := range []instanceCase{
		pairInstances(t),
		safetyInstances(t),
		deadlineInstances(t),
		correlatedInstances(t),
	} {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expand(tc.contract)
			require.Greater(t, len(expanded.Rules), len(tc.contract.Rules), "the case has Rule instances")
			for _, run := range tc.runs {
				require.Empty(t, instanceDifference(t, tc.world, tc.contract, expanded, run))
			}
			// Swapping two instances' values must be seen, or the comparison proves nothing.
			swapped := proto.CloneOf(tc.contract)
			rule := swapped.Rules[slices.IndexFunc(swapped.Rules, func(rule *testpilotspb.ContractRule) bool { return len(rule.Instances) > 1 })]
			first, second := rule.Instances[0].Assignments[0], rule.Instances[1].Assignments[0]
			first.Value, second.Value = second.Value, first.Value
			differs := false
			for _, run := range tc.runs {
				differs = differs || instanceDifference(t, tc.world, swapped, expanded, run) != ""
			}
			require.True(t, differs, "swapping two instances' values changes some Run's outcome")
		})
	}
}

// A Contract whose expansion is rejected is rejected with the same diagnostic, including on the
// Contract's surface size, which only the expansion's repeated Rules exceed.
func TestRuleInstancesRejectOnTheExpansionsCeiling(t *testing.T) {
	for _, tc := range []struct {
		name   string
		build  func(*testpilotspb.Contract, *testpilotspb.ContractLimits)
		within func(*testing.T, *ir.Error)
	}{
		{"states", func(c *testpilotspb.Contract, limits *testpilotspb.ContractLimits) {
			instanced(c.Rules[0], "a", "b", "c")
			c.Rules[0].Transitions[0].Predicate = readsInstanceValue()
			limits.MaxStates = 8
		}, nil},
		{"surface size", func(c *testpilotspb.Contract, _ *testpilotspb.ContractLimits) {
			large := strings.Repeat("x", 6<<20)
			instanced(c.Rules[0], large, large+"y")
			c.Rules[0].Transitions[0].Predicate = all(readsInstanceValue(), readsInstanceValue())
		}, func(t *testing.T, err *ir.Error) {
			require.True(t, strings.HasPrefix(err.Path, "$.rules."), err.Path)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, limits := fixture(t)
			tc.build(c, limits)
			world := instanceWorld{catalog: catalog, view: view, limits: limits}
			_, err := Prepare(expand(c), world.catalog, world.view, world.limits, nil)
			require.Error(t, err, "the expansion is rejected")
			_, _, err = prepareBoth(t, world, c, expand(c))
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			if tc.within != nil {
				tc.within(t, diagnostic)
			}
		})
	}
}

// The authored Contract is bounded as written before it is walked, so values its predicates never
// read still count toward its surface size: the one direction in which an instanced Contract is
// rejected though its expansion, which drops them, is admitted.
func TestRuleInstancesBoundTheAuthoredSurface(t *testing.T) {
	c, catalog, view, limits := fixture(t)
	instanced(c.Rules[0], "a", "b", "c")
	c.Rules[0].Transitions[0].Predicate = readsInstanceValue()
	c.Rules[0].InstanceValues = append(c.Rules[0].InstanceValues, &testpilotspb.ContractInstanceValue{InstanceValueId: "unread", Type: singular(scalar(testpilotspb.SCALAR_KIND_TEXT))})
	for _, instance := range c.Rules[0].Instances {
		instance.Assignments = append(instance.Assignments, assignment("unread", text(strings.Repeat("x", 6<<20))))
	}
	_, err := Prepare(expand(c), catalog, view, limits, nil)
	require.NoError(t, err, "the expansion drops the unread values")
	_, err = Prepare(c, catalog, view, limits, nil)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.LimitExceeded, diagnostic.Category)
	require.True(t, strings.HasPrefix(diagnostic.Path, "$.rules."), diagnostic.Path)
}

// prepareBoth prepares an instanced Contract and its expansion and requires the same admission: both
// prepared, or both rejected with the same diagnostic, which it returns.
func prepareBoth(t *testing.T, world instanceWorld, source, expanded *testpilotspb.Contract) (instancedContract, expandedContract *PreparedContract, err error) {
	t.Helper()
	instancedContract, instancedErr := Prepare(source, world.catalog, world.view, world.limits, world.correlated)
	expandedContract, expandedErr := Prepare(expanded, world.catalog, world.view, world.limits, world.correlated)
	if expandedErr == nil {
		require.NoError(t, instancedErr)
		return instancedContract, expandedContract, nil
	}
	var want, got *ir.Error
	require.ErrorAs(t, expandedErr, &want)
	require.ErrorAs(t, instancedErr, &got)
	require.Equal(t, want, got)
	return nil, nil, instancedErr
}

// instanceDifference evaluates run through the instanced Contract and its expansion, and names the
// first way their outcomes differ, or returns "" when they are identical. It requires each Contract's
// online and offline Verdicts to be byte-identical.
func instanceDifference(t *testing.T, world instanceWorld, source, expanded *testpilotspb.Contract, run namedRun) string {
	t.Helper()
	instancedContract, expandedContract, err := prepareBoth(t, world, source, expanded)
	require.NoError(t, err)
	got := evaluateInstanceRun(t, world, instancedContract, run.run)
	want := evaluateInstanceRun(t, world, expandedContract, run.run)
	var verdict testpilotspb.Verdict
	require.NoError(t, proto.Unmarshal(want.online, &verdict))
	require.Equal(t, run.want, verdict.Status, run.name)
	if difference := firstDifferingRule(t, want.online, got.online); difference != "" {
		return fmt.Sprintf("run %s: %s", run.name, difference)
	}
	if !slices.Equal(want.online, got.online) {
		return fmt.Sprintf("run %s: Verdict bytes differ", run.name)
	}
	if !slices.EqualFunc(want.violations, got.violations, func(a, b Violation) bool {
		return a.RuleID == b.RuleID && a.Sequence == b.Sequence && a.Kind == b.Kind && slices.Equal(a.ObservationIDs, b.ObservationIDs)
	}) {
		return fmt.Sprintf("run %s: violations differ: %v, %v", run.name, want.violations, got.violations)
	}
	if !slices.Equal(want.trace, got.trace) {
		return fmt.Sprintf("run %s: transition traces differ: %v, %v", run.name, want.trace, got.trace)
	}
	if want.stop != got.stop {
		return fmt.Sprintf("run %s: Executor stop differs: %d, %d", run.name, want.stop, got.stop)
	}
	return ""
}

func evaluateInstanceRun(t *testing.T, world instanceWorld, contract *PreparedContract, run *testpilotspb.Run) instanceOutcome {
	t.Helper()
	var outcome instanceOutcome
	monitor, err := contract.New(context.Background(), world.view)
	require.NoError(t, err)
	for _, event := range run.Events {
		decision, err := monitor.Observe(context.Background(), event)
		require.NoError(t, err)
		if decision == execution.Stop && outcome.stop == 0 {
			outcome.stop = event.Sequence
		}
	}
	outcome.trace = monitor.(*Evaluator).trace
	live, err := monitor.Close(context.Background(), run)
	require.NoError(t, err)
	offline, violations, err := contract.Evaluate(context.Background(), run)
	require.NoError(t, err)
	outcome.violations = violations
	outcome.online, err = proto.MarshalOptions{Deterministic: true}.Marshal(live)
	require.NoError(t, err)
	outcome.offline, err = proto.MarshalOptions{Deterministic: true}.Marshal(offline)
	require.NoError(t, err)
	require.Equal(t, outcome.online, outcome.offline, "online and offline Verdicts are byte-identical")
	return outcome
}

// firstDifferingRule names the differing rule-ID lists, or the first rule ID whose RuleVerdict
// differs, or returns "" when every RuleVerdict matches.
func firstDifferingRule(t *testing.T, want, got []byte) string {
	t.Helper()
	var wantVerdict, gotVerdict testpilotspb.Verdict
	require.NoError(t, proto.Unmarshal(want, &wantVerdict))
	require.NoError(t, proto.Unmarshal(got, &gotVerdict))
	ids := func(verdict *testpilotspb.Verdict) []string {
		var result []string
		for _, rule := range verdict.Rules {
			result = append(result, rule.RuleId)
		}
		return result
	}
	if !slices.Equal(ids(&wantVerdict), ids(&gotVerdict)) {
		return fmt.Sprintf("rule IDs differ: %v, %v", ids(&wantVerdict), ids(&gotVerdict))
	}
	for i, rule := range wantVerdict.Rules {
		if !proto.Equal(rule, gotVerdict.Rules[i]) {
			return fmt.Sprintf("rule %s differs: %v, %v", rule.RuleId, rule, gotVerdict.Rules[i])
		}
	}
	return ""
}

func nexusWorld(t *testing.T) instanceWorld {
	t.Helper()
	prepared, view := nexusCorrelationFixture(t)
	return instanceWorld{catalog: prepared.catalog, view: view, limits: &testpilotspb.ContractLimits{MaxRules: 16, MaxStates: 64, MaxTransitions: 64, MaxExpressionDepth: 12, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 16, MaxCaptureBytes: 65536}}
}

func scheduledOperation() *testpilotspb.Expression {
	return nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_scheduled_event_attributes"), "operation")
}

// selectsOperation matches a scheduled event of operation, the pair Case's capture selector.
func selectsOperation(operation *testpilotspb.Expression) *testpilotspb.Expression {
	return all(present(nexusObservation()), equal(scheduledOperation(), operation))
}

// pairRule is the pair Case's capture shape: capture the scheduled event of operation, then match
// the completion that names it. A crossed completion leaves it pending.
func pairRule(id string, operation *testpilotspb.Expression) *testpilotspb.ContractRule {
	captured := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_CaptureId{CaptureId: "nexusOperationScheduled-relation"}}}}
	rule := &testpilotspb.ContractRule{
		RuleId: id, Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "pending",
		States: []*testpilotspb.ContractState{
			{StateId: "pending", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
			{StateId: "nexusOperationScheduled", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
			{StateId: "satisfied", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
		},
		Captures: []*testpilotspb.ContractCapture{{CaptureId: "nexusOperationScheduled-relation", Type: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.api.history.v1.HistoryEvent"}}}}},
		Transitions: []*testpilotspb.ContractTransition{
			nexusTransition("capture-nexusOperationScheduled-relation", "pending", "nexusOperationScheduled", selectsOperation(operation)),
			nexusTransition("match-nexusOperationCompleted-relation", "nexusOperationScheduled", "satisfied", all(present(captured), equal(nexusPath(captured, "event_id"), nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_completed_event_attributes"), "scheduled_event_id")))),
		},
	}
	rule.Transitions[0].CaptureAssignments = []*testpilotspb.ContractCaptureAssignment{{CaptureId: "nexusOperationScheduled-relation", ObservationId: "history-event"}}
	return rule
}

func nexusRun(t *testing.T, disposition testpilotspb.RunDisposition, events ...*historypb.HistoryEvent) *testpilotspb.Run {
	t.Helper()
	run := &testpilotspb.Run{RunId: "run", CaseId: "case", ProgramId: "program", Disposition: disposition, Events: []*testpilotspb.RunEvent{event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED)}}
	for _, history := range events {
		run.Events = append(run.Events, nexusHistoryRunEvent(t, int64(len(run.Events)+1), history))
	}
	closure := int64(len(run.Events) + 1)
	run.Events = append(run.Events, event(closure, closure, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED))
	return run
}

func nexusScheduled(id int64, operation string) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{EventId: id, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{Operation: operation, RequestId: fmt.Sprint("request-", id)}}}
}

// nexusStarted carries no operation, so it carries no instance's value.
func nexusStarted(id, scheduled int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{EventId: id, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: scheduled}}}
}

func nexusCompleted(id, scheduled int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{EventId: id, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: scheduled}}}
}

// pairInstances is the pair Case's Rule over two instances, beside a plain Rule and a one-instance
// Rule of the same shape.
func pairInstances(t *testing.T) (tc instanceCase) {
	relation := pairRule("relation", instanceValue("op"))
	instanced(relation, "complete-1", "complete-2")
	single := pairRule("single", instanceValue("op"))
	instanced(single, "complete-2")
	tc.name, tc.world = "pair", nexusWorld(t)
	tc.contract = &testpilotspb.Contract{ContractId: "contract", Rules: []*testpilotspb.ContractRule{pairRule("plain", textLiteral("complete-1")), relation, single}}
	completed := testpilotspb.RUN_DISPOSITION_COMPLETED
	incomplete := nexusRun(t, testpilotspb.RUN_DISPOSITION_INCOMPLETE, nexusScheduled(1, "complete-1"), nexusScheduled(2, "complete-2"), nexusCompleted(3, 1), nexusCompleted(4, 2))
	incomplete.Events[3].ExecutionIncomplete = true
	tc.runs = []namedRun{
		{"satisfied", nexusRun(t, completed, nexusScheduled(1, "complete-1"), nexusScheduled(2, "complete-2"), nexusCompleted(3, 1), nexusCompleted(4, 2)), testpilotspb.VERDICT_STATUS_SATISFIED},
		{"crossed completion", nexusRun(t, completed, nexusScheduled(1, "complete-1"), nexusScheduled(2, "complete-2"), nexusCompleted(3, 2), nexusCompleted(4, 999)), testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{"no instance's value", nexusRun(t, completed, nexusStarted(1, 7), nexusScheduled(2, "complete-9"), nexusCompleted(3, 2)), testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{"incomplete", incomplete, testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
	}
	return tc
}

// safetyInstances rejects its instance's operation, so a scheduled event of that operation violates
// that instance alone and stops the Run.
func safetyInstances(t *testing.T) (tc instanceCase) {
	forbid := &testpilotspb.ContractRule{
		RuleId: "forbid", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "pending",
		States: []*testpilotspb.ContractState{
			{StateId: "pending", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
			{StateId: "satisfied", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
			{StateId: "violated", Status: testpilotspb.CONTRACT_STATE_STATUS_VIOLATED},
		},
		Transitions: []*testpilotspb.ContractTransition{
			nexusTransition("reject", "pending", "violated", selectsOperation(instanceValue("op"))),
			nexusTransition("finish", "pending", "satisfied", present(nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_completed_event_attributes"), "scheduled_event_id"))),
		},
	}
	instanced(forbid, "complete-1", "complete-2", "complete-3")
	tc.name, tc.world = "safety", nexusWorld(t)
	tc.contract = &testpilotspb.Contract{ContractId: "contract", Rules: []*testpilotspb.ContractRule{forbid, pairRule("plain", textLiteral("complete-9"))}}
	tc.runs = []namedRun{
		{"violated", nexusRun(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, nexusScheduled(1, "complete-2"), nexusCompleted(2, 1)), testpilotspb.VERDICT_STATUS_VIOLATED},
		{"matching no instance", nexusRun(t, testpilotspb.RUN_DISPOSITION_COMPLETED, nexusScheduled(1, "complete-9"), nexusCompleted(2, 1)), testpilotspb.VERDICT_STATUS_SATISFIED},
	}
	return tc
}

// deadlineInstances gives each instance its own rule_events counter: the instance whose operation
// is scheduled is satisfied and stops counting, the other expires.
func deadlineInstances(t *testing.T) (tc instanceCase) {
	seen := &testpilotspb.ContractRule{
		RuleId: "seen", Kind: testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, InitialStateId: "pending",
		States: []*testpilotspb.ContractState{
			{StateId: "pending", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
			{StateId: "satisfied", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
			{StateId: "violated", Status: testpilotspb.CONTRACT_STATE_STATUS_VIOLATED},
		},
		Transitions: []*testpilotspb.ContractTransition{nexusTransition("see", "pending", "satisfied", selectsOperation(instanceValue("op")))},
		Deadline:    &testpilotspb.Deadline{ViolationStateId: "violated", Bound: &testpilotspb.Deadline_RuleEvents{RuleEvents: 4}},
	}
	instanced(seen, "complete-1", "complete-2")
	tc.name, tc.world = "deadline", nexusWorld(t)
	tc.contract = &testpilotspb.Contract{ContractId: "contract", Rules: []*testpilotspb.ContractRule{seen}}
	tc.runs = []namedRun{
		{"deadline expired", nexusRun(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, nexusScheduled(1, "complete-1"), nexusStarted(2, 1), nexusStarted(3, 1)), testpilotspb.VERDICT_STATUS_VIOLATED},
		{"both seen", nexusRun(t, testpilotspb.RUN_DISPOSITION_COMPLETED, nexusScheduled(1, "complete-2"), nexusScheduled(2, "complete-1")), testpilotspb.VERDICT_STATUS_SATISFIED},
	}
	return tc
}

// correlatedInstances places the correlated rule's verdict after the Rule instances'.
func correlatedInstances(t *testing.T) (tc instanceCase) {
	contract, catalog, view, limits, correlated := correlatedFixture(t, 1)
	evidence := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ObservationId{ObservationId: "evidence"}}}}
	operation := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: evidence, Path: "operation"}}}
	rule := &testpilotspb.ContractRule{
		RuleId: "operation", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "pending",
		States: []*testpilotspb.ContractState{
			{StateId: "pending", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
			{StateId: "satisfied", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
		},
		Transitions: []*testpilotspb.ContractTransition{transition("see", "pending", "satisfied", all(present(evidence), equal(operation, instanceValue("op"))))},
	}
	instanced(rule, "a", "b")
	contract.Rules = []*testpilotspb.ContractRule{rule}
	tc.name, tc.world, tc.contract = "correlated", instanceWorld{catalog: catalog, view: view, limits: limits, correlated: correlated}, contract
	run := func(disposition testpilotspb.RunDisposition, operations ...string) *testpilotspb.Run {
		events := []*testpilotspb.RunEvent{event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED)}
		for i, operation := range operations {
			events = append(events, correlatedEvent(t, int64(i+2), correlatedEvidence(int64(i), "both", operation)))
		}
		closure := int64(len(events) + 1)
		events = append(events, event(closure, closure, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED))
		return &testpilotspb.Run{RunId: "run", CaseId: "correlated.case", ProgramId: "correlated.program", Disposition: disposition, Events: events}
	}
	tc.runs = []namedRun{
		{"satisfied", run(testpilotspb.RUN_DISPOSITION_COMPLETED, "a", "b"), testpilotspb.VERDICT_STATUS_SATISFIED},
		{"one operation", run(testpilotspb.RUN_DISPOSITION_COMPLETED, "a"), testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
	}
	return tc
}
