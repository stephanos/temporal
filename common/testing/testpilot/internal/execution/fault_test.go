package execution

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func faultNode(id, role string, kind testpilotspb.FaultKind) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{
		InstructionId: id,
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: role, Kind: kind}}},
		Limits:        &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 1000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}},
	}
}

func faultFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	c, catalog, policy := fixture(t)
	policy.Opcodes = append(policy.Opcodes, contract.InjectFault)
	addWorker(c, &policy)
	c.Program.Entrypoints[0].Instructions = []*testpilotspb.InstructionNode{
		faultNode("stop", "queue", testpilotspb.FAULT_KIND_WORKER_STOP),
		faultNode("resume", "queue", testpilotspb.FAULT_KIND_WORKER_RESUME),
	}
	c.Program.Entrypoints[0].Instructions[1].Guard = alwaysRuns()
	return c, catalog, policy
}

// The Opcode list, the opcodes table and the Instruction oneof are three hand-maintained lists.
// Pinning them to each other is what stops a new instruction from landing in only one of them; the
// facade re-exports the contract leaf's Opcode by alias, so it adds no fourth list to pin. A removed
// arm's successors move up, so an arm's Opcode is its field number and the numbers stay dense from 1.
func TestInstructionOpcodesCoverTheInstructionTable(t *testing.T) {
	oneof := (&testpilotspb.Instruction{}).ProtoReflect().Descriptor().Oneofs().ByName("instruction")
	require.NotNil(t, oneof)
	require.Equal(t, int(contract.MaxOpcode), oneof.Fields().Len())

	seen := map[contract.Opcode]bool{}
	var previous protoreflect.FieldNumber
	for i := range oneof.Fields().Len() {
		field := oneof.Fields().Get(i)
		t.Run(string(field.Name()), func(t *testing.T) {
			instruction := &testpilotspb.Instruction{}
			instruction.ProtoReflect().Mutable(field)
			opcode := InstructionOpcode(instruction)
			// The arm's field number is its opcode, and the numbers are dense from 1: the two
			// lists cannot be reordered apart, and a removed arm's successors move up.
			require.Equal(t, previous+1, field.Number())
			require.Equal(t, contract.Opcode(field.Number()), opcode)
			previous = field.Number()
			require.False(t, seen[opcode])
			seen[opcode] = true
			require.Equal(t, field.Name(), opcodes[opcode].arm)
			require.NotZero(t, opcodes[opcode].context)
			require.NotNil(t, opcodes[opcode].bind)
		})
	}
}

func TestPrepareAdmitsFaultInjection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mutate   func(*testpilotspb.Case, *Profile)
		category ir.ErrorCategory
	}{
		{"admitted", func(*testpilotspb.Case, *Profile) {}, ""},
		// Whether a Driver can hold a delivery is the Driver's to answer at Validate, as a worker
		// outage is: admission takes every declared kind.
		{"delivery control", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_DELIVERY_HOLD
			c.Program.Entrypoints[0].Instructions[1].Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_DELIVERY_RELEASE
		}, ""},
		{"undeclared fault kind", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_DELIVERY_RELEASE + 1
		}, ir.Unknown},
		{"missing opcode", func(_ *testpilotspb.Case, p *Profile) {
			p.Opcodes = p.Opcodes[:len(p.Opcodes)-1]
		}, ir.Unsupported},
		{"undeclared role", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().RoleId = "missing"
		}, ir.Malformed},
		{"non task queue role", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().RoleId = "worker"
		}, ir.Malformed},
		{"unknown fault kind", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_UNSPECIFIED
		}, ir.Malformed},
		{"outside the controller context", func(c *testpilotspb.Case, _ *Profile) {
			workflow := c.Program.Entrypoints[len(c.Program.Entrypoints)-1]
			workflow.Instructions = []*testpilotspb.InstructionNode{faultNode("stop", "queue", testpilotspb.FAULT_KIND_WORKER_STOP)}
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
	for _, tc := range []struct {
		name   string
		status testpilotspb.InstructionOutcomeStatus
		want   []*testpilotspb.FaultInjected
	}{
		{"realized", testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, []*testpilotspb.FaultInjected{
			{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_STOP},
			{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_RESUME},
		}},
		// A Driver that reported the outage as not realized has produced intent, not evidence.
		{"reported unrealized", testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, policy := faultFixture(t)
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			var dispatched []string
			host := &testsupport.Session{OnInjectFault: func(_ context.Context, _ contract.Coordinate, roleID string, kind testpilotspb.FaultKind) (contract.EffectHandle, error) {
				dispatched = append(dispatched, roleID+"/"+kind.String())
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: tc.status}}, nil
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
			require.Len(t, recorded, len(tc.want))
			for i, want := range tc.want {
				require.True(t, proto.Equal(want, recorded[i]))
			}
		})
	}
}

