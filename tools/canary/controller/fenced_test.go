package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/gen/umpire/run/v1"
	"umpire.local/model/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// stubDriver records the requests it is asked to start and whether it was closed.
type stubDriver struct {
	started []string
	closed  bool
}

func (d *stubDriver) Start(_ context.Context, _ uint64, payload *anypb.Any) (runner.Exchange, error) {
	d.started = append(d.started, string(payload.MessageName()))
	return nil, nil
}

func (d *stubDriver) Close() error {
	d.closed = true
	return nil
}

func (d *stubDriver) Redact(*runv1.Event) bool { return true }

func packed(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	payload, err := anypb.New(message)
	require.NoError(t, err)
	return payload
}

// A fenced Driver sends only the canary World's activity calls on its own activity, alone or as a
// read instruction, and refuses everything else before it reaches the wrapped Driver; it keeps the
// wrapped Driver's Close and redaction.
func TestTheFenceAdmitsOnlyItsOwnActivity(t *testing.T) {
	id := activityID(1)
	inner := &stubDriver{}
	newDriver := fenced(func(context.Context) (runner.Driver, error) { return inner, nil }, id)
	driver, err := newDriver(t.Context())
	require.NoError(t, err)

	read := func(request proto.Message) *temporalv1.ObserveActivity {
		return &temporalv1.ObserveActivity{Request: packed(t, request), PollIntervalMs: 1, BackoffMs: 1, ReadTimeoutMs: 1}
	}
	for name, admitted := range map[string]proto.Message{
		"its start":              &workflowservice.StartActivityExecutionRequest{ActivityId: id},
		"its cancel":             &workflowservice.RequestCancelActivityExecutionRequest{ActivityId: id},
		"its describe":           &workflowservice.DescribeActivityExecutionRequest{ActivityId: id},
		"a read of its describe": read(&workflowservice.DescribeActivityExecutionRequest{ActivityId: id}),
		"a read of its poll":     read(&workflowservice.PollActivityExecutionRequest{ActivityId: id}),
	} {
		_, err := driver.Start(t.Context(), 1, packed(t, admitted))
		require.NoError(t, err, name)
	}
	require.Len(t, inner.started, 5)
	for name, refused := range map[string]proto.Message{
		"another activity's start":            &workflowservice.StartActivityExecutionRequest{ActivityId: activityID(2)},
		"another activity's cancel":           &workflowservice.RequestCancelActivityExecutionRequest{ActivityId: activityID(2)},
		"a read of another activity":          read(&workflowservice.DescribeActivityExecutionRequest{ActivityId: activityID(2)}),
		"a workflow":                          &workflowservice.StartWorkflowExecutionRequest{WorkflowId: id},
		"a read that starts a workflow":       read(&workflowservice.StartWorkflowExecutionRequest{WorkflowId: "customer-workflow"}),
		"a read that starts its own activity": read(&workflowservice.StartActivityExecutionRequest{ActivityId: id}),
		"a signal-with-start":                 &workflowservice.SignalWithStartWorkflowExecutionRequest{WorkflowId: "customer-workflow"},
		"a multi-operation":                   &workflowservice.ExecuteMultiOperationRequest{},
		"a schedule":                          &workflowservice.CreateScheduleRequest{ScheduleId: id},
		"a batch operation":                   &workflowservice.StartBatchOperationRequest{JobId: id},
	} {
		_, err := driver.Start(t.Context(), 3, packed(t, refused))
		require.ErrorIs(t, err, ErrOutsideFence, name)
	}
	require.Len(t, inner.started, 5, "a refused request never reaches the wrapped Driver")

	require.True(t, driver.(runner.Redactor).Redact(&runv1.Event{}))
	require.NoError(t, driver.(interface{ Close() error }).Close())
	require.True(t, inner.closed)

	failing := fenced(func(context.Context) (runner.Driver, error) { return nil, errors.New("no Driver") }, id)
	_, err = failing(t.Context())
	require.Error(t, err)
}

func TestActivityIDsAreTheCanarysOwn(t *testing.T) {
	require.True(t, IsActivityID(newActivityID()))
	first, second := newActivityID(), newActivityID()
	require.NotEqual(t, first, second)
	for _, id := range []string{"", "umpire-canary-", "umpire-canary-lease", "customer-activity", "umpire-canary-NOT-A-UUID",
		"umpire-canary-00000000-0000-4000-8000-00000000000G"} {
		require.False(t, IsActivityID(id), id)
	}
}
