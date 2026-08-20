package temporal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/participant"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type staticHistorySource struct {
	snapshot CorroboratingHistory
	err      error
}

func (s staticHistorySource) ReadHistory(context.Context, HistoryRequest) (CorroboratingHistory, error) {
	return s.snapshot, s.err
}

func TestSDKObservationUsesTargetSpecificHistoryEvidence(t *testing.T) {
	t.Parallel()

	events := map[enumspb.EventType]historyEventPosition{
		enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED:             {sequence: 1},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED:               {sequence: 2},
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED:  {sequence: 1},
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED: {sequence: 2},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:             {sequence: 3},
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED:           {sequence: 4},
		enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:             {sequence: 5},
		enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT:           {sequence: 6},
		enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED:    {sequence: 7},
		enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED:  {sequence: 8},
		enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED:   {sequence: 9},
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:        {sequence: 10},
	}
	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{
			{Identifier: "request", Kind: "request-cancellation"},
			{Identifier: "commit", Kind: "commit-cancellation"},
			{Identifier: "worker", Kind: "worker-returns-success"},
			{Identifier: "persist", Kind: "persist-success"},
			{Identifier: "callback", Kind: "record-callback-response"},
			{Identifier: "callback-register", Kind: "register-callback"},
			{Identifier: "link", Kind: "link-nexus-activity"},
			{Identifier: "route", Kind: "route-workflow-task"},
			{Identifier: "speculative-create", Kind: "create-speculative-workflow-task"},
			{Identifier: "speculative-commit", Kind: "commit-speculative-workflow-task"},
			{Identifier: "dispatch-progress", Kind: "dispatch-assurance-workflow-task"},
			{Identifier: "progress", Kind: "progress-entity"},
		}},
		results: map[string]participant.Result{
			"request": {Status: "completed"}, "commit": {Status: "completed"},
			"worker": {Status: "suppressed"}, "persist": {Status: "suppressed"},
			"callback": {
				Status: "completed", Source: "temporal-nexus-completion-callback-receiver",
				Reference: "workflow/run/callback/delivered", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"callback-register": {
				Status: "completed", Source: "temporal-completion-callback-registration",
				Reference: "workflow/run/callback/registered", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"link": {Status: "completed"},
			"route": {
				Status: "completed", Source: "temporal-sdk-routing", SourceIdentity: "task-queue",
				Reference: "workflow/run/task-queue/task-queue", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"speculative-create": {
				Status: "completed", Source: "temporal-sdk-speculative-update",
				Reference: "workflow/run/speculative-update/create", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"speculative-commit": {
				Status: "completed", Source: "temporal-sdk-speculative-update",
				Reference: "workflow/run/speculative-update/commit", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"dispatch-progress": {
				Status: "completed", Source: "temporal-sdk-workflow-progress",
				Reference: "workflow/run/workflow-progress/dispatch", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"progress": {
				Status: "completed", Source: "temporal-sdk-workflow-progress",
				Reference: "workflow/run/workflow-progress/progress", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
		},
	}
	history := historyPosition{
		events: events, callbackRegistered: true, taskQueues: map[string]bool{"task-queue": true},
		nexusActivityForwardLinked: true, nexusActivityReverseLinked: true,
		nexusTimeoutType: enumspb.TIMEOUT_TYPE_START_TO_CLOSE, nexusTimeoutMessage: "operation timed out",
	}
	session.taskQueue = "task-queue"
	for _, observation := range []string{
		"cancellation-accepted", "cancellation-won", "stale-success-absent",
		"update-accepted", "update-completed", "workflow-task-acknowledged",
		"speculative-task-valid", "workflow-task-not-starved", "nexus-operation-closed",
		"nexus-activity-links-consistent", "nexus-timeout-valid", "callback-reference-valid",
		"callback-response-consistent", "entity-progressed",
		"workflow-routing-isolated",
	} {
		require.True(t, session.observationSatisfied(observation, history), observation)
	}
}

func TestSDKObservationFailsClosedOnMissingTargetEventOrFailedCommand(t *testing.T) {
	t.Parallel()

	session := &sdkSession{results: map[string]participant.Result{"action": {Status: "completed"}}}
	history := historyPosition{events: map[enumspb.EventType]historyEventPosition{
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED: {sequence: 1},
	}}
	require.False(t, session.observationSatisfied("nexus-activity-links-consistent", history))
	require.False(t, session.observationSatisfied("nexus-timeout-valid", history))
	require.False(t, session.observationSatisfied("unknown-observation", history))

	session.results["action"] = participant.Result{
		Status: "failed", TerminalState: "failed",
		TerminalDisposition: participant.TerminalDispositionFailure,
	}
	history.events[enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED] = historyEventPosition{sequence: 2}
	history.events[enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED] = historyEventPosition{sequence: 3}
	require.True(t, session.observationSatisfied("nexus-operation-closed", history))
	missingCallbackResult := &sdkSession{results: map[string]participant.Result{"action": {Status: "completed"}}}
	require.False(t, missingCallbackResult.observationSatisfied("callback-response-consistent", historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED: {sequence: 1},
		},
		callbackRegistered: true,
	}))
}

func TestSDKCallbackObservationRequiresMechanismReceipts(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{
			{Identifier: "register", Kind: "register-callback"},
			{Identifier: "respond", Kind: "record-callback-response"},
		}},
		results: map[string]participant.Result{
			"register": {
				Status: "completed", Source: "generic-sdk", Reference: "register",
				WorkflowID: "workflow", RunID: "run", Lineage: []string{"run"},
			},
			"respond": {
				Status: "completed", Source: "generic-sdk", Reference: "respond",
				WorkflowID: "workflow", RunID: "run", Lineage: []string{"run"},
			},
		},
	}
	history := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:   {sequence: 1},
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED: {sequence: 2},
		},
		callbackRegistered: true,
	}
	require.False(t, session.observationSatisfied("callback-reference-valid", history))
	require.False(t, session.observationSatisfied("callback-response-consistent", history))

	register := session.results["register"]
	register.Source = "temporal-completion-callback-registration"
	session.results["register"] = register
	require.True(t, session.observationSatisfied("callback-reference-valid", history))
	require.False(t, session.observationSatisfied("callback-response-consistent", history))

	response := session.results["respond"]
	response.Source = "temporal-nexus-completion-callback-receiver"
	session.results["respond"] = response
	require.True(t, session.observationSatisfied("callback-response-consistent", history))
}

