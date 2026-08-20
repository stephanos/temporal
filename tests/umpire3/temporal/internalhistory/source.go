package internalhistory

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	umpire3temporal "go.temporal.io/server/tests/umpire3/temporal"
	"google.golang.org/grpc"
)

const (
	maximumPageSize = 1000
	maximumPages    = 1000
	maximumEvents   = 100000
)

type historyClient interface {
	GetWorkflowExecutionHistory(
		context.Context,
		*historyservice.GetWorkflowExecutionHistoryRequest,
		...grpc.CallOption,
	) (*historyservice.GetWorkflowExecutionHistoryResponse, error)
}

type Source struct {
	client         historyClient
	namespaceID    string
	sourceIdentity string
}

func New(client historyClient, namespaceID string, clusterIdentity string) (*Source, error) {
	if client == nil || namespaceID == "" || clusterIdentity == "" {
		return nil, errors.New("internal history source requires client, namespace, and cluster identities")
	}
	return &Source{
		client: client, namespaceID: namespaceID, sourceIdentity: clusterIdentity + "/history-service",
	}, nil
}

func (s *Source) ReadHistory(
	ctx context.Context,
	request umpire3temporal.HistoryRequest,
) (umpire3temporal.CorroboratingHistory, error) {
	if request.Namespace == "" || request.WorkflowID == "" || request.RunID == "" {
		return umpire3temporal.CorroboratingHistory{}, errors.New("complete history request identity is required")
	}
	result := umpire3temporal.CorroboratingHistory{
		Source: "temporal-history-service", SourceIdentity: s.sourceIdentity,
		ClockDomain: "temporal-history-service-event-id",
	}
	var nextPageToken []byte
	for page := 0; page < maximumPages; page++ {
		response, err := s.client.GetWorkflowExecutionHistory(ctx, &historyservice.GetWorkflowExecutionHistoryRequest{
			NamespaceId: s.namespaceID,
			Request: &workflowservice.GetWorkflowExecutionHistoryRequest{
				Namespace:       request.Namespace,
				Execution:       &commonpb.WorkflowExecution{WorkflowId: request.WorkflowID, RunId: request.RunID},
				MaximumPageSize: maximumPageSize, NextPageToken: nextPageToken,
				HistoryEventFilterType: enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT, SkipArchival: true,
			},
		})
		if err != nil {
			return umpire3temporal.CorroboratingHistory{}, fmt.Errorf("read History service page: %w", err)
		}
		if response == nil || response.GetResponse() == nil {
			return umpire3temporal.CorroboratingHistory{}, errors.New("history service returned an incomplete response")
		}
		history := response.GetHistory()
		if history == nil || len(history.GetEvents()) == 0 {
			history = response.GetResponse().GetHistory()
		}
		for _, event := range history.GetEvents() {
			if len(result.Events) == maximumEvents {
				return umpire3temporal.CorroboratingHistory{}, errors.New("internal history event limit exceeded")
			}
			corroborating := umpire3temporal.CorroboratingHistoryEvent{
				Type: event.GetEventType(), ID: event.GetEventId(), TimeUnixNano: event.GetEventTime().AsTime().UnixNano(),
				Reference: fmt.Sprintf("history-service/%s/%s/%s/history/%d",
					s.namespaceID, request.WorkflowID, request.RunID, event.GetEventId()),
			}
			if started := event.GetWorkflowExecutionStartedEventAttributes(); started != nil {
				corroborating.ContinuedExecutionRunID = started.GetContinuedExecutionRunId()
				corroborating.OriginalExecutionRunID = started.GetOriginalExecutionRunId()
				corroborating.FirstExecutionRunID = started.GetFirstExecutionRunId()
				corroborating.TaskQueue = started.GetTaskQueue().GetName()
				corroborating.CallbackRegistered = len(started.GetCompletionCallbacks()) != 0
			}
			if scheduled := event.GetWorkflowTaskScheduledEventAttributes(); scheduled != nil {
				corroborating.TaskQueue = scheduled.GetTaskQueue().GetName()
			}
			if nexusStarted := event.GetNexusOperationStartedEventAttributes(); nexusStarted != nil &&
				nexusStarted.GetOperationToken() != "" {
				corroborating.CallbackRegistered = true
			}
			for _, link := range event.GetLinks() {
				if activity := link.GetActivity(); activity != nil && activity.GetNamespace() != "" &&
					activity.GetActivityId() != "" && activity.GetRunId() != "" {
					corroborating.NexusActivityForwardLinked = true
				}
				if operation := link.GetNexusOperation(); operation != nil && operation.GetNamespace() != "" &&
					operation.GetOperationId() != "" && operation.GetRunId() != "" {
					corroborating.NexusActivityReverseLinked = true
				}
			}
			if timedOut := event.GetNexusOperationTimedOutEventAttributes(); timedOut != nil {
				cause := timedOut.GetFailure().GetCause()
				corroborating.NexusTimeoutType = cause.GetTimeoutFailureInfo().GetTimeoutType()
				corroborating.NexusTimeoutMessage = cause.GetMessage()
			}
			result.Events = append(result.Events, corroborating)
		}
		next := response.GetResponse().GetNextPageToken()
		if len(next) == 0 {
			if len(result.Events) == 0 {
				return umpire3temporal.CorroboratingHistory{}, errors.New("history service returned no events")
			}
			return result, nil
		}
		if bytes.Equal(next, nextPageToken) {
			return umpire3temporal.CorroboratingHistory{}, errors.New("history service repeated a page token")
		}
		nextPageToken = append(nextPageToken[:0], next...)
	}
	return umpire3temporal.CorroboratingHistory{}, errors.New("internal history page limit exceeded")
}
