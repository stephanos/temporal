package execution

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
)

// Every outcome a reservation can settle with is judged by one table. This enumerates every
// entrypoint kind, every outcome status and every activity attempt response, an unset and an
// out-of-range value of each included, with the attempt fact absent and present, and holds the
// verdict to the combinations written out here: everything else is rejected.
func TestReservationOutcomesAreJudgedByOneClosedTable(t *testing.T) {
	const workflow, handler, activity = contract.WorkflowEntrypoint, contract.NexusHandlerEntrypoint, contract.ActivityEntrypoint
	succeeded, failed, canceled := testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
	type combination struct {
		kind     contract.EntrypointKind
		status   testpilotspb.InstructionOutcomeStatus
		response testpilotspb.ActivityAttemptResponse
		attempt  bool
		unused   bool
	}
	allowed := map[combination]reservationVerdict{
		{kind: workflow, status: succeeded}:               reservationRecorded,
		{kind: workflow, status: succeeded, unused: true}: reservationRecorded,
		{kind: workflow, status: canceled, unused: true}:  reservationRecorded,
		{kind: handler, status: succeeded}:                reservationRecorded,
		{kind: handler, status: succeeded, unused: true}:  reservationRecorded,
		{kind: handler, status: canceled, unused: true}:   reservationRecorded,
	}
	for response, verdict := range map[testpilotspb.ActivityAttemptResponse]struct {
		status  testpilotspb.InstructionOutcomeStatus
		verdict reservationVerdict
	}{
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED:            {succeeded, reservationRecorded},
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE:     {succeeded, reservationRecorded},
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE: {succeeded, reservationRecorded},
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED:             {succeeded, reservationRecorded},
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED:                      {failed, reservationRecordedThenFailed},
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED:                   {canceled, reservationRecorded},
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_WITHHELD:                     {succeeded, reservationRecorded},
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING:                      {succeeded, reservationRecorded},
	} {
		for _, unused := range []bool{false, true} {
			allowed[combination{kind: activity, status: verdict.status, response: response, attempt: true, unused: unused}] = verdict.verdict
		}
	}

	// wellFormed is the attempt fact a response requires at the second position of an activity.
	wellFormed := func(response testpilotspb.ActivityAttemptResponse) *testpilotspb.ActivityAttempt {
		if response == testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED {
			return &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", Response: response}
		}
		return &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", SdkAttempt: 2, DeliveryId: "delivery-2", Response: response}
	}
	kinds := []contract.EntrypointKind{0, contract.ControllerEntrypoint, workflow, activity, handler, 99}
	statuses := []testpilotspb.InstructionOutcomeStatus{99}
	for number := range testpilotspb.InstructionOutcomeStatus_name {
		statuses = append(statuses, testpilotspb.InstructionOutcomeStatus(number))
	}
	responses := []testpilotspb.ActivityAttemptResponse{99}
	for number := range testpilotspb.ActivityAttemptResponse_name {
		responses = append(responses, testpilotspb.ActivityAttemptResponse(number))
	}
	numbering := &testpilotspb.AttemptNumbering{First: 1, OneRun: true}
	judged := 0
	for _, kind := range kinds {
		for _, status := range statuses {
			for _, unused := range []bool{false, true} {
				absent := combination{kind: kind, status: status, unused: unused}
				require.Equal(t, allowed[absent], judgeReservation(kind, numbering, 0, unused, 1, "activity-run", &testpilotspb.InstructionOutcome{Status: status}), fmt.Sprint(absent))
				judged++
				for _, response := range responses {
					present := combination{kind: kind, status: status, response: response, attempt: true, unused: unused}
					outcome := &testpilotspb.InstructionOutcome{Status: status, ActivityAttempt: wellFormed(response)}
					require.Equal(t, allowed[present], judgeReservation(kind, numbering, 0, unused, 1, "activity-run", outcome), fmt.Sprint(present))
					judged++
				}
			}
		}
	}
	require.Equal(t, len(kinds)*len(statuses)*2*(1+len(responses)), judged)
	require.Len(t, statuses, 7)
	require.Len(t, responses, 10)
	// The table holds nothing the list above does not: one row per allowed combination, whatever
	// the entrypoint performs.
	require.Len(t, reservationOutcomes, 12)
	require.Equal(t, reservationRejected, judgeReservation(activity, numbering, 0, false, 1, "activity-run", nil))
}

