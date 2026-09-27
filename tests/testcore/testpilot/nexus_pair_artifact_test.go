package testpilot

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	nexusPairArtifactNamespace        = "nexus-pair-namespace"
	nexusPairArtifactTaskQueue        = "nexus-pair-task-queue"
	nexusPairArtifactHandlerTaskQueue = "nexus-pair-handler-task-queue"
	nexusPairArtifactEndpoint         = "nexus-pair-endpoint"
	nexusPairRelationDefinitionID     = "temporal.nexus.pair.property.completionReferencesSchedule"
)

// TestNexusPairCaseCarriesOneCaptureRulePerInstance prepares the unchanged pair Case bytes offline.
// The Model's one field relation, over two instances of the operation, lowers to one capture rule
// per instance: each retains the scheduled event that records its own operation and matches the
// completion's reference against it; the two operations share no handler, slot or instruction; and
// the correlated capability the Query's Property lowered into sits beside them.
func TestNexusPairCaseCarriesOneCaptureRulePerInstance(t *testing.T) {
	source := loadLeanCase(t, NexusPairFixture)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	rules := source.GetContract().GetRules()
	require.Len(t, rules, 2)
	for index, ruleID := range []string{NexusPairFirstRuleID, NexusPairSecondRuleID} {
		rule := rules[index]
		require.Equal(t, ruleID, rule.GetRuleId())
		require.Equal(t, nexusPairRelationDefinitionID+"."+ruleID, localNameDefinition(t, source, ruleID))
		require.Equal(t, testpilotspb.CONTRACT_RULE_KIND_SAFETY, rule.GetKind())
		require.Len(t, rule.GetCaptures(), 1)
		require.Len(t, rule.GetTransitions(), 2)
		require.Equal(t, "capture-nexusOperationScheduled-"+ruleID, rule.GetTransitions()[0].GetTransitionId())
		require.Equal(t, "match-nexusOperationCompleted-"+ruleID, rule.GetTransitions()[1].GetTransitionId())
	}
	require.NotNil(t, source.GetContract().GetCorrelated())

	var handlers []string
	for _, entrypoint := range source.GetProgram().GetEntrypoints() {
		if handler := entrypoint.GetNexusHandler(); handler != nil {
			handlers = append(handlers, handler.GetOperation())
		}
	}
	require.Equal(t, []string{NexusPairFirstOperation, NexusPairSecondOperation}, handlers)
	require.Len(t, source.GetProgram().GetSlots(), 2)

	prepared, err := testpilot.Prepare(source, NexusPairProfile(catalog, NexusCallerEnvironment{
		Namespace: nexusPairArtifactNamespace, TaskQueue: nexusPairArtifactTaskQueue,
		HandlerTaskQueue: nexusPairArtifactHandlerTaskQueue, NexusEndpoint: nexusPairArtifactEndpoint,
	}))
	require.NoError(t, err)
	require.Equal(t, source.GetCaseId(), prepared.Snapshot().GetCaseId())
}

// TestNexusPairCaseReadsEveryInstanceScheduledEvent runs the pair Case offline against a history
// that grows the way the live server's does. The workflow schedules the second operation only once
// the first has started, so until the workflow closes the history holds the first operation's
// scheduled and started events and nothing of the second: a scheduled read answered there lifts one
// scheduled event, and the second operation's started event then reaches the monitor unauthorized.
// The Case must read every instance's scheduled event whenever the double answers.
func TestNexusPairCaseReadsEveryInstanceScheduledEvent(t *testing.T) {
	source := loadLeanCase(t, NexusPairFixture)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, NexusPairProfile(catalog, NexusCallerEnvironment{
		Namespace: nexusPairArtifactNamespace, TaskQueue: nexusPairArtifactTaskQueue,
		HandlerTaskQueue: nexusPairArtifactHandlerTaskQueue, NexusEndpoint: nexusPairArtifactEndpoint,
	}))
	require.NoError(t, err)

	run, verdict, err := prepared.Run(t.Context(), &nexusPairGapDriver{identity: prepared.Identity()})
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), "diagnostics: %v", run.GetDiagnostics())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	for index, ruleID := range []string{NexusPairFirstRuleID, NexusPairSecondRuleID} {
		require.Equal(t, ruleID, verdict.GetRules()[index].GetRuleId())
		require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_SATISFIED, verdict.GetRules()[index].GetStatus())
	}
}

// nexusPairGapDriver answers the pair Case's reads from one scripted history: the gap history
// until the workflow's close read resolves, and the whole history after it.
type nexusPairGapDriver struct {
	identity testpilot.DriverIdentity
}

