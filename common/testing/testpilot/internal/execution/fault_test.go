package execution

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func faultNode(id, role string, kind testpilotspb.FaultKind) *testpilotspb.InstructionDefinition {
	return &testpilotspb.InstructionDefinition{
		InstructionId: id,
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: role, Kind: kind}}},
		Outcome:       statusSchema(),
		Limits:        &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 8},
	}
}

func faultFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Policy) {
	t.Helper()
	c, catalog, policy := fixture(t)
	policy.Capabilities = append(policy.Capabilities, InjectFault)
	addWorker(c, &policy)
	c.Program.Entrypoints[0].Instructions = []*testpilotspb.InstructionDefinition{
		faultNode("stop", "queue", testpilotspb.FAULT_KIND_WORKER_STOP),
		faultNode("resume", "queue", testpilotspb.FAULT_KIND_WORKER_RESUME),
	}
	c.Program.Entrypoints[0].Instructions[1].Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "stop"}}
	return c, catalog, policy
}

// The Opcode list, the instruction-to-opcode switch and the Instruction oneof are three
// hand-maintained lists. Pinning them to each other is what stops a new instruction from landing
// in only one of them; Capability alignment is asserted from the facade package, which owns it.
func TestInstructionOpcodesCoverTheInstructionTable(t *testing.T) {
	table := []*testpilotspb.Instruction{
		{Instruction: &testpilotspb.Instruction_InvokeRpc{}},
		{Instruction: &testpilotspb.Instruction_AwaitSlot{}},
		{Instruction: &testpilotspb.Instruction_CompleteNexusOperation{}},
		{Instruction: &testpilotspb.Instruction_StartNexusOperation{}},
		{Instruction: &testpilotspb.Instruction_AwaitOutcome{}},
		{Instruction: &testpilotspb.Instruction_Finish{}},
		{Instruction: &testpilotspb.Instruction_RespondNexus{}},
		{Instruction: &testpilotspb.Instruction_InjectFault{}},
	}
	oneof := (&testpilotspb.Instruction{}).ProtoReflect().Descriptor().Oneofs().ByName("instruction")
	require.NotNil(t, oneof)
	require.Equal(t, oneof.Fields().Len(), len(table))

	seen := map[Opcode]bool{}
	for i, instruction := range table {
		opcode := instructionOpcode(instruction)
		require.Equal(t, Opcode(i+1), opcode)
		require.False(t, seen[opcode])
		seen[opcode] = true
		require.NotEqual(t, testpilotspb.ENTRYPOINT_KIND_UNSPECIFIED, opcodeContext(opcode))
	}
	require.Equal(t, InjectFault, Opcode(len(table)))
}

func TestPrepareAdmitsFaultInjection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mutate   func(*testpilotspb.Case, *Policy)
		category ir.ErrorCategory
	}{
		{"admitted", func(*testpilotspb.Case, *Policy) {}, ""},
		{"missing capability", func(_ *testpilotspb.Case, p *Policy) {
			p.Capabilities = p.Capabilities[:len(p.Capabilities)-1]
		}, ir.Unsupported},
		{"undeclared role", func(c *testpilotspb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().RoleId = "missing"
		}, ir.Malformed},
		{"non task queue role", func(c *testpilotspb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().RoleId = "worker"
		}, ir.Malformed},
		{"unknown fault kind", func(c *testpilotspb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_UNSPECIFIED
		}, ir.Malformed},
		{"outside the controller context", func(c *testpilotspb.Case, _ *Policy) {
			workflow := c.Program.Entrypoints[len(c.Program.Entrypoints)-1]
			workflow.Instructions = []*testpilotspb.InstructionDefinition{faultNode("stop", "queue", testpilotspb.FAULT_KIND_WORKER_STOP)}
		}, ir.Unsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, policy := faultFixture(t)
			tc.mutate(c, &policy)
			_, err := Prepare(c, catalog, policy)
			if tc.category == "" {
				require.NoError(t, err)
				return
			}
			var admissionErr *ir.Error
			require.ErrorAs(t, err, &admissionErr)
			require.Equal(t, tc.category, admissionErr.Category)
		})
	}
}

// One realized fault is one recorded fact. The event has to survive the recorder's own append
// validation, which is what proves the new kind is admitted end to end rather than only produced.
func TestSchedulerRecordsOneFaultEventPerInstruction(t *testing.T) {
	c, catalog, policy := faultFixture(t)
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	var dispatched []string
	host := &schedulerHost{fault: func(_ context.Context, _ Coordinate, roleID string, kind testpilotspb.FaultKind) (EffectHandle, error) {
		dispatched = append(dispatched, roleID+"/"+kind.String())
		return &schedulerEffect{wait: func(context.Context) (EffectResult, error) {
			return EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
		}}, nil
	}}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()

	require.Equal(t, []string{"queue/WorkerStop", "queue/WorkerResume"}, dispatched)
	var recorded []*testpilotspb.FaultInjected
	for _, event := range s.recorder.run.Events {
		if event.Kind == testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED {
			recorded = append(recorded, event.GetFaultInjected())
		}
	}
	require.Len(t, recorded, 2)
	require.True(t, proto.Equal(&testpilotspb.FaultInjected{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_STOP}, recorded[0]))
	require.True(t, proto.Equal(&testpilotspb.FaultInjected{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_RESUME}, recorded[1]))
}
