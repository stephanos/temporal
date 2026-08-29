package nexus

import (
	"errors"
	"fmt"

	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

type historyIterator interface {
	HasNext() bool
	Next() (*historypb.HistoryEvent, error)
}

var errHistoryCapacity = errors.New("history projection exceeds capacity")

// projectTerminalHistory is the only SDK-history-to-runtime-fact boundary.
// It deliberately retains no event attributes, payloads, headers, or errors.
func projectTerminalHistory(
	command umpireruntime.Command,
	iterator historyIterator,
	correlations adapterCorrelations,
	cancellationCallbacks uint64,
) ([]umpireruntime.Fact, error) {
	if iterator == nil || correlations.workflow == "" || correlations.operation == "" {
		return nil, errors.New("history projection is unavailable")
	}
	maximumFacts := command.Limit().MaxRecords()
	if maximumFacts <= 1 {
		return nil, errors.New("history projection has no capacity")
	}
	// adapterReceipt appends one participant-output fact to this same receipt.
	maximumHistoryFacts := maximumFacts - 1
	facts := make([]umpireruntime.Fact, 0, 64)
	eventCounts := make(map[enumspb.EventType]uint64)
	previousFact := ""
	previousEventID := int64(0)
	lastType := enumspb.EVENT_TYPE_UNSPECIFIED
	for iterator.HasNext() {
		if uint64(len(facts)) >= maximumHistoryFacts {
			return facts, errHistoryCapacity
		}
		event, err := iterator.Next()
		if err != nil {
			return facts, errors.New("history iteration failed")
		}
		if event == nil || event.GetEventId() <= previousEventID ||
			event.GetEventType() == enumspb.EVENT_TYPE_UNSPECIFIED {
			return facts, errors.New("history event is malformed or out of order")
		}
		fact, err := historyFact(
			command,
			event.GetEventId(),
			event.GetEventType(),
			previousFact,
			correlations,
		)
		if err != nil {
			return facts, fmt.Errorf("project history event: %w", err)
		}
		facts = append(facts, fact)
		previousFact = fact.DefinitionID()
		previousEventID = event.GetEventId()
		lastType = event.GetEventType()
		eventCounts[lastType]++
	}
	if cancellationCallbacks != 1 ||
		lastType != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED ||
		eventCounts[enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED] != 1 ||
		eventCounts[enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED] != 1 ||
		eventCounts[enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUESTED] != 1 ||
		eventCounts[enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED] != 1 ||
		eventCounts[enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED] != 1 {
		return facts, errors.New("history closure is incomplete")
	}
	return facts, nil
}
