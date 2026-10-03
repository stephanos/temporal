package delivery

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
)

// scheduledFixture is a workflow start that carries its workflow and the three declared attempts of
// the activity the workflow's schedule command "schedule" reaches.
type scheduledFixture struct {
	*fixture
	attempts []*testsupport.Reservation
}

func newScheduledFixture(t *testing.T) *scheduledFixture {
	t.Helper()
	ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8}})
	require.NoError(t, err)
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start-workflow", Attempt: 1}
	plan := testpilot.ReservationCarrierPlan{
		EndpointRoleID: "temporal",
		Method:         primitive.StartWorkflowPath,
		Reservations: []testpilot.ReservationTopology{
			{EntrypointID: "workflow", Kind: testpilot.WorkflowEntrypoint, Count: 1},
			{EntrypointID: "activity", Kind: testpilot.ActivityEntrypoint, Count: 3},
		},
		Routes: []testpilot.ReservationRoute{{WorkflowEntrypointID: "workflow", SourceInstructionID: "schedule", HandlerEntrypointID: "activity"}},
	}
	workflow := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "workflow"})
	reservations := []*testsupport.Reservation{workflow}
	var attempts []*testsupport.Reservation
	for ordinal := range int64(3) {
		attempt := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "activity", Ordinal: ordinal, ID: fmt.Sprintf("attempt-%d", ordinal+1)})
		attempts = append(attempts, attempt)
		reservations = append(reservations, attempt)
	}
	var handles []testpilot.ReservationHandle
	// Retained out of order, as a carrier may hand them over.
	for _, index := range []int{3, 0, 1, 2} {
		retained, err := ledger.RetainReservation(context.Background(), reservations[index])
		require.NoError(t, err)
		handles = append(handles, retained)
	}
	binding := WorkflowBinding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	bundle, err := ledger.CreateBundle(context.Background(), origin, plan, binding, handles)
	require.NoError(t, err)
	return &scheduledFixture{fixture: &fixture{ledger: ledger, origin: origin, plan: plan, binding: binding, workflow: workflow, bundle: bundle}, attempts: attempts}
}

func scheduledAttempt(header *commonpb.Header, workflowRunID, activityID string, attempt int32) ActivityDelivery {
	return ActivityDelivery{Header: header, Namespace: "namespace", ActivityID: activityID, ActivityType: "activity-type", TaskQueue: "task-queue", WorkflowRunID: workflowRunID, Attempt: attempt, DeliveryID: fmt.Sprintf("delivery-%d", attempt)}
}

// A workflow's schedule command carries the route of the activity it reaches, and the server's
// attempt N is admitted under that activity's reservation of ordinal N-1, whatever order the
// attempts arrive in; a second delivery of an attempt replays its admission. An attempt the script
// does not declare, one of another workflow run, or one under another activity ID is refused.
func TestAdmitScheduledActivityBindsEachAttemptToItsWorkflowsReservation(t *testing.T) {
	f := newScheduledFixture(t)
	workflow := admitWorkflow(t, f.fixture, "workflow-run")
	_, err := f.ledger.PrepareActivity(context.Background(), workflow, "another-command")
	require.ErrorIs(t, err, ErrRouteCrossed)
	dispatch, err := f.ledger.PrepareActivity(context.Background(), workflow, "schedule")
	require.NoError(t, err)
	name, payload := dispatch.Header()
	require.Equal(t, ScheduledActivityHeader, name)
	header := &commonpb.Header{Fields: map[string]*commonpb.Payload{name: payload}}
	delivered := ActivityDelivery{Header: header}
	require.True(t, delivered.Scheduled())
	entrypoint, err := f.ledger.ScheduledEntrypoint(delivered)
	require.NoError(t, err)
	require.Equal(t, "activity", entrypoint)

	second, err := f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 2))
	require.NoError(t, err)
	require.Equal(t, "attempt-2", second.Reservation().ID)
	require.Equal(t, "workflow-run", second.TemporalRunID())
	first, err := f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 1))
	require.NoError(t, err)
	require.Equal(t, "attempt-1", first.Reservation().ID)
	require.False(t, first.Replay())
	again, err := f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 1))
	require.NoError(t, err)
	require.True(t, again.Replay())
	require.Equal(t, int64(1), f.attempts[0].Consumes())

	_, err = f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 4))
	require.ErrorIs(t, err, ErrAttemptUndeclared)
	_, err = f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "another-run", "7", 3))
	require.ErrorIs(t, err, ErrRouteConflict)
	_, err = f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "8", 3))
	require.ErrorIs(t, err, ErrRouteConflict)
	misbound := scheduledAttempt(header, "workflow-run", "7", 3)
	misbound.Namespace = "another-namespace"
	_, err = f.ledger.AdmitScheduledActivity(context.Background(), misbound)
	require.ErrorIs(t, err, ErrBindingMismatch)
	require.Zero(t, f.attempts[2].Consumes())

	// A standalone activity's admission does not read a scheduled route, nor the other way round.
	_, err = f.ledger.AdmitActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 3))
	require.ErrorIs(t, err, ErrRouteMissing)
}

