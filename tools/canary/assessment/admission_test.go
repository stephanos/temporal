package assessment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/evaluation"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
)

func currentRecorded(t *testing.T) recordedrun.Decoded {
	t.Helper()
	return recordedIn(t, "nexus-workflow-syncCompletion-current-run.json")
}

func recordedIn(t *testing.T, name string) recordedrun.Decoded {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name))
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

// The Runs recorded of the Cases the canary pinned before are kept with their Cases. Both were
// recorded under retired Case format 1.0, so each Case is refused before its Run payload is read,
// and against the pinned Case each record is crossed.
func TestRetiredRecordsAreRefusedBeforeTheirRuns(t *testing.T) {
	catalog := currentRecorded(t).Driver.Catalog
	for prior, run := range map[string]string{
		"nexus-workflow-syncCompletion-historical-case.json": "nexus-workflow-syncCompletion-run.json",
		"nexusCallerCanary-syncCompletion-case.json":         "nexusCallerCanary-syncCompletion-run.json",
		"nexus-caller-syncCompletion-historical-case.json":   "nexus-caller-syncCompletion-historical-run.json",
	} {
		t.Run(run, func(t *testing.T) {
			caseBytes, err := os.ReadFile(filepath.Join("testdata", prior))
			require.NoError(t, err)
			recorded, err := os.ReadFile(filepath.Join("testdata", run))
			require.NoError(t, err)
			_, err = recordedrun.Decode(recorded)
			require.Error(t, err, "the retired record's Run payload is not format 4.0")

			subject, err := evaluation.Admit(caseBytes, recorded, catalog)
			require.Nil(t, subject)
			rejection, ok := evaluation.IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, evaluation.ReasonIncompatible, rejection.Reason, rejection.Detail)
			require.Contains(t, rejection.Detail, "unsupported Case version 1.0")

			crossed, err := evaluation.Admit(casebinding.Case(), recorded, catalog)
			require.Nil(t, crossed)
			rejection, ok = evaluation.IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, evaluation.ReasonCrossed, rejection.Reason, rejection.Detail)
		})
	}
}

// Every iteration that is not a closed Run of the pinned Case under the canary's Profile and the
// tree's catalog is refused: a lost one has no subject at all, and the rest are fn-26 rejections.
func TestAdmitRefusesEveryIterationItCannotStandBehind(t *testing.T) {
	canary := committed(t)
	baseline := currentRecorded(t)
	subject, err := Admit(canary, baseline.Driver, baseline.Run)
	require.NoError(t, err)
	profile, err := LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	require.Equal(t, evaluation.DecisionAccepted, evaluation.Assess(subject, *profile, nil).Outcome)
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
			fixture := currentRecorded(t)
			test.edit(&edited, &fixture.Driver, fixture.Run)
			subject, err := Admit(&edited, fixture.Driver, fixture.Run)
			require.Nil(t, subject)
			rejection, ok := evaluation.IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, test.reason, rejection.Reason, rejection.Detail)
		})
	}

	t.Run("a lost iteration", func(t *testing.T) {
		subject, err := Admit(canary, currentRecorded(t).Driver, nil)
		require.Nil(t, subject)
		require.ErrorIs(t, err, ErrLost)
		_, rejected := evaluation.IsRejection(err)
		require.False(t, rejected, "a lost iteration is not a subject to reject; it has none")
	})

	_, err = Admit(nil, currentRecorded(t).Driver, currentRecorded(t).Run)
	require.Error(t, err)
}
