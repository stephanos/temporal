package temporaldriver

import (
	"context"
	"strings"
	"testing"

	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestFactoryRefusesMismatchedProgramGuardsBeforePolling(t *testing.T) {
	for _, kind := range []string{"raw", "sdk"} {
		for _, guard := range []string{"namespace", "task queue"} {
			t.Run(kind+"/"+guard, func(t *testing.T) {
				p := &temporalv1.WorkerProgram{Namespace: "protected", TaskQueue: "protected-queue"}
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
				if err != nil {
					t.Fatal(err)
				}
				driver, err := factory.New(context.Background())
				if err == nil && driver != nil {
					if closeErr := driver.(interface{ Close() error }).Close(); closeErr != nil {
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
