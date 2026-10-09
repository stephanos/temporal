package replay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/protobuf/proto"
)

// This historical negative-control pair retains the Case, Profile and Run captured by the
// original scoped test-cluster recording. Current generated control recordings are maintained
// separately by `make umpire-rerecord-pinned-runs` and the Model-assessment probe.
const (
	controlCasePath       = "testdata/nexusCallerControl-forgedCompletion-case.json"
	controlRunPath        = "testdata/nexusCallerControl-forgedCompletion-run.json"
	currentControlRunPath = "testdata/nexusCallerControl-forgedCompletion-current-run.json"
	controlProfile        = "nexus-control-forgedCompletion-profile"
	controlKey            = "temporal.features.nexuscaller.system.property.forgedSuccess.fact-nexusOperationCompleted@correlated.violated[temporal.features.nexuscaller.evidence.failed]"
)

// controlPreparer prepares under the names the live control test binds to, so the recorded
// identity's bindings fingerprint is reached again.
func controlPreparer(t testing.TB) Preparer {
	t.Helper()
	return preparerIn(t, testpilotdriver.Environment{
		Namespace: "umpire-control", TaskQueue: "umpire-control-queue",
		HandlerTaskQueue: "umpire-control-queue-handler", NexusEndpoint: "umpire-control-endpoint",
	})
}

// The fresh companion Run pins the correlated key: it admits under its recorded identity, replays
// offline to its recorded Verdict, and its key names the control's violated rule, the
// correlated terminal state and the failed event's evidence, all in historical Definition IDs.
func TestControlRecordPinsTheCorrelatedKey(t *testing.T) {
	caseBytes, err := os.ReadFile(filepath.Clean(controlCasePath))
	require.NoError(t, err)
	recorded, err := os.ReadFile(currentControlRunPath)
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(recorded)
	require.NoError(t, err)

	subject, err := Admit(t.Context(), caseBytes, recorded, controlPreparer(t))
	require.NoError(t, err)
	require.Equal(t, decoded.Driver, subject.Driver)
	require.Equal(t, subject.Prepared.Identity(), subject.Driver)
	require.True(t, proto.Equal(decoded.Run, subject.Run))
	require.True(t, proto.Equal(decoded.Run.GetVerdict(), subject.Verdict))
	require.Equal(t, controlProfile, subject.Driver.Profile)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, subject.Run.GetDisposition())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, subject.Run.GetCleanup().GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, subject.Verdict.GetStatus())
	require.Equal(t, []int64{8, 27, 32}, subject.Verdict.GetSupportingEventSequences())
	require.Len(t, subject.Replay.Violations, 1)
	require.Equal(t, controlKey, subject.Key.String())
}

func TestHistoricalControlRecordIsStaleUnderCurrentDriver(t *testing.T) {
	caseBytes, err := os.ReadFile(controlCasePath)
	require.NoError(t, err)
	recorded, err := os.ReadFile(controlRunPath)
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(recorded)
	require.NoError(t, err)
	require.Equal(t, "3364057f225cf6fd6116023ef36573038a9acd29f0d13df2da98c76062e9c3c0", decoded.Driver.Catalog)

	subject, err := Admit(t.Context(), caseBytes, recorded, controlPreparer(t))
	require.Nil(t, subject)
	rejection, ok := IsRejection(err)
	require.True(t, ok, "not a rejection: %v", err)
	require.Equal(t, ReasonStale, rejection.Reason)
	require.Contains(t, rejection.Detail, decoded.Driver.Catalog)
	require.Contains(t, rejection.Detail, "12fa287d45f6e40502f05d1c89643a59d612ddd53e7fde025dc90abfb7c744f6")
}