func TestSDKLineageObservationsRequireTypedMechanismReceipts(t *testing.T) {
	t.Parallel()

	continuation := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{
			Identifier: "continue", Kind: "continue-workflow",
		}}},
		results: map[string]participant.Result{"continue": {
			Status: "completed", Source: "generic-sdk", Reference: "workflow/successor/continued-from/predecessor",
			WorkflowID: "workflow", RunID: "successor",
			Lineage: []string{"experiment", "workflow", "predecessor", "successor"},
		}},
	}
	continuationHistory := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED: {sequence: 1},
		},
		continuedExecutionRunID: "predecessor",
		originalExecutionRunID:  "successor",
		firstExecutionRunID:     "predecessor",
	}
	require.False(t, continuation.observationSatisfied(
		"workflow-continuation-lineage-valid", continuationHistory))
	result := continuation.results["continue"]
	result.Source = "temporal-sdk-continuation"
	continuation.results["continue"] = result
	require.True(t, continuation.observationSatisfied(
		"workflow-continuation-lineage-valid", continuationHistory))

	reset := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{
			Identifier: "reset", Kind: "reset-workflow",
		}}},
		results: map[string]participant.Result{"reset": {
			Status: "completed", Source: "temporal-sdk-reset", Reference: "workflow/reset/reset-from/base",
			WorkflowID: "workflow", RunID: "reset", Lineage: []string{"experiment", "workflow", "base", "reset"},
		}},
	}
	resetHistory := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED: {sequence: 1},
		},
		originalExecutionRunID: "base",
		firstExecutionRunID:    "base",
	}
	require.True(t, reset.observationSatisfied("workflow-reset-lineage-valid", resetHistory))
	resetHistory.firstExecutionRunID = "unrelated"
	require.False(t, reset.observationSatisfied("workflow-reset-lineage-valid", resetHistory))
}

func TestSDKOwnershipObservationRequiresTypedOrderedFencingEvidence(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{
			Identifier: "fence", Kind: "fence-workflow-owner",
		}}},
		results: map[string]participant.Result{"fence": {
			Status: "completed", Source: "generic-sdk", SourceIdentity: "generic-sdk",
			Reference:  "workflow/run/workflow-task/2/fenced-before/5",
			WorkflowID: "workflow", RunID: "run", Lineage: []string{"run"},
		}},
	}
	history := historyPosition{events: map[enumspb.EventType]historyEventPosition{
		enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED:    {sequence: 4},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED: {sequence: 7},
	}}
	require.False(t, session.observationSatisfied("workflow-ownership-fenced", history))

	result := session.results["fence"]
	result.Source = "umpire3-workflow-task-fencer"
	result.SourceIdentity = result.Source
	session.results["fence"] = result
	require.True(t, session.observationSatisfied("workflow-ownership-fenced", history))

	result.Reference = "workflow/run/workflow-task/5/fenced-before/2"
	session.results["fence"] = result
	require.False(t, session.observationSatisfied("workflow-ownership-fenced", history))

	result.Reference = "workflow/run/workflow-task/2/fenced-before/5"
	session.results["fence"] = result
	history.events[enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED] = historyEventPosition{sequence: 8}
	require.False(t, session.observationSatisfied("workflow-ownership-fenced", history))
}

