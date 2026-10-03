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
	controlKey      = "temporal.nexus.control.property.forgedSuccess.fact-nexusOperationCompleted@correlated.violated[temporal.nexus.caller.evidence.failed]"
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

	// The core reads the scheduled, started and failed evidence. The inspected prefix and
	// completion controller are scaffolding, as are the history events outside that core.
	core := EvidenceCore(subject.Verdict)
	require.Equal(t, []int64{8, 27, 32}, core)
	outside := map[int64]string{}
	for _, event := range OutsideCore(subject.Run, core) {
		require.NotContains(t, core, event.Sequence)
		outside[event.Sequence] = event.InstructionID
	}
	require.Equal(t, map[int64]string{
		3: "start-workflow", 4: "start-workflow",
		5: "await-scheduled", 7: "await-scheduled",
		9: "inspect-workflow", 10: "inspect-workflow",
		11: "inspect-workflow-2", 12: "inspect-workflow-2",
		13: "await-completion-authority", 14: "await-completion-authority",
		15: "fail-nexus-operation", 16: "fail-nexus-operation",
		17: "await-close", 19: "await-close",
		20: "history", 21: "history", 22: "history", 23: "history", 24: "history", 25: "history", 26: "history",
		28: "history", 29: "history", 30: "history", 31: "history",
		33: "history", 34: "history", 35: "history", 36: "history",
	}, outside)
}
