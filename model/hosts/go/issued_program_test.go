package temporal_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/temporal/internal/programfixture"
	"umpire.local/temporal/reference/worker"
	workersdk "umpire.local/temporal/reference/worker/sdk"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestIssuedProgramAdmissionControls(t *testing.T) {
	original := programfixture.Issue(t, receiptNamespace, "queue", "ModelSlice")
	for _, control := range []string{"missing-heartbeat-child", "zero-attempt", "malformed-key", "multiple-payloads"} {
		t.Run(control, func(t *testing.T) {
			envelope := new(anypb.Any)
			require.NoError(t, proto.Unmarshal(original, envelope))
			program := new(temporalv1.WorkerProgram)
			require.NoError(t, envelope.UnmarshalTo(program))
			changed := false
			for i, row := range program.GetRules() {
				key := new(temporalv1.ActivityTaskKey)
				require.NoError(t, row.GetKey().UnmarshalTo(key))
				switch control {
				case "missing-heartbeat-child":
					if len(key.GetHeartbeatHistory()) == 0 {
						continue
					}
					program.Rules = append(program.Rules[:i], program.Rules[i+1:]...)
				case "zero-attempt":
					key.Attempt = 0
					row.Key = pack(t, key)
				case "malformed-key":
					row.Key.Value = []byte{0xff}
				case "multiple-payloads":
					if !row.GetRespond()[0].MessageIs(new(workflowservice.RespondActivityTaskCompletedRequest)) {
						continue
					}
					response := new(workflowservice.RespondActivityTaskCompletedRequest)
					require.NoError(t, row.GetRespond()[0].UnmarshalTo(response))
					response.Result.Payloads = append(response.Result.Payloads, proto.CloneOf(response.Result.Payloads[0]))
					row.Respond = []*anypb.Any{pack(t, response)}
				}
				changed = true
				break
			}
			require.True(t, changed)
			data, err := proto.Marshal(pack(t, program))
			require.NoError(t, err)
			require.Error(t, workersdk.Validate(data))
			if control != "multiple-payloads" {
				_, err := worker.Decode(data)
				require.Error(t, err)
			}
		})
	}
}