func TestSDKSpeculativeObservationRequiresTypedReceiptsAndUpdateHistory(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{
			{Identifier: "create", Kind: "create-speculative-workflow-task"},
			{Identifier: "commit", Kind: "commit-speculative-workflow-task"},
		}},
		results: map[string]participant.Result{
			"create": {
				Status: "completed", Source: "temporal-sdk-participant",
				Reference: "workflow/run/create", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"commit": {
				Status: "completed", Source: "temporal-sdk-participant",
				Reference: "workflow/run/commit", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
		},
	}
	history := historyPosition{events: map[enumspb.EventType]historyEventPosition{
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED:  {sequence: 3},
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED: {sequence: 4},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:             {sequence: 5},
	}}
	require.False(t, session.observationSatisfied("speculative-task-valid", history))

	for _, identifier := range []string{"create", "commit"} {
		result := session.results[identifier]
		result.Source = "temporal-sdk-speculative-update"
		result.SourceIdentity = "umpire3-program"
		result.Reference = "workflow/run/speculative-update/" + identifier
		session.results[identifier] = result
	}
	require.True(t, session.observationSatisfied("speculative-task-valid", history))

	delete(history.events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED)
	require.False(t, session.observationSatisfied("speculative-task-valid", history))
}

func TestSDKProgressObservationsRequireTypedReceiptsAndOrderedHistory(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{
			{Identifier: "dispatch", Kind: "dispatch-assurance-workflow-task"},
			{Identifier: "progress", Kind: "progress-entity"},
		}},
		results: map[string]participant.Result{
			"dispatch": {
				Status: "completed", Source: "temporal-sdk-participant",
				Reference: "workflow/run/dispatch", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
			"progress": {
				Status: "completed", Source: "temporal-sdk-participant",
				Reference: "workflow/run/progress", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"run"},
			},
		},
	}
	history := historyPosition{events: map[enumspb.EventType]historyEventPosition{
		enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED:            {sequence: 2},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED:              {sequence: 3},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:            {sequence: 4},
		enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED: {sequence: 6},
	}}
	require.False(t, session.observationSatisfied("workflow-task-not-starved", history))
	require.False(t, session.observationSatisfied("entity-progressed", history))

	for _, identifier := range []string{"dispatch", "progress"} {
		result := session.results[identifier]
		result.Source = "temporal-sdk-workflow-progress"
		result.SourceIdentity = "umpire3-program"
		result.Reference = "workflow/run/workflow-progress/" + identifier
		session.results[identifier] = result
	}
	require.True(t, session.observationSatisfied("workflow-task-not-starved", history))
	require.True(t, session.observationSatisfied("entity-progressed", history))

	history.events[enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED] = historyEventPosition{sequence: 5}
	require.False(t, session.observationSatisfied("workflow-task-not-starved", history))
}

func TestSDKNexusClosureRequiresOperationTerminalBeforeCallerClose(t *testing.T) {
	t.Parallel()

	session := &sdkSession{results: map[string]participant.Result{"action": {Status: "completed"}}}
	tests := []struct {
		name      string
		events    map[enumspb.EventType]historyEventPosition
		satisfied bool
	}{
		{
			name: "settled before close",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED:    {sequence: 9},
				enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED: {sequence: 10},
			},
			satisfied: true,
		},
		{
			name: "settled after close",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED: {sequence: 10},
				enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED:    {sequence: 11},
			},
		},
		{
			name: "caller close only",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED: {sequence: 10},
			},
		},
		{
			name: "operation terminal only",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCELED: {sequence: 9},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.satisfied, session.observationSatisfied(
				"nexus-operation-closed", historyPosition{events: test.events}))
		})
	}
}

func TestSDKNexusTimeoutObservationRejectsWrongMetadata(t *testing.T) {
	t.Parallel()

	session := &sdkSession{results: map[string]participant.Result{"action": {Status: "completed"}}}
	history := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT: {sequence: 1},
		},
		nexusTimeoutType: enumspb.TIMEOUT_TYPE_SCHEDULE_TO_CLOSE, nexusTimeoutMessage: "operation timed out",
	}
	require.False(t, session.observationSatisfied("nexus-timeout-valid", history))
	history.nexusTimeoutType = enumspb.TIMEOUT_TYPE_START_TO_CLOSE
	history.nexusTimeoutMessage = "unrelated failure"
	require.False(t, session.observationSatisfied("nexus-timeout-valid", history))
	history.nexusTimeoutMessage = "operation timed out after 2s"
	require.True(t, session.observationSatisfied("nexus-timeout-valid", history))
}

