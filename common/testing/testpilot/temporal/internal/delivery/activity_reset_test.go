package delivery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
)

// A reset of a held attempt makes the server's next delivery attempt 1 again, under a new task
// token. The plan declares the reservation that delivery runs as, so a fresh delivery of attempt 1
// consumes that group, while the held attempt's own delivery again replays its admission exactly
// once and consumes nothing. Neither identity is read from the other: the SDK attempt stays the
// server's number and the reservation names the activation.
func TestAdmitActivityTakesTheDeclaredFreshGroupForAResetFirstAttempt(t *testing.T) {
	plan := activityPlanOf(2)
	plan.Reservations[0].Restart = 1
	f := newActivityFixtureWith(t, "run", "session", plan)
	header := activityHeader(t, f)
	consumed := func() []int64 { return []int64{f.reservations[0].Consumes(), f.reservations[1].Consumes()} }

	held, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "held-delivery"))
	require.NoError(t, err)
	require.Equal(t, admission{Coordinate: testpilot.Coordinate{RunID: "run", EntrypointID: "activity", ActivationID: "session-activity"}, Reservation: "session-activity", ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "held-delivery"}, admissionOf(held))
	require.Equal(t, []int64{1, 0}, consumed())

	again, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "held-delivery"))
	require.NoError(t, err)
	require.True(t, again.Replay())
	require.Equal(t, "session-activity", again.Reservation().ID)
	require.Equal(t, []int64{1, 0}, consumed())

	fresh, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "fresh-delivery"))
	require.NoError(t, err)
	require.Equal(t, admission{Coordinate: testpilot.Coordinate{RunID: "run", EntrypointID: "activity", ActivationID: "session-activity-1"}, Reservation: "session-activity-1", ActivityRunID: "activity-run", Attempt: 1, DeliveryID: "fresh-delivery"}, admissionOf(fresh))
	require.Equal(t, []int64{1, 1}, consumed())

	for name, delivery := range map[string]ActivityDelivery{
		"the fresh delivery again":      f.attempt(header, "activity-run", 1, "fresh-delivery"),
		"the held delivery once more":   f.attempt(header, "activity-run", 1, "held-delivery"),
		"a third delivery of attempt 1": f.attempt(header, "activity-run", 1, "third-delivery"),
	} {
		replay, err := f.ledger.AdmitActivity(context.Background(), delivery)
		require.NoError(t, err, name)
		require.True(t, replay.Replay(), name)
	}
	require.Equal(t, []int64{1, 1}, consumed())

	_, err = f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 2, "delivery-2"))
	require.ErrorIs(t, err, ErrAttemptUndeclared, "the reset leaves no reservation numbered 2")
}

// Without a declared restart a second delivery of attempt 1 is a redelivery, never a later group.
func TestAdmitActivityNeverTakesALaterGroupForAnUndeclaredRewind(t *testing.T) {
	f := newActivityFixtureOf(t, "run", "session", 2)
	header := activityHeader(t, f)
	_, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "held-delivery"))
	require.NoError(t, err)
	rewound, err := f.ledger.AdmitActivity(context.Background(), f.attempt(header, "activity-run", 1, "fresh-delivery"))
	require.NoError(t, err)
	require.True(t, rewound.Replay())
	require.Equal(t, "session-activity", rewound.Reservation().ID)
	require.Equal(t, []int64{1, 0}, []int64{f.reservations[0].Consumes(), f.reservations[1].Consumes()})
}

// A restart is a reservation of the activity's own topology after its first: none at or past the
// count, and none on a negative ordinal, is carried.
func TestCreateActivityBundleRefusesAnInvalidRestart(t *testing.T) {
	for name, restart := range map[string]int64{"past the count": 2, "negative": -1} {
		t.Run(name, func(t *testing.T) {
			ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8}})
			require.NoError(t, err)
			origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start-activity", Attempt: 1}
			plan := activityPlanOf(2)
			plan.Reservations[0].Restart = restart
			var handles []testpilot.ReservationHandle
			for ordinal, id := range []string{"session-activity", "session-activity-1"} {
				retained, err := ledger.RetainReservation(context.Background(), testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "activity", Ordinal: int64(ordinal), ID: id}))
				require.NoError(t, err)
				handles = append(handles, retained)
			}
			_, err = ledger.CreateActivityBundle(context.Background(), origin, plan, ActivityBinding{Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue"}, handles)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}
