package toolchain

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/runtimesemantics"
	"go.temporal.io/server/tools/gomad3/backends/native/toolchain/version"
)

var runtimeSemanticFunctions = strings.Fields(`gomadDiagnosticAllocations gomadChoiceRecordSelectResult gomadChoiceSelectOrigin gomadChoiceSelectReadiness gomadChoiceWaiterPresent gomadChoiceSeedRandom gomadTimerRand gomadRuntimeRand gomadRuntimeCheapRand gomadSeededDrawCheck gomadChoiceRunqSeeded gomadChoiceRunnextSeeded gomadChoiceRandom gomadChoiceSelectSeeded gomadChoiceEncodeRecord gomadChoiceZero gomadChoiceEqual gomadChoiceSite gomadChoiceRootIdentity gomadChoiceAssignGoroutineIdentity gomadChoiceTimerCreated gomadChoiceTimerFire gomadChoiceTimerFired gomadChoiceRunqIndex gomadChoiceSelectPollIndex gomadChoiceSelectIdentity`)

func runtimeSemanticInventoryProblems(source []byte) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "gomad_semantics.go", source, 0)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]bool)
	for _, name := range append(slices.Clone(runtimeSemanticFunctions), "gomadChoiceTimerIdentity", "gomadChoiceRecordValue") {
		wanted[name] = true
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			delete(wanted, declaration.Name.Name)
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if spec, ok := spec.(*ast.TypeSpec); ok {
					delete(wanted, spec.Name.Name)
				}
			}
		}
	}
	var problems []string
	for name := range wanted {
		problems = append(problems, "missing runtime semantic declaration: "+name)
	}
	slices.Sort(problems)
	return problems, nil
}

func TestRuntimeSemanticsInventory(t *testing.T) {
	source, err := os.ReadFile("../../../runtimesemantics/gomad_semantics.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	problems, err := runtimeSemanticInventoryProblems(source)
	if err != nil || len(problems) != 0 {
		t.Fatalf("runtime semantics inventory: %v %v", problems, err)
	}
	for _, name := range append(slices.Clone(runtimeSemanticFunctions), "gomadChoiceTimerIdentity", "gomadChoiceRecordValue") {
		t.Run(name, func(t *testing.T) {
			changed := regexp.MustCompile(`\b`+name+`\b`).ReplaceAll(source, []byte(name+"Renamed"))
			problems, err := runtimeSemanticInventoryProblems(changed)
			if err != nil || !slices.Equal(problems, []string{"missing runtime semantic declaration: " + name}) {
				t.Fatalf("renamed declaration not diagnosed: %v %v", problems, err)
			}
		})
	}
}

func TestNativeOverlayRuntimeSemanticsIdentity(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"runtimesemantics", "backends/native/toolchain/runtime/overlay", "backends/native/toolchain/version"} {
		if err := copyTree(filepath.Join("..", "..", "..", directory), filepath.Join(root, directory)); err != nil {
			t.Fatal(err)
		}
	}
	descriptor, err := version.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	config := BuildSpec{Root: root, ToolchainRoot: t.TempDir(), Patch: filepath.Join("runtime", "go1.27.1.patch"), Overlay: filepath.Join(root, "backends/native/toolchain/runtime/overlay")}
	bootstrap := bootstrapIdentity{version: "stock", hostOS: "linux", hostArch: "amd64", bashVersion: "bash"}
	config.BuildPath, config.BuildBash = "/bin", "/bin/bash"
	baseline, err := snapshotInputs(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := baseline.remove(); err != nil {
			t.Error(err)
		}
	})
	key, err := computeBuildKey(config, descriptor, bootstrap, baseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gomad_semantics.go", "gomad_choicewire_generated.go"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "runtimesemantics", name+".txt")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			copyPath := filepath.Join(baseline.overlay, "src/runtime", name)
			copied, err := os.ReadFile(copyPath)
			if err != nil || !bytes.Equal(source, copied) {
				t.Fatalf("native overlay %s differs from canonical bytes: %v", name, err)
			}
			if err := os.WriteFile(path, append(slices.Clone(source), '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			changed, err := snapshotInputs(config)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := changed.remove(); err != nil {
					t.Error(err)
				}
			})
			changedKey, err := computeBuildKey(config, descriptor, bootstrap, changed)
			if err != nil || changedKey == key {
				t.Fatalf("native identity did not bind %s: %v", name, err)
			}
			if err := runtimesemantics.Verify(changed.semantics, filepath.Join(baseline.overlay, "src/runtime")); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("stale native copy not diagnosed: %v", err)
			}
			if err := runtimesemantics.Verify(changed.semantics, filepath.Join(changed.overlay, "src/runtime")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, source, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRuntimeSemanticsPreservesReviewedDrawInventory(t *testing.T) {
	root := t.TempDir()
	if err := copyTree("runtime/overlay", root); err != nil {
		t.Fatal(err)
	}
	sources, err := runtimesemantics.Read("../../../runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range sources {
		if err := os.WriteFile(filepath.Join(root, "src/runtime", name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sharedPath := func(key string) bool {
		return strings.Contains(key, "runtime/gomad.go ") || strings.Contains(key, "runtime/gomad_semantics.go ")
	}
	got, err := runtimeDrawReferences(root)
	if err != nil {
		t.Fatal(err)
	}
	for key := range got {
		if !sharedPath(key) {
			delete(got, key)
		}
	}
	reviewed := parseReviewedDrawReferences(t)
	for key := range reviewed {
		if !sharedPath(key) {
			delete(reviewed, key)
		}
	}
	if problems := drawInventoryProblems(got, reviewed); len(problems) != 0 {
		t.Fatalf("shared draw inventory drift: %v", problems)
	}
	seeded, err := seededDrawReferences(root, version.SupportedPlatforms[:])
	if err != nil {
		t.Fatal(err)
	}
	for key := range seeded {
		if !sharedPath(key) {
			delete(seeded, key)
		}
	}
	var reviewedSeeded []seededDrawReference
	for _, reference := range reviewedSeededDrawReferences {
		if reference.file == "runtime/gomad.go" || reference.file == "runtime/gomad_semantics.go" {
			reviewedSeeded = append(reviewedSeeded, reference)
		}
	}
	if problems := seededDrawProblems(seeded, reviewedSeeded); len(problems) != 0 {
		t.Fatalf("shared seeded draw inventory drift: %v", problems)
	}
}
