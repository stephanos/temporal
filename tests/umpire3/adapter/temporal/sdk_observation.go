package temporal

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/server/tests/umpire3/execution/observation"
	"go.temporal.io/server/tests/umpire3/execution/participant"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

type sdkFactNormalizer struct {
	experiment protocolexperiment.Experiment
	namespace  string
	taskQueue  string
	results    map[string]participant.Result
}

func (n sdkFactNormalizer) Normalize(
	checkpoint protocolexperiment.Checkpoint,
	history historyPosition,
) ([]observation.Fact, error) {
	result := n.latestResult()
	if result.WorkflowID == "" || result.RunID == "" {
		return nil, errors.New("SDK participant identity evidence is incomplete")
	}
	if history.sourceIdentity == "" || history.clockDomain == "" || history.sequence <= 0 ||
		history.reference == "" {
		return nil, errors.New("SDK history source evidence is incomplete")
	}

	facts := make([]observation.Fact, 0, 24)
	appendHistory := func(kind string, position historyEventPosition) {
		if position.sequence > 0 && position.reference != "" {
			facts = append(facts, n.historyFact(result, history, position, kind))
		}
	}
	if position, ok := history.events[enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED]; ok {
		appendHistory(observation.WorkflowUpdateAccepted, position)
	}
	if position, ok := history.events[enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED]; ok {
		appendHistory(observation.WorkflowUpdateCompleted, position)
	}
	if position, ok := history.events[enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED]; ok {
		if n.hasActionStatus("request-cancellation", "completed") {
			appendHistory(observation.NexusCancellationAccepted, position)
		}
		if n.hasActionStatus("commit-cancellation", "completed") {
			appendHistory(observation.NexusCancellationCommitted, position)
		}
	}
	if n.hasActionStatus("persist-success", "completed") {
		if position, ok := history.latest(
			enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
			enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED,
		); ok {
			fact := n.historyFact(result, history, position, observation.NexusSuccessRecorded)
			committed := n.hasActionStatus("commit-cancellation", "completed")
			ownerEpoch, currentOwnerEpoch := int64(0), int64(0)
			fact.History.CancellationCommitted = &committed
			fact.History.OwnerEpoch = &ownerEpoch
			fact.History.CurrentOwnerEpoch = &currentOwnerEpoch
			fact.History.OperationID = result.WorkflowID
			facts = append(facts, fact)
		}
	}

	n.appendSpeculativeFacts(&facts, result, history)
	n.appendNexusClosureFacts(&facts, result, history)
	n.appendNexusActivityLinkFacts(&facts, result, history)
	n.appendNexusTimeoutFacts(&facts, result, history)
	n.appendCallbackFacts(&facts, result, history)
	n.appendProgressFacts(&facts, result, history)
	n.appendLineageFacts(&facts, result, history)
	n.appendRoutingFacts(&facts, result, history)
	n.appendOwnershipFacts(&facts, result, history)

	catalog, err := observation.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	program, ok := catalog.Program(protocolcatalog.ObservationID(checkpoint.Observation))
	if !ok {
		return nil, fmt.Errorf("observation %q has no generated interpreter program", checkpoint.Observation)
	}
	for _, closure := range program.Closures {
		facts = append(facts, n.windowFact(result, history, checkpoint, closure.Kind))
	}
	return facts, nil
}

func (n sdkFactNormalizer) latestResult() participant.Result {
	var latest participant.Result
	for _, action := range n.experiment.Actions {
		if result, exists := n.results[action.Identifier]; exists {
			latest = result
		}
	}
	return latest
}

func (n sdkFactNormalizer) hasActionStatus(kind string, status string) bool {
	for _, action := range n.experiment.Actions {
		if action.Kind == kind && n.results[action.Identifier].Status == status {
			return true
		}
	}
	return false
}

func (n sdkFactNormalizer) qualifiedActionResult(
	kind string,
	referenceMarker string,
	allowedSources ...string,
) (participant.Result, bool) {
	for _, action := range n.experiment.Actions {
		if action.Kind != kind {
			continue
		}
		result, ok := n.results[action.Identifier]
		if !ok || result.Status != "completed" || result.Reference == "" || result.WorkflowID == "" ||
			result.RunID == "" || len(result.Lineage) == 0 || !slices.Contains(allowedSources, result.Source) {
			continue
		}
		if referenceMarker == "" || strings.Contains(result.Reference, referenceMarker) {
			return result, true
		}
	}
	return participant.Result{}, false
}

