package temporal

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	environment "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/observation"
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

func testSDKResult(status string, source string) participant.Result {
	return participant.Result{
		Status: status, Source: source, Reference: "workflow/run/participant",
		WorkflowID: "workflow", RunID: "run", Lineage: []string{"workflow", "run"},
	}
}

func completeSDKHistory(history historyPosition) historyPosition {
	if history.source == "" {
		history.source = "temporal-public-history"
	}
	if history.sourceIdentity == "" {
		history.sourceIdentity = "namespace/public-history"
	}
	if history.clockDomain == "" {
		history.clockDomain = "temporal-history-event-id"
	}
	for eventType, position := range history.events {
		if position.reference == "" {
			position.reference = "workflow/run/history/" + strconv.FormatInt(position.sequence, 10)
		}
		if position.timestamp == 0 {
			position.timestamp = time.Unix(position.sequence, 0).UnixNano()
		}
		history.events[eventType] = position
		if position.sequence > history.sequence {
			history.sequence = position.sequence
			history.timestamp = position.timestamp
			history.reference = position.reference
		}
	}
	if history.sequence == 0 {
		history.sequence = 1
		history.timestamp = time.Unix(1, 0).UnixNano()
		history.reference = "workflow/run/history/1"
	}
	return history
}

func sdkObservationTruth(
	t *testing.T,
	session *sdkSession,
	observationID string,
	history historyPosition,
) observation.Truth {
	t.Helper()
	namespace := session.namespace
	if namespace == "" {
		namespace = "namespace"
	}
	facts, err := (sdkFactNormalizer{
		experiment: session.experiment,
		namespace:  namespace,
		taskQueue:  session.taskQueue,
		results:    session.results,
	}).Normalize(protocol.Checkpoint{Identifier: observationID, Observation: observationID}, completeSDKHistory(history))
	require.NoError(t, err)
	for _, fact := range facts {
		require.NoError(t, fact.Validate())
	}
	catalog, err := observation.DefaultCatalog()
	require.NoError(t, err)
	program, ok := catalog.Program(protocol.ObservationID(observationID))
	require.True(t, ok)
	return program.Evaluate(facts).Value
}

func TestSDKSessionEmitsFactsWithoutPropertyTruth(t *testing.T) {
	t.Parallel()

	var session any = &sdkSession{}
	_, emitsFacts := session.(environment.FactSession)
	require.True(t, emitsFacts)
}

func TestSDKFactNormalizerFeedsGeneratedObservationProgram(t *testing.T) {
	t.Parallel()

	normalizer := sdkFactNormalizer{
		experiment: protocol.Experiment{ExperimentID: "experiment", Actions: []protocol.Action{{
			Identifier: "update", Kind: "complete-update",
		}}},
		namespace: "namespace",
		results: map[string]participant.Result{"update": {
			Status: "completed", Source: "temporal-sdk-participant", Reference: "participant/update",
			WorkflowID: "workflow", RunID: "run", Lineage: []string{"workflow", "run"},
		}},
	}
	history := historyPosition{
		source: "temporal-public-history", sourceIdentity: "namespace/history",
		clockDomain: "temporal-history-event-id", sequence: 7,
		reference: "workflow/run/history/7",
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED: {
				sequence: 7, timestamp: time.Unix(10, 0).UnixNano(), reference: "workflow/run/history/7",
			},
		},
	}
	facts, err := normalizer.Normalize(protocol.Checkpoint{
		Identifier: "accepted", Observation: "update-accepted", Ordering: "source-sequence",
	}, history)
	require.NoError(t, err)
	require.NotEmpty(t, facts)
	for _, fact := range facts {
		require.NoError(t, fact.Validate())
	}
	catalog, err := observation.DefaultCatalog()
	require.NoError(t, err)
	program, ok := catalog.Program(protocol.ObservationIDUpdateAccepted)
	require.True(t, ok)
	require.Equal(t, observation.True, program.Evaluate(facts).Value)
}

