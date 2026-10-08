package preparation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestInspectSourcePublicHostRefusal(t *testing.T) {
	host := runtime.GOOS + "/" + runtime.GOARCH
	if slices.Contains(deterministicio.BoundaryPlatforms(), host) {
		t.Skip("public refusal control requires an unsupported actual host")
	}
	_, err := Inspect(context.Background(), target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: t.TempDir()})
	if err == nil || StageOf(err) != StageAdapters || !strings.Contains(err.Error(), "host is "+host) {
		t.Fatalf("actual-host public refusal = %v, stage=%s", err, StageOf(err))
	}
}

func TestInspectSourceCleanupPreservesOriginalErrorText(t *testing.T) {
	primary, cleanup := errors.New("review failed"), errors.New("remove failed")
	_, err := inspectWith(t.Context(), target.Spec{Kind: target.KindGoRun, Source: "."}, inspectionServices{
		adapters: func(_ context.Context, spec target.Spec) (target.Spec, []deterministicio.BuildAdapter, error) {
			return spec, []deterministicio.BuildAdapter{}, nil
		},
		review: func(context.Context, target.Spec) (target.CapabilityReview, error) {
			return target.CapabilityReview{}, primary
		},
		remove: func(root string) error {
			t.Cleanup(func() {
				if err := os.RemoveAll(root); err != nil {
					t.Error(err)
				}
			})
			return cleanup
		},
	})
	if err == nil || err.Error() != "review failed\nremove failed" || !errors.Is(err, primary) || !errors.Is(err, cleanup) || StageOf(err) != StageReview {
		t.Fatalf("owner error text/identity/stage = %v, %s", err, StageOf(err))
	}
	operation, removal := InspectionErrorParts(err)
	if !errors.Is(operation, primary) || !errors.Is(removal, cleanup) || StageOf(operation) != StageReview || StageOf(removal) != StageCleanup || !errors.Is(err, operation) || !errors.Is(err, removal) {
		t.Fatalf("typed owner parts = %v, %v", operation, removal)
	}
	owner, ok := err.(*inspectionFailure)
	if !ok {
		t.Fatalf("owner error type = %T", err)
	}
	parts := owner.Unwrap()
	if !reflect.DeepEqual(parts, []error{operation, removal}) {
		t.Fatalf("owner error traversal order = %v", parts)
	}
	for _, outer := range []error{nil, primary, removal, errors.Join(err, errors.New("unrelated sibling")), fmt.Errorf("outer context: %w", err), &stageError{stage: StageAdapters, err: err}} {
		actual, cleanup := InspectionErrorParts(outer)
		if actual != outer || cleanup != nil {
			t.Fatalf("projected unrelated wrapper %T = %v, %v", outer, actual, cleanup)
		}
	}
}