func (n sdkFactNormalizer) appendSpeculativeFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	_, created := n.qualifiedActionResult(
		"create-speculative-workflow-task", "/speculative-update/", "temporal-sdk-speculative-update")
	_, committed := n.qualifiedActionResult(
		"commit-speculative-workflow-task", "/speculative-update/", "temporal-sdk-speculative-update")
	accepted, acceptedObserved := history.events[enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED]
	completed, completedObserved := history.events[enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED]
	taskCompleted, taskCompletedObserved := history.events[enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED]
	if created && acceptedObserved {
		*facts = append(*facts, n.historyFact(result, history, accepted, observation.UpdateRequested))
	}
	if created && taskCompletedObserved {
		*facts = append(*facts, n.historyFact(result, history, taskCompleted, observation.SpeculativeTaskCreated))
	}
	if committed && completedObserved {
		*facts = append(*facts, n.historyFact(result, history, completed, observation.SpeculativeTaskCommitted))
	}
}

func (n sdkFactNormalizer) appendNexusClosureFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	operation, operationObserved := history.latest(
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCELED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT,
	)
	workflow, workflowObserved := history.latest(
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT,
	)
	if operationObserved {
		*facts = append(*facts, n.historyFact(result, history, operation, observation.NexusOperationSettled))
	}
	if workflowObserved {
		*facts = append(*facts, n.historyFact(result, history, workflow, observation.CallerWorkflowClosed))
	}
	if workflowObserved && (!operationObserved || operation.sequence > workflow.sequence) {
		*facts = append(*facts, n.historyFact(
			result, history, workflow, observation.CallerClosedWithOpenOperation,
		))
	}
}

func (n sdkFactNormalizer) appendNexusActivityLinkFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	if !n.hasActionStatus("link-nexus-activity", "completed") {
		return
	}
	position, observed := history.latest(enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED)
	if !observed {
		return
	}
	if history.nexusActivityForwardLinked {
		*facts = append(*facts, n.mechanismFact(
			result, history, position, observation.NexusOperationLinkedActivity, "linked",
		))
	}
	if history.nexusActivityReverseLinked {
		*facts = append(*facts, n.mechanismFact(
			result, history, position, observation.ActivityLinkedNexusOperation, "linked",
		))
	}
	if history.nexusActivityForwardLinked != history.nexusActivityReverseLinked {
		*facts = append(*facts, n.mechanismFact(
			result, history, position, observation.NexusActivityLinkOneSided, "one-sided",
		))
	}
}

func (n sdkFactNormalizer) appendNexusTimeoutFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	position, observed := history.latest(enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT)
	if !observed {
		return
	}
	*facts = append(*facts,
		n.historyFact(result, history, position, observation.NexusTimeoutConfigured),
		n.historyFact(result, history, position, observation.NexusOperationTimedOut),
	)
	if history.nexusTimeoutType != enumspb.TIMEOUT_TYPE_START_TO_CLOSE ||
		!strings.Contains(history.nexusTimeoutMessage, "operation timed out") {
		*facts = append(*facts, n.historyFact(
			result, history, position, observation.NexusTimeoutMetadataInvalid,
		))
	}
}

func (n sdkFactNormalizer) appendCallbackFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	registered, registeredQualified := n.qualifiedActionResult(
		"register-callback", "", "temporal-completion-callback-registration",
		"temporal-shared-handler-registration", "temporal-nexus-callback-registration",
	)
	started, startedObserved := history.latest(
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED,
	)
	if history.callbackRegistered && registeredQualified && startedObserved {
		*facts = append(*facts,
			n.historyFact(result, history, started, observation.CallbackAttached),
			n.historyFact(result, history, started, observation.NexusOperationStarted),
			n.historyFact(result, history, started, observation.CallbackRegistered),
		)
		if registered.WorkflowID != result.WorkflowID || registered.RunID != result.RunID {
			*facts = append(*facts, n.historyFact(
				result, history, started, observation.CallbackReferenceMismatch,
			))
		}
	}
	settled, settledObserved := history.latest(
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED,
	)
	if settledObserved {
		*facts = append(*facts, n.historyFact(
			result, history, settled, observation.CallbackOperationSettled,
		))
	}
	response, responseQualified := n.qualifiedActionResult(
		"record-callback-response", "", "temporal-nexus-completion-callback-receiver",
		"temporal-shared-handler-completion", "temporal-nexus-callback-rejection",
	)
	if responseQualified && settledObserved {
		*facts = append(*facts, n.historyFact(
			result, history, settled, observation.CallbackResponseRecorded,
		))
		if response.WorkflowID != result.WorkflowID || response.RunID != result.RunID {
			*facts = append(*facts, n.historyFact(
				result, history, settled, observation.CallbackResponseConflict,
			))
		}
	}
}

