package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/umpire/run/v1"
	"umpire.local/model/umpire/runner"
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

// A fenced Driver starts its own activity and any read, and refuses any other start before it
// reaches the wrapped Driver; it keeps the wrapped Driver's Close and redaction.
func TestTheFenceAdmitsOnlyItsOwnActivity(t *testing.T) {
	id := activityID(1)
	inner := &stubDriver{}
	newDriver := fenced(func(context.Context) (runner.Driver, error) { return inner, nil }, id)
	driver, err := newDriver(t.Context())
	require.NoError(t, err)

	_, err = driver.Start(t.Context(), 1, packed(t, &workflowservice.StartActivityExecutionRequest{ActivityId: id}))
	require.NoError(t, err)
	_, err = driver.Start(t.Context(), 2, packed(t, &workflowservice.DescribeActivityExecutionRequest{ActivityId: "anything"}))
	require.NoError(t, err)
	for name, refused := range map[string]proto.Message{
		"another activity":  &workflowservice.StartActivityExecutionRequest{ActivityId: activityID(2)},
		"a workflow":        &workflowservice.StartWorkflowExecutionRequest{WorkflowId: id},
		"a batch operation": &workflowservice.StartBatchOperationRequest{JobId: id},
	} {
		_, err := driver.Start(t.Context(), 3, packed(t, refused))
		require.ErrorIs(t, err, ErrOutsideFence, name)
	}
	require.Equal(t, []string{
		"temporal.api.workflowservice.v1.StartActivityExecutionRequest",
		"temporal.api.workflowservice.v1.DescribeActivityExecutionRequest",
	}, inner.started, "a refused start never reaches the wrapped Driver")

	require.True(t, driver.(runner.Redactor).Redact(&runv1.Event{}))
	require.NoError(t, driver.(interface{ Close() error }).Close())
	require.True(t, inner.closed)

	failing := fenced(func(context.Context) (runner.Driver, error) { return nil, errors.New("no Driver") }, id)
	_, err = failing(t.Context())
	require.Error(t, err)
}

func TestActivityIDsAreTheCanarysOwn(t *testing.T) {
	require.True(t, IsActivityID(newActivityID()))
	require.NotEqual(t, newActivityID(), newActivityID())
	for _, id := range []string{"", "umpire-canary-", "umpire-canary-lease", "customer-activity", "umpire-canary-NOT-A-UUID",
		"umpire-canary-00000000-0000-4000-8000-00000000000G"} {
		require.False(t, IsActivityID(id), id)
	}
}
