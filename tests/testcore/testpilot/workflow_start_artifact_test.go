package testpilot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/protobuf/proto"
)

// localNameDefinition returns the Definition ID the Case's provenance maps a Case-local name to; a
// name no row maps is its own Definition ID.
func localNameDefinition(t testing.TB, source *testpilotspb.Case, localName string) string {
	t.Helper()
	for _, row := range source.GetProvenance().GetLocalNames() {
		if row.GetLocalName() == localName {
			return row.GetDefinitionId()
		}
	}
	return localName
}

// workflowStartArtifactPrepared prepares the unchanged workflow-start Case bytes against the
// Profile the live test uses, with the physical environment values this package owns. The Case
// names its monitor rule by its Case-local name; its provenance maps it to the relation's
// Definition ID.
func workflowStartArtifactPrepared(t testing.TB) (*testpilot.PreparedCase, *testpilotspb.Case) {
	t.Helper()
	source := loadLeanCase(t, WorkflowStartFixture)
	require.Equal(t, WorkflowStartRuleDefinitionID, localNameDefinition(t, source, WorkflowStartRuleID))
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	return prepareUnchanged(t, source, WorkflowStartProfile(catalog, WorkflowStartEnvironment{
		Namespace: workflowStartArtifactNamespace, TaskQueue: workflowStartArtifactTaskQueue,
	})), source
}

// monitorRule is the one monitor rule of a Verdict: the relation's, beside the correlated
// capability the Query's Property lowered into.
func monitorRule(t testing.TB, verdict *testpilotspb.Verdict) *testpilotspb.RuleVerdict {
	t.Helper()
	for _, rule := range verdict.GetRules() {
		if rule.GetRuleId() == WorkflowStartRuleID {
			return rule
		}
	}
	require.FailNow(t, "verdict carries no monitor rule", "%s", WorkflowStartRuleID)
	return nil
}

// TestWorkflowStartCaseTenfoldRecordedTypeVariation runs the Case ten times against histories that
// differ only in the workflow type the started event records. The one history recording the
// submitted type satisfies the clause; the nine tampered ones violate it, which is the answer the
// model Property gives and a distinct answer from the missing evidence below.
func TestWorkflowStartCaseTenfoldRecordedTypeVariation(t *testing.T) {
	prepared, _ := workflowStartArtifactPrepared(t)
	runIDs := make(map[string]struct{}, 10)
	for index := range 10 {
		recorded := WorkflowStartWorkflowType
		if index > 0 {
			recorded = WorkflowStartWorkflowType + "-tampered-" + strconv.Itoa(index)
		}
		t.Run("recorded-"+recorded, func(t *testing.T) {
			driver := &workflowStartDriver{identity: prepared.Identity(), recordedType: recorded}
			run, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.True(t, proto.Equal(verdict, run.GetVerdict()))

			if index == 0 {
				require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
				require.Equal(t, "satisfied", monitorRule(t, verdict).GetTerminalStateId())
			} else {
				require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, verdict.GetStatus())
				require.Equal(t, "violated", monitorRule(t, verdict).GetTerminalStateId())
			}
			require.Len(t, monitorRule(t, verdict).GetSupportingEventSequences(), 1)
			requireStartedEventRecords(t, run, monitorRule(t, verdict).GetSupportingEventSequences()[0], recorded)

			require.NotContains(t, runIDs, run.GetRunId())
			runIDs[run.GetRunId()] = struct{}{}
		})
	}
	require.Len(t, runIDs, 10)
}

// TestWorkflowStartCaseMissingRecordedTypeStaysInconclusive checks the third answer. A history whose
// events never establish the compared field leaves the rule pending, and a pending safety rule
// closes inconclusive rather than either satisfied or violated.
func TestWorkflowStartCaseMissingRecordedTypeStaysInconclusive(t *testing.T) {
	prepared, _ := workflowStartArtifactPrepared(t)
	for _, test := range []struct {
		name   string
		driver *workflowStartDriver
	}{
		{name: "no started event", driver: &workflowStartDriver{omitStarted: true}},
		{name: "started event without a workflow type", driver: &workflowStartDriver{omitWorkflowType: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.driver.identity = prepared.Identity()
			test.driver.recordedType = WorkflowStartWorkflowType
			run, verdict, err := prepared.Run(t.Context(), test.driver)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
			require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
			require.Empty(t, monitorRule(t, verdict).GetTerminalStateId())
			require.Empty(t, monitorRule(t, verdict).GetSupportingEventSequences())
		})
	}
}

