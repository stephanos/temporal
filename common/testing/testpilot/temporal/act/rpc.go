package act

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
)

// service is the generated WorkflowService descriptor every proposed request is sent through.
var service = workflowservice.File_temporal_api_workflowservice_v1_service_proto.Services().ByName("WorkflowService")

// errMalformed marks evidence the Driver refuses to report: it never reaches Judge or Answer.
var errMalformed = errors.New("malformed evidence")

// method resolves the one WorkflowService method whose input is the request's message.
func method(request *anypb.Any) (protoreflect.MethodDescriptor, error) {
	name := protoreflect.FullName(request.GetTypeUrl()[strings.LastIndexByte(request.GetTypeUrl(), '/')+1:])
	var found protoreflect.MethodDescriptor
	methods := service.Methods()
	for i := range methods.Len() {
		if methods.Get(i).Input().FullName() != name {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("%s is the input of more than one WorkflowService method", name)
		}
		found = methods.Get(i)
	}
	if found == nil {
		return nil, fmt.Errorf("%q is the input of no WorkflowService method", request.GetTypeUrl())
	}
	return found, nil
}

func fullMethod(m protoreflect.MethodDescriptor) string {
	return "/" + string(m.Parent().FullName()) + "/" + string(m.Name())
}

// raw carries exact wire bytes through gRPC, so the request sent is the proposal's bytes and the
// response recorded is the bytes the server sent.
type raw struct{ data []byte }

type rawCodec struct{}

func (rawCodec) Marshal(v any) (mem.BufferSlice, error) {
	message, ok := v.(*raw)
	if !ok {
		return nil, fmt.Errorf("act: cannot send %T", v)
	}
	return mem.BufferSlice{mem.SliceBuffer(message.data)}, nil
}

func (rawCodec) Unmarshal(data mem.BufferSlice, v any) error {
	message, ok := v.(*raw)
	if !ok {
		return fmt.Errorf("act: cannot receive %T", v)
	}
	message.data = data.Materialize()
	return nil
}

func (rawCodec) Name() string { return "proto" }

// completion records whether a call's request left for the server, and whether the server ended
// the call with its own status. A call it never ended (unreachable, a local deadline, a
// cancellation) is a transport failure, not evidence.
type completion struct{}

type progressKey struct{}

// progress is how far one call got.
type progress struct{ sent, answered atomic.Bool }

func (completion) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context { return ctx }
func (completion) HandleRPC(ctx context.Context, s stats.RPCStats) {
	p, ok := ctx.Value(progressKey{}).(*progress)
	if !ok {
		return
	}
	switch s.(type) {
	case *stats.OutHeader:
		p.sent.Store(true)
	case *stats.InTrailer:
		p.answered.Store(true)
	default:
	}
}
func (completion) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context { return ctx }
func (completion) HandleConn(context.Context, stats.ConnStats)                       {}

// call sends one proposed request unchanged and returns the invocation's Observation and how far
// the call got. A call the server did not answer has no Observation; evidence that is not well
// formed returns an error wrapping errMalformed.
func call(ctx context.Context, conn *grpc.ClientConn, request *anypb.Any) (_ *realizationv1.Observation, _ *progress, _ error) {
	reached := new(progress)
	m, err := method(request)
	if err != nil {
		return nil, reached, err
	}
	output, err := protoregistry.GlobalTypes.FindMessageByName(m.Output().FullName())
	if err != nil {
		return nil, reached, err
	}
	response := new(raw)
	err = conn.Invoke(context.WithValue(ctx, progressKey{}, reached), fullMethod(m), &raw{data: request.GetValue()}, response, grpc.ForceCodecV2(rawCodec{}))
	observation := &realizationv1.Observation{Method: fullMethod(m), Request: request}
	if err != nil {
		if !reached.answered.Load() {
			return nil, reached, err
		}
		code := status.Code(err)
		if code == codes.OK || code > codes.Unauthenticated {
			return nil, reached, fmt.Errorf("%w: status code %d", errMalformed, code)
		}
		observation.Result = &realizationv1.Observation_Error{Error: &realizationv1.Error{Code: int32(code), Message: status.Convert(err).Message()}}
		return observation, reached, nil
	}
	reached.answered.Store(true)
	if err := (proto.UnmarshalOptions{}).Unmarshal(response.data, output.New().Interface()); err != nil {
		return nil, reached, fmt.Errorf("%w: the %s response does not decode: %v", errMalformed, m.Name(), err)
	}
	observation.Result = &realizationv1.Observation_Response{Response: &anypb.Any{
		TypeUrl: "type.googleapis.com/" + string(m.Output().FullName()), Value: response.data,
	}}
	return observation, reached, nil
}

// pack wraps evidence in an Any with deterministic bytes, so a Run records the same bytes for the
// same evidence.
func pack(message proto.Message) (*anypb.Any, error) {
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	if err != nil {
		return nil, err
	}
	return &anypb.Any{TypeUrl: "type.googleapis.com/" + string(message.ProtoReflect().Descriptor().FullName()), Value: data}, nil
}
