package temporaldriver

import (
	"strings"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestFactoryRefusesMismatchedProgramGuardsBeforePolling(t *testing.T) {
	for _, kind := range []string{"raw", "sdk"} {
		for _, guard := range []string{"namespace", "task queue"} {
			t.Run(kind+"/"+guard, func(t *testing.T) {
				key, err := anypb.New(&temporalv1.ActivityTaskKey{Scenario: "Complete", Attempt: 1, ActivityType: "ModelSlice"})
				if err != nil {
					t.Fatal(err)
				}
				response, err := anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("Complete")}}}})
				if err != nil {
					t.Fatal(err)
				}
				p := &temporalv1.WorkerProgram{Namespace: "protected", TaskQueue: "protected-queue", Rules: []*temporalv1.Rule{{Key: key, Respond: []*anypb.Any{response}}}}
				if guard == "namespace" {
					p.Namespace = "other"
				} else {
					p.TaskQueue = "other"
				}
				a, err := anypb.New(p)
				if err != nil {
					t.Fatal(err)
				}
				data, err := proto.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				factory, err := NewFactory(act.Options{Address: "127.0.0.1:1", Namespace: "protected", TaskQueue: "protected-queue", ExecutorKind: kind, Program: data})
				var driver interface{ Close() error }
				if err == nil {
					created, createErr := factory.New(t.Context())
					err = createErr
					if created != nil {
						driver = created.(interface{ Close() error })
					}
				}
				if err == nil && driver != nil {
					if closeErr := driver.Close(); closeErr != nil {
						t.Fatal(closeErr)
					}
				}
				if err == nil || !strings.Contains(err.Error(), guard) {
					t.Fatalf("mismatched %s error = %v", guard, err)
				}
			})
		}
	}
}
