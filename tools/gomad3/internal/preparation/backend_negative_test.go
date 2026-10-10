package preparation

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func externalFixture() (target.Spec, target.Prepared) {
	spec := target.Spec{Kind: target.KindGoRun, Source: ".", Args: []string{"argument"}, Backend: "fixture"}
	data := []byte(`{"profile":"observed"}`)
	reference := record.BackendPayload{Schema: "fixture/v1", File: "backend/provenance.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}
	return spec, target.Prepared{Kind: spec.Kind, Source: spec.Source, Argv: []string{"gomad3-target", "argument"}, Backend: &record.BackendMetadata{Name: "fixture", ReplayMode: record.ReplayObserved, Provenance: reference}, BackendPayloads: []target.BackendPayload{{Reference: reference, Data: data}}}
}

func TestPrepareExternalBackendRejectsIdentityAndPayloadChanges(t *testing.T) {
	cases := []struct {
		name   string
		change func(*target.Prepared)
	}{
		{"backend", func(p *target.Prepared) { p.Backend.Name = "other" }},
		{"kind", func(p *target.Prepared) { p.Kind = target.KindExec }},
		{"source", func(p *target.Prepared) { p.Source = "other" }},
		{"argv0", func(p *target.Prepared) { p.Argv[0] = "other" }},
		{"argument", func(p *target.Prepared) { p.Argv[1] = "other" }},
		{"missing-argv", func(p *target.Prepared) { p.Argv = nil }},
		{"adapters", func(p *target.Prepared) { p.Adapters = []record.TargetAdapter{{Module: "native"}} }},
		{"digest", func(p *target.Prepared) { p.Backend.Provenance.SHA256 = "invalid" }},
		{"payload", func(p *target.Prepared) { p.BackendPayloads[0].Data[0] = 'x' }},
		{"reference", func(p *target.Prepared) { p.BackendPayloads[0].Reference.File = "backend/other" }},
		{"missing-payload", func(p *target.Prepared) { p.BackendPayloads = nil }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			spec, prepared := externalFixture()
			test.change(&prepared)
			called := false
			_, err := Prepare(t.Context(), Request{Target: spec, Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return prepared, nil }), Validate: func(target.Spec, target.Prepared, []string) error { called = true; return nil }})
			if err == nil || StageOf(err) != StageValidation || called {
				t.Fatalf("identity rejection: err=%v stage=%s validator=%v", err, StageOf(err), called)
			}
		})
	}
}

func TestPrepareExternalBackendFailureStagesAndOwnership(t *testing.T) {
	spec, provided := externalFixture()
	sentinel := errors.New("fixture failure")
	for _, stage := range []Stage{StageTarget, StageValidation} {
		t.Run(string(stage), func(t *testing.T) {
			_, err := Prepare(t.Context(), Request{Target: spec, Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) {
				if stage == StageTarget {
					return target.Prepared{}, sentinel
				}
				return provided, nil
			}), Validate: func(target.Spec, target.Prepared, []string) error { return sentinel }})
			if !errors.Is(err, sentinel) || StageOf(err) != stage {
				t.Fatalf("failure stage %s: %v", StageOf(err), err)
			}
		})
	}
	for _, request := range []Request{{Target: spec}, {Target: spec, Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return provided, nil })}} {
		_, err := Prepare(t.Context(), request)
		if err == nil || StageOf(err) != StageTarget {
			t.Fatalf("missing provider: %v", err)
		}
	}
	original := provided.CloneBackend()
	got, err := Prepare(t.Context(), Request{Target: spec, Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return provided, nil }), Validate: func(_ target.Spec, p target.Prepared, _ []string) error {
		p.Backend.Name = "mutated"
		p.BackendPayloads[0].Data[0] = 'x'
		return nil
	}})
	if err != nil || !reflect.DeepEqual(original, got) || !reflect.DeepEqual(original, provided) {
		t.Fatalf("validation alias: %v", err)
	}
	recorded := got.RecordTarget()
	recorded.Backend.Name = "changed-record"
	provided.Backend.Name = "changed-provider"
	provided.BackendPayloads[0].Data[0] = 'x'
	if !reflect.DeepEqual(original, got) {
		t.Fatal("provider or recorded target retained preparation aliases")
	}
}

func TestPrepareNativeIntentRejectsBackendAfterNativeValidation(t *testing.T) {
	_, prepared := externalFixture()
	sentinel := errors.New("native validation first")
	request := Request{Target: target.Spec{Kind: target.KindGoRun, Source: "."}, Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return prepared, nil }), Validate: func(target.Spec, target.Prepared, []string) error {
		t.Fatal("external callback selected for native intent")
		return nil
	}}
	_, err := prepareWith(t.Context(), request, preparationServices{validate: func(target.Spec, target.Prepared, []string) error { return sentinel }})
	if !errors.Is(err, sentinel) || StageOf(err) != StageValidation {
		t.Fatalf("native precedence: %v", err)
	}
	_, err = prepareWith(t.Context(), request, preparationServices{validate: func(target.Spec, target.Prepared, []string) error { return nil }})
	if err == nil || StageOf(err) != StageValidation {
		t.Fatalf("native backend metadata accepted: %v", err)
	}
}

func TestPrepareExternalBackendArgumentsRemainDetached(t *testing.T) {
	for _, boundary := range []string{"validator", "provider"} {
		t.Run(boundary, func(t *testing.T) {
			spec, provided := externalFixture()
			original := append([]string{}, spec.Args...)
			_, err := Prepare(t.Context(), Request{Target: spec, Preparer: testPreparer(func(_ context.Context, s target.Spec) (target.Prepared, error) {
				if boundary == "provider" {
					s.Args[0] = "altered"
					provided.Argv[1] = "altered"
				}
				return provided, nil
			}), Validate: func(s target.Spec, p target.Prepared, _ []string) error {
				if boundary == "validator" {
					s.Args[0] = "altered"
					p.Argv[1] = "altered"
				}
				return nil
			}})
			if boundary == "provider" && err == nil {
				t.Fatal("provider altered requested identity")
			}
			if !reflect.DeepEqual(spec.Args, original) {
				t.Fatal("boundary mutated caller arguments")
			}
			if boundary == "validator" && provided.Argv[1] != "argument" {
				t.Fatal("validator retained prepared argument aliases")
			}
		})
	}
}
