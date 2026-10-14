//go:build test_dep

package conformance

// The held race with its release decided by the observer a live Driver uses (control.Deliveries),
// fed the answers a server gives, and the Run that follows assessed. What the observer makes of a
// sequence of answers is thereby read through to the Verdict and the claim.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/api/historyservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/namespace"
	serviceerrors "go.temporal.io/server/common/serviceerror"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/control"
)

// admissionAnswer is one answer of history's admission for the held delivery.
type admissionAnswer struct {
	response any
	err      error
}

// observedRelease is the release of the held race as the observer decides it from the answers the
// server gives the released delivery, in order: a success with what it recorded, or, where it has no
// decision when the instruction's deadline ends it, a timeout that records none.
func observedRelease(t *testing.T, answers ...admissionAnswer) func(runID string) effect {
	return func(runID string) effect {
		scope := namespace.ID("namespace-id")
		hooks := testhooks.NewTestHooks()
		deliveries, err := control.NewDeliveries(func(h testhooks.Hook) func() { return h.Apply(hooks, scope) })
		require.NoError(t, err)
		defer deliveries.Close()
		dispatch, ok := testhooks.Get(hooks, testhooks.ActivityDispatch, scope)
		require.True(t, ok)
		respond, ok := testhooks.Get(hooks, testhooks.GRPCResponseFaultGeneratorByNamespaceID, scope)
		require.True(t, ok)
		key := chasm.ExecutionKey{NamespaceID: string(scope), BusinessID: runID, RunID: "activity-run"}
		component := chasm.NewComponentRefByArchetypeID(key, 1)
		ref, err := component.Serialize(nil)
		require.NoError(t, err)

		held, err := deliveries.Hold(runID)
		require.NoError(t, err)
		dispatched := make(chan error, 1)
		go func() { dispatched <- dispatch(t.Context(), testhooks.ActivityDelivery{Execution: key, Stamp: 1}) }()
		require.NoError(t, held.Await(t.Context()))
		deadline, expire := context.WithCancel(t.Context())
		defer expire()
		admission, err := held.Release(deadline, func(polling context.Context) error {
			require.NoError(t, <-dispatched)
			for _, answer := range answers {
				respond(polling, "", &historyservice.RecordActivityTaskStartedRequest{ComponentRef: ref, Stamp: 1}, answer.response, answer.err)
			}
			expire()
			<-polling.Done()
			return polling.Err()
		})
		if err != nil {
			require.ErrorIs(t, err, context.Canceled)
			return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, "", nil)
		}
		return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, "", admission)
	}
}

// The server answers a delivery as obsolete when its stamp is no longer the activity's, which commits
// nothing, and also when the activity has already started. A redelivery meets the second after an
// admission that committed and whose answer was lost, so an obsolete answer that follows a lost one
// is no rejection: the release records no decision, and the Run is inconclusive, never satisfied. The
// same obsolete answer with nothing lost before it is the rejection, and the Run is satisfied.
func TestTheHeldRaceReadsNoRejectionAfterALostAnswer(t *testing.T) {
	obsolete := serviceerrors.NewObsoleteMatchingTask("invalid transition")
	for _, test := range []struct {
		name     string
		answers  []admissionAnswer
		verdict  testpilotspb.VerdictStatus
		property testpilot.PropertyStatus
	}{
		{"the stale message is refused", []admissionAnswer{{nil, obsolete}},
			testpilotspb.VERDICT_STATUS_SATISFIED, testpilot.PropertySatisfied},
		{"an answer is lost and the redelivery is refused", []admissionAnswer{{nil, serviceerror.NewUnavailable("lost")}, {nil, obsolete}},
			testpilotspb.VERDICT_STATUS_INCONCLUSIVE, testpilot.PropertyInconclusive},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := loweredRace(t)
			run, verdict, live, err := b.assessed.Run(t.Context(), &raceDriver{identity: b.plain.Identity(),
				hold:    func() effect { return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, "", nil) },
				release: observedRelease(t, test.answers...)})
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), "%v", run.GetDiagnostics())
			require.Equal(t, test.verdict, verdict.GetStatus())
			require.Nil(t, live.Failure)
			require.Equal(t, testpilot.ConformanceConformant, live.Conformance.Status, live.Conformance.Detail)
			require.Equal(t, test.property, raceClaim(t, live, raceProperty).Status)
			for _, property := range live.Properties {
				require.NotEqual(t, testpilot.PropertyViolated, property.Status, property.ID)
			}
			replayed, evaluation, err := b.assessed.Evaluate(t.Context(), run, live)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, live, evaluation.Assessment)
		})
	}
}
