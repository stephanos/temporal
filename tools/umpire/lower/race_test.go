package lower

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
)

// The held race lowers to a Case (Realization.scala, heldDelivery; admission/Model.scala, heldAdmission): the
// controller starts the activity, holds what its dispatch sent to the task queue, pauses it, reads
// the pause back and releases the stale message, and no worker runs. The hold and the release are the
// Driver's delivery controls of the control's task-queue role. The Case carries the dispatch, the
// pause and the rejection as the path's evidence, the rejection as the release's own record under the
// committed decision, keyed by the activity the record names; and the admission the path does not
// take, since that kind is exhaustive, so a Run that records one is read. The control is accounted
// for by the two instructions that realize it. A Profile whose environment supplies no delivery
// control refuses the Case, naming the hold.
func TestTheHeldRaceLowers(t *testing.T) {
	p, err := NewProducer(loaded(t, "activity-race"))
	require.NoError(t, err)
	l, err := p.Lower("heldAdmission.staleDelivery", cp.IdentityFor("temporal.case", "standaloneActivityRace", "heldAdmission.staleDelivery"))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing, "%v", l.Unsupported)
	require.Empty(t, l.OffPath)
	require.Equal(t, map[string][]string{"controller": {"start-activity", "hold-dispatch", "pause-activity", "await-paused", "release-dispatch"}},
		instructionIDs(l.Case))
	fault := func(kind testpilotspb.FaultKind) *testpilotspb.Instruction {
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{
			RoleId: "temporal.task-queue", Kind: kind}}}
	}
	protorequire.ProtoEqual(t, fault(testpilotspb.FAULT_KIND_DELIVERY_HOLD), instruction(t, l.Case, "controller", "hold-dispatch").GetInstruction())
	protorequire.ProtoEqual(t, fault(testpilotspb.FAULT_KIND_DELIVERY_RELEASE), instruction(t, l.Case, "controller", "release-dispatch").GetInstruction())

	type source struct {
		kind        testpilotspb.RunEventKind
		instruction string
		operation   string
		keyedByRun  bool
	}
	carried := map[string]source{}
	for _, e := range l.Case.GetProgram().GetEvidence() {
		record := e.GetRunEvent()
		carried[e.GetEvidenceId()] = source{record.GetKind(), record.GetInstruction().GetInstructionId(), e.GetOperation(), record.GetRunKeyed()}
	}
	completed := testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED
	require.Equal(t, map[string]source{
		"evidence.dispatchSent":      {completed, "hold-dispatch", "", true},
		"evidence.statusPaused":      {0, "", "activity_id", false},
		"evidence.admissionRejected": {completed, "release-dispatch", "delivery_admission.activity_id", false},
		"evidence.attemptAdmitted":   {completed, "release-dispatch", "delivery_admission.activity_id", false},
	}, carried)

	var control []Entry
	for _, entry := range l.Inventory {
		if entry.Kind == "controls" {
			control = append(control, Entry{Kind: entry.Kind, ID: entry.ID, Disposition: entry.Disposition, As: entry.As})
		}
	}
	require.Equal(t, []Entry{{Kind: "controls", ID: "hold-dispatch", Disposition: InCase, As: []string{
		"program.entrypoints[controller].instructions[hold-dispatch]", "program.entrypoints[controller].instructions[release-dispatch]"}}}, control)

	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := temporal.Environment{Identity: "race", Namespace: "namespace", TaskQueue: "task-queue"}
	uncontrolled, err := temporal.DeriveProfile(l.Case, catalog, environment)
	require.NoError(t, err)
	_, err = testpilot.Prepare(l.Case, uncontrolled)
	require.ErrorContains(t, err, "unsupported at controller.hold-dispatch")
	environment.DeliveryControl = true
	controlled, err := temporal.DeriveProfile(l.Case, catalog, environment)
	require.NoError(t, err)
	_, err = testpilot.Prepare(l.Case, controlled)
	require.NoError(t, err)
}
