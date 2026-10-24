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
	runv1 "umpire.local/model/gen/umpire/run/v1"
	"umpire.local/model/runner"
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

// Worker preserves the executor identity of the wrapped Driver.
func (f *fencedDriver) Worker() *runv1.Worker {
	if recorder, ok := f.Driver.(runner.WorkerRecorder); ok {
		return recorder.Worker()
	}
	return nil
}

// within admits only the allowlisted activity requests on the fenced activity, unwrapping the
// read instruction; anything else, a workflow start, a signal-with-start, a multi-operation or a
// schedule among them, is outside the fence.
func (f *fencedDriver) within(payload *anypb.Any) error {
	allowed := unaryRequests
	if payload.MessageIs(new(temporalv1.TemporalInstruction)) {
		instruction := new(temporalv1.TemporalInstruction)
		if err := payload.UnmarshalTo(instruction); err != nil {
			return err
		}
		cancellation := new(workflowservice.RequestCancelActivityExecutionRequest)
		if !instruction.GetRequest().MessageIs(cancellation) {
			return fmt.Errorf("%w: instruction names no cancellation", ErrOutsideFence)
		}
		if err := instruction.GetRequest().UnmarshalTo(cancellation); err != nil {
			return err
		}
		if err := f.within(instruction.GetRequest()); err != nil {
			return err
		}
		if instruction.GetNotFoundRead() != nil {
			read := new(temporalv1.ObserveActivity)
			if err := instruction.GetNotFoundRead().UnmarshalTo(read); err != nil {
				return err
			}
			describe := new(workflowservice.DescribeActivityExecutionRequest)
			if err := read.GetRequest().UnmarshalTo(describe); err != nil {
				return fmt.Errorf("%w: reconciliation names no Describe", ErrOutsideFence)
			}
			if describe.GetNamespace() != cancellation.GetNamespace() || describe.GetRunId() != cancellation.GetRunId() {
				return fmt.Errorf("%w: reconciliation crosses cancellation identity", ErrOutsideFence)
			}
			if err := f.within(instruction.GetNotFoundRead()); err != nil {
				return err
			}
		}
		return nil
	}
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
