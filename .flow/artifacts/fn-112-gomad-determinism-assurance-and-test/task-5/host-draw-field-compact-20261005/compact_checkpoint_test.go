package toolchain

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// This retained checkpoint test measures canonical output using the existing
// pinned-source helpers. It is loaded through a Go overlay, not shipped.
func TestCompactCheckpointAlignment(t *testing.T) {
	root, archive, _ := pinnedArchive(t)
	candidate := materializePinnedSource(t, root, archive, "")
	phase := os.Getenv("COMPACT_PHASE")
	output := os.Getenv("COMPACT_CAPTURE_DIR")
	if phase == "" || output == "" {
		t.Fatal("checkpoint measurement requires phase and capture directory")
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
	before := compactMFieldLines(t, filepath.Join(pristine, "go", "src", "runtime", "runtime2.go"))
	after := compactMFieldLines(t, filepath.Join(candidate, "src", "runtime", "runtime2.go"))
	var changed []string
	for name, line := range before {
		other, found := after[name]
		if found && line != other && strings.Join(strings.Fields(line), " ") == strings.Join(strings.Fields(other), " ") {
			changed = append(changed, name)
		}
	}
	t.Logf("unchanged m fields with avoidable alignment edits=%d", len(changed))
	if len(changed) != 0 {
		t.Fatalf("canonical output realigns %d otherwise unchanged m fields", len(changed))
	}
}

func TestCompactCheckpointDrawInventories(t *testing.T) {
	root, archive, _ := pinnedArchive(t)
	candidate := materializePinnedSource(t, root, archive, "")
	if err := copyOverlay(filepath.Join(root, "toolchain", "runtime", "overlay"), candidate); err != nil {
		t.Fatal(err)
	}
	draws, err := runtimeDrawReferences(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if problems := drawInventoryProblems(draws, parseReviewedDrawReferences(t)); len(problems) != 0 {
		t.Fatalf("draw inventory: %s", strings.Join(problems, "\n"))
	}
	seeded, err := seededDrawReferences(candidate, gomadversion.SupportedPlatforms[:])
	if err != nil {
		t.Fatal(err)
	}
	if problems := seededDrawProblems(seeded, reviewedSeededDrawReferences); len(problems) != 0 {
		t.Fatalf("seeded inventory: %s", strings.Join(problems, "\n"))
	}
	t.Logf("draw inventory rows=%d, seeded rows=%d; platforms=%v", len(draws), len(seeded), gomadversion.SupportedPlatforms)
}

func compactMFieldLines(t *testing.T, path string) map[string]string {
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
		if !ok || spec.Name.Name != "m" {
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
		t.Fatal("m field selection is empty")
	}
	return result
}