func TestSDKNexusActivityObservationRequiresReciprocalLinks(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{Identifier: "link", Kind: "link-nexus-activity"}}},
		results:    map[string]participant.Result{"link": {Status: "completed"}},
	}
	history := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED: {sequence: 1},
		},
		nexusActivityForwardLinked: true,
	}
	require.False(t, session.observationSatisfied("nexus-activity-links-consistent", history))
	history.nexusActivityReverseLinked = true
	require.True(t, session.observationSatisfied("nexus-activity-links-consistent", history))
}

func TestSDKCapabilitiesAreConservativeOrExplicitlyAttested(t *testing.T) {
	t.Parallel()

	capabilities, err := normalizeSDKCapabilities(SDKFactoryOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"history-observation", "update", "workflow-task-control"}, capabilities)

	capabilities, err = normalizeSDKCapabilities(SDKFactoryOptions{
		NexusEndpoint: "endpoint", NexusService: "service", NexusOperation: "operation",
	})
	require.NoError(t, err)
	require.Equal(t, []string{
		"history-observation", "nexus", "nexus-observation", "nexus-worker-control", "update", "workflow-task-control",
	}, capabilities)

	_, err = normalizeSDKCapabilities(SDKFactoryOptions{Capabilities: []string{"unknown"}})
	require.ErrorContains(t, err, "unknown capability")
	_, err = normalizeSDKCapabilities(SDKFactoryOptions{Capabilities: []string{"update", "update"}})
	require.ErrorContains(t, err, "duplicate capability")
}

func TestSDKSessionCorroboratesThroughIndependentHistorySource(t *testing.T) {
	t.Parallel()

	session := &corroboratingSDKSession{
		sdkSession: &sdkSession{
			experiment: protocol.Experiment{ExperimentID: "experiment", Actions: []protocol.Action{{
				Identifier: "action", Kind: "progress-entity",
			}}},
			namespace: "namespace",
			results: map[string]participant.Result{"action": {
				Status: "completed", Source: "temporal-sdk-workflow-progress",
				Reference: "participant/workflow-progress/action", WorkflowID: "workflow", RunID: "run",
				Lineage: []string{"workflow", "run"}, PayloadDigest: "",
			}},
		},
		sources: []CorroboratingHistorySource{staticHistorySource{snapshot: CorroboratingHistory{
			Source: "temporal-history-service", SourceIdentity: "cluster/history-service",
			ClockDomain: "temporal-history-service-event-id",
			Events: []CorroboratingHistoryEvent{{
				Type: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED,
				ID:   17, TimeUnixNano: time.Unix(10, 0).UnixNano(), Reference: "history-service/workflow/run/17",
			}},
		}}},
	}

	observations, err := session.Corroborate(context.Background(), protocol.Checkpoint{
		Identifier: "progress", Observation: "entity-progressed", Ordering: "causal",
	}, environment.Bindings{})
	require.NoError(t, err)
	require.Len(t, observations, 1)
	require.Equal(t, environment.Observation{
		CheckpointID: "progress", Kind: "entity-progressed", Satisfied: true,
		Source: "temporal-history-service", SourceIdentity: "cluster/history-service",
		ClockDomain: "temporal-history-service-event-id", SourceSequence: 17,
		AuthoritativeTimeUnixNano: time.Unix(10, 0).UnixNano(),
		ObservedAtUnixNano:        observations[0].ObservedAtUnixNano,
		Reference:                 "history-service/workflow/run/17/progress",
		CausalReference:           "history-service/workflow/run/17",
		CausalReferences:          []string{"participant/workflow-progress/action"},
		EntityIdentity:            "workflow/run",
		Lineage:                   []string{"experiment", "workflow", "run"},
	}, observations[0])
}

func TestSDKSessionFailsClosedWhenIndependentHistorySourceFails(t *testing.T) {
	t.Parallel()

	session := &corroboratingSDKSession{
		sdkSession: &sdkSession{
			experiment: protocol.Experiment{ExperimentID: "experiment", Actions: []protocol.Action{{Identifier: "action"}}},
			results:    map[string]participant.Result{"action": {WorkflowID: "workflow", RunID: "run"}},
		},
		sources: []CorroboratingHistorySource{staticHistorySource{err: errors.New("internal history unavailable")}},
	}

	_, err := session.Corroborate(context.Background(), protocol.Checkpoint{}, nil)
	require.ErrorContains(t, err, "internal history unavailable")
}