// TestWorkflowStartCaseBoundedHistoryLoad grows the history the Case reads until it no longer fits
// the response budget its Profile allows. Inside the budget the clause is decided as before;
// past it the instruction reports its own resource failure and the Run closes inconclusive. No
// load turns an unread field into a satisfied clause.
func TestWorkflowStartCaseBoundedHistoryLoad(t *testing.T) {
	prepared, _ := workflowStartArtifactPrepared(t)
	limits, _, _ := temporal.DefaultCeilings()
	payloadBudget := limits.GetMaxInstructionResponseBytes()
	collectionBudget := limits.GetMaxPathFanout()
	require.Positive(t, payloadBudget)
	require.Positive(t, collectionBudget)

	// The two bounds are exercised one at a time. The collection cases carry empty filler names so
	// their responses stay far inside the payload budget, and the payload case carries four events
	// so it stays far inside the collection budget. The started event the clause reads counts
	// against the collection budget alongside the filler.
	for _, test := range []struct {
		name     string
		filler   int
		bytes    int
		run      testpilotspb.RunDisposition
		verdict  testpilotspb.VerdictStatus
		terminal string
	}{
		{
			name: "inside both budgets", filler: 4, bytes: 64,
			run: testpilotspb.RUN_DISPOSITION_COMPLETED, verdict: testpilotspb.VERDICT_STATUS_SATISFIED,
			terminal: "satisfied",
		},
		{
			name: "at the collection budget", filler: int(collectionBudget) - 1, bytes: 0,
			run: testpilotspb.RUN_DISPOSITION_COMPLETED, verdict: testpilotspb.VERDICT_STATUS_SATISFIED,
			terminal: "satisfied",
		},
		{
			name: "one past the collection budget", filler: int(collectionBudget), bytes: 0,
			run: testpilotspb.RUN_DISPOSITION_INCOMPLETE, verdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
		},
		{
			name: "past the payload budget", filler: 4, bytes: int(payloadBudget) * 2,
			run: testpilotspb.RUN_DISPOSITION_INCOMPLETE, verdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &workflowStartDriver{
				identity: prepared.Identity(), recordedType: WorkflowStartWorkflowType,
				fillerEvents: test.filler, fillerBytes: test.bytes,
			}
			size, observations := int64(proto.Size(driver.historyResponse(""))), int64(test.filler)+1
			if test.bytes == 0 || test.verdict == testpilotspb.VERDICT_STATUS_SATISFIED {
				require.Less(t, size, payloadBudget, "a collection case stays inside the payload budget")
			} else {
				require.Greater(t, size, payloadBudget, "the payload case exceeds the payload budget")
				require.LessOrEqual(t, observations, collectionBudget,
					"the payload case stays inside the collection budget")
			}
			run, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, test.run, run.GetDisposition())
			require.Equal(t, test.verdict, verdict.GetStatus())
			require.Equal(t, test.terminal, monitorRule(t, verdict).GetTerminalStateId())
		})
	}
}

// TestWorkflowStartCaseSequentialAndConcurrentRunIsolation reuses one prepared Case across sequential
// and concurrent Runs driven by two different histories. Each Run answers from its own evidence,
// and no Run carries a state, capture or verdict into another.
func TestWorkflowStartCaseSequentialAndConcurrentRunIsolation(t *testing.T) {
	prepared, _ := workflowStartArtifactPrepared(t)
	matching := &workflowStartDriver{identity: prepared.Identity(), recordedType: WorkflowStartWorkflowType}
	tampered := &workflowStartDriver{identity: prepared.Identity(), recordedType: WorkflowStartWorkflowType + "-tampered"}
	drivers := []*workflowStartDriver{matching, tampered}
	expected := []testpilotspb.VerdictStatus{
		testpilotspb.VERDICT_STATUS_SATISFIED, testpilotspb.VERDICT_STATUS_VIOLATED,
	}

	type outcome struct {
		index   int
		run     *testpilotspb.Run
		verdict *testpilotspb.Verdict
		err     error
	}
	results := make(chan outcome, 20)
	execute := func(index int) {
		run, verdict, err := prepared.Run(t.Context(), drivers[index%len(drivers)])
		results <- outcome{index: index % len(drivers), run: run, verdict: verdict, err: err}
	}
	for index := range 10 {
		execute(index)
	}
	var concurrent sync.WaitGroup
	for index := range 10 {
		concurrent.Go(func() { execute(index) })
	}
	concurrent.Wait()
	close(results)

	runIDs := make(map[string]struct{}, 20)
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, expected[result.index], result.verdict.GetStatus())
		require.Len(t, monitorRule(t, result.verdict).GetSupportingEventSequences(), 1)
		require.True(t, proto.Equal(result.verdict, result.run.GetVerdict()))
		require.NotContains(t, runIDs, result.run.GetRunId())
		runIDs[result.run.GetRunId()] = struct{}{}
	}
	require.Len(t, runIDs, 20)
	require.Equal(t, int64(20), matching.opens.Load()+tampered.opens.Load())
}

