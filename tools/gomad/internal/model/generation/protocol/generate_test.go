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
	stale, err := Generate(moduleRoot, SharedTargets(), SharedHeaders, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale shared outputs %v; run go run ./internal/model/cmd/protocolgen", stale)
	}
}

func TestSharedSetStaysInTheParentModule(t *testing.T) {
	for _, target := range SharedTargets() {
		for _, path := range []string{target.Template, target.Path} {
			if strings.HasPrefix(path, "internal/backends/native/") {
				t.Errorf("shared target %s reads or writes the native tree: %s", target.Path, path)
			}
		}
	}
	for _, path := range choiceImplementationSources {
		if strings.HasPrefix(path, "internal/backends/native/") {
			t.Errorf("choice implementation input is in the native tree: %s", path)
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
	if stale, err := Generate(root, SharedTargets(), SharedHeaders, false); err != nil || stale != nil {
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
	if stale, err := Generate(root, SharedTargets(), SharedHeaders, true); err != nil || len(stale) != 0 {
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
		stale, err := Generate(root, SharedTargets(), SharedHeaders, true)
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
	if stale, err := Generate(root, SharedTargets(), SharedHeaders, true); err != nil || !slices.Equal(stale, []string{missing}) {
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
	if stale, err := Generate(root, SharedTargets(), SharedHeaders, true); err == nil || stale != nil {
		t.Fatalf("broken schema = %v, %v", stale, err)
	}
	if _, err := Generate(moduleRoot, SharedTargets()[:1], nil, true); err == nil || !strings.Contains(err.Error(), "header") {
		t.Fatalf("missing header = %v", err)
	}
}
