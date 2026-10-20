package archcheck_test

import (
	"bytes"
	"go/build/constraint"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

// nestedModuleDirectories returns the in-tree directories root's go.mod
// replaces nested modules with (today only the native backend).
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

// requiresNativeSelection: the build constraint is false whenever gomad_native is off.
func requiresNativeSelection(source []byte) bool {
	for _, line := range strings.Split(string(source), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if !constraint.IsGoBuild(line) {
			continue
		}
		expression, err := constraint.Parse(line)
		if err != nil {
			return false
		}
		var tags []string
		expression.Eval(func(tag string) bool {
			if tag != "gomad_native" && !slices.Contains(tags, tag) {
				tags = append(tags, tag)
			}
			return false
		})
		for assignment := range 1 << len(tags) {
			satisfied := expression.Eval(func(tag string) bool {
				index := slices.Index(tags, tag)
				return index >= 0 && assignment&(1<<index) != 0
			})
			if satisfied {
				return false
			}
		}
		return true
	}
	return false
}

// defaultSetReferences lists default-set files naming a nested module's tree,
// exempting Markdown, go.mod/go.sum, Make files (see
// TestDefaultMakeTargetsSelectNoNative), build outputs and gomad_native-only Go.
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
			if slices.Contains(needles, relative) || relative == ".toolchain" || relative == ".bin" || strings.HasPrefix(relative, "internal/backends/wasm/wasmhost/target") {
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
		if strings.HasSuffix(name, ".go") && requiresNativeSelection(contents) {
			return nil
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

func TestNoDefaultSourceNamesNativeTree(t *testing.T) {
	if len(nestedModuleDirectories(t, ".")) == 0 {
		if nested, _ := filepath.Glob("internal/backends/*/go.mod"); len(nested) != 0 {
			t.Fatalf("nested backend modules %v are not replaced by go.mod; the native-path rule has no needle", nested)
		}
		t.Skip("no nested in-tree module is checked out; nothing to keep out of the default set")
	}
	for _, reference := range defaultSetReferences(t, ".") {
		t.Errorf("default-set file names a nested module tree: %s", reference)
	}
}

func TestNativePathRuleRejectsNewOccurrence(t *testing.T) {
	root := t.TempDir()
	const nested = "internal/backends/legacy"
	files := map[string]string{
		"go.mod":                       "module example.invalid/rule\n\ngo 1.27.1\n\nreplace example.invalid/rule/legacy => ./" + nested + "\n",
		nested + "/go.mod":             "module example.invalid/rule/legacy\n",
		nested + "/inside.go":          "package legacy // " + nested + "\n",
		"README.md":                    nested + "\n",
		"Makefile":                     "x:\n\techo " + nested + "\n",
		"tagged.go":                    "//go:build gomad_native && linux\n\npackage rule // " + nested + "\n",
		"untagged.go":                  "package rule // " + nested + "\n",
		"joined.go":                    "package rule\nvar _ = []string{\"internal\", \"backends\", \"legacy\", \"x\"}\n",
		"default_only.go":              "//go:build !gomad_native\n\npackage rule // " + nested + "\n",
		"either.go":                    "//go:build gomad_native || linux\n\npackage rule // " + nested + "\n",
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
	want := []string{"default_only.go: " + nested, "either.go: " + nested, "joined.go: " + nested, "testdata/corpus.json: " + nested, "untagged.go: " + nested}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("references = %q, want %q", got, want)
	}
}

func TestDefaultMakeTargetsSelectNoNative(t *testing.T) {
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
	for _, forbidden := range append([]string{"gomad_native"}, nestedModuleDirectories(t, ".")...) {
		if bytes.Contains(output, []byte(forbidden)) {
			t.Errorf("default targets select %q:\n%s", forbidden, output)
		}
	}
	if !bytes.Contains(output, []byte("go run ./internal/model/cmd/protocolgen -check")) {
		t.Errorf("validate does not check the shared protocol outputs:\n%s", output)
	}
}