// What admission committed for a released delivery is the outcome of the release that delivered it
// and of nothing else: a release that succeeded without it, and any other instruction that reports
// one, are malformed outcomes, never evidence.
func TestDeliveryAdmissionIsCarriedOnlyByASuccessfulRelease(t *testing.T) {
	admission := &testpilotspb.DeliveryAdmission{ActivityId: "activity", ActivityRunId: "run", DeliveryId: "7",
		Decision: testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED}
	for _, tc := range []struct {
		name     string
		outcome  func(testpilotspb.FaultKind) *testpilotspb.InstructionOutcome
		recorded *testpilotspb.DeliveryAdmission
	}{
		{"release carries the decision", func(kind testpilotspb.FaultKind) *testpilotspb.InstructionOutcome {
			if kind == testpilotspb.FAULT_KIND_DELIVERY_RELEASE {
				return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, DeliveryAdmission: admission}
			}
			return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
		}, admission},
		{"release without a decision", func(testpilotspb.FaultKind) *testpilotspb.InstructionOutcome {
			return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
		}, nil},
		{"hold with a decision", func(testpilotspb.FaultKind) *testpilotspb.InstructionOutcome {
			return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, DeliveryAdmission: admission}
		}, nil},
		{"undecided release with a decision", func(kind testpilotspb.FaultKind) *testpilotspb.InstructionOutcome {
			if kind == testpilotspb.FAULT_KIND_DELIVERY_RELEASE {
				return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, DeliveryAdmission: admission}
			}
			return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
		}, nil},
		{"unspecified decision", func(kind testpilotspb.FaultKind) *testpilotspb.InstructionOutcome {
			if kind == testpilotspb.FAULT_KIND_DELIVERY_RELEASE {
				return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, DeliveryAdmission: &testpilotspb.DeliveryAdmission{ActivityId: "activity"}}
			}
			return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
		}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, policy := faultFixture(t)
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_DELIVERY_HOLD
			c.Program.Entrypoints[0].Instructions[1].Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_DELIVERY_RELEASE
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			host := &testsupport.Session{OnInjectFault: func(_ context.Context, _ contract.Coordinate, _ string, kind testpilotspb.FaultKind) (contract.EffectHandle, error) {
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: tc.outcome(kind)}, nil
				}}, nil
			}}
			s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			executeErr := s.execute(context.Background())
			s.waits.Wait()
			var completed []*testpilotspb.InstructionOutcome
			for _, event := range s.recorder.run.Events {
				if event.Kind == testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED {
					completed = append(completed, event.GetOutcome())
				}
			}
			if tc.recorded == nil {
				require.Error(t, executeErr)
				for _, outcome := range completed {
					require.Nil(t, outcome.GetDeliveryAdmission())
				}
				return
			}
			require.NoError(t, executeErr)
			require.Len(t, completed, 2)
			require.True(t, proto.Equal(tc.recorded, completed[1].GetDeliveryAdmission()))
		})
	}
}