// observationValue is the value one Run event carries under an Observation ID. A history read
// emits each event under the history Observation and the correlated-evidence one, so the value is
// selected by ID rather than by position.
func observationValue(t testing.TB, event *testpilotspb.RunEvent, observationID string) *celpb.Value {
	t.Helper()
	for _, observation := range event.GetObservations() {
		if observation.GetObservationId() == observationID {
			return observation.GetValue()
		}
	}
	require.FailNow(t, "run event carries no observation", "%s", observationID)
	return nil
}

// requireStartedEventRecords reads one supporting Observation back out of the Run and checks it is
// the started event whose recorded workflow type the rule compared.
func requireStartedEventRecords(t testing.TB, run *testpilotspb.Run, sequence int64, recorded string) {
	t.Helper()
	require.Positive(t, sequence)
	require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
	event := run.GetEvents()[sequence-1]
	var historyEvent historypb.HistoryEvent
	require.NoError(t, observationValue(t, event, "history-event").GetObjectValue().UnmarshalTo(&historyEvent))
	require.Equal(t, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED, historyEvent.GetEventType())
	require.Equal(t, recorded,
		historyEvent.GetWorkflowExecutionStartedEventAttributes().GetWorkflowType().GetName())
}

// contractReads renders every field read of one Contract rule's transitions in order, each as the
// transition it belongs to, the operand's Observation and path, the operator and the literal it is
// compared with; a presence read carries no operator or literal. It is the structural view the
// migration compares: identity, provenance and fingerprints differ from the hand-written Case, the
// reads must not.
func contractReads(t testing.TB, rule *testpilotspb.ContractRule) []string {
	t.Helper()
	var reads []string
	for _, transition := range rule.GetTransitions() {
		expression := transition.GetPredicate()
		bindings := map[string]*testpilotspb.ExpressionBinding{}
		for _, binding := range expression.GetBindings() {
			bindings[binding.GetVariable()] = binding
		}
		var walk func(string, *celpb.Expr)
		walk = func(transitionID string, node *celpb.Expr) {
			call := node.GetCallExpr()
			require.NotNil(t, call, "%s: %v", transitionID, node)
			switch call.GetFunction() {
			case "_&&_":
				for _, operand := range call.GetArgs() {
					walk(transitionID, operand)
				}
			case "!_":
				require.Len(t, call.GetArgs(), 1)
				walk(transitionID+"/not", call.GetArgs()[0])
			case "hasValue":
				require.Empty(t, call.GetArgs())
				require.NotNil(t, call.GetTarget().GetIdentExpr())
				binding := bindings[call.GetTarget().GetIdentExpr().GetName()]
				require.NotNil(t, binding)
				read := fmt.Sprintf("%s present %s", transitionID, binding.GetReference().GetObservationId())
				if binding.GetPath() != "" {
					read += " " + binding.GetPath()
				}
				reads = append(reads, read)
			case "_==_":
				require.Len(t, call.GetArgs(), 2)
				operand := call.GetArgs()[0].GetCallExpr()
				require.NotNil(t, operand)
				require.Equal(t, "value", operand.GetFunction())
				require.Empty(t, operand.GetArgs())
				require.NotNil(t, operand.GetTarget().GetIdentExpr())
				binding := bindings[operand.GetTarget().GetIdentExpr().GetName()]
				require.NotNil(t, binding)
				require.NotNil(t, call.GetArgs()[1].GetConstExpr())
				reads = append(reads, fmt.Sprintf("%s compare %s %s %s %q", transitionID,
					binding.GetReference().GetObservationId(), binding.GetPath(), call.GetFunction(),
					call.GetArgs()[1].GetConstExpr().GetStringValue()))
			default:
				require.FailNow(t, "unexpected contract expression", "%s: %v", transitionID, node)
			}
		}
		walk(transition.GetTransitionId(), expression.GetCel().GetExpr())
	}
	return reads
}

