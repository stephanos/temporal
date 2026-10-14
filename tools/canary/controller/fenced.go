package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	// ErrSecondOpen is a fenced Driver asked to open a second Run: each iteration's Driver opens
	// one, so a concurrent or repeated Run fails closed.
	ErrSecondOpen = errors.New("a fenced canary Driver opens one Run")
	// ErrOutsideFence is a workflow start whose ID is not the fenced Run's.
	ErrOutsideFence = errors.New("the canary starts only the workflow its fence names")
)

// startWorkflowExecution is the one method that creates a workflow the fence must name.
var startWorkflowExecution = workflowservice.File_temporal_api_workflowservice_v1_service_proto.
	Services().ByName("WorkflowService").Methods().ByName("StartWorkflowExecution").FullName()

// Fencer records a Run's ID on the lease before anything runs.
type Fencer func(ctx context.Context, runID string) error

// FencedDriver wraps a Testpilot Driver so a Run is fenced before it is opened: Open signals the
// Run's ID to the lease and only then delegates, and the Session it returns refuses to start any
// workflow but the one that ID names, which is the Case's own workflow ID.
type FencedDriver struct {
	driver testpilot.Driver
	fence  Fencer
	mu     sync.Mutex
	opened string
}

// NewFencedDriver fences driver's Runs with fence.
func NewFencedDriver(driver testpilot.Driver, fence Fencer) *FencedDriver {
	return &FencedDriver{driver: driver, fence: fence}
}

func (f *FencedDriver) Identity(ctx context.Context) (testpilot.DriverIdentity, error) {
	return f.driver.Identity(ctx)
}

func (f *FencedDriver) Validate(ctx context.Context, program testpilot.PreparedProgram) error {
	return f.driver.Validate(ctx, program)
}

// Open fences runID and then opens it. A second Open, or a fence that fails, opens nothing.
func (f *FencedDriver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.opened != "" {
		return nil, ErrSecondOpen
	}
	if !testpilot.IsRunID(runID) {
		return nil, errors.New("a fenced canary Run needs a Testpilot Run ID")
	}
	if err := f.fence(ctx, runID); err != nil {
		return nil, err
	}
	f.opened = runID
	session, err := f.driver.Open(ctx, runID, program)
	if err != nil {
		return nil, err
	}
	return &fencedSession{Session: session, runID: runID}, nil
}

// Opened is the Run ID this Driver fenced and opened, or "".
func (f *FencedDriver) Opened() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opened
}

type fencedSession struct {
	testpilot.Session
	runID string
}

func (s *fencedSession) InvokeRPC(ctx context.Context, coordinate testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.EffectHandle, error) {
	if err := s.within(method, request); err != nil {
		return nil, err
	}
	return s.Session.InvokeRPC(ctx, coordinate, role, method, request)
}

func (s *fencedSession) PollRPC(ctx context.Context, coordinate testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message, interval time.Duration, satisfied testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	if err := s.within(method, request); err != nil {
		return nil, err
	}
	return s.Session.PollRPC(ctx, coordinate, role, method, request, interval, satisfied)
}

// within refuses a workflow start whose ID is not the fenced Run's, so cleanup, which acts only on
// the fence's IDs, can never miss a workflow the canary started.
func (s *fencedSession) within(method protoreflect.MethodDescriptor, request proto.Message) error {
	if method == nil || method.FullName() != startWorkflowExecution {
		return nil
	}
	wire, err := proto.Marshal(request)
	if err != nil {
		return err
	}
	var start workflowservice.StartWorkflowExecutionRequest
	if err := proto.Unmarshal(wire, &start); err != nil {
		return err
	}
	if start.GetWorkflowId() != s.runID {
		return fmt.Errorf("%w: the start names another workflow", ErrOutsideFence)
	}
	return nil
}
