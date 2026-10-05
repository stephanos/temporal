package target

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/internal/sourceinventory"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target/internal/livecap"
)

// TestCapabilityReviewGoldenCanonicalBytes pins the canonical bytes of
// reviews over fixed evidence: ordered closure findings, linked and guarded
// projections with live, guarded and eliminated blockers, recorded closures,
// and adapter replacement inventories. The goldens were captured before
// collection, evaluation and linked projection were separated.
func TestCapabilityReviewGoldenCanonicalBytes(t *testing.T) {
	findings := goldenFindingsReview(t)
	sources := goldenSourcesReview(t)
	recorded, err := evaluateGoldenRecordedClosure(sources.Closure)
	requireTestNoError(t, err)
	for _, test := range []struct {
		name   string
		review CapabilityReview
	}{
		{name: "closure-findings", review: findings},
		{name: "closure-sources", review: sources},
		{name: "closure-adapter", review: goldenAdapterReview(t)},
		{name: "linked", review: projectGoldenLinkedReview(findings, CapabilityModeLinked)},
		{name: "guarded", review: projectGoldenLinkedReview(findings, CapabilityModeGuarded)},
		{name: "recorded-closure", review: recorded},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := canonicaljson.CanonicalJSON(test.review)
			requireTestNoError(t, err)
			want, err := os.ReadFile(filepath.Join("testdata", "capability-review", test.name+".json"))
			requireTestNoError(t, err)
			if !bytes.Equal(got, bytes.TrimSuffix(want, []byte("\n"))) {
				t.Fatalf("canonical review differs from golden:\n got %s\nwant %s", got, want)
			}
		})
	}
}

func goldenFindingsReview(t *testing.T) CapabilityReview {
	t.Helper()
	directory := t.TempDir()
	writeGoldenSource(t, directory, "target.go", "package main\n\n//go:linkname malformed\nfunc malformed()\n")
	writeGoldenSource(t, directory, "escape.c", "int escape;\n")
	review, err := projectCapabilityReview([]listedPackage{
		{
			ImportPath: "example.com/dependency", Name: "dependency", DepOnly: true,
			Imports: []string{"syscall"}, Module: &listedModule{Path: "example.com/dependency", Version: "v1.0.0", Sum: "h1:dependency"},
		},
		{
			ImportPath: "example.com/target", Name: "main", Dir: directory, GoFiles: []string{"target.go"},
			Imports: []string{"example.com/dependency", "os/exec", "os/user"}, CFiles: []string{"escape.c"}, Module: &listedModule{Path: "example.com/target", Main: true},
		},
		{ImportPath: "os", Name: "os", Standard: true, Imports: []string{"syscall"}},
	}, nil, []string{"gomad_fixture"})
	requireTestNoError(t, err)
	return review
}

func goldenSourcesReview(t *testing.T) CapabilityReview {
	t.Helper()
	directory := t.TempDir()
	writeGoldenSource(t, directory, "main.go", "package main\n\nfunc main() {}\n")
	writeGoldenSource(t, directory, "link.go", "package main\n\nimport _ \"unsafe\"\n\n//go:linkname escape syscall.Syscall\nfunc escape()\n")
	writeGoldenSource(t, directory, "api.h", "#define VALUE 1\n")
	writeGoldenSource(t, directory, "raw_arm64.s", "TEXT ·raw(SB),$0\n")
	writeGoldenSource(t, directory, "main_test.go", "package main\n")
	overlay := filepath.Join(t.TempDir(), "overlay.go")
	writeGoldenSource(t, filepath.Dir(overlay), filepath.Base(overlay), "package main\n\nfunc overlaid() {}\n")
	review, err := projectCapabilityReview([]listedPackage{
		{
			ImportPath: "example.com/target", Name: "main", Dir: directory, GoFiles: []string{"main.go", "link.go", "main.go"},
			Imports: []string{"os/signal", "fmt", "os/signal"}, HFiles: []string{"api.h"}, SFiles: []string{"raw_arm64.s"},
			Module: &listedModule{Path: "example.com/target", Main: true},
		},
		{
			ImportPath: "example.com/target [example.com/target.test]", ForTest: "example.com/target", Name: "main", Dir: directory,
			GoFiles: []string{"main.go", "main_test.go"}, Imports: []string{"plugin"}, Module: &listedModule{Path: "example.com/target", Main: true},
		},
		{
			ImportPath: "example.com/target.test", Name: "main", GoFiles: []string{filepath.Join(t.TempDir(), "_testmain.go")},
			Imports: []string{"os", "testing"}, Module: &listedModule{Path: "example.com/target", Main: true},
		},
		{
			ImportPath: "example.com/replaced", Name: "replaced", DepOnly: true, Dir: directory, GoFiles: []string{"main.go"},
			Imports: []string{"golang.org/x/sys/unix"},
			Module:  &listedModule{Path: "example.com/replaced", Version: "v1.0.0", Sum: "h1:replaced", Replace: &listedModule{Path: "example.com/fork", Version: "v1.0.1", Sum: "h1:fork"}},
		},
		{
			ImportPath: "example.com/local", Name: "local", DepOnly: true, Dir: directory, GoFiles: []string{"main.go"},
			Module: &listedModule{Path: "example.com/local", Version: "v1.0.0", Replace: &listedModule{Path: "../local", Dir: directory}},
		},
	}, map[string]string{filepath.Join(directory, "main.go"): overlay}, []string{"gomad_fixture", "test_dep"})
	requireTestNoError(t, err)
	return review
}