func TestSDKFactNormalizerSupportsEverySDKObservation(t *testing.T) {
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
		source: "temporal-public-history", sourceIdentity: "namespace/history",
		clockDomain: "temporal-history-event-id", sequence: 10,
		reference: "workflow/run/history/10",
		events:    events, callbackRegistered: true, taskQueues: map[string]bool{"task-queue": true},
		nexusActivityForwardLinked: true, nexusActivityReverseLinked: true,
		nexusTimeoutType: enumspb.TIMEOUT_TYPE_START_TO_CLOSE, nexusTimeoutMessage: "operation timed out",
	}
	for eventType, position := range history.events {
		position.timestamp = time.Unix(position.sequence, 0).UnixNano()
		position.reference = "workflow/run/history/" + strconv.FormatInt(position.sequence, 10)
		history.events[eventType] = position
	}
	session.namespace = "namespace"
	session.taskQueue = "task-queue"
	catalog, err := observation.DefaultCatalog()
	require.NoError(t, err)
	normalizer := sdkFactNormalizer{
		experiment: session.experiment, namespace: session.namespace,
		taskQueue: session.taskQueue, results: session.results,
	}
	for _, observationID := range []string{
		"cancellation-accepted", "cancellation-won", "stale-success-absent",
		"update-accepted", "update-completed",
		"speculative-task-valid", "workflow-task-not-starved", "nexus-operation-closed",
		"nexus-activity-links-consistent", "nexus-timeout-valid", "callback-reference-valid",
		"callback-response-consistent", "entity-progressed",
		"workflow-routing-isolated",
	} {
		facts, err := normalizer.Normalize(protocol.Checkpoint{
			Identifier: observationID, Observation: observationID, Ordering: "causal",
		}, history)
		require.NoError(t, err, observationID)
		program, ok := catalog.Program(protocol.ObservationID(observationID))
		require.True(t, ok, observationID)
		require.Equal(t, observation.True, program.Evaluate(facts).Value, observationID)
	}
	linkFacts, err := normalizer.Normalize(protocol.Checkpoint{
		Identifier: "links", Observation: "nexus-activity-links-consistent", Ordering: "causal",
	}, history)
	require.NoError(t, err)
	var linkReceipts []string
	for _, fact := range linkFacts {
		if fact.Mechanism != nil && (fact.Mechanism.Action == observation.NexusOperationLinkedActivity ||
			fact.Mechanism.Action == observation.ActivityLinkedNexusOperation) {
			linkReceipts = append(linkReceipts, fact.Mechanism.Action)
		}
		if fact.History != nil {
			require.NotContains(t, []string{
				observation.NexusOperationLinkedActivity,
				observation.ActivityLinkedNexusOperation,
			}, fact.History.EventType)
		}
	}
	require.ElementsMatch(t, []string{
		observation.NexusOperationLinkedActivity,
		observation.ActivityLinkedNexusOperation,
	}, linkReceipts)
}

func TestSDKObservationFailsClosedOnMissingTargetEventOrFailedCommand(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{
			Identifier: "action", Kind: "link-nexus-activity",
		}}},
		results: map[string]participant.Result{"action": testSDKResult("completed", "temporal-sdk-participant")},
	}
	history := historyPosition{events: map[enumspb.EventType]historyEventPosition{
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED: {sequence: 1},
	}}
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "nexus-activity-links-consistent", history))
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "nexus-timeout-valid", history))
	_, err := (sdkFactNormalizer{
		experiment: session.experiment, namespace: "namespace", results: session.results,
	}).Normalize(protocol.Checkpoint{Observation: "unknown-observation"}, completeSDKHistory(history))
	require.ErrorContains(t, err, "no generated interpreter")

	failed := testSDKResult("failed", "temporal-sdk-participant")
	failed.TerminalState = "failed"
	failed.TerminalDisposition = participant.TerminalDispositionFailure
	session.results["action"] = failed
	history.events[enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED] = historyEventPosition{sequence: 2}
	history.events[enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED] = historyEventPosition{sequence: 3}
	require.Equal(t, observation.True, sdkObservationTruth(t, session, "nexus-operation-closed", history))
	missingCallbackResult := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{
			Identifier: "action", Kind: "record-callback-response",
		}}},
		results: map[string]participant.Result{"action": testSDKResult("completed", "generic-sdk")},
	}
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, missingCallbackResult, "callback-response-consistent", historyPosition{
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
			"register": testSDKResult("completed", "generic-sdk"),
			"respond":  testSDKResult("completed", "generic-sdk"),
		},
	}
	history := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:   {sequence: 1},
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED: {sequence: 2},
		},
		callbackRegistered: true,
	}
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "callback-reference-valid", history))
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "callback-response-consistent", history))

	register := session.results["register"]
	register.Source = "temporal-completion-callback-registration"
	session.results["register"] = register
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "callback-reference-valid", history))
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "callback-response-consistent", history))

	response := session.results["respond"]
	response.Source = "temporal-nexus-completion-callback-receiver"
	session.results["respond"] = response
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "callback-response-consistent", history))
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
	require.Equal(t, observation.Unknown, sdkObservationTruth(
		t, continuation, "workflow-continuation-lineage-valid", continuationHistory))
	result := continuation.results["continue"]
	result.Source = "temporal-sdk-continuation"
	continuation.results["continue"] = result
	require.Equal(t, observation.True, sdkObservationTruth(
		t, continuation, "workflow-continuation-lineage-valid", continuationHistory))

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
	require.Equal(t, observation.True,
		sdkObservationTruth(t, reset, "workflow-reset-lineage-valid", resetHistory))
	resetHistory.firstExecutionRunID = "unrelated"
	require.Equal(t, observation.False,
		sdkObservationTruth(t, reset, "workflow-reset-lineage-valid", resetHistory))
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
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "workflow-ownership-fenced", history))

	result := session.results["fence"]
	result.Source = "umpire3-workflow-task-fencer"
	result.SourceIdentity = result.Source
	session.results["fence"] = result
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "workflow-ownership-fenced", history))

	result.Reference = "workflow/run/workflow-task/5/fenced-before/2"
	session.results["fence"] = result
	require.Equal(t, observation.False,
		sdkObservationTruth(t, session, "workflow-ownership-fenced", history))

	result.Reference = "workflow/run/workflow-task/2/fenced-before/5"
	session.results["fence"] = result
	history.events[enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED] = historyEventPosition{sequence: 8}
	require.Equal(t, observation.False,
		sdkObservationTruth(t, session, "workflow-ownership-fenced", history))
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
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "speculative-task-valid", history))

	for _, identifier := range []string{"create", "commit"} {
		result := session.results[identifier]
		result.Source = "temporal-sdk-speculative-update"
		result.SourceIdentity = "umpire3-program"
		result.Reference = "workflow/run/speculative-update/" + identifier
		session.results[identifier] = result
	}
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "speculative-task-valid", history))

	delete(history.events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED)
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "speculative-task-valid", history))
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
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "workflow-task-not-starved", history))
	require.Equal(t, observation.Unknown,
		sdkObservationTruth(t, session, "entity-progressed", history))

	for _, identifier := range []string{"dispatch", "progress"} {
		result := session.results[identifier]
		result.Source = "temporal-sdk-workflow-progress"
		result.SourceIdentity = "umpire3-program"
		result.Reference = "workflow/run/workflow-progress/" + identifier
		session.results[identifier] = result
	}
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "workflow-task-not-starved", history))
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "entity-progressed", history))

	history.events[enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED] = historyEventPosition{sequence: 5}
	require.Equal(t, observation.False,
		sdkObservationTruth(t, session, "workflow-task-not-starved", history))
}

