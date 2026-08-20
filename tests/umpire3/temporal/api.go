package temporal

import (
	"errors"

	"go.temporal.io/api/workflowservice/v1"
)

var (
	ErrMissingNamespace = errors.New("missing namespace")
	ErrMissingOperation = errors.New("missing operation ID")
	ErrMissingRequestID = errors.New("missing request ID")
)

type CancelCommand struct {
	Namespace   string
	OperationID string
	RunID       string
	RequestID   string
	Reason      string
}

func InterpretCancelRequest(request *workflowservice.RequestCancelNexusOperationExecutionRequest) (CancelCommand, error) {
	if request == nil || request.GetNamespace() == "" {
		return CancelCommand{}, ErrMissingNamespace
	}
	if request.GetOperationId() == "" {
		return CancelCommand{}, ErrMissingOperation
	}
	if request.GetRequestId() == "" {
		return CancelCommand{}, ErrMissingRequestID
	}
	return CancelCommand{
		Namespace:   request.GetNamespace(),
		OperationID: request.GetOperationId(),
		RunID:       request.GetRunId(),
		RequestID:   request.GetRequestId(),
		Reason:      request.GetReason(),
	}, nil
}