// The workflow closing ends its activities: the declared attempts after the last one delivered are
// released as not needed, once, and a later attempt is stale. An activity none of whose attempts
// was delivered is not explained by the closure, so its attempts stay reserved.
func TestParentTerminalReleasesTheScheduledAttemptsNoDeliveryNeeds(t *testing.T) {
	for name, delivered := range map[string][]int32{"one attempt delivered": {1}, "none delivered": nil} {
		t.Run(name, func(t *testing.T) {
			f := newScheduledFixture(t)
			workflow := admitWorkflow(t, f.fixture, "workflow-run")
			dispatch, err := f.ledger.PrepareActivity(context.Background(), workflow, "schedule")
			require.NoError(t, err)
			name, payload := dispatch.Header()
			header := &commonpb.Header{Fields: map[string]*commonpb.Payload{name: payload}}
			for _, attempt := range delivered {
				_, err := f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", attempt))
				require.NoError(t, err)
			}
			release, err := f.ledger.ParentTerminal(context.Background(), workflow)
			require.NoError(t, err)
			again, err := f.ledger.ParentTerminal(context.Background(), workflow)
			require.NoError(t, err)
			require.Empty(t, again.NotNeeded())
			if delivered == nil {
				require.Empty(t, release.NotNeeded())
			} else {
				var released []string
				for _, identity := range release.NotNeeded() {
					released = append(released, identity.ID)
				}
				require.Equal(t, []string{"attempt-2", "attempt-3"}, released)
			}
			_, err = f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 2))
			require.ErrorIs(t, err, ErrRouteStale)
			for _, attempt := range f.attempts {
				require.Zero(t, attempt.Cancels())
			}
		})
	}
}

// An attempt's route leaves the ledger when its reservation settles, but the activity's later
// attempts are still admitted under the route its schedule command carries.
func TestAdmitScheduledActivityAfterAnEarlierAttemptSettled(t *testing.T) {
	f := newScheduledFixture(t)
	workflow := admitWorkflow(t, f.fixture, "workflow-run")
	dispatch, err := f.ledger.PrepareActivity(context.Background(), workflow, "schedule")
	require.NoError(t, err)
	name, payload := dispatch.Header()
	header := &commonpb.Header{Fields: map[string]*commonpb.Payload{name: payload}}
	first, err := f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 1))
	require.NoError(t, err)
	f.attempts[0].Complete()
	for _, handle := range f.bundle.Handles() {
		if handle.(testpilot.ReservationHandle).Identity().ID == first.Reservation().ID {
			_, err := handle.Wait(context.Background())
			require.NoError(t, err)
		}
	}
	require.NotContains(t, f.ledger.routes, first.Reservation().ID)
	second, err := f.ledger.AdmitScheduledActivity(context.Background(), scheduledAttempt(header, "workflow-run", "7", 2))
	require.NoError(t, err)
	require.Equal(t, "attempt-2", second.Reservation().ID)
}