func goldenAdapterReview(t *testing.T) CapabilityReview {
	t.Helper()
	replacement := filepath.Join(t.TempDir(), "replacement")
	packageDirectory := filepath.Join(replacement, "internal")
	requireTestNoError(t, os.MkdirAll(packageDirectory, 0o700))
	contents := "package internal\n"
	writeGoldenSource(t, packageDirectory, "internal.go", contents)
	writeGoldenSource(t, replacement, "go.mod", "module example.com/adapter\n")
	writeGoldenSource(t, replacement, "adapter.go", "package adapter\n")
	inventory, err := sourceinventory.Digest(replacement)
	requireTestNoError(t, err)
	moduleSum := "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	adapter := AdapterReplacement{
		Original:        ModuleIdentity{Path: "example.com/adapter", Version: "v1.2.3", Sum: moduleSum},
		ReplacementPath: replacement,
		PreparedPackage: "example.com/adapter/internal",
		ProfileName:     "gomad3-deterministic/v1", ProfileImplementationSHA256: "sha256:" + strings.Repeat("1", 64),
		Adapter:                          ModuleIdentity{Path: "example.com/adapter", Version: "v1.2.3-gomad", Sum: "h1:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="},
		OriginalSourceInventorySHA256:    "sha256:" + strings.Repeat("2", 64),
		ReplacementSourceInventorySHA256: inventory,
		PreparedSourceSetSHA256: compatibility.DigestSources([]compatibility.Source{{
			Name: "internal.go", SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(contents))),
		}}),
	}
	module := &listedModule{Path: "example.com/adapter", Version: "v1.2.3", Replace: &listedModule{Dir: replacement}}
	review, err := projectCapabilityReview([]listedPackage{
		{ImportPath: "example.com/adapter/internal", Name: "internal", DepOnly: true, Dir: packageDirectory, GoFiles: []string{"internal.go"}, Module: module},
		{ImportPath: "example.com/adapter", Name: "adapter", DepOnly: true, Dir: replacement, GoFiles: []string{"adapter.go"}, Imports: []string{"example.com/adapter/internal"}, Module: module},
		{ImportPath: "example.com/target", Name: "main", Standard: true},
	}, nil, nil, []AdapterReplacement{adapter})
	requireTestNoError(t, err)
	return review
}

func goldenLinkedRecord() livecap.Record {
	return livecap.Record{
		Manifest: livecap.Manifest{
			Schema: "gomad3.live-capability-manifest/v2", CapabilityUniverseSHA256: "sha256:" + strings.Repeat("a", 64),
			ProducerImplementationSHA256: "sha256:" + strings.Repeat("b", 64), GuardImplementationSHA256: "sha256:" + strings.Repeat("c", 64),
			Facts: []livecap.Fact{
				{Kind: livecap.FactKindCapability, Capability: "import:syscall", OwnerPackage: "example.com/dependency", OwnerSymbol: "example.com/dependency.call"},
				{Kind: livecap.FactKindGuard, Capability: "import:os/exec", OwnerPackage: "example.com/target", OwnerSymbol: "main.run"},
				{Kind: livecap.FactKindBoundary, Capability: "filesystem.readlink", Disposition: livecap.DispositionDenied, OwnerPackage: "main", OwnerSymbol: "main.main"},
				{Kind: livecap.FactKindBoundary, Capability: "filesystem.readlink", Disposition: livecap.DispositionDenied, OwnerPackage: "example.com/target", OwnerSymbol: "main.other"},
				{Kind: livecap.FactKindGuard, Capability: "process.start", OwnerPackage: "example.com/dependency", OwnerSymbol: "example.com/dependency.start"},
				{Kind: livecap.FactKindBoundary, Capability: "filesystem.open", Disposition: livecap.DispositionModeled, OwnerPackage: "os", OwnerSymbol: "os.OpenFile"},
			},
		},
		Payload: []byte("golden linked capability payload"),
		SHA256:  record.SHA256("sha256:" + strings.Repeat("d", 64)),
	}
}

func writeGoldenSource(t *testing.T, directory, name, contents string) {
	t.Helper()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600))
}

func evaluateGoldenRecordedClosure(closure CapabilityClosure) (CapabilityReview, error) {
	review, err := reviewRecordedClosure(closure, nil)
	if err != nil {
		return CapabilityReview{}, err
	}
	return projectLinkedCapabilityReview(review, goldenLinkedRecord(), CapabilityModeGuarded), nil
}

func projectGoldenLinkedReview(review CapabilityReview, mode CapabilityMode) CapabilityReview {
	return projectLinkedCapabilityReview(review, goldenLinkedRecord(), mode)
}
