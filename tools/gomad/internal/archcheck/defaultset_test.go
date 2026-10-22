package archcheck_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/archcheck"
	"golang.org/x/mod/modfile"
)

// nestedModuleDirectories returns the in-tree directories root's go.mod
// replaces nested modules with.
func nestedModuleDirectories(t *testing.T, root string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatal(err)
	}
	var directories []string
	for _, replacement := range file.Replace {
		directory := filepath.ToSlash(filepath.Clean(filepath.FromSlash(replacement.New.Path)))
		if replacement.New.Version == "" && !filepath.IsAbs(replacement.New.Path) && directory != "." && !strings.HasPrefix(directory, "..") {
			directories = append(directories, directory)
		}
	}
	return directories
}

// defaultSetReferences lists default-set files naming a nested module's tree,
// exempting Markdown, go.mod/go.sum, Make files (see
// TestDefaultMakeTargetsKeepSharedValidation), and build outputs.
func defaultSetReferences(t *testing.T, root string) []string {
	t.Helper()
	needles := nestedModuleDirectories(t, root)
	var found []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if slices.Contains(needles, relative) || relative == ".toolchain" || relative == ".bin" || strings.HasPrefix(relative, "rust/target") {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") || name == "go.mod" || name == "go.sum" || name == "Makefile" || strings.HasSuffix(name, ".mk") || !entry.Type().IsRegular() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, needle := range needles {
			// Also catch the path spelled as separate filepath.Join arguments.
			split := strings.Join(strings.Split(needle, "/"), `", "`)
			if bytes.Contains(contents, []byte(needle)) || bytes.Contains(contents, []byte(split)) {
				found = append(found, relative+": "+needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestNoDefaultSourceNamesNestedTree(t *testing.T) {
	if len(nestedModuleDirectories(t, ".")) == 0 {
		if nested, _ := filepath.Glob("internal/backends/*/go.mod"); len(nested) != 0 {
			t.Fatalf("nested backend modules %v are not replaced by go.mod; the nested-path rule has no needle", nested)
		}
		t.Skip("no nested in-tree module is checked out; nothing to keep out of the default set")
	}
	for _, reference := range defaultSetReferences(t, ".") {
		t.Errorf("default-set file names a nested module tree: %s", reference)
	}
}

func TestNestedPathRuleRejectsNewOccurrence(t *testing.T) {
	root := t.TempDir()
	const nested = "internal/backends/legacy"
	files := map[string]string{
		"go.mod":                       "module example.invalid/rule\n\ngo 1.27.1\n\nreplace example.invalid/rule/legacy => ./" + nested + "\n",
		nested + "/go.mod":             "module example.invalid/rule/legacy\n",
		nested + "/inside.go":          "package legacy // " + nested + "\n",
		"README.md":                    nested + "\n",
		"Makefile":                     "x:\n\techo " + nested + "\n",
		"untagged.go":                  "package rule // " + nested + "\n",
		"joined.go":                    "package rule\nvar _ = []string{\"internal\", \"backends\", \"legacy\", \"x\"}\n",
		"testdata/corpus.json":         `{"path": "` + nested + `"}`,
		"internal/clean/clean_test.go": "package clean\n",
	}
	for name, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := defaultSetReferences(t, root)
	want := []string{"joined.go: " + nested, "testdata/corpus.json: " + nested, "untagged.go: " + nested}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("references = %q, want %q", got, want)
	}
}

func TestDefaultMakeTargetsKeepSharedValidation(t *testing.T) {
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-n", "generate", "validate", "test-wasm")
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "GOFLAGS=") && !strings.HasPrefix(value, "MAKEFLAGS=") {
			command.Env = append(command.Env, value)
		}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("expand default targets: %v: %s", err, output)
	}
	for _, forbidden := range nestedModuleDirectories(t, ".") {
		if bytes.Contains(output, []byte(forbidden)) {
			t.Errorf("default targets select %q:\n%s", forbidden, output)
		}
	}
	for _, required := range []string{"--locked --lib", "./internal/model/...", "./internal/backends/wasm/cluster", "./internal/backends/wasm/toolchain/...", "GOMAD_WASM_CONFORMANCE_REQUIRED=1"} {
		if !bytes.Contains(output, []byte(required)) {
			t.Errorf("validate omits retained conformance gate %q", required)
		}
	}
	if !bytes.Contains(output, []byte("go run ./internal/model/cmd/protocolgen -check")) {
		t.Errorf("validate does not check the shared protocol outputs:\n%s", output)
	}
}

// Inventory ownership cannot be observed through a public CLI fixture.
func TestInventorySeparatesRetainedGoFromRetiringHost(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                         "module example.invalid/retention\n\ngo 1.27.1\n",
		"internal/model/world/source.go": "package world\n",
		"internal/backends/wasm/toolchain/source.go": "package toolchain\n",
		"internal/archcheck/source.go":               "package archcheck\n",
		"internal/runner/source.go":                  "package runner\n",
		"cmd/gomad/source.go":                        "package main\nfunc main() {}\n",
	}
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	inventory, findings, err := archcheck.Discover(root, sourceGoCommand(), archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	var got struct{ RetainedGo, RetiringGo []string }
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	wantRetained := []string{"example.invalid/retention/internal/archcheck", "example.invalid/retention/internal/backends/wasm/toolchain", "example.invalid/retention/internal/model/world"}
	wantRetiring := []string{"example.invalid/retention/cmd/gomad", "example.invalid/retention/internal/runner"}
	for _, imported := range wantRetiring {
		if !slices.ContainsFunc(findings, func(f archcheck.Finding) bool { return f.Category == "retired-host-source" && f.Path == imported }) {
			t.Errorf("retired default source %s was admitted: %v", imported, findings)
		}
	}
	if !slices.Equal(got.RetainedGo, wantRetained) || !slices.Equal(got.RetiringGo, wantRetiring) {
		t.Fatalf("inventory retention = %+v, want retained %v, retiring %v", got, wantRetained, wantRetiring)
	}
}
