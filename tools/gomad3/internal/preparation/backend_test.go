package preparation

import (
	"context"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestPrepareExternalBackendUsesValidatedTypedTarget(t *testing.T) {
	spec := target.Spec{Kind: target.KindGoRun, Source: ".", Backend: "fixture"}
	payload := []byte(`{"profile":"fixture-observed/v1"}`)
	want := target.Prepared{
		Kind: spec.Kind, Source: spec.Source, Argv: []string{"gomad3-target"},
		GoVersion: "go1.27.1", TargetGOOS: "wasip1", TargetGOARCH: "wasm",
		Adapters: []record.TargetAdapter{},
		Backend: &record.BackendMetadata{
			Name: "fixture", ReplayMode: record.ReplayObserved,
			Provenance: record.BackendPayload{Schema: "fixture-provenance/v1", File: "backend/provenance.json", SHA256: record.HashBytes(payload), Bytes: record.Uint64String(len(payload))},
		},
	}
	want.BackendPayloads = []target.BackendPayload{{Reference: want.Backend.Provenance, Data: payload}}
	var sequence []string
	got, err := Prepare(t.Context(), Request{
		Target: spec,
		Preparer: testPreparer(func(_ context.Context, actual target.Spec) (target.Prepared, error) {
			sequence = append(sequence, "target")
			if !reflect.DeepEqual(spec, actual) {
				t.Fatalf("external specification changed: %#v", actual)
			}
			return want, nil
		}),
		Validate: func(actual target.Spec, prepared target.Prepared, _ []string) error {
			sequence = append(sequence, "validation")
			if !reflect.DeepEqual(actual, spec) || !reflect.DeepEqual(prepared, want) {
				t.Fatalf("external validation received changed values")
			}
			return nil
		},
	})
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(sequence, []string{"target", "validation"}) {
		t.Fatalf("external preparation = %#v, %v, sequence %v", got, err, sequence)
	}
}
