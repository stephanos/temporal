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
	"google.golang.org/protobuf/proto"
)

// recorded is the test-cluster Run of the canary Case pinned before the kind extraction.
// Its immutable Case sits beside it; it is not a Run of the current production pin.
func recorded(t *testing.T) recordedrun.Decoded {
	t.Helper()
	return recordedIn(t, "nexus-workflow-syncCompletion-run.json")
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

// The closed Run keeps the exact Case it was recorded against before the kind extraction.
// Admission retains its Verdict, disposition and cleanup; the current pin refuses that identity.
func TestHistoricalClosedRunKeepsAdmissionAndDecision(t *testing.T) {
	canary := committed(t)
	fixture := recorded(t)
	prior, err := os.ReadFile(filepath.Join("testdata", "nexus-workflow-syncCompletion-historical-case.json"))
	require.NoError(t, err)
	identity, err := recordedrun.CaseIdentity(prior)
	require.NoError(t, err)
	require.Equal(t, fixture.Case, identity, "the immutable Case is the recorded Run's")
	require.NotEqual(t, canary.CaseIdentity, fixture.Case)
	then := *canary
	then.CaseIdentity = identity
	before := proto.CloneOf(fixture.Run)

	subject, err := admit(prior, &then, fixture.Driver, fixture.Run)
	require.NoError(t, err)
	require.Equal(t, fixture.Driver, subject.Driver)
	require.Equal(t, fixture.Case, subject.CaseIdentity)
	require.Equal(t, fixture.Run.GetRunId(), subject.RunID)
	require.Equal(t, fixture.Run.GetDisposition(), subject.Disposition)
	require.Equal(t, fixture.Run.GetCleanup().GetStatus(), subject.Cleanup)
	require.True(t, proto.Equal(fixture.Run.GetVerdict(), subject.Verdict), "the recorded Verdict is the subject's")
	require.True(t, proto.Equal(before, fixture.Run), "admission never changes the Run")
	require.Empty(t, subject.KnownGaps)

	profile, err := LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	decision := evaluation.Assess(subject, *profile, nil)
	require.Equal(t, evaluation.DecisionAccepted, decision.Outcome)
	recordedBytes, err := os.ReadFile(filepath.Join("testdata", "nexus-workflow-syncCompletion-run.json"))
	require.NoError(t, err)
	crossed, err := evaluation.Admit(casebinding.Case(), recordedBytes, fixture.Driver.Catalog)
	require.Nil(t, crossed)
	rejection, ok := evaluation.IsRejection(err)
	require.True(t, ok, "not a rejection: %v", err)
	require.Equal(t, evaluation.ReasonCrossed, rejection.Reason, rejection.Detail)
}

// The Run recorded of the Case the canary pinned before is kept unchanged, with that Case beside it.
// Under the identities it was recorded with it is admitted and accepted as it was; under the
// committed policy it is crossed, so it is never read as a Run of the Case pinned now.
func TestTheRunOfThePriorPinnedCaseKeepsItsDecision(t *testing.T) {
	prior, err := os.ReadFile(filepath.Join("testdata", "nexusCallerCanary-syncCompletion-case.json"))
	require.NoError(t, err)
	fixture := recordedIn(t, "nexusCallerCanary-syncCompletion-run.json")
	identity, err := recordedrun.CaseIdentity(prior)
	require.NoError(t, err)
	require.Equal(t, fixture.Case, identity)
	canary := committed(t)
	require.NotEqual(t, canary.CaseIdentity, fixture.Case)
	before := proto.CloneOf(fixture.Run)

	then := *canary
	then.CaseIdentity = fixture.Case
	recorded, err := recordedrun.Encode(fixture.Case, fixture.Driver, fixture.Run)
	require.NoError(t, err)
	subject, err := evaluation.Admit(prior, recorded, fixture.Driver.Catalog)
	require.NoError(t, err)
	require.Equal(t, fixture.Driver, subject.Driver)
	require.Equal(t, fixture.Case, subject.CaseIdentity)
	profile, err := LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	require.Equal(t, evaluation.DecisionAccepted, evaluation.Assess(subject, *profile, nil).Outcome)

	_, err = admit(prior, &then, fixture.Driver, fixture.Run)
	rejection, ok := evaluation.IsRejection(err)
	require.True(t, ok, "not a rejection: %v", err)
	require.Equal(t, evaluation.ReasonStale, rejection.Reason)

	for name, caseBytes := range map[string][]byte{"the prior Case": prior, "the pinned Case": nil} {
		t.Run(name, func(t *testing.T) {
			var crossed *evaluation.Subject
			var err error
			if caseBytes == nil {
				crossed, err = Admit(canary, fixture.Driver, fixture.Run)
			} else {
				crossed, err = admit(caseBytes, canary, fixture.Driver, fixture.Run)
			}
			require.Nil(t, crossed)
			rejection, ok := evaluation.IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, evaluation.ReasonCrossed, rejection.Reason, rejection.Detail)
		})
	}
	require.True(t, proto.Equal(before, fixture.Run), "admission never changes the Run")
}

func TestThePriorNexusWorkflowRecordRetainsItsCaseAndVerdict(t *testing.T) {
	prior, err := os.ReadFile(filepath.Join("testdata", "nexus-caller-syncCompletion-historical-case.json"))
	require.NoError(t, err)
	fixture := recordedIn(t, "nexus-caller-syncCompletion-historical-run.json")
	identity, err := recordedrun.CaseIdentity(prior)
	require.NoError(t, err)
	require.Equal(t, identity, fixture.Case)
	canary := committed(t)
	require.NotEqual(t, canary.CaseIdentity, fixture.Case)
	encoded, err := recordedrun.Encode(fixture.Case, fixture.Driver, fixture.Run)
	require.NoError(t, err)
	subject, err := evaluation.Admit(prior, encoded, fixture.Driver.Catalog)
	require.NoError(t, err)
	profile, err := LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	require.Equal(t, evaluation.DecisionAccepted, evaluation.Assess(subject, *profile, nil).Outcome)
	_, err = Admit(canary, fixture.Driver, fixture.Run)
	rejection, ok := evaluation.IsRejection(err)
	require.True(t, ok, "not a rejection: %v", err)
	require.Equal(t, evaluation.ReasonCrossed, rejection.Reason, rejection.Detail)
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
