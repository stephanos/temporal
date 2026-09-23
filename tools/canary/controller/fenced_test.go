package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func workflowServiceMethod(name protoreflect.Name) protoreflect.MethodDescriptor {
	return workflowservice.File_temporal_api_workflowservice_v1_service_proto.Services().ByName("WorkflowService").Methods().ByName(name)
}

// recordingSession is a Session that records the RPCs it is asked to make.
type recordingSession struct {
	testpilot.Session
	invoked []string
}

func (s *recordingSession) InvokeRPC(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	s.invoked = append(s.invoked, string(method.Name()))
	return nil, nil
}

func (s *recordingSession) PollRPC(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message, _ time.Duration, _ testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	s.invoked = append(s.invoked, "poll "+string(method.Name()))
	return nil, nil
}

// stubDriver opens a recordingSession and says whether the fence ran before it did.
type stubDriver struct {
	fenced  *[]string
	opens   int
	session *recordingSession
	// fencedBeforeOpen is how many IDs were fenced when Open was reached.
	fencedBeforeOpen int
}

func (d *stubDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return testpilot.DriverIdentity{Profile: "stub"}, nil
}

func (d *stubDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }

func (d *stubDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	d.opens++
	if d.fenced != nil {
		d.fencedBeforeOpen = len(*d.fenced)
	}
	d.session = &recordingSession{}
	return d.session, nil
}

// Open fences the Run before the wrapped Driver opens it, and opens one Run only.
func TestFencedDriverFencesBeforeItOpens(t *testing.T) {
	var fenced []string
	inner := &stubDriver{fenced: &fenced}
	driver := NewFencedDriver(inner, func(_ context.Context, runID string) error {
		fenced = append(fenced, runID)
		return nil
	})
	identity, err := driver.Identity(t.Context())
	require.NoError(t, err)
	require.Equal(t, "stub", identity.Profile)
	require.NoError(t, driver.Validate(t.Context(), testpilot.PreparedProgram{}))

	session, err := driver.Open(t.Context(), "run-1", testpilot.PreparedProgram{})
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, []string{"run-1"}, fenced)
	require.Equal(t, 1, inner.fencedBeforeOpen, "the Run is fenced before it is opened")
	require.Equal(t, "run-1", driver.Opened())

	_, err = driver.Open(t.Context(), "run-2", testpilot.PreparedProgram{})
	require.ErrorIs(t, err, ErrSecondOpen)
	require.Equal(t, 1, inner.opens, "a second or concurrent Run opens nothing")
	require.Equal(t, []string{"run-1"}, fenced)
}

// A fence that fails -- a stale lease run, a lost connection -- opens nothing.
func TestFencedDriverOpensNothingWhenTheFenceFails(t *testing.T) {
	inner := &stubDriver{}
	stale := errors.New("the lease run is closed")
	driver := NewFencedDriver(inner, func(context.Context, string) error { return stale })
	_, err := driver.Open(t.Context(), "run-1", testpilot.PreparedProgram{})
	require.ErrorIs(t, err, stale)
	require.Zero(t, inner.opens)
	require.Empty(t, driver.Opened())

	_, err = NewFencedDriver(inner, func(context.Context, string) error { return nil }).Open(t.Context(), "", testpilot.PreparedProgram{})
	require.Error(t, err, "a Run with no ID cannot be fenced")
	require.Zero(t, inner.opens)
}

// The fenced Session starts only the workflow its Run ID names; every other call passes through.
func TestTheFencedSessionStartsOnlyTheFencedWorkflow(t *testing.T) {
	inner := &stubDriver{}
	driver := NewFencedDriver(inner, func(context.Context, string) error { return nil })
	session, err := driver.Open(t.Context(), "testpilot.run.1", testpilot.PreparedProgram{})
	require.NoError(t, err)
	start := workflowServiceMethod("StartWorkflowExecution")
	history := workflowServiceMethod("GetWorkflowExecutionHistory")

	_, err = session.InvokeRPC(t.Context(), testpilot.Coordinate{}, "role", start, &workflowservice.StartWorkflowExecutionRequest{WorkflowId: "testpilot.run.1"})
	require.NoError(t, err)
	_, err = session.InvokeRPC(t.Context(), testpilot.Coordinate{}, "role", start, &workflowservice.StartWorkflowExecutionRequest{WorkflowId: "customer-workflow"})
	require.ErrorIs(t, err, ErrOutsideFence)
	_, err = session.PollRPC(t.Context(), testpilot.Coordinate{}, "role", start, &workflowservice.StartWorkflowExecutionRequest{WorkflowId: "customer-workflow"}, time.Second, nil)
	require.ErrorIs(t, err, ErrOutsideFence)
	_, err = session.InvokeRPC(t.Context(), testpilot.Coordinate{}, "role", history, &workflowservice.GetWorkflowExecutionHistoryRequest{})
	require.NoError(t, err)
	require.Equal(t, []string{"StartWorkflowExecution", "GetWorkflowExecutionHistory"}, inner.session.invoked,
		"a start outside the fence never reaches the target")
}
