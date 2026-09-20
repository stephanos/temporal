package verification

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	historypb "go.temporal.io/api/history/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// The shipped worker-outage Case. The agreement below has to hold for the Contract the live Run is
// judged by, and the two things it needs sit on opposite sides of one boundary: the generator
// publishes that Contract to the functional fixture tree, and `PreparedContract.Evaluate` is
// internal to this package, so no test outside it can call the offline path. The read crosses once,
// here, rather than a second copy of the fixture being checked in to avoid it.
const workerOutageFixture = "../../../../../tests/testcore/testpilot/testdata/workerOutageTests-survived-case.json"

// TestWorkerOutageContractAgreesOnlineAndOffline replays one recorded outage Run through the
// shipped Contract twice: once event by event as the runtime does, and once offline over the whole
// recorded Run. Both answers must be the same Verdict, because the event-count deadline reaches its
// counter through the one helper both paths call. The last case is the outage that never ended: it
// expires on the count, and it expires identically either way.
func TestWorkerOutageContractAgreesOnlineAndOffline(t *testing.T) {
	contract, view := workerOutageFixtureContract(t)

	satisfied := workerOutageRun(t, true)
	live, offline := nexusEvaluateLiveAndOffline(t, contract, view, satisfied)
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, live.GetStatus())
	require.True(t, proto.Equal(live, offline))
	require.Len(t, live.GetRules(), 1)
	require.Equal(t, "worker-outage-order", live.GetRules()[0].GetRuleId())
	require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_SATISFIED, live.GetRules()[0].GetStatus())

	// A resume that never arrives leaves the rule counting, and the count is what ends it.
	expired := workerOutageRun(t, false)
	live, offline = nexusEvaluateLiveAndOffline(t, contract, view, expired)
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, live.GetStatus())
	require.True(t, proto.Equal(live, offline))
	require.Equal(t, "expired", live.GetRules()[0].GetTerminalStateId())
}

// workerOutageFixtureContract prepares the shipped Contract's monitor rules against a minimal
// Program. The Case's Contract also carries the correlated capability the Model's completion clause
// lowered into, which reads the correlated-evidence Observation the live Program lifts; what is
// under test here is the derived outage-order rule's event-count deadline, so the rules are prepared
// alone and the capability stays with the live Run.
func workerOutageFixtureContract(t testing.TB) (*PreparedContract, execution.ProgramView) {
	t.Helper()
	encoded, err := os.ReadFile(workerOutageFixture)
	require.NoError(t, err)
	artifact := &testpilotspb.Case{}
	require.NoError(t, protojson.Unmarshal(encoded, artifact))
	// A moved or renamed fixture fails here rather than silently evaluating a different Contract.
	require.Equal(t, "temporal.case.workerOutageTests.survived", artifact.GetCaseId())
	contract := &testpilotspb.Contract{ContractId: artifact.GetContract().GetContractId(), Rules: artifact.GetContract().GetRules()}
	require.Len(t, contract.GetRules(), 1)

	catalog, err := ir.NewCatalog(nexusDescriptorClosure(historypb.File_temporal_api_history_v1_message_proto))
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{
		MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32,
		MaxRunEvents: 512, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096,
		MaxResponseBytes: 8192, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000,
		MaxInstructionEmittedEvents: 64, MaxInstructionResponseBytes: 8192,
	}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Contract: &testpilotspb.Contract{ContractId: "contract"}, Program: &testpilotspb.Program{
		ProgramId:    "program",
		Observations: []*testpilotspb.Observation{{ObservationId: "history-event", Type: nexusMessageType("temporal.api.history.v1.HistoryEvent")}},
		Entrypoints:  []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}},
		Cleanup:      &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}}
	program, err := execution.Prepare(source, catalog, execution.Profile{Identity: "profile", CatalogIdentity: catalog.Identity(), Limits: limits})
	require.NoError(t, err)
	ceiling := &testpilotspb.ContractLimits{MaxRules: 4, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 12, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 4, MaxCaptureBytes: 8192}
	prepared, err := Prepare(contract, catalog, program.View(), ceiling, nil)
	require.NoError(t, err)
	return prepared, program.View()
}

// workerOutageRun records the shape the live Run has: the stop, the start, the resume when the
// outage ended, and the closing history event the resumed worker produced.
func workerOutageRun(t testing.TB, resumed bool) *testpilotspb.Run {
	t.Helper()
	fault := func(sequence int64, kind testpilotspb.FaultKind, instruction string) *testpilotspb.RunEvent {
		return &testpilotspb.RunEvent{
			Sequence: sequence, ElapsedMilliseconds: sequence * 10,
			Kind:        testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED,
			Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", InstructionId: instruction, ActivationId: "controller-1", Attempt: 1},
			Payload:     &testpilotspb.RunEvent_FaultInjected{FaultInjected: &testpilotspb.FaultInjected{RoleId: "temporal.task-queue", Kind: kind}},
		}
	}
	completed := &historypb.HistoryEvent{
		EventId:    9,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{WorkflowTaskCompletedEventId: 8}},
	}
	value, err := anypb.New(completed)
	require.NoError(t, err)

	events := []*testpilotspb.RunEvent{
		{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED},
		fault(2, testpilotspb.FAULT_KIND_WORKER_STOP, "stop-worker"),
		{Sequence: 3, ElapsedMilliseconds: 30, Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", InstructionId: "start-workflow", ActivationId: "controller-1", Attempt: 1}},
	}
	if resumed {
		events = append(events, fault(4, testpilotspb.FAULT_KIND_WORKER_RESUME, "resume-worker"))
	} else {
		// The outage never ends, so the rule keeps counting past its declared bound.
		for sequence := int64(4); sequence <= 24; sequence++ {
			events = append(events, &testpilotspb.RunEvent{Sequence: sequence, ElapsedMilliseconds: sequence * 10, Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC})
		}
	}
	next := events[len(events)-1].GetSequence() + 1
	events = append(events, &testpilotspb.RunEvent{
		Sequence: next, ElapsedMilliseconds: next * 10, Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
		Coordinates:  &testpilotspb.RunEventCoordinates{EntrypointId: "controller", InstructionId: "history", ActivationId: "controller-1", Attempt: 1},
		Observations: []*testpilotspb.ObservationResult{{ObservationId: "history-event", Value: &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: value}}}},
	})
	closure := next + 1
	events = append(events, &testpilotspb.RunEvent{Sequence: closure, ElapsedMilliseconds: closure * 10, Kind: testpilotspb.RUN_EVENT_KIND_RUN_CLOSED})

	status := testpilotspb.RUN_DISPOSITION_COMPLETED
	if !resumed {
		status = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
	}
	return &testpilotspb.Run{RunId: "run", CaseId: "temporal.case.workerOutageTests.survived", ProgramId: "program", Disposition: status, Events: events}
}
