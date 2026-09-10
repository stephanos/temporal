package testpilot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

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
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	typedUnaryArtifactNamespace = "typed-unary-namespace"
	typedUnaryArtifactTaskQueue = "typed-unary-task-queue"
	typedUnaryRuleID            = "temporal.nexus.success.typed-unary.property.submitted-workflow-type.recorded-workflow-type"
)

// typedUnaryArtifactPrepared prepares the unchanged typed unary Case bytes against the Profile the
// live test uses, with the physical environment values this package owns.
func typedUnaryArtifactPrepared(t testing.TB) (*testpilot.PreparedCase, *testpilotspb.Case) {
	t.Helper()
	source := loadLeanCase(t, "typed-unary")
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, TypedUnaryProfile(catalog, source, TypedUnaryEnvironment{
		Namespace: typedUnaryArtifactNamespace, TaskQueue: typedUnaryArtifactTaskQueue,
	}))
	require.NoError(t, err)
	return prepared, source
}

// TestTypedUnaryCaseTenfoldRecordedTypeVariation runs the Case ten times against histories that
// differ only in the workflow type the started event records. The one history recording the
// submitted type satisfies the clause; the nine tampered ones violate it, which is the answer the
// model Property gives and a distinct answer from the missing evidence below.
func TestTypedUnaryCaseTenfoldRecordedTypeVariation(t *testing.T) {
	prepared, _ := typedUnaryArtifactPrepared(t)
	runIDs := make(map[string]struct{}, 10)
	for index := range 10 {
		recorded := TypedUnaryWorkflowType
		if index > 0 {
			recorded = TypedUnaryWorkflowType + "-tampered-" + strconv.Itoa(index)
		}
		t.Run("recorded-"+recorded, func(t *testing.T) {
			driver := &typedUnaryDriver{identity: prepared.Identity(), recordedType: recorded}
			run, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.True(t, proto.Equal(verdict, run.GetVerdict()))
			require.Len(t, verdict.GetRules(), 1)
			require.Equal(t, typedUnaryRuleID, verdict.GetRules()[0].GetRuleId())

			if index == 0 {
				require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
				require.Equal(t, "satisfied", verdict.GetRules()[0].GetTerminalStateId())
			} else {
				require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, verdict.GetStatus())
				require.Equal(t, "violated", verdict.GetRules()[0].GetTerminalStateId())
			}
			require.Len(t, verdict.GetRules()[0].GetSupportingEventSequences(), 1)
			requireStartedEventRecords(t, run, verdict.GetRules()[0].GetSupportingEventSequences()[0], recorded)

			require.NotContains(t, runIDs, run.GetRunId())
			runIDs[run.GetRunId()] = struct{}{}
		})
	}
	require.Len(t, runIDs, 10)
}

// TestTypedUnaryCaseMissingRecordedTypeStaysInconclusive checks the third answer. A history whose
// events never establish the compared field leaves the rule pending, and a pending safety rule
// closes inconclusive rather than either satisfied or violated.
func TestTypedUnaryCaseMissingRecordedTypeStaysInconclusive(t *testing.T) {
	prepared, _ := typedUnaryArtifactPrepared(t)
	for _, test := range []struct {
		name   string
		driver *typedUnaryDriver
	}{
		{name: "no started event", driver: &typedUnaryDriver{omitStarted: true}},
		{name: "started event without a workflow type", driver: &typedUnaryDriver{omitWorkflowType: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.driver.identity = prepared.Identity()
			test.driver.recordedType = TypedUnaryWorkflowType
			run, verdict, err := prepared.Run(t.Context(), test.driver)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RUN_STATUS_COMPLETED, run.GetStatus())
			require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
			require.Len(t, verdict.GetRules(), 1)
			require.Empty(t, verdict.GetRules()[0].GetTerminalStateId())
			require.Empty(t, verdict.GetRules()[0].GetSupportingEventSequences())
		})
	}
}