func TestInspectSourceOwnsWorkspaceAndCompleteEvidence(t *testing.T) {
	spec := target.Spec{Kind: target.KindGoRun, Source: "example.com/inspection"}
	adapters := []deterministicio.BuildAdapter{{Module: "example.com/adapter", Version: "v1.0.0", Sum: "h1:identity", BuildModFile: "prepared.mod", Source: "original.go", ReplacementRoot: "adapter-root", Replacement: "replacement.go", PreparedPackage: "example.com/adapter/pkg", SourceSHA256: "source", ReplacementSHA256: "replacement", OriginalSourceInventorySHA256: "original-inventory", ReplacementSourceInventorySHA256: "replacement-inventory", PreparedSourceSetSHA256: "prepared-sources"}}
	review := target.CapabilityReview{Schema: target.CapabilityReviewSchema, BuildTags: []string{"test_dep"}, CapabilityMode: target.CapabilityModeClosure, Roots: []target.CapabilityPackageReference{{ImportPath: spec.Source, Name: "main"}}, Closure: target.CapabilityClosure{Schema: target.CapabilityClosureSchema, Packages: []target.CapabilityPackage{{ImportPath: spec.Source, Name: "main", Root: true}}}, Packs: []target.CompatibilityPackEvidence{}, Findings: []target.CapabilityFinding{}, GuardedFindings: []target.CapabilityFinding{}, EliminatedFindings: []target.CapabilityFinding{}}
	var root string
	var calls []string
	services := inspectionServices{
		adapters: func(_ context.Context, observed target.Spec) (target.Spec, []deterministicio.BuildAdapter, error) {
			calls = append(calls, "adapters")
			root = observed.PreparationRoot
			info, err := os.Stat(root)
			if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
				t.Fatalf("private workspace = %v, %v", info, err)
			}
			observed.PreparationRoot = ""
			if !reflect.DeepEqual(observed, spec) {
				t.Fatalf("adapter request = %#v", observed)
			}
			observed.PreparationRoot, observed.BuildModFile = root, "prepared.mod"
			return observed, adapters, nil
		},
		review: func(_ context.Context, observed target.Spec) (target.CapabilityReview, error) {
			calls = append(calls, "review")
			if observed.PreparationRoot != root || observed.BuildModFile != "prepared.mod" {
				t.Fatalf("review request = %#v", observed)
			}
			return review, nil
		},
		remove: func(path string) error {
			calls = append(calls, "remove")
			if path != root {
				t.Fatalf("removed root %q, want %q", path, root)
			}
			return os.RemoveAll(path)
		},
	}
	inspected, err := inspectWith(t.Context(), spec, services)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inspected.Review, review) || !reflect.DeepEqual(inspected.BuildAdapters, adapters) || !reflect.DeepEqual(inspected.Adapters, []deterministicio.Adapter{{Module: "example.com/adapter", Version: "v1.0.0", Sum: "h1:identity"}}) || inspected.Spec.BuildModFile != "prepared.mod" {
		t.Fatalf("incomplete inspection = %#v", inspected)
	}
	if _, err := os.Stat(root); err != nil || !reflect.DeepEqual(calls, []string{"adapters", "review"}) {
		t.Fatalf("workspace removed before Close: %v, calls=%v", err, calls)
	}
	if err := inspected.Close(); err != nil {
		t.Fatal(err)
	}
	if err := inspected.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) || !reflect.DeepEqual(calls, []string{"adapters", "review", "remove"}) {
		t.Fatalf("workspace after Close: %v, calls=%v", err, calls)
	}
}

func TestInspectSourcePreservesStageAndPrimaryErrors(t *testing.T) {
	primary := errors.New("primary")
	cleanup := errors.New("cleanup")
	for _, test := range []struct {
		name  string
		stage Stage
		err   error
	}{
		{name: "invalid adapter", stage: StageAdapters, err: &deterministicio.InvalidBuildAdapterConfigurationError{Err: primary}},
		{name: "invalid review", stage: StageReview, err: &target.InvalidCapabilityReviewError{Err: primary}},
		{name: "unsupported", stage: StageReview, err: &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}},
		{name: "malformed linked", stage: StageReview, err: primary},
	} {
		t.Run(test.name, func(t *testing.T) {
			services := inspectionServices{
				adapters: func(_ context.Context, spec target.Spec) (target.Spec, []deterministicio.BuildAdapter, error) {
					if test.stage == StageAdapters {
						return target.Spec{}, nil, test.err
					}
					return spec, []deterministicio.BuildAdapter{}, nil
				},
				review: func(context.Context, target.Spec) (target.CapabilityReview, error) {
					return target.CapabilityReview{}, test.err
				},
				remove: func(root string) error {
					if err := os.RemoveAll(root); err != nil {
						t.Fatal(err)
					}
					return cleanup
				},
			}
			_, err := inspectWith(t.Context(), target.Spec{}, services)
			if !errors.Is(err, test.err) || !errors.Is(err, cleanup) || StageOf(err) != test.stage || deterministicio.IsInvalidBuildAdapterConfiguration(err) != (test.stage == StageAdapters) || target.IsInvalidCapabilityReview(err) != (test.name == "invalid review") || target.IsUnsupportedCapability(err) != (test.name == "unsupported") {
				t.Fatalf("error = %v, stage=%s", err, StageOf(err))
			}
		})
	}
}

func TestInspectSourceFailedCloseRetiresOwnership(t *testing.T) {
	cleanup := errors.New("cleanup")
	calls := 0
	root := t.TempDir()
	inspected := Inspection{root: root, remove: func(path string) error { calls++; return cleanup }}
	if err := inspected.Close(); !errors.Is(err, cleanup) || StageOf(err) != StageCleanup {
		t.Fatalf("Close = %v", err)
	}
	if err := inspected.Close(); err != nil || calls != 1 {
		t.Fatalf("second Close = %v, attempts=%d", err, calls)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
	var absent *Inspection
	if err := absent.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInspectSourceRefusesCallerRootBeforeServices(t *testing.T) {
	_, err := inspectWith(t.Context(), target.Spec{PreparationRoot: t.TempDir()}, inspectionServices{})
	if err == nil || StageOf(err) != "" {
		t.Fatalf("caller root = %v", err)
	}
}
