package replay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
)

// The negative control's Case under testdata is the one its recorded Run beside it names: one live
// Run of it against the test cluster, captured by
// `TestTestpilotNexusControlForgedCompletionIsViolated` with UMPIRE_CONTROL_RECORD naming the file.
// The record names the Case's canonical bytes, and the live test now runs the control lowered from
// the Scala model, so recording it again live (`make umpire-rerecord-pinned-runs`) makes it a Run of
// that Case, which then replaces the one kept here together with the recorded Profile name.
const (
	controlCasePath = "testdata/nexusCallerControl-forgedCompletion-case.json"
	controlRunPath  = "testdata/nexusCallerControl-forgedCompletion-run.json"
	controlProfile  = "nexus-control-forgedCompletion-profile"
	controlKey      = "temporal.features.nexuscaller.system.property.forgedSuccess.fact-nexusOperationCompleted@correlated.violated[temporal.features.nexuscaller.evidence.failed]"
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

// The recorded control Run pins the correlated key: it admits under its recorded identity, replays
// offline to its recorded Verdict, and its key names the control's violated rule, the
// correlated terminal state and the failed event's evidence, all in Definition IDs. When the
// control's fixture or the runtime changes the record, re-record it through the live test.
func TestControlRecordPinsTheCorrelatedKey(t *testing.T) {
	caseBytes, err := os.ReadFile(filepath.Clean(controlCasePath))
	require.NoError(t, err)
	recorded, err := os.ReadFile(controlRunPath)
	require.NoError(t, err)

	subject, err := Admit(t.Context(), caseBytes, recorded, controlPreparer(t))
	require.NoError(t, err)
	require.Equal(t, controlProfile, subject.Driver.Profile)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, subject.Run.GetDisposition())
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, subject.Verdict.GetStatus())
	require.Len(t, subject.Replay.Violations, 1)
	require.Equal(t, controlKey, subject.Key.String())
}
