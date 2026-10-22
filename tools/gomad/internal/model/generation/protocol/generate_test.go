package protocol

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var moduleRoot = filepath.Join("..", "..", "..", "..")

func TestSharedSetIsCurrent(t *testing.T) {
	stale, err := Generate(moduleRoot, moduleRoot, SharedTargets(), SharedHeaders, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale shared outputs %v; run go run ./internal/model/cmd/protocolgen", stale)
	}
}

func TestSharedSetStaysInTheParentModule(t *testing.T) {
	// Shared inputs live under internal/model; shared outputs land in the
	// model or the WASM backend, never in another backend's tree.
	for _, target := range SharedTargets() {
		if !strings.HasPrefix(target.Template, "internal/model/") {
			t.Errorf("shared target %s reads a template outside internal/model: %s", target.Path, target.Template)
		}
		if !strings.HasPrefix(target.Path, "internal/model/") && !strings.HasPrefix(target.Path, "internal/backends/wasm/") && !strings.HasPrefix(target.Path, "rust/wasmhost/") {
			t.Errorf("shared target writes outside internal/model and the WASM backend: %s", target.Path)
		}
	}
	for _, path := range choiceImplementationSources {
		if !strings.HasPrefix(path, "internal/model/") {
			t.Errorf("choice implementation input is outside internal/model: %s", path)
		}
	}
}

func TestSharedOutputsCarryTheProtocolgenHeader(t *testing.T) {
	for _, target := range SharedTargets() {
		if path.Base(target.Template) == "iowire_test.go.tmpl" {
			// The I/O wire test template never carried a header; adding one
			// would also change the native gomadwire test it renders.
			continue
		}
		contents, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(target.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if first, _, _ := strings.Cut(string(contents), "\n"); first != SharedHeader {
			t.Errorf("%s header = %q", target.Path, first)
		}
	}
}

// sharedInputs lists every file the shared set reads.
func sharedInputs() []string {
	inputs := []string{
		"internal/model/schema/iowire.json",
		"internal/model/schema/modelwire.json",
		"internal/model/schema/processwire.json",
		"internal/model/schema/networkwire.json",
		"internal/model/schema/volumewire.json",
		"internal/model/choice/schema/choicewire.json",
		"internal/model/world/schema/snapshotwire.json",
	}
	inputs = append(inputs, choiceImplementationSources...)
	for _, target := range SharedTargets() {
		inputs = append(inputs, target.Template)
	}
	slices.Sort(inputs)
	return slices.Compact(inputs)
}

func copyInputs(t *testing.T, root string, inputs []string) {
	t.Helper()
	for _, relative := range inputs {
		contents, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerateWritesAndChecksEverySharedOutput(t *testing.T) {
	root := t.TempDir()
	copyInputs(t, root, sharedInputs())
	if stale, err := Generate(root, root, SharedTargets(), SharedHeaders, false); err != nil || stale != nil {
		t.Fatalf("generate = %v, %v", stale, err)
	}
	for _, target := range SharedTargets() {
		want, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(target.Path)))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(target.Path)))
		if err != nil || string(got) != string(want) {
			t.Fatalf("isolated generation of %s differs from the tree: %v", target.Path, err)
		}
	}
	// A second generation must preserve every byte, including Rust codec output.
	before := make(map[string]string)
	for _, target := range SharedTargets() {
		value, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(target.Path)))
		if err != nil {
			t.Fatal(err)
		}
		before[target.Path] = string(value)
	}
	if _, err := Generate(root, root, SharedTargets(), SharedHeaders, false); err != nil {
		t.Fatal(err)
	}
	for path, want := range before {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || string(got) != want {
			t.Fatalf("second generation differs at %s: %v", path, err)
		}
	}
	if stale, err := Generate(root, root, SharedTargets(), SharedHeaders, true); err != nil || len(stale) != 0 {
		t.Fatalf("check after generate = %v, %v", stale, err)
	}
	for _, target := range SharedTargets() {
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		current, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(current, "stale"...), 0o600); err != nil {
			t.Fatal(err)
		}
		stale, err := Generate(root, root, SharedTargets(), SharedHeaders, true)
		if err != nil || !slices.Equal(stale, []string{target.Path}) {
			t.Fatalf("check after editing %s = %v, %v", target.Path, stale, err)
		}
		if err := os.WriteFile(path, current, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	missing := SharedTargets()[0].Path
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(missing))); err != nil {
		t.Fatal(err)
	}
	if stale, err := Generate(root, root, SharedTargets(), SharedHeaders, true); err != nil || !slices.Equal(stale, []string{missing}) {
		t.Fatalf("check with %s missing = %v, %v", missing, stale, err)
	}
}

func TestGenerateReportsSchemaAndTemplateErrors(t *testing.T) {
	root := t.TempDir()
	copyInputs(t, root, sharedInputs())
	broken := filepath.Join(root, "internal", "model", "schema", "networkwire.json")
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if stale, err := Generate(root, root, SharedTargets(), SharedHeaders, true); err == nil || stale != nil {
		t.Fatalf("broken schema = %v, %v", stale, err)
	}
	if _, err := Generate(moduleRoot, moduleRoot, SharedTargets()[:1], nil, true); err == nil || !strings.Contains(err.Error(), "header") {
		t.Fatalf("missing header = %v", err)
	}
}

// CLI execution cannot prove that regeneration owns the binary codec.
func TestSharedSetIncludesLinkedChoiceCodec(t *testing.T) {
	for _, target := range SharedTargets() {
		if target.Path == "rust/wasmhost/src/model/choice_wire_generated.rs" && target.Family == FamilyChoice && target.Unformatted {
			return
		}
	}
	t.Fatal("shared generation omits the linked Rust choice codec")
}