// TestTypedUnaryCaseBoundedHistoryLoad grows the history the Case reads until it no longer fits
// the response budget the Program declared. Inside the budget the clause is decided as before;
// past it the instruction reports its own resource failure and the Run closes inconclusive. No
// load turns an unread field into a satisfied clause.
func TestTypedUnaryCaseBoundedHistoryLoad(t *testing.T) {
	prepared, source := typedUnaryArtifactPrepared(t)
	payloadBudget := source.GetProgram().GetLimits().GetMaxResponseBytes()
	collectionBudget := source.GetProgram().GetLimits().GetMaxPathFanout()
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
		run      testpilotspb.RunStatus
		verdict  testpilotspb.VerdictStatus
		terminal string
	}{
		{
			name: "inside both budgets", filler: 4, bytes: 64,
			run: testpilotspb.RUN_STATUS_COMPLETED, verdict: testpilotspb.VERDICT_STATUS_SATISFIED,
			terminal: "satisfied",
		},
		{
			name: "at the collection budget", filler: int(collectionBudget) - 1, bytes: 0,
			run: testpilotspb.RUN_STATUS_COMPLETED, verdict: testpilotspb.VERDICT_STATUS_SATISFIED,
			terminal: "satisfied",
		},
		{
			name: "one past the collection budget", filler: int(collectionBudget), bytes: 0,
			run: testpilotspb.RUN_STATUS_INCOMPLETE, verdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
		},
		{
			name: "past the payload budget", filler: 4, bytes: int(payloadBudget) * 2,
			run: testpilotspb.RUN_STATUS_INCOMPLETE, verdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &typedUnaryDriver{
				identity: prepared.Identity(), recordedType: TypedUnaryWorkflowType,
				fillerEvents: test.filler, fillerBytes: test.bytes,
			}
			size, observations := int64(proto.Size(driver.historyResponse())), int64(test.filler)+1
			if test.bytes == 0 || test.verdict == testpilotspb.VERDICT_STATUS_SATISFIED {
				require.Less(t, size, payloadBudget, "a collection case stays inside the payload budget")
			} else {
				require.Greater(t, size, payloadBudget, "the payload case exceeds the payload budget")
				require.LessOrEqual(t, observations, collectionBudget,
					"the payload case stays inside the collection budget")
			}
			run, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, test.run, run.GetStatus())
			require.Equal(t, test.verdict, verdict.GetStatus())
			require.Len(t, verdict.GetRules(), 1)
			require.Equal(t, test.terminal, verdict.GetRules()[0].GetTerminalStateId())
		})
	}
}

// TestTypedUnaryCaseSequentialAndConcurrentRunIsolation reuses one prepared Case across sequential
// and concurrent Runs driven by two different histories. Each Run answers from its own evidence,
// and no Run carries a state, capture or verdict into another.
func TestTypedUnaryCaseSequentialAndConcurrentRunIsolation(t *testing.T) {
	prepared, _ := typedUnaryArtifactPrepared(t)
	matching := &typedUnaryDriver{identity: prepared.Identity(), recordedType: TypedUnaryWorkflowType}
	tampered := &typedUnaryDriver{identity: prepared.Identity(), recordedType: TypedUnaryWorkflowType + "-tampered"}
	drivers := []*typedUnaryDriver{matching, tampered}
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
		require.Len(t, result.verdict.GetRules(), 1)
		require.Len(t, result.verdict.GetRules()[0].GetSupportingEventSequences(), 1)
		require.True(t, proto.Equal(result.verdict, result.run.GetVerdict()))
		require.NotContains(t, runIDs, result.run.GetRunId())
		runIDs[result.run.GetRunId()] = struct{}{}
	}
	require.Len(t, runIDs, 20)
	require.Equal(t, int64(20), matching.opens.Load()+tampered.opens.Load())
}

// requireStartedEventRecords reads one supporting Observation back out of the Run and checks it is
// the started event whose recorded workflow type the rule compared.
func requireStartedEventRecords(t testing.TB, run *testpilotspb.Run, sequence int64, recorded string) {
	t.Helper()
	require.Positive(t, sequence)
	require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
	event := run.GetEvents()[sequence-1]
	require.Len(t, event.GetObservations(), 1)
	require.Equal(t, "history-event", event.GetObservations()[0].GetObservationId())
	var historyEvent historypb.HistoryEvent
	require.NoError(t, event.GetObservations()[0].GetValue().GetMessageValue().UnmarshalTo(&historyEvent))
	require.Equal(t, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED, historyEvent.GetEventType())
	require.Equal(t, recorded,
		historyEvent.GetWorkflowExecutionStartedEventAttributes().GetWorkflowType().GetName())
}

