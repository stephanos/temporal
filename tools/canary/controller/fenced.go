package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/umpire/run/v1"
	"umpire.local/model/umpire/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// ErrOutsideFence is a request the fence does not admit: one on another activity, or any request
// but the canary World's activity calls.
var ErrOutsideFence = errors.New("the canary starts only the activity its fence names")

// activityPrefix begins every activity ID the canary fences, a fresh UUID after it.
const activityPrefix = "umpire-canary-"

// newActivityID is a fresh activity ID for one iteration.
func newActivityID() string { return activityPrefix + uuid.NewString() }

// IsActivityID reports whether id is one the canary fences: its prefix and a UUID.
func IsActivityID(id string) bool {
	rest, ok := strings.CutPrefix(id, activityPrefix)
	if !ok {
		return false
	}
	parsed, err := uuid.Parse(rest)
	return err == nil && parsed.String() == rest
}

// activityRequest is a WorkflowService request on one activity, which the fence admits only for
// its own activity ID.
type activityRequest interface {
	proto.Message
	GetActivityId() string
}

// unaryRequests and readRequests are the only requests the fence admits, sent alone or inside the
// Temporal Driver's read instruction: the canary World's activity calls. A start is admitted only
// unary, never as a read.
var (
	unaryRequests = map[protoreflect.FullName]func() activityRequest{
		name(&workflowservice.StartActivityExecutionRequest{}):         func() activityRequest { return new(workflowservice.StartActivityExecutionRequest) },
		name(&workflowservice.RequestCancelActivityExecutionRequest{}): func() activityRequest { return new(workflowservice.RequestCancelActivityExecutionRequest) },
		name(&workflowservice.DescribeActivityExecutionRequest{}):      func() activityRequest { return new(workflowservice.DescribeActivityExecutionRequest) },
		name(&workflowservice.PollActivityExecutionRequest{}):          func() activityRequest { return new(workflowservice.PollActivityExecutionRequest) },
	}
	readRequests = map[protoreflect.FullName]func() activityRequest{
		name(&workflowservice.DescribeActivityExecutionRequest{}): func() activityRequest { return new(workflowservice.DescribeActivityExecutionRequest) },
		name(&workflowservice.PollActivityExecutionRequest{}):     func() activityRequest { return new(workflowservice.PollActivityExecutionRequest) },
	}
	readInstruction = name(&temporalv1.ObserveActivity{})
)

func name(message proto.Message) protoreflect.FullName {
	return message.ProtoReflect().Descriptor().FullName()
}

// fencedDriver wraps one Run's Driver so the Run can send nothing but the activity calls on the
// activity the lease already names: cleanup and reconcile act only on the fence's IDs, so they can never miss an
// execution the canary started.
type fencedDriver struct {
	runner.Driver
	activityID string
}

// fenced makes newDriver's Drivers fenced to activityID.
func fenced(newDriver runner.NewDriver, activityID string) runner.NewDriver {
	return func(ctx context.Context) (runner.Driver, error) {
		driver, err := newDriver(ctx)
		if err != nil {
			return nil, err
		}
		return &fencedDriver{Driver: driver, activityID: activityID}, nil
	}
}

func (f *fencedDriver) Start(ctx context.Context, request uint64, payload *anypb.Any) (runner.Exchange, error) {
	if err := f.within(payload); err != nil {
		return nil, err
	}
	return f.Driver.Start(ctx, request, payload)
}

// Close closes the wrapped Driver, when it has anything to close.
func (f *fencedDriver) Close() error {
	if closer, ok := f.Driver.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Redact keeps the wrapped Driver's redaction, when it has one.
func (f *fencedDriver) Redact(event *runv1.Event) bool {
	if redactor, ok := f.Driver.(runner.Redactor); ok {
		return redactor.Redact(event)
	}
	return false
}

// within admits only the allowlisted activity requests on the fenced activity, unwrapping the
// read instruction; anything else, a workflow start, a signal-with-start, a multi-operation or a
// schedule among them, is outside the fence.
func (f *fencedDriver) within(payload *anypb.Any) error {
	allowed := unaryRequests
	if payload.MessageName() == readInstruction {
		instruction := new(temporalv1.ObserveActivity)
		if err := payload.UnmarshalTo(instruction); err != nil {
			return err
		}
		payload, allowed = instruction.GetRequest(), readRequests
	}
	newRequest, ok := allowed[payload.MessageName()]
	if !ok {
		return fmt.Errorf("%w: %s", ErrOutsideFence, payload.MessageName())
	}
	request := newRequest()
	if err := payload.UnmarshalTo(request); err != nil {
		return err
	}
	if request.GetActivityId() != f.activityID {
		return fmt.Errorf("%w: %s names another activity", ErrOutsideFence, payload.MessageName())
	}
	return nil
}
