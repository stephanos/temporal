package replay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
)

// The negative control's fixture is rendered by `umpire-gen-case-runtime-conformance`; its
// recorded Run under testdata is one live Run of it against the test cluster, captured by
// `TestTestpilotNexusControlForgedCompletionIsViolated` with UMPIRE_CONTROL_RECORD naming the file.
// The record names the fixture's canonical bytes, so any change to the control's definitions makes
// it crossed until it is recorded again live.
const (
	controlCasePath = "../../../tests/testcore/testpilot/testdata/nexusCallerControl-forgedCompletion-case.json"
	controlRunPath  = "testdata/nexusCallerControl-forgedCompletion-run.json"
	controlProfile  = "nexusCallerControl-forgedCompletion-profile"
	controlKey      = "temporal.nexus.control.property.forgedSuccess.fact-nexusOperationCompleted@correlated.violated[temporal.nexus.caller.evidence.failed];" +
		"temporal.nexus.control.property.forgedSuccess.state-succeeded@correlated.violated[temporal.nexus.caller.evidence.failed]"
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
// offline to its recorded Verdict, and its key names the control's two violated rules, the
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
	require.Len(t, subject.Replay.Violations, 2)
	require.Equal(t, controlKey, subject.Key.String())

	// The core is the two events the correlated rules read, the scheduled event's lift and the
	// history read that carries the failed event. Every other instruction event is scaffolding
	// the core omits: the workflow's start, the wait for the close, and the other events the
	// same two instructions lifted, named by instruction id.
	core := EvidenceCore(subject.Verdict)
	require.Equal(t, []int64{9, 19}, core)
	outside := map[int64]string{}
	for _, event := range OutsideCore(subject.Run, core) {
		require.NotContains(t, core, event.Sequence)
		outside[event.Sequence] = event.InstructionID
	}
	require.Equal(t, map[int64]string{
		3: "start-workflow", 4: "start-workflow",
		5: "await-scheduled", 8: "await-scheduled",
		10: "await-close", 11: "await-close",
		12: "history", 13: "history", 14: "history", 15: "history", 16: "history", 17: "history", 18: "history",
		20: "history", 21: "history", 22: "history", 23: "history",
	}, outside)
}