// TestWorkflowStartCaseReadsMatchTypedUnaryBaseline compares the migrated Case's monitor rule with
// the hand-written typed unary Contract it replaced: the same safety rule over the same recorded
// field, matched against the workflow type the Program submits. The rule's name and its literal
// are the identity's, everything else is the baseline's, and the first read that differs is named.
func TestWorkflowStartCaseReadsMatchTypedUnaryBaseline(t *testing.T) {
	source := loadLeanCase(t, WorkflowStartFixture)
	require.Len(t, source.GetContract().GetRules(), 1)
	rule := source.GetContract().GetRules()[0]
	require.Equal(t, WorkflowStartRuleID, rule.GetRuleId())
	require.Nil(t, rule.GetDeadline())
	require.Equal(t, "pending", rule.GetInitialStateId())

	recorded := "attributes<workflow_execution_started_event_attributes>.workflow_type.name"
	expected := []string{
		"match-" + WorkflowStartRuleID + " present history-event",
		fmt.Sprintf("match-%s compare history-event %s _==_ %q", WorkflowStartRuleID, recorded, WorkflowStartWorkflowType),
		"reject-" + WorkflowStartRuleID + " present history-event",
		"reject-" + WorkflowStartRuleID + " present history-event " + recorded,
		fmt.Sprintf("reject-%s/not compare history-event %s _==_ %q", WorkflowStartRuleID, recorded, WorkflowStartWorkflowType),
	}
	actual := contractReads(t, rule)
	for index := range min(len(expected), len(actual)) {
		require.Equalf(t, expected[index], actual[index], "first differing read at %d", index)
	}
	require.Equal(t, expected, actual)
}

// workflowStartDriver answers the typed unary Program's two RPCs from a scripted history. Only the
// history varies across the tests above; the request the Program builds is validated identically
// on every Run, so a Case that stopped constructing the submitted workflow type fails here.
type workflowStartDriver struct {
	identity         testpilot.DriverIdentity
	recordedType     string
	fillerEvents     int
	fillerBytes      int
	omitStarted      bool
	omitWorkflowType bool
	opens            atomic.Int64
}

func (d *workflowStartDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (*workflowStartDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (d *workflowStartDriver) Open(_ context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if program.Snapshot().GetProgramId() != WorkflowStartProgramID {
		return nil, temporal.ErrInvalid
	}
	d.opens.Add(1)
	return &scriptedSession{
		runID: runID, namespace: workflowStartArtifactNamespace, taskQueue: workflowStartArtifactTaskQueue,
		start: func(request *workflowservice.StartWorkflowExecutionRequest) (*testpilotspb.InstructionOutcome, error) {
			if request.GetWorkflowType().GetName() != WorkflowStartWorkflowType {
				return nil, fmt.Errorf("invalid workflow type for run %q: %w", runID, temporal.ErrInvalid)
			}
			return nil, nil
		},
		// The close-event read is answered with the close event alone; the full read with the
		// scripted history the tests vary.
		history: func(instructionID string) []*historypb.HistoryEvent {
			if instructionID == "await-close" {
				return []*historypb.HistoryEvent{closedEvent(2)}
			}
			return d.historyResponse(runID).GetHistory().GetEvents()
		},
	}, nil
}

// historyResponse builds the scripted history: the started event the clause reads, preceded by the
// filler events the load tests use to grow the response past its declared budget. runID is the Run
// the session was opened for; the started event it scripts names the workflow by it, so concurrent
// sessions of one Driver each script their own.
func (d *workflowStartDriver) historyResponse(runID string) *workflowservice.GetWorkflowExecutionHistoryResponse {
	events := make([]*historypb.HistoryEvent, 0, d.fillerEvents+1)
	for index := range d.fillerEvents {
		events = append(events, &historypb.HistoryEvent{
			EventId: int64(index + 1), EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{
				WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
					TaskQueue: &taskqueuepb.TaskQueue{Name: strings.Repeat("f", d.fillerBytes)},
				},
			},
		})
	}
	if !d.omitStarted {
		attributes := &historypb.WorkflowExecutionStartedEventAttributes{
			TaskQueue:           &taskqueuepb.TaskQueue{Name: workflowStartArtifactTaskQueue},
			FirstExecutionRunId: runID,
		}
		if !d.omitWorkflowType {
			attributes.WorkflowType = &commonpb.WorkflowType{Name: d.recordedType}
		}
		events = append(events, &historypb.HistoryEvent{
			EventId: int64(len(events) + 1), EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: attributes,
			},
		})
	}
	return &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: events}}
}
