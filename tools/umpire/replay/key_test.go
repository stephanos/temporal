package replay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
)

// The negative control's fixture is rendered by `umpire-gen-case-runtime-conformance`; its
// recorded Run under testdata is one live Run of it against the test cluster, captured by
// `TestTestpilotNexusControlForgedCompletionIsViolated` with UMPIRE_CONTROL_RECORD naming the file.
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
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	return func(identity string, source *testpilotspb.Case) (*testpilot.PreparedCase, error) {
		profile, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
			Identity: identity, Namespace: "umpire-control", TaskQueue: "umpire-control-queue",
			HandlerTaskQueue: "umpire-control-queue-handler", NexusEndpoint: "umpire-control-endpoint",
		})
		if err != nil {
			return nil, err
		}
		return testpilot.Prepare(source, profile)
	}
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
	// history read that carries the failed event; the scaffolding around them, from the start
	// of the workflow to the handler's reply, supports no violated rule and is named by
	// instruction id.
	core := EvidenceCore(subject.Verdict)
	require.Len(t, core, 2)
	for _, sequence := range core {
		require.Equal(t, "controller", subject.Run.GetEvents()[sequence-1].GetCoordinates().GetEntrypointId())
	}
	outside := OutsideCore(subject.Run, core)
	require.NotEmpty(t, outside)
	instructions := map[string]bool{}
	for _, event := range outside {
		require.NotContains(t, core, event.Sequence)
		require.NotEmpty(t, event.InstructionID)
		instructions[event.InstructionID] = true
	}
	require.Greater(t, len(instructions), 1, "more than one scaffolding instruction lies outside the core: %v", instructions)
}
