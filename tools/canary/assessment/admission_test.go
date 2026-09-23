package assessment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/recordedrun"
	"google.golang.org/protobuf/proto"
)

// recorded is a Run of the pinned canary Case against the test cluster, as the lifecycle test
// records it with UMPIRE_CANARY_RECORD set: a test cluster's Run is the only one ever written.
func recorded(t *testing.T) recordedrun.Decoded {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", "nexusCallerCanary-syncCompletion-run.json"))
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(encoded)
	require.NoError(t, err)
	return decoded
}

func committed(t *testing.T) *policy.Policy {
	t.Helper()
	canary, err := policy.Embedded()
	require.NoError(t, err)
	return canary
}

// A closed Run of the pinned Case admits as fn-26's subject with its recorded Verdict,
// disposition and cleanup unchanged, and is accepted under the canary's Evaluation Profile.
func TestAdmitRecordsAClosedRunAndAdmitsIt(t *testing.T) {
	canary := committed(t)
	fixture := recorded(t)
	require.Equal(t, canary.CaseIdentity, fixture.Case)
	before := proto.CloneOf(fixture.Run)

	subject, err := Admit(canary, fixture.Driver, fixture.Run)
	require.NoError(t, err)
	require.Equal(t, fixture.Driver, subject.Driver)
	require.Equal(t, canary.CaseIdentity, subject.CaseIdentity)
	require.Equal(t, fixture.Run.GetRunId(), subject.RunID)
	require.Equal(t, fixture.Run.GetDisposition(), subject.Disposition)
	require.Equal(t, fixture.Run.GetCleanup().GetStatus(), subject.Cleanup)
	require.True(t, proto.Equal(fixture.Run.GetVerdict(), subject.Verdict), "the recorded Verdict is the subject's")
	require.True(t, proto.Equal(before, fixture.Run), "admission never changes the Run")
	require.Empty(t, subject.KnownGaps)

	profile, err := LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	decision := evaluation.Assess(subject, *profile)
	require.Equal(t, evaluation.DecisionAccepted, decision.Outcome)
}

// Every iteration that is not a closed Run of the pinned Case under the canary's Profile and the
// tree's catalog is refused: a lost one has no subject at all, and the rest are fn-26 rejections.
func TestAdmitRefusesEveryIterationItCannotStandBehind(t *testing.T) {
	canary := committed(t)
	for name, test := range map[string]struct {
		edit   func(canary *policy.Policy, driver *testpilot.DriverIdentity, run *testpilotspb.Run)
		reason string
	}{
		"a foreign Profile": {func(_ *policy.Policy, d *testpilot.DriverIdentity, _ *testpilotspb.Run) {
			d.Profile = "local-ephemeral"
		}, evaluation.ReasonCrossed},
		"a policy naming another Case": {func(c *policy.Policy, _ *testpilot.DriverIdentity, _ *testpilotspb.Run) {
			c.CaseIdentity = "0000000000000000000000000000000000000000000000000000000000000000"
		}, evaluation.ReasonCrossed},
		"another catalog": {func(_ *policy.Policy, d *testpilot.DriverIdentity, _ *testpilotspb.Run) {
			d.Catalog = "1111111111111111111111111111111111111111111111111111111111111111"
		}, evaluation.ReasonStale},
		"a Run of another Case": {func(_ *policy.Policy, _ *testpilot.DriverIdentity, r *testpilotspb.Run) {
			r.CaseId = "temporal.case.nexusCallerTests.syncCompletion"
		}, evaluation.ReasonCrossed},
		"an open Run": {func(_ *policy.Policy, _ *testpilot.DriverIdentity, r *testpilotspb.Run) {
			r.Disposition = testpilotspb.RUN_DISPOSITION_UNSPECIFIED
		}, evaluation.ReasonOpen},
		"a Run with no Verdict": {func(_ *policy.Policy, _ *testpilot.DriverIdentity, r *testpilotspb.Run) {
			r.Verdict = nil
		}, evaluation.ReasonOpen},
		"a Verdict the Run does not support": {func(_ *policy.Policy, _ *testpilot.DriverIdentity, r *testpilotspb.Run) {
			r.Verdict.Status = testpilotspb.VERDICT_STATUS_VIOLATED
		}, evaluation.ReasonInconsistent},
		"a Run past the event cap": {func(_ *policy.Policy, _ *testpilot.DriverIdentity, r *testpilotspb.Run) {
			for len(r.Events) <= evaluation.MaxRunEvents {
				r.Events = append(r.Events, &testpilotspb.RunEvent{Sequence: int64(len(r.Events) + 1)})
			}
		}, evaluation.ReasonOversized},
	} {
		t.Run(name, func(t *testing.T) {
			edited := *canary
			fixture := recorded(t)
			test.edit(&edited, &fixture.Driver, fixture.Run)
			subject, err := Admit(&edited, fixture.Driver, fixture.Run)
			require.Nil(t, subject)
			rejection, ok := evaluation.IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, test.reason, rejection.Reason, rejection.Detail)
		})
	}

	t.Run("a lost iteration", func(t *testing.T) {
		subject, err := Admit(canary, recorded(t).Driver, nil)
		require.Nil(t, subject)
		require.ErrorIs(t, err, ErrLost)
		_, rejected := evaluation.IsRejection(err)
		require.False(t, rejected, "a lost iteration is not a subject to reject; it has none")
	})

	_, err := Admit(nil, recorded(t).Driver, recorded(t).Run)
	require.Error(t, err)
}
