package execution

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

// Each Case carries two defects that admission checks at different points, so the first rejection
// pins which check runs first. The corpus carries one defect per entry and cannot tell the order
// apart: the opcode's context and capability before its bounds, the bounds before the opcode's own
// binding, every instruction's binding before any dataflow, and a node's guard before its opcode's
// dataflow.
func TestPrepareRejectsTheFirstOfTwoDefects(t *testing.T) {
	overAttempts := func(n *testpilotspb.InstructionNode) {
		n.Limits.Attempts = &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 33}
	}
	missingSlot := present(slot("missing"))
	undeclared := ir.Error{Category: ir.Unknown, Path: "expression", Detail: "reference is not declared in this environment"}
	const (
		unsupported = "unsupported instruction context or Driver capability"
		overBounds  = "instruction bounds exceed Profile ceilings"
	)
	for _, tc := range []struct {
		name     string
		fixture  func(*testing.T) (*testpilotspb.Case, *ir.Catalog, Profile)
		mutate   func(*testpilotspb.Case, *Profile)
		expected ir.Error
	}{
		{"missing capability before bounds", handleFixture, func(c *testpilotspb.Case, p *Profile) {
			p.Opcodes = slices.DeleteFunc(p.Opcodes, func(opcode contract.Opcode) bool { return opcode == contract.InvokeRPC })
			overAttempts(c.Program.Entrypoints[0].Instructions[0])
		}, ir.Error{Category: ir.Unsupported, Path: "controller.call", Detail: "instruction invoke_rpc the Profile does not authorize"}},
		{"missing instruction before bounds", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction = &testpilotspb.Instruction{}
			overAttempts(c.Program.Entrypoints[0].Instructions[0])
		}, ir.Error{Category: ir.Unsupported, Path: "controller.call", Detail: unsupported}},
		{"wrong context before bounds and binding", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction = c.Program.Entrypoints[1].Instructions[1].Instruction
			overAttempts(c.Program.Entrypoints[0].Instructions[0])
		}, ir.Error{Category: ir.Unsupported, Path: "controller.call", Detail: unsupported}},
		{"bounds before RPC binding", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method = "/example.Service/Missing"
			overAttempts(c.Program.Entrypoints[0].Instructions[0])
		}, ir.Error{Category: ir.LimitExceeded, Path: "controller.call", Detail: overBounds}},
		{"bounds before AwaitSlot binding", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[1].Instruction.GetAwaitSlot().SlotId = "missing"
			overAttempts(c.Program.Entrypoints[0].Instructions[1])
		}, ir.Error{Category: ir.LimitExceeded, Path: "controller.ready", Detail: overBounds}},
		{"bounds before completion binding", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[2].Instruction.GetNexusOperationCompletion().HandleSlotId = "missing"
			overAttempts(c.Program.Entrypoints[0].Instructions[2])
		}, ir.Error{Category: ir.LimitExceeded, Path: "controller.complete", Detail: overBounds}},
		{"bounds before workflow command binding", handleFixture, func(c *testpilotspb.Case, p *Profile) {
			p.CommandTypes = nil
			overAttempts(c.Program.Entrypoints[1].Instructions[0])
		}, ir.Error{Category: ir.LimitExceeded, Path: "workflow.start", Detail: overBounds}},
		{"bounds before Await binding", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[1].Instructions[1].Instruction.GetAwaitInstruction().Instruction.InstructionId = "missing"
			overAttempts(c.Program.Entrypoints[1].Instructions[1])
		}, ir.Error{Category: ir.LimitExceeded, Path: "workflow.await", Detail: overBounds}},
		{"bounds before fault binding", handleFixture, func(c *testpilotspb.Case, p *Profile) {
			p.Opcodes = append(p.Opcodes, contract.InjectFault)
			c.Program.Entrypoints[0].Instructions[0] = faultNode("call", "endpoint", testpilotspb.FAULT_KIND_WORKER_STOP)
			overAttempts(c.Program.Entrypoints[0].Instructions[0])
		}, ir.Error{Category: ir.LimitExceeded, Path: "controller.call", Detail: overBounds}},
		{"bounds before reply binding", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", &testpilotspb.NexusHandlerReply{})
			overAttempts(c.Program.Entrypoints[2].Instructions[0])
		}, ir.Error{Category: ir.LimitExceeded, Path: "handler.respond", Detail: overBounds}},
		{"bounds before ReadEvidence binding", evidenceFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[2].Instruction.GetReadEvidence().EvidenceId = "missing"
			overAttempts(c.Program.Entrypoints[0].Instructions[2])
		}, ir.Error{Category: ir.LimitExceeded, Path: "controller.pending-attempts", Detail: overBounds}},
		{"a later instruction's binding before an earlier node's dataflow", handleFixture, func(c *testpilotspb.Case, p *Profile) {
			c.Program.Entrypoints[0].Instructions[2].Guard = alwaysRuns()
			p.CommandTypes = nil
		}, ir.Error{Category: ir.Unsupported, Path: "program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.command_type", Detail: "command type COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION the Profile does not admit"}},
		{"guard before RPC assignments", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			call := c.Program.Entrypoints[0].Instructions[0]
			call.Guard = missingSlot
			call.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "missing", Value: textLiteral("value")}}
		}, undeclared},
		{"guard before AwaitSlot writer", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints = c.Program.Entrypoints[:2]
			c.Program.Entrypoints[0].Instructions[1].Guard = missingSlot
		}, undeclared},
		{"guard before completion readiness", handleFixture, func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[2].Guard = missingSlot
		}, undeclared},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, p := tc.fixture(t)
			tc.mutate(c, &p)
			_, err := Prepare(c, catalog, p)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.expected, *diagnostic)
		})
	}
}
