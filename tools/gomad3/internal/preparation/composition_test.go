package preparation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestPrepareCompositionOrderAndCleanupIsolation(t *testing.T) {
	root := t.TempDir()
	sibling := filepath.Join(root, ".adapter-work-caller")
	writeTestFile(t, sibling, "durable caller data")
	adapter := deterministicio.Default().Adapters()[0]
	var sequence []string
	var workspace string
	spec := target.Spec{Kind: target.KindGoRun, Source: ".", PreparationRoot: root}
	want := validPrepared()
	want.Adapters = []record.TargetAdapter{{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}}
	got, err := prepareWith(t.Context(), Request{Target: spec}, preparationServices{
		adapters: func(_ context.Context, actual target.Spec) (target.Spec, []deterministicio.BuildAdapter, error) {
			sequence = append(sequence, "adapters")
			workspace = actual.PreparationRoot
			if filepath.Dir(workspace) != root || !strings.HasPrefix(filepath.Base(workspace), ".adapter-work-") || workspace == sibling {
				t.Fatalf("adapter workspace = %s", workspace)
			}
			actual.BuildModFile = filepath.Join(workspace, "gomad.mod")
			return actual, []deterministicio.BuildAdapter{{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}}, nil
		},
		target: func(_ context.Context, actual target.Spec) (target.Prepared, error) {
			sequence = append(sequence, "target")
			if actual.PreparationRoot != root || actual.BuildModFile != filepath.Join(workspace, "gomad.mod") {
				t.Fatalf("target spec = %#v", actual)
			}
			return validPrepared(), nil
		},
		validate: func(actual target.Spec, prepared target.Prepared, environment []string) error {
			sequence = append(sequence, "validation")
			if !reflect.DeepEqual(prepared, want) {
				t.Fatalf("validation received = %#v, want %#v", prepared, want)
			}
			return deterministicio.Default().ValidatePreparedTarget(actual, prepared, environment)
		},
		remove: func(path string) error {
			sequence = append(sequence, "cleanup")
			if path != workspace {
				t.Fatalf("cleanup path = %s, workspace = %s", path, workspace)
			}
			return os.RemoveAll(path)
		},
	})
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(sequence, []string{"adapters", "target", "validation", "cleanup"}) {
		t.Fatalf("prepared = %#v, error = %v, sequence = %v", got, err, sequence)
	}
	data, err := os.ReadFile(sibling)
	if err != nil || string(data) != "durable caller data" {
		t.Fatalf("caller sibling changed: %q, %v", data, err)
	}
	if _, err := os.Stat(workspace); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("workspace survived: %v", err)
	}
}

func TestPrepareCompositionFailuresRetainIdentityAndCleanup(t *testing.T) {
	primary := errors.New("primary failure")
	cleanup := errors.New("cleanup failure")
	for _, stage := range []Stage{StageAdapters, StageTarget, StageValidation, StageCleanup} {
		t.Run(string(stage), func(t *testing.T) {
			root := t.TempDir()
			var sequence []Stage
			_, err := prepareWith(t.Context(), Request{Target: target.Spec{PreparationRoot: root}}, preparationServices{
				adapters: func(_ context.Context, spec target.Spec) (target.Spec, []deterministicio.BuildAdapter, error) {
					sequence = append(sequence, StageAdapters)
					if stage == StageAdapters {
						return target.Spec{}, nil, primary
					}
					return spec, nil, nil
				},
				target: func(context.Context, target.Spec) (target.Prepared, error) {
					sequence = append(sequence, StageTarget)
					if stage == StageTarget {
						return target.Prepared{}, primary
					}
					return target.Prepared{}, nil
				},
				validate: func(target.Spec, target.Prepared, []string) error {
					sequence = append(sequence, StageValidation)
					if stage == StageValidation {
						return primary
					}
					return nil
				},
				remove: func(path string) error {
					sequence = append(sequence, StageCleanup)
					if filepath.Dir(path) != root || !strings.HasPrefix(filepath.Base(path), ".adapter-work-") {
						t.Fatalf("cleanup path = %s", path)
					}
					if err := os.RemoveAll(path); err != nil {
						return err
					}
					return cleanup
				},
			})
			want := []Stage{StageAdapters, StageTarget, StageValidation, StageCleanup}
			switch stage {
			case StageAdapters:
				want = []Stage{StageAdapters, StageCleanup}
			case StageTarget:
				want = []Stage{StageAdapters, StageTarget, StageCleanup}
			case StageValidation, StageReview, StageCleanup:
			}
			if StageOf(err) != stage || !errors.Is(err, cleanup) || stage != StageCleanup && !errors.Is(err, primary) || !reflect.DeepEqual(sequence, want) {
				t.Fatalf("error = %v, stage = %s, sequence = %v, want %v", err, StageOf(err), sequence, want)
			}
		})
	}
}

func TestPrepareCustomTargetIdentityValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*target.Prepared)
	}{
		{"kind", func(prepared *target.Prepared) { prepared.Kind = target.KindExec }},
		{"source", func(prepared *target.Prepared) { prepared.Source = "./other" }},
		{"arguments", func(prepared *target.Prepared) { prepared.Argv = []string{"gomad3-target", "extra"} }},
		{"Go version", func(prepared *target.Prepared) { prepared.GoVersion = "go1.0" }},
		{"platform", func(prepared *target.Prepared) { prepared.TargetGOOS = "plan9" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepared := validPrepared()
			test.mutate(&prepared)
			_, err := Prepare(t.Context(), Request{
				Target:   target.Spec{Kind: target.KindGoRun, Source: "."},
				Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return prepared, nil }),
			})
			if err == nil || StageOf(err) != StageValidation || !strings.Contains(err.Error(), "deterministic I/O") || strings.Contains(err.Error(), "host is") {
				t.Fatalf("identity validation = %v, stage = %s", err, StageOf(err))
			}
		})
	}
}