func (d *nexusPairGapDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}

func (*nexusPairGapDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }

func (*nexusPairGapDriver) Open(_ context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if program.Snapshot().GetProgramId() != "temporal.case.nexusPairTests.bothComplete.program" {
		return nil, temporal.ErrInvalid
	}
	return &nexusPairGapSession{artifactSession: artifactSession{runID: runID}, bridge: &nexusPairBridge{}}, nil
}

type nexusPairGapSession struct {
	artifactSession
	bridge *nexusPairBridge
	closed atomic.Bool
}

func (s *nexusPairGapSession) InvokeRPC(_ context.Context, coordinate testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.EffectHandle, error) {
	if method == nil {
		return nil, temporal.ErrInvalid
	}
	switch coordinate.InstructionID {
	case "start-workflow":
		return &artifactEffect{result: succeededResult(&workflowservice.StartWorkflowExecutionResponse{RunId: s.runID})}, nil
	case "await-close":
		var typed workflowservice.GetWorkflowExecutionHistoryRequest
		if err := decodeArtifactRequest(request, &typed); err != nil {
			return nil, fmt.Errorf("decode close request: %w", err)
		}
		if typed.GetHistoryEventFilterType() != enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT {
			return nil, temporal.ErrInvalid
		}
		s.closed.Store(true)
		events := nexusPairHistory(s.runID)
		return &artifactEffect{result: succeededResult(&workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: events[len(events)-1:]}})}, nil
	case "await-scheduled", "history":
		events := nexusPairHistory(s.runID)
		if !s.closed.Load() {
			// The first operation scheduled and started; the workflow has not yet scheduled the
			// second.
			events = events[:2]
		}
		return &artifactEffect{result: succeededResult(&workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: events}})}, nil
	default:
		return nil, temporal.ErrInvalid
	}
}

// PollRPC answers the scheduled poll once, from the history as it stands when the poll is issued.
func (s *nexusPairGapSession) PollRPC(ctx context.Context, coordinate testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message, interval time.Duration, satisfied testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	if coordinate.InstructionID != "await-scheduled" || interval <= 0 || satisfied == nil {
		return nil, temporal.ErrInvalid
	}
	handle, err := s.InvokeRPC(ctx, coordinate, role, method, request)
	if err != nil {
		return nil, err
	}
	result, err := handle.Wait(ctx)
	if err != nil {
		return nil, err
	}
	done, err := satisfied(ctx, result.Response)
	if err != nil {
		return nil, err
	}
	if !done {
		return nil, fmt.Errorf("scheduled poll unsatisfied for run %q: %w", s.runID, temporal.ErrInvalid)
	}
	return handle, nil
}

func (s *nexusPairGapSession) Bridge(context.Context) (testpilot.HandleBridge, error) {
	return s.bridge, nil
}

// nexusPairBridge holds one published authority per instance, each ready at once and consumed once.
type nexusPairBridge struct {
	consumed sync.Map
}

func (*nexusPairBridge) Publish(context.Context, testpilot.Coordinate, string, testpilot.OpaqueHandle) error {
	return nil
}
func (*nexusPairBridge) Await(context.Context, string) error { return nil }
func (b *nexusPairBridge) Consume(_ context.Context, slot string) (testpilot.OpaqueHandle, error) {
	if _, loaded := b.consumed.LoadOrStore(slot, true); loaded {
		return nil, temporal.ErrInvalid
	}
	return &struct{}{}, nil
}

// nexusPairHistory is the pair's history in the order the live server records it when the first
// operation completes before the second is scheduled.
func nexusPairHistory(runID string) []*historypb.HistoryEvent {
	scheduled := func(id int64, operation string) *historypb.HistoryEvent {
		return &historypb.HistoryEvent{
			EventId: id, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
			Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{
				Endpoint: nexusPairArtifactEndpoint, Service: NexusPairService, Operation: operation, RequestId: runID + "-" + operation,
			}},
		}
	}
	started := func(id, scheduledID int64) *historypb.HistoryEvent {
		return &historypb.HistoryEvent{
			EventId: id, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED,
			Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: scheduledID}},
		}
	}
	completed := func(id, scheduledID int64) *historypb.HistoryEvent {
		return &historypb.HistoryEvent{
			EventId: id, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
			Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: scheduledID}},
		}
	}
	return []*historypb.HistoryEvent{
		scheduled(5, NexusPairFirstOperation), started(6, 5), completed(7, 5),
		scheduled(8, NexusPairSecondOperation), started(9, 8), completed(10, 8),
		{
			EventId: 11, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{}},
		},
	}
}
