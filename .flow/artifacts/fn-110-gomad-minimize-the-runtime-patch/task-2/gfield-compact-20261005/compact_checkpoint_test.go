package toolchain

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestGFieldCompactCanonicalAlignment(t *testing.T) {
	root, archive, _ := pinnedArchive(t)
	candidate := materializePinnedSource(t, root, archive, "")
	phase, output := os.Getenv("COMPACT_PHASE"), os.Getenv("COMPACT_CAPTURE_DIR")
	if phase == "" || output == "" {
		t.Fatal("measurement requires phase and capture directory")
	}
	for _, contextLines := range []int{1, 3} {
		generated := regeneratePinnedPatch(t, root, archive, candidate, contextLines)
		contents, err := os.ReadFile(generated)
		if err != nil {
			t.Fatal(err)
		}
		suffix := "-U1.patch"
		if contextLines == 3 {
			suffix = "-U3.patch"
		}
		if err := os.WriteFile(filepath.Join(output, phase+suffix), contents, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("canonical U%d bytes=%d lines=%d", contextLines, len(contents), strings.Count(string(contents), "\n"))
	}
	pristine := filepath.Join(t.TempDir(), "pristine")
	if err := ExtractSource(context.Background(), archive, pristine); err != nil {
		t.Fatal(err)
	}
	before := compactGFieldLines(t, filepath.Join(pristine, "go", "src", "runtime", "runtime2.go"))
	after := compactGFieldLines(t, filepath.Join(candidate, "src", "runtime", "runtime2.go"))
	var changed []string
	for name, line := range before {
		other, found := after[name]
		if found && line != other && strings.Join(strings.Fields(line), " ") == strings.Join(strings.Fields(other), " ") {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	t.Logf("otherwise unchanged g fields with avoidable alignment edits=%d: %v", len(changed), changed)
	if len(changed) != 0 {
		t.Fatalf("canonical output realigns %d otherwise unchanged g fields", len(changed))
	}
}

func TestGFieldCompactSourceInventories(t *testing.T) {
	root, archive, _ := pinnedArchive(t)
	candidate := materializePinnedSource(t, root, archive, "")
	if err := copyOverlay(filepath.Join(root, "toolchain", "runtime", "overlay"), candidate); err != nil {
		t.Fatal(err)
	}
	platforms := gomadversion.SupportedPlatforms[:]
	draws, err := runtimeDrawReferences(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if problems := drawInventoryProblems(draws, parseReviewedDrawReferences(t)); len(problems) != 0 {
		t.Fatal(strings.Join(problems, "\n"))
	}
	seeded, err := seededDrawReferences(candidate, platforms)
	if err != nil {
		t.Fatal(err)
	}
	if problems := seededDrawProblems(seeded, reviewedSeededDrawReferences); len(problems) != 0 {
		t.Fatal(strings.Join(problems, "\n"))
	}
	clocks, err := hostClockReferences(candidate, platforms)
	if err != nil {
		t.Fatal(err)
	}
	if len(clocks) != len(reviewedHostClockReferences) {
		t.Fatalf("clock rows=%d, want=%d", len(clocks), len(reviewedHostClockReferences))
	}
	for _, reference := range reviewedHostClockReferences {
		if clocks[reference.key()] != reference.count {
			t.Fatalf("clock inventory: %s", reference.key())
		}
	}
	for _, entry := range []struct{ file, function, hostCall string }{
		{"runtime/time_nofake.go", "nanotime", "nanotime1"},
		{"runtime/time.go", "time_runtimeNow", "time_now"},
	} {
		guard, host, err := guardAndHostCallPositions(filepath.Join(candidate, "src", entry.file), entry.function, entry.hostCall)
		if err != nil || !guard.IsValid() || !host.IsValid() || guard.Offset > host.Offset {
			t.Fatalf("clock activation: %s: %v", entry.function, err)
		}
	}
	creations, err := goroutineCreationSites(candidate, platforms)
	if err != nil {
		t.Fatal(err)
	}
	if problems := goroutineCreationProblems(creations, reviewedGoroutineCreations); len(problems) != 0 {
		t.Fatal(strings.Join(problems, "\n"))
	}
	t.Logf("platforms=%v draw rows=%d seeded rows=%d clock rows=%d goroutine rows=%d", platforms, len(draws), len(seeded), len(clocks), len(creations))
}

func compactGFieldLines(t *testing.T, path string) map[string]string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, contents, 0)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(contents), "\n")
	result := make(map[string]string)
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "g" {
			return true
		}
		for _, field := range spec.Type.(*ast.StructType).Fields.List {
			for _, name := range field.Names {
				result[name.Name] = lines[set.Position(field.Pos()).Line-1]
			}
		}
		return false
	})
	if len(result) == 0 {
		t.Fatal("g field selection is empty")
	}
	return result
}