func TestSDKEntityProgressObservationAcceptsHighLevelProgressReceipt(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{
			Identifier: "progress", Kind: "progress-entity",
		}}},
		results: map[string]participant.Result{
			"progress": {
				Status: "completed", Source: "temporal-sdk-workflow-progress",
				Reference:  "workflow/run/workflow-progress/progress",
				WorkflowID: "workflow", RunID: "run", Lineage: []string{"run"},
			},
		},
	}
	history := historyPosition{events: map[enumspb.EventType]historyEventPosition{
		enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED:            {sequence: 2},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED:              {sequence: 3},
		enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:            {sequence: 4},
		enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED: {sequence: 6},
	}}
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "entity-progressed", history))
}

func TestSDKNexusClosureRequiresOperationTerminalBeforeCallerClose(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{Identifier: "action", Kind: "complete-operation"}}},
		results:    map[string]participant.Result{"action": testSDKResult("completed", "temporal-sdk-participant")},
	}
	tests := []struct {
		name   string
		events map[enumspb.EventType]historyEventPosition
		value  observation.Truth
	}{
		{
			name: "settled before close",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED:    {sequence: 9},
				enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED: {sequence: 10},
			},
			value: observation.True,
		},
		{
			name: "settled after close",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED: {sequence: 10},
				enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED:    {sequence: 11},
			},
			value: observation.False,
		},
		{
			name: "caller close only",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED: {sequence: 10},
			},
			value: observation.False,
		},
		{
			name: "operation terminal only",
			events: map[enumspb.EventType]historyEventPosition{
				enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCELED: {sequence: 9},
			},
			value: observation.Unknown,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.value, sdkObservationTruth(
				t, session, "nexus-operation-closed", historyPosition{events: test.events}))
		})
	}
}

func TestSDKNexusTimeoutObservationRejectsWrongMetadata(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{Identifier: "action", Kind: "timeout-operation"}}},
		results:    map[string]participant.Result{"action": testSDKResult("completed", "temporal-sdk-participant")},
	}
	history := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT: {sequence: 1},
		},
		nexusTimeoutType: enumspb.TIMEOUT_TYPE_SCHEDULE_TO_CLOSE, nexusTimeoutMessage: "operation timed out",
	}
	require.Equal(t, observation.False,
		sdkObservationTruth(t, session, "nexus-timeout-valid", history))
	history.nexusTimeoutType = enumspb.TIMEOUT_TYPE_START_TO_CLOSE
	history.nexusTimeoutMessage = "unrelated failure"
	require.Equal(t, observation.False,
		sdkObservationTruth(t, session, "nexus-timeout-valid", history))
	history.nexusTimeoutMessage = "operation timed out after 2s"
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "nexus-timeout-valid", history))
}

