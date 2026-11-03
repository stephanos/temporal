package temporal

import (
	"context"
	"errors"
	annotationsv1 "github.com/temporalio/umpire/hosts/go/gen/umpire/annotations/v1"
	model "github.com/temporalio/umpire/hosts/go/host"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// Probe carries activity catalog operations with bounded observation metadata.
func Probe(ctx context.Context, host *model.Host, request *temporalv1.ProbeRequest) (*annotationsv1.ProbeResponse, error) {
	if request == nil || len(request.Observations) > 32768 {
		return nil, errors.New("probe request exceeds metadata budget")
	}
	return model.Probe(ctx, host, request)
}

// DispatchProbe admits only the workflow service used by this slice.
func DispatchProbe(ctx context.Context, connection grpc.ClientConnInterface, prepared *annotationsv1.ProbeResponse) (proto.Message, error) {
	return model.DispatchProbe(ctx, connection, prepared, "temporal.api.workflowservice.v1.WorkflowService")
}
