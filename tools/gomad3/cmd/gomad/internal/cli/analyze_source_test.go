package cli

import (
	"bytes"
	"context"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/preparation"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestAnalyzeSourceInspectionForwardsCompleteEvidence(t *testing.T) {
	spec := target.Spec{Kind: target.KindGoTest, Source: "./fixture", Args: []string{"-test.run=Fixture"}, BuildTags: []string{"test_dep"}, WorkingDir: "/source", ToolchainRoot: "/toolchain", PreparationRoot: "/owned", BuildModFile: "/owned/prepared.mod", CapabilityMode: target.CapabilityModeClosure}
	review := target.CapabilityReview{Schema: target.CapabilityReviewSchema, BuildTags: spec.BuildTags, Roots: []target.CapabilityPackageReference{{ImportPath: "example.com/fixture", Name: "fixture", ForTest: "example.com/fixture"}}, Closure: target.CapabilityClosure{Schema: target.CapabilityClosureSchema, Packages: []target.CapabilityPackage{{ImportPath: "example.com/fixture", Name: "fixture", Root: true}}, Compatibility: []target.CompatibilityIdentity{}}, Packs: []target.CompatibilityPackEvidence{}, CapabilityMode: target.CapabilityModeClosure, Findings: []target.CapabilityFinding{}, GuardedFindings: []target.CapabilityFinding{}, EliminatedFindings: []target.CapabilityFinding{}}
	adapters := []deterministicio.Adapter{{Module: "example.com/adapter", Version: "v1.0.0", Sum: "h1:identity"}}
	identity := target.ToolchainIdentity{GoVersion: "go1.27.1", BuildKey: strings.Repeat("7", 64), TargetGOOS: "linux", TargetGOARCH: "amd64"}
	var order []string
	var stdout, stderr bytes.Buffer
	status := executeAnalysis(context.Background(), &stdout, &stderr, "text", target.Spec{Kind: target.KindGoTest}, identity, analyzeDependencies{
		inspect: func(_ context.Context, observed target.Spec) (preparation.Inspection, error) {
			order = append(order, "inspect")
			if observed.Kind != target.KindGoTest || observed.PreparationRoot != "" {
				t.Fatalf("inspection request=%#v", observed)
			}
			return preparation.Inspection{Spec: spec, Review: review, Adapters: adapters}, nil
		},
		build: func(input capabilityanalysis.Input) (capabilityanalysis.Report, error) {
			order = append(order, "build")
			if !reflect.DeepEqual(input.Spec, spec) || !reflect.DeepEqual(input.Review, review) || !reflect.DeepEqual(input.Adapters, adapters) || !reflect.DeepEqual(input.Toolchain, identity) || input.IOProfile.Identity() != deterministicio.Default().Identity() {
				t.Fatalf("forwarded inspection input=%#v", input)
			}
			return capabilityanalysis.Report{Classification: capabilityanalysis.ClassificationUnsupported}, nil
		},
	})
	if status != 1 || stderr.Len() != 0 || !slices.Equal(order, []string{"inspect", "build"}) {
		t.Fatalf("status=%d diagnostics=%q order=%v", status, stderr.String(), order)
	}
}

func TestAnalyzeSourcePublicHostRefusal(t *testing.T) {
	host := runtime.GOOS + "/" + runtime.GOARCH
	if slices.Contains(deterministicio.BoundaryPlatforms(), host) {
		t.Skip("public refusal control requires an unsupported actual host")
	}
	var stdout, stderr bytes.Buffer
	status := executeAnalysis(context.Background(), &stdout, &stderr, "json", target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: t.TempDir()}, target.ToolchainIdentity{}, analyzeDependencies{
		inspect: preparation.Inspect,
		build: func(capabilityanalysis.Input) (capabilityanalysis.Report, error) {
			t.Fatal("public host refusal reached report construction")
			return capabilityanalysis.Report{}, nil
		},
	})
	if status != 3 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "host is "+host) {
		t.Fatalf("actual-host public refusal status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