func (n sdkFactNormalizer) appendProgressFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	_, dispatchQualified := n.qualifiedActionResult(
		"dispatch-assurance-workflow-task", "/workflow-progress/", "temporal-sdk-workflow-progress")
	_, progressQualified := n.qualifiedActionResult(
		"progress-entity", "/workflow-progress/", "temporal-sdk-workflow-progress")
	workflowTaskQualified := dispatchQualified || progressQualified
	scheduled, scheduledObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED)
	started, startedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED)
	completed, completedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
	if workflowTaskQualified && scheduledObserved {
		*facts = append(*facts, n.historyFact(result, history, scheduled, observation.WorkflowTaskQueued))
	}
	if workflowTaskQualified && startedObserved {
		*facts = append(*facts, n.historyFact(result, history, started, observation.WorkflowWorkerAvailable))
	}
	if workflowTaskQualified && completedObserved {
		*facts = append(*facts, n.historyFact(result, history, completed, observation.WorkflowTaskCompleted))
	}
	if workflowTaskQualified && scheduledObserved && startedObserved && completedObserved &&
		(scheduled.sequence >= started.sequence || started.sequence >= completed.sequence) {
		*facts = append(*facts, n.historyFact(result, history, completed, observation.WorkflowTaskStarved))
	}
	progressed, progressedObserved := history.latest(
		enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED,
		enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED,
	)
	if progressQualified && scheduledObserved {
		*facts = append(*facts, n.historyFact(result, history, scheduled, observation.EntityPending))
	}
	if progressQualified && progressedObserved {
		*facts = append(*facts, n.historyFact(result, history, progressed, observation.EntityProgressed))
	}
}

func (n sdkFactNormalizer) appendLineageFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	started, observed := history.latest(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED)
	if !observed {
		return
	}
	if receipt, qualified := n.qualifiedActionResult(
		"continue-workflow", "", "temporal-sdk-continuation",
	); qualified {
		*facts = append(*facts, n.historyFact(result, history, started, observation.WorkflowContinued))
		valid := len(receipt.Lineage) >= 4 &&
			history.continuedExecutionRunID == receipt.Lineage[len(receipt.Lineage)-2] &&
			history.originalExecutionRunID == receipt.RunID &&
			history.firstExecutionRunID == receipt.Lineage[len(receipt.Lineage)-2]
		kind := observation.WorkflowLineageRecorded
		if !valid {
			kind = observation.WorkflowContinuationLineageInvalid
		}
		*facts = append(*facts, n.historyFact(result, history, started, kind))
	}
	if receipt, qualified := n.qualifiedActionResult(
		"reset-workflow", "", "temporal-sdk-reset",
	); qualified {
		*facts = append(*facts, n.historyFact(result, history, started, observation.WorkflowReset))
		valid := len(receipt.Lineage) >= 4 &&
			history.originalExecutionRunID == receipt.Lineage[len(receipt.Lineage)-2] &&
			history.originalExecutionRunID != receipt.RunID &&
			history.firstExecutionRunID == receipt.Lineage[len(receipt.Lineage)-2]
		kind := observation.WorkflowLineageRecorded
		if !valid {
			kind = observation.WorkflowResetLineageInvalid
		}
		*facts = append(*facts, n.historyFact(result, history, started, kind))
	}
}

func (n sdkFactNormalizer) appendRoutingFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	receipt, qualified := n.qualifiedActionResult(
		"route-workflow-task", "/task-queue/", "temporal-sdk-routing")
	if !qualified {
		return
	}
	scheduled, scheduledObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED)
	completed, completedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
	if scheduledObserved && history.taskQueues[n.taskQueue] {
		*facts = append(*facts,
			n.historyFact(result, history, scheduled, observation.WorkflowTaskRouted),
			n.historyFact(result, history, scheduled, observation.WorkflowPollerRouted),
		)
	}
	if completedObserved && receipt.SourceIdentity == n.taskQueue && history.taskQueues[n.taskQueue] {
		*facts = append(*facts, n.historyFact(
			result, history, completed, observation.WorkflowTaskReserved,
		))
	} else if completedObserved {
		*facts = append(*facts, n.historyFact(
			result, history, completed, observation.WorkflowTaskCrossRoute,
		))
	}
}

