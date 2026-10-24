package host

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

// Annotate transports an actual RPC message through the guest's Rust catalog.
func Annotate(ctx context.Context, host *Host, request *annotationsv1.AnnotationRequest) (*annotationsv1.AnnotationResponse, error) {
	input, err := anypb.New(request)
	if err != nil {
		return nil, err
	}
	response, err := host.Annotate(ctx, &modelv1.AnnotateRequest{Input: input})
	if err != nil {
		return nil, err
	}
	output := new(annotationsv1.AnnotationResponse)
	if err := response.GetOutput().UnmarshalTo(output); err != nil {
		return nil, fmt.Errorf("decode annotation response: %w", err)
	}
	return output, nil
}