// An attempt fact is admitted only with the identities its response requires: a delivered attempt
// names its activity run, the SDK attempt of its own position as the entrypoint numbers attempts and
// its delivery, and a position that was not needed names the run alone and follows a recorded
// attempt. Where the entrypoint declares every attempt of one run, every outcome of one activity names
// one activity run, the one its first recorded attempt named.
func TestReservationOutcomeRequiresTheIdentitiesOfItsResponse(t *testing.T) {
	temporal := &testpilotspb.AttemptNumbering{First: 1, OneRun: true}
	offered := func(mutate func(*testpilotspb.ActivityAttempt)) *testpilotspb.InstructionOutcome {
		attempt := &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", SdkAttempt: 2, DeliveryId: "delivery-2", Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED}
		mutate(attempt)
		return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ActivityAttempt: attempt}
	}
	notNeeded := func(mutate func(*testpilotspb.ActivityAttempt)) *testpilotspb.InstructionOutcome {
		attempt := &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED}
		mutate(attempt)
		return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, ActivityAttempt: attempt}
	}
	for name, test := range map[string]struct {
		outcome  *testpilotspb.InstructionOutcome
		priorRun string
		want     reservationVerdict
	}{
		"a first delivered attempt":               {offered(func(*testpilotspb.ActivityAttempt) {}), "", reservationRecorded},
		"a later delivered attempt":               {offered(func(*testpilotspb.ActivityAttempt) {}), "activity-run", reservationRecorded},
		"a delivered attempt of another run":      {offered(func(*testpilotspb.ActivityAttempt) {}), "another-run", reservationRejected},
		"a delivered attempt with no run":         {offered(func(a *testpilotspb.ActivityAttempt) { a.ActivityRunId = "" }), "", reservationRejected},
		"a delivered attempt with no delivery":    {offered(func(a *testpilotspb.ActivityAttempt) { a.DeliveryId = "" }), "activity-run", reservationRejected},
		"a delivered attempt with no number":      {offered(func(a *testpilotspb.ActivityAttempt) { a.SdkAttempt = 0 }), "activity-run", reservationRejected},
		"a delivered attempt of another position": {offered(func(a *testpilotspb.ActivityAttempt) { a.SdkAttempt = 1 }), "activity-run", reservationRejected},
		"a position not needed":                   {notNeeded(func(*testpilotspb.ActivityAttempt) {}), "activity-run", reservationRecorded},
		"one in another run":                      {notNeeded(func(*testpilotspb.ActivityAttempt) {}), "another-run", reservationRejected},
		"one before any attempt":                  {notNeeded(func(*testpilotspb.ActivityAttempt) {}), "", reservationRejected},
		"one with no run before any attempt":      {notNeeded(func(a *testpilotspb.ActivityAttempt) { a.ActivityRunId = "" }), "", reservationRejected},
		"one with no run":                         {notNeeded(func(a *testpilotspb.ActivityAttempt) { a.ActivityRunId = "" }), "activity-run", reservationRejected},
		"one that names an SDK attempt":           {notNeeded(func(a *testpilotspb.ActivityAttempt) { a.SdkAttempt = 2 }), "activity-run", reservationRejected},
		"one that names a delivery":               {notNeeded(func(a *testpilotspb.ActivityAttempt) { a.DeliveryId = "delivery-2" }), "activity-run", reservationRejected},
		"one that claims a heartbeat invocation":  {notNeeded(func(a *testpilotspb.ActivityAttempt) { a.HeartbeatInvoked = true }), "activity-run", reservationRejected},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, judgeReservation(contract.ActivityEntrypoint, temporal, 0, false, 1, test.priorRun, test.outcome))
		})
	}
	// The numbering is the entrypoint's: counted from another first, the position takes another
	// number, without one run an attempt may name another run, and with no numbering no delivered
	// attempt is admitted.
	for name, test := range map[string]struct {
		numbering *testpilotspb.AttemptNumbering
		outcome   *testpilotspb.InstructionOutcome
		priorRun  string
		want      reservationVerdict
	}{
		"numbered from 2": {&testpilotspb.AttemptNumbering{First: 2, OneRun: true}, offered(func(a *testpilotspb.ActivityAttempt) { a.SdkAttempt = 3 }),
			"activity-run", reservationRecorded},
		"numbered from 1 where the entrypoint numbers from 2": {&testpilotspb.AttemptNumbering{First: 2, OneRun: true}, offered(func(*testpilotspb.ActivityAttempt) {}),
			"activity-run", reservationRejected},
		"another run where runs may differ":     {&testpilotspb.AttemptNumbering{First: 1}, offered(func(*testpilotspb.ActivityAttempt) {}), "another-run", reservationRecorded},
		"one not needed where runs may differ":  {&testpilotspb.AttemptNumbering{First: 1}, notNeeded(func(*testpilotspb.ActivityAttempt) {}), "another-run", reservationRecorded},
		"a delivered attempt with no numbering": {nil, offered(func(*testpilotspb.ActivityAttempt) {}), "activity-run", reservationRejected},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, judgeReservation(contract.ActivityEntrypoint, test.numbering, 0, false, 1, test.priorRun, test.outcome))
		})
	}
}
