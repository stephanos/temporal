package testpilot

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

const (
	nexusPairArtifactNamespace        = "nexus-pair-namespace"
	nexusPairArtifactTaskQueue        = "nexus-pair-task-queue"
	nexusPairArtifactHandlerTaskQueue = "nexus-pair-handler-task-queue"
	nexusPairArtifactEndpoint         = "nexus-pair-endpoint"
	nexusPairRelationDefinitionID     = "temporal.nexus.pair.property.completionReferencesSchedule"
	nexusPairRuleID                   = "relation"
	nexusPairInstanceValueID          = "operation"
)

// TestNexusPairCaseCarriesOneCaptureRuleWithTwoInstances prepares the unchanged pair Case bytes
// offline. The Model's one field relation, over two instances of the operation, lowers to one
// capture Rule with one Rule instance per operation: the Rule retains the scheduled event that
// records the operation its instance names and matches the completion's reference against it; the
// two operations share no handler, slot or instruction; and the correlated capability the Query's
// Property lowered into sits beside them.
func TestNexusPairCaseCarriesOneCaptureRuleWithTwoInstances(t *testing.T) {
	source := loadLeanCase(t, NexusPairFixture)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	rules := source.GetContract().GetRules()
	require.Len(t, rules, 1)
	rule := rules[0]
	require.Equal(t, nexusPairRuleID, rule.GetRuleId())
	require.Equal(t, nexusPairRelationDefinitionID+"."+nexusPairRuleID, localNameDefinition(t, source, nexusPairRuleID))
	require.Nil(t, rule.GetDeadline())
	require.Len(t, rule.GetCaptures(), 1)
	require.Equal(t, "nexusOperationScheduled-"+nexusPairRuleID, rule.GetCaptures()[0].GetCaptureId())
	require.Len(t, rule.GetTransitions(), 2)
	require.Equal(t, "capture-nexusOperationScheduled-"+nexusPairRuleID, rule.GetTransitions()[0].GetTransitionId())
	require.Equal(t, "match-nexusOperationCompleted-"+nexusPairRuleID, rule.GetTransitions()[1].GetTransitionId())
	require.Len(t, rule.GetInstanceValues(), 1)
	require.Equal(t, nexusPairInstanceValueID, rule.GetInstanceValues()[0].GetInstanceValueId())
	require.Equal(t, testpilotspb.SCALAR_KIND_TEXT, rule.GetInstanceValues()[0].GetType().GetScalar().GetKind())
	require.Len(t, rule.GetInstances(), 2)
	for index, instance := range []struct{ ruleID, operation string }{
		{NexusPairFirstRuleID, NexusPairFirstOperation},
		{NexusPairSecondRuleID, NexusPairSecondOperation},
	} {
		ruleInstance := rule.GetInstances()[index]
		require.Equal(t, instance.ruleID, ruleInstance.GetRuleId())
		require.Equal(t, nexusPairRelationDefinitionID+"."+instance.ruleID, localNameDefinition(t, source, instance.ruleID))
		require.Len(t, ruleInstance.GetAssignments(), 1)
		require.Equal(t, nexusPairInstanceValueID, ruleInstance.GetAssignments()[0].GetInstanceValueId())
		require.Equal(t, instance.operation, ruleInstance.GetAssignments()[0].GetValue().GetStringValue())
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

	prepareUnchanged(t, source, nexusPairArtifactProfile(catalog))
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
	prepared := prepareUnchanged(t, source, nexusPairArtifactProfile(catalog))

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
	var closed atomic.Bool
	return &scriptedSession{
		runID: runID, namespace: nexusPairArtifactNamespace, taskQueue: nexusPairArtifactTaskQueue, bridge: &nexusPairBridge{},
		history: func(instructionID string) []*historypb.HistoryEvent {
			events := nexusPairHistory(runID)
			switch {
			case instructionID == "await-close":
				closed.Store(true)
				return events[len(events)-1:]
			case !closed.Load():
				// The first operation scheduled and started; the workflow has not yet scheduled
				// the second.
				return events[:2]
			default:
				return events
			}
		},
	}, nil
}

// nexusPairArtifactProfile is the pair Profile over this package's physical environment values.
func nexusPairArtifactProfile(catalog *testpilot.Catalog) testpilot.ProfileSpec {
	return NexusPairProfile(catalog, NexusCallerEnvironment{
		Namespace: nexusPairArtifactNamespace, TaskQueue: nexusPairArtifactTaskQueue,
		HandlerTaskQueue: nexusPairArtifactHandlerTaskQueue, NexusEndpoint: nexusPairArtifactEndpoint,
	})
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
		closedEvent(11),
	}
}