func (n sdkFactNormalizer) appendOwnershipFacts(
	facts *[]observation.Fact,
	result participant.Result,
	history historyPosition,
) {
	receipt, qualified := n.qualifiedActionResult(
		"fence-workflow-owner", "/workflow-task/", "umpire3-workflow-task-fencer")
	if !qualified || receipt.SourceIdentity != "umpire3-workflow-task-fencer" {
		return
	}
	failed, failedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED)
	completed, completedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
	staleStarted, currentStarted, referenceValid := parseWorkflowOwnerFencingReference(receipt.Reference)
	if failedObserved {
		*facts = append(*facts,
			n.historyFact(result, history, failed, observation.WorkflowTaskDispatched),
			n.historyFact(result, history, failed, observation.WorkflowOwnerRotated),
		)
	}
	if referenceValid && failedObserved && completedObserved && failed.sequence < completed.sequence {
		*facts = append(*facts,
			n.historyFact(result, history, failed, observation.StaleCompletionRejected),
			n.historyFact(result, history, completed, observation.CurrentCompletionRecorded),
		)
	} else if completedObserved && (!referenceValid || staleStarted >= currentStarted || failed.sequence >= completed.sequence) {
		*facts = append(*facts, n.historyFact(
			result, history, completed, observation.StaleCompletionRecorded,
		))
	}
}

func (n sdkFactNormalizer) historyFact(
	result participant.Result,
	history historyPosition,
	position historyEventPosition,
	kind string,
) observation.Fact {
	lineage := append([]string(nil), result.Lineage...)
	if n.experiment.ExperimentID != "" {
		lineage = append([]string{n.experiment.ExperimentID}, lineage...)
	}
	causalReferences := []string(nil)
	if result.Reference != "" {
		causalReferences = []string{result.Reference}
	}
	fact := observation.Fact{
		Identifier: fmt.Sprintf("history/%s/%s/%d", history.sourceIdentity, kind, position.sequence),
		Source: observation.Source{
			Identity: history.sourceIdentity, ClockDomain: history.clockDomain,
			Sequence: position.sequence, Reference: position.reference,
			CausalReferences: causalReferences,
			EntityIdentity:   result.WorkflowID + "/" + result.RunID,
			Lineage:          lineage,
			PayloadDigest:    result.PayloadDigest,
		},
		History: &observation.HistoryEvent{
			EventType: kind, EventID: position.sequence,
			WorkflowID: result.WorkflowID, RunID: result.RunID,
		},
	}
	if kind == observation.NexusCancellationAccepted || kind == observation.NexusCancellationCommitted ||
		kind == observation.NexusSuccessRecorded {
		fact.History.OperationID = result.WorkflowID
	}
	return fact
}

func (n sdkFactNormalizer) mechanismFact(
	result participant.Result,
	history historyPosition,
	position historyEventPosition,
	kind string,
	outcome string,
) observation.Fact {
	fact := n.historyFact(result, history, position, kind)
	fact.Identifier = fmt.Sprintf("mechanism/%s/%s/%d", history.sourceIdentity, kind, position.sequence)
	fact.History = nil
	fact.Mechanism = &observation.MechanismReceipt{
		Action: kind, Resource: result.WorkflowID + "/" + result.RunID,
		Attempt: position.sequence, OwnerEpoch: 0, Outcome: outcome,
	}
	return fact
}

func (n sdkFactNormalizer) windowFact(
	result participant.Result,
	history historyPosition,
	checkpoint protocolexperiment.Checkpoint,
	purpose string,
) observation.Fact {
	lineage := append([]string(nil), result.Lineage...)
	if n.experiment.ExperimentID != "" {
		lineage = append([]string{n.experiment.ExperimentID}, lineage...)
	}
	causalReferences := []string(nil)
	if result.Reference != "" {
		causalReferences = []string{result.Reference}
	}
	return observation.Fact{
		Identifier: fmt.Sprintf("window/%s/%s/%d", history.sourceIdentity, purpose, history.sequence),
		Source: observation.Source{
			Identity: history.sourceIdentity, ClockDomain: history.clockDomain,
			Sequence:         history.sequence,
			Reference:        history.reference + "/window/" + checkpoint.Identifier,
			CausalReferences: causalReferences,
			EntityIdentity:   result.WorkflowID + "/" + result.RunID,
			Lineage:          lineage,
			PayloadDigest:    result.PayloadDigest,
		},
		Window: &observation.EvidenceWindow{
			Purpose: purpose, Closed: true, ThroughSequence: history.sequence,
		},
	}
}