// typedUnaryDriver answers the typed unary Program's two RPCs from a scripted history. Only the
// history varies across the tests above; the request the Program builds is validated identically
// on every Run, so a Case that stopped constructing the submitted workflow type fails here.
type typedUnaryDriver struct {
	identity         testpilot.DriverIdentity
	recordedType     string
	fillerEvents     int
	fillerBytes      int
	omitStarted      bool
	omitWorkflowType bool
	opens            atomic.Int64
}

func (d *typedUnaryDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (*typedUnaryDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (d *typedUnaryDriver) Open(_ context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if program.Snapshot().GetProgramId() != "temporal.case.typed-unary.program" {
		return nil, temporal.ErrInvalid
	}
	d.opens.Add(1)
	return &typedUnarySession{runID: runID, driver: d}, nil
}

type typedUnarySession struct {
	runID  string
	driver *typedUnaryDriver
}

func (s *typedUnarySession) Reserve(_ context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	result := make([]testpilot.ReservationHandle, request.Count)
	for ordinal := range request.Count {
		result[ordinal] = &artifactReservation{
			identity: testpilot.ReservationIdentity{
				Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: ordinal,
				ID: request.EntrypointID + ".reservation." + strconv.FormatInt(ordinal, 10),
			},
			artifactEffect: artifactEffect{result: succeededResult(nil)},
		}
	}
	return result, nil
}

func (s *typedUnarySession) InvokeRPC(_ context.Context, coordinate testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.EffectHandle, error) {
	if method == nil {
		return nil, temporal.ErrInvalid
	}
	switch coordinate.InstructionID {
	case "start-workflow":
		if string(method.FullName()) != "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution" {
			return nil, temporal.ErrInvalid
		}
		var typed workflowservice.StartWorkflowExecutionRequest
		if err := decodeArtifactRequest(request, &typed); err != nil {
			return nil, fmt.Errorf("decode start request: %w", err)
		}
		if typed.GetNamespace() != typedUnaryArtifactNamespace ||
			typed.GetTaskQueue().GetName() != typedUnaryArtifactTaskQueue ||
			typed.GetWorkflowType().GetName() != TypedUnaryWorkflowType ||
			typed.GetWorkflowId() != s.runID || typed.GetRequestId() != s.runID {
			return nil, fmt.Errorf("invalid start request for run %q: %w", s.runID, temporal.ErrInvalid)
		}
		return &artifactEffect{result: succeededResult(&workflowservice.StartWorkflowExecutionResponse{RunId: s.runID})}, nil
	case "history":
		if string(method.FullName()) != "temporal.api.workflowservice.v1.WorkflowService.GetWorkflowExecutionHistory" {
			return nil, temporal.ErrInvalid
		}
		var typed workflowservice.GetWorkflowExecutionHistoryRequest
		if err := decodeArtifactRequest(request, &typed); err != nil {
			return nil, fmt.Errorf("decode history request: %w", err)
		}
		if typed.GetNamespace() != typedUnaryArtifactNamespace || typed.GetExecution().GetWorkflowId() != s.runID {
			return nil, fmt.Errorf("invalid history request for run %q: %w", s.runID, temporal.ErrInvalid)
		}
		return &artifactEffect{result: succeededResult(s.driver.historyResponse())}, nil
	default:
		return nil, temporal.ErrInvalid
	}
}

func (*typedUnarySession) InvokeCapability(context.Context, testpilot.Coordinate, testpilot.OpaqueCapability, proto.Message) (testpilot.EffectHandle, error) {
	return &artifactEffect{result: succeededResult(nil)}, nil
}
func (*typedUnarySession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	return nil, temporal.ErrInvalid
}
func (*typedUnarySession) Bridge(context.Context) (testpilot.CapabilityBridge, error) {
	return nil, temporal.ErrInvalid
}
func (*typedUnarySession) Quarantine(context.Context, testpilot.EffectHandle) error { return nil }
func (*typedUnarySession) Close(context.Context) error                              { return nil }
func (*typedUnarySession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

// historyResponse builds the scripted history: the started event the clause reads, preceded by the
// filler events the load tests use to grow the response past its declared budget.
func (d *typedUnaryDriver) historyResponse() *workflowservice.GetWorkflowExecutionHistoryResponse {
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
			TaskQueue: &taskqueuepb.TaskQueue{Name: typedUnaryArtifactTaskQueue},
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