func TestSDKNexusActivityObservationRequiresReciprocalLinks(t *testing.T) {
	t.Parallel()

	session := &sdkSession{
		experiment: protocol.Experiment{Actions: []protocol.Action{{Identifier: "link", Kind: "link-nexus-activity"}}},
		results:    map[string]participant.Result{"link": testSDKResult("completed", "temporal-sdk-participant")},
	}
	history := historyPosition{
		events: map[enumspb.EventType]historyEventPosition{
			enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED: {sequence: 1},
		},
		nexusActivityForwardLinked: true,
	}
	require.Equal(t, observation.False,
		sdkObservationTruth(t, session, "nexus-activity-links-consistent", history))
	history.nexusActivityReverseLinked = true
	require.Equal(t, observation.True,
		sdkObservationTruth(t, session, "nexus-activity-links-consistent", history))
}

func TestSDKCapabilitiesAreConservativeOrExplicitlyAttested(t *testing.T) {
	t.Parallel()

	capabilities, err := normalizeSDKCapabilities(SDKFactoryOptions{})
	require.NoError(t, err)
	require.Equal(t, []protocol.CapabilityID{
		protocol.CapabilityIDHistoryObservation, protocol.CapabilityIDUpdate,
		protocol.CapabilityIDWorkflowTaskControl,
	}, capabilities)

	capabilities, err = normalizeSDKCapabilities(SDKFactoryOptions{
		NexusEndpoint: "endpoint", NexusService: "service", NexusOperation: "operation",
	})
	require.NoError(t, err)
	require.Equal(t, []protocol.CapabilityID{
		protocol.CapabilityIDHistoryObservation, protocol.CapabilityIDNexus,
		protocol.CapabilityIDNexusObservation, protocol.CapabilityIDNexusWorkerControl,
		protocol.CapabilityIDUpdate, protocol.CapabilityIDWorkflowTaskControl,
	}, capabilities)

}

func TestSDKSessionEmitsIndependentlyNormalizableCorroboratingFacts(t *testing.T) {
	t.Parallel()

	session := &corroboratingSDKSession{
		sdkSession: &sdkSession{
			experiment: protocol.Experiment{ExperimentID: "experiment", Actions: []protocol.Action{
				{Identifier: "dispatch", Kind: "dispatch-assurance-workflow-task"},
				{Identifier: "progress", Kind: "progress-entity"},
			}},
			namespace: "namespace",
			results: map[string]participant.Result{
				"dispatch": {
					Status: "completed", Source: "temporal-sdk-workflow-progress",
					Reference: "participant/workflow-progress/dispatch", WorkflowID: "workflow", RunID: "run",
					Lineage: []string{"workflow", "run"},
				},
				"progress": {
					Status: "completed", Source: "temporal-sdk-workflow-progress",
					Reference: "participant/workflow-progress/progress", WorkflowID: "workflow", RunID: "run",
					Lineage: []string{"workflow", "run"},
				},
			},
		},
		sources: []CorroboratingHistorySource{staticHistorySource{snapshot: CorroboratingHistory{
			Source: "temporal-history-service", SourceIdentity: "cluster/history-service",
			ClockDomain: "temporal-history-service-event-id",
			Events: []CorroboratingHistoryEvent{
				{
					Type: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
					ID:   14, TimeUnixNano: time.Unix(7, 0).UnixNano(), Reference: "history-service/workflow/run/14",
				},
				{
					Type: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
					ID:   15, TimeUnixNano: time.Unix(8, 0).UnixNano(), Reference: "history-service/workflow/run/15",
				},
				{
					Type: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
					ID:   16, TimeUnixNano: time.Unix(9, 0).UnixNano(), Reference: "history-service/workflow/run/16",
				},
				{
					Type: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED,
					ID:   17, TimeUnixNano: time.Unix(10, 0).UnixNano(), Reference: "history-service/workflow/run/17",
				},
			},
		}}},
	}

	factSets, err := session.CorroborateFacts(context.Background(), protocol.Checkpoint{
		Identifier: "progress", Observation: "entity-progressed", Ordering: "causal",
	}, environment.Bindings{})
	require.NoError(t, err)
	require.Len(t, factSets, 1)
	for _, fact := range factSets[0] {
		require.NoError(t, fact.Validate())
		require.Equal(t, "cluster/history-service", fact.Source.Identity)
		require.Equal(t, "temporal-history-service-event-id", fact.Source.ClockDomain)
	}
	catalog, err := observation.DefaultCatalog()
	require.NoError(t, err)
	program, ok := catalog.Program(protocol.ObservationIDEntityProgressed)
	require.True(t, ok)
	require.Equal(t, observation.True, program.Evaluate(factSets[0]).Value)
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

	_, err := session.CorroborateFacts(context.Background(), protocol.Checkpoint{}, nil)
	require.ErrorContains(t, err, "internal history unavailable")
}
