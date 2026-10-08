package execution

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
)

func TestPrepareRefusesNonlinearActivityAttempts(t *testing.T) {
	source, catalog, policy := activityFixture(t)
	retried(source, &policy)
	policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 2
	source.Program.Entrypoints[1].Instructions[1].After = &testpilotspb.After{}
	_, err := Prepare(source, catalog, policy)
	require.Error(t, err, "two unordered dispositions cannot denote successive SDK attempts")
}

func heartbeatNode(id string) *testpilotspb.InstructionNode {
	return activityNode(id, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityHeartbeat{ActivityHeartbeat: &testpilotspb.ActivityHeartbeat{Details: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("beat")}}}}}})
}

func TestPrepareGroupsHeartbeatWithItsActivityDisposition(t *testing.T) {
	source, catalog, policy := activityFixture(t)
	retried(source, &policy)
	policy.Opcodes = append(policy.Opcodes, contract.ActivityHeartbeat)
	policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 2
	entry := source.Program.Entrypoints[1]
	entry.Instructions = append([]*testpilotspb.InstructionNode{heartbeatNode("heartbeat")}, entry.Instructions...)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	groups := prepared.Entrypoints()[1].ActivityAttempts()
	require.Equal(t, [][]int{{0, 1}, {2}}, groups)
	plan, ok := prepared.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, int64(2), plan.Reservations[0].Count)
	groups[0][0] = 99
	groups[1] = nil
	entry.Instructions[0].Instruction.GetActivityHeartbeat().Details.Payloads[0].Data[0] = 'X'
	require.Equal(t, [][]int{{0, 1}, {2}}, prepared.Entrypoints()[1].ActivityAttempts())
	require.Equal(t, []byte("beat"), prepared.Entrypoints()[1].Instructions()[0].Source().GetInstruction().GetActivityHeartbeat().GetDetails().Payloads[0].Data)
}

func TestPrepareRefusesUnsupportedActivityHeartbeatGroups(t *testing.T) {
	for name, modify := range map[string]func(*testpilotspb.Case, *Profile){
		"missing terminal": func(source *testpilotspb.Case, _ *Profile) {
			source.Program.Entrypoints[1].Instructions = source.Program.Entrypoints[1].Instructions[:1]
		},
		"multiple prefixes": func(source *testpilotspb.Case, _ *Profile) {
			entry := source.Program.Entrypoints[1]
			entry.Instructions = append([]*testpilotspb.InstructionNode{heartbeatNode("another-heartbeat")}, entry.Instructions...)
		},
		"wrong context": func(source *testpilotspb.Case, _ *Profile) {
			source.Program.Entrypoints[0].Instructions[0].Instruction = source.Program.Entrypoints[1].Instructions[0].Instruction
		},
		"unauthorized opcode": func(_ *testpilotspb.Case, policy *Profile) { policy.Opcodes = policy.Opcodes[:len(policy.Opcodes)-1] },
		"unknown mode": func(source *testpilotspb.Case, policy *Profile) {
			policy.Opcodes = append(policy.Opcodes, contract.ActivityAttemptWithholding)
			source.Program.Entrypoints[1].Instructions[1].Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: 99}}}
		},
		"excessive details": func(source *testpilotspb.Case, policy *Profile) {
			source.Program.Entrypoints[1].Instructions[0].Instruction.GetActivityHeartbeat().Details.Payloads[0].Data = []byte(strings.Repeat("x", int(policy.Limits.MaxRequestBytes)+1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			policy.Opcodes = append(policy.Opcodes, contract.ActivityHeartbeat)
			entry := source.Program.Entrypoints[1]
			entry.Instructions = append([]*testpilotspb.InstructionNode{heartbeatNode("heartbeat")}, entry.Instructions...)
			modify(source, &policy)
			_, err := Prepare(source, catalog, policy)
			require.Error(t, err)
		})
	}
}
