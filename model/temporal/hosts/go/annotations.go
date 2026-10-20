package temporal

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"
	model "umpire.local/model"
	modelv1 "umpire.local/model/umpire/model/v1"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// Annotate transports an actual RPC message through the guest's Rust catalog.
func Annotate(ctx context.Context, host *model.Host, request *temporalv1.AnnotationRequest) (*temporalv1.AnnotationResponse, error) {
	input, err := anypb.New(request)
	if err != nil {
		return nil, err
	}
	response, err := host.Annotate(ctx, &modelv1.AnnotateRequest{Input: input})
	if err != nil {
		return nil, err
	}
	output := new(temporalv1.AnnotationResponse)
	if err := response.GetOutput().UnmarshalTo(output); err != nil {
		return nil, fmt.Errorf("decode annotation response: %w", err)
	}
	return output, nil
}
