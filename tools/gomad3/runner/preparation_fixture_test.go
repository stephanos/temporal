package runner

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/preparation"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

const scriptedBootstrapMarker = "synthetic scripted Runner bootstrap; not a toolchain frame"

func scriptedPreparationDependencies(t *testing.T, preparer Preparer, executor executionRunner) executionDependencies {
	t.Helper()
	if preparer == nil || executor == nil {
		t.Fatal("scripted preparation requires an explicit preparer and executor")
	}
	return executionDependencies{
		executor: executor,
		prepare: func(ctx context.Context, request preparation.Request) (target.Prepared, error) {
			if !reflect.DeepEqual(request.Preparer, preparer) || request.Target.PreparationRoot == "" || len(request.Target.AdapterReplacements) != 0 {
				t.Fatalf("scripted preparation request = %#v", request)
			}
			prepared, err := preparer.Prepare(ctx, request.Target)
			if err != nil {
				return target.Prepared{}, err
			}
			if prepared.Kind != request.Target.Kind || prepared.Source != request.Target.Source || len(prepared.Argv) == 0 || prepared.Argv[0] != "gomad3-target" || !slices.Equal(prepared.Argv[1:], request.Target.Args) || len(prepared.Adapters) != 0 {
				t.Fatalf("scripted prepared target = %#v", prepared)
			}
			if err := prepared.Verify(); err != nil {
				t.Fatal(err)
			}
			prepared.Adapters = []record.TargetAdapter{}
			return prepared, nil
		},
		bootstrap: func(deterministicio.Spec, target.Prepared, string, uint64) ([]byte, error) {
			return []byte(scriptedBootstrapMarker), nil
		},
	}
}
