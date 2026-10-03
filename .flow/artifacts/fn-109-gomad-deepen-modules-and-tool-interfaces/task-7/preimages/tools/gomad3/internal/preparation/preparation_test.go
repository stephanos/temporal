package preparation

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

type testPreparer func(context.Context, target.Spec) (target.Prepared, error)

func (prepare testPreparer) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return prepare(ctx, spec)
}

func validPrepared() target.Prepared {
	contract := deterministicio.Default().TargetContract()
	return target.Prepared{
		Kind: target.KindGoRun, Source: ".", Argv: []string{"gomad3-target"},
		GoVersion: contract.GoVersion, TargetGOOS: contract.GOOS, TargetGOARCH: contract.GOARCH,
	}
}

func TestPrepareCustomPreparerSkipsAdaptersAndValidates(t *testing.T) {
	want := validPrepared()
	want.Adapters = []record.TargetAdapter{}
	provided := want
	provided.Adapters = []record.TargetAdapter{{Module: "unselected"}}
	got, err := Prepare(t.Context(), Request{
		Target: target.Spec{Kind: target.KindGoRun, Source: ".", PreparationRoot: t.TempDir()},
		Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return provided, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prepared = %#v, want %#v", got, want)
	}
}

func TestPrepareKeepsTargetFailureStageAndIdentity(t *testing.T) {
	want := errors.New("preparer failed")
	_, err := Prepare(t.Context(), Request{
		Target: target.Spec{Kind: target.KindGoRun, Source: ".", PreparationRoot: t.TempDir()},
		Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return target.Prepared{}, want }),
	})
	if !errors.Is(err, want) || StageOf(err) != StageTarget || err.Error() != want.Error() {
		t.Fatalf("Prepare() error = %v, stage = %s", err, StageOf(err))
	}
}
