package temporal_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelv1 "github.com/temporalio/umpire/hosts/go/gen/umpire/model/v1"
	realizationv1 "github.com/temporalio/umpire/hosts/go/gen/umpire/realization/v1"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"umpire.local/temporal/reference/worker"
	workersdk "umpire.local/temporal/reference/worker/sdk"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestResetProgramRowsAndScenarioHeader(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for attempts := int32(1); attempts <= 3; attempts++ {
		h := unboundGuest(t)
		empty, err := h.Reset(ctx, &modelv1.ResetRequest{})
		require.NoError(t, err)
		require.Nil(t, empty.GetProgram())
		require.Equal(t, modelv1.ProofContract_PROOF_CONTRACT_WHOLE_PREFIX_V1, empty.GetProofContract())
		captured := new(realizationv1.Observation)
		require.NoError(t, startObservation(t, attempts).UnmarshalTo(captured))
		configured := new(workflowservice.StartActivityExecutionRequest)
		require.NoError(t, captured.GetRequest().UnmarshalTo(configured))
		configured.Input = &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("application input")}}}
		reset, err := h.Reset(ctx, &modelv1.ResetRequest{Configuration: pack(t, configured)})
		require.NoError(t, err)
		require.Equal(t, modelv1.ProofContract_PROOF_CONTRACT_WHOLE_PREFIX_V1, reset.GetProofContract())
		program := new(temporalv1.WorkerProgram)
		require.NoError(t, reset.GetProgram().UnmarshalTo(program))
		require.Equal(t, receiptNamespace, program.GetNamespace())
		require.Equal(t, "queue", program.GetTaskQueue())
		require.Len(t, program.GetRules(), []int{39, 416, 3211}[attempts-1])
		require.Len(t, program.GetContinuationDomains(), []int{12, 128, 988}[attempts-1])
		encoded, err := proto.Marshal(reset.GetProgram())
		require.NoError(t, err)
		table, err := worker.Decode(encoded)
		require.NoError(t, err)
		require.NoError(t, workersdk.Validate(encoded))
		names := programPolicyNames(t, program)
		require.Len(t, names, []int{15, 85, 435}[attempts-1])
		groups := map[string][]string{}
		for _, row := range table.Rows() {
			key := row.Key
			require.Equal(t, "ModelSlice", key.GetActivityType())
			require.GreaterOrEqual(t, key.GetAttempt(), int32(1))
			require.LessOrEqual(t, key.GetAttempt(), attempts)
			require.True(t, proto.Equal(row.Response, table.Match(key.GetScenario(), key.GetAttempt(), key.GetActivityType(), key.GetHeartbeatHistory()...)))
			bits := ""
			for _, bit := range key.GetHeartbeatHistory() {
				if bit {
					bits += "1"
				} else {
					bits += "0"
				}
			}
			choice := ""
			switch {
			case row.Response.MessageIs(new(workflowservice.RespondActivityTaskCompletedRequest)):
				response := new(workflowservice.RespondActivityTaskCompletedRequest)
				require.NoError(t, row.Response.UnmarshalTo(response))
				require.Empty(t, response.GetTaskToken())
				require.Empty(t, response.GetIdentity())
				want := &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete")}}}
				require.True(t, proto.Equal(want, response.GetResult()))
				choice = "c"
			case row.Response.MessageIs(new(workflowservice.RespondActivityTaskFailedRequest)):
				response := new(workflowservice.RespondActivityTaskFailedRequest)
				require.NoError(t, row.Response.UnmarshalTo(response))
				require.Empty(t, response.GetTaskToken())
				require.Empty(t, response.GetIdentity())
				require.Equal(t, "MiniActivityFailure", response.GetFailure().GetApplicationFailureInfo().GetType())
				choice, message := "r", "Retry"
				if response.GetFailure().GetApplicationFailureInfo().GetNonRetryable() {
					choice, message = "f", "Fail"
				}
				require.Equal(t, message, response.GetFailure().GetMessage())
				groups[key.GetScenario()] = append(groups[key.GetScenario()], fmt.Sprintf("%dh%s=%s", key.GetAttempt(), bits, choice))
				continue
			case row.Response.MessageIs(new(workflowservice.RecordActivityTaskHeartbeatRequest)):
				choice = "h"
			case row.Response.MessageIs(new(workflowservice.RespondActivityTaskCanceledRequest)):
				choice = "a"
			default:
				t.Fatalf("unknown response: %v", row.Response)
			}
			groups[key.GetScenario()] = append(groups[key.GetScenario()], fmt.Sprintf("%dh%s=%s", key.GetAttempt(), bits, choice))
		}
		for name, rows := range groups {
			require.Equal(t, "activity-v1_"+strings.Join(rows, "_"), name)
		}
		original, err := proto.Marshal(reset)
		require.NoError(t, err)
		again, err := h.Reset(ctx, &modelv1.ResetRequest{})
		require.NoError(t, err)
		repeated, err := proto.Marshal(again)
		require.NoError(t, err)
		require.True(t, bytes.Equal(original, repeated))
		seed := len(names) - 1
		name := names[seed]
		{
			proposal, err := h.Propose(ctx, &modelv1.ProposeRequest{Seed: uint64(seed)})
			require.NoError(t, err)
			start := new(workflowservice.StartActivityExecutionRequest)
			require.NoError(t, proposal.GetRequest().UnmarshalTo(start))
			tag := start.GetHeader().GetFields()["umpire-scenario"]
			require.Equal(t, name, string(tag.GetData()))
			require.Equal(t, "binary/plain", string(tag.GetMetadata()["encoding"]))
			require.True(t, proto.Equal(start.GetInput(), configured.GetInput()))
		}
	}
}

func programPolicyNames(t *testing.T, program *temporalv1.WorkerProgram) []string {
	t.Helper()
	var names []string
	for _, rule := range program.GetRules() {
		key := new(temporalv1.ActivityTaskKey)
		require.NoError(t, rule.GetKey().UnmarshalTo(key))
		if !slices.Contains(names, key.GetScenario()) {
			names = append(names, key.GetScenario())
		}
	}
	return names
}

func witnessName(label string) string {
	switch label {
	case "Complete":
		return "activity-v1_1h=c"
	case "Fail":
		return "activity-v1_1h=f"
	case "RetryThenComplete":
		return "activity-v1_1h=r_2h=c"
	default:
		panic("unknown mnemonic witness: " + label)
	}
}
