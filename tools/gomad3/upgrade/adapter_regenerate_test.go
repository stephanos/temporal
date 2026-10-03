package upgrade

import (
	"bytes"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
)

func TestAdapterRegenerationDryRunLeavesCheckoutUntouched(t *testing.T) {
	checkout := t.TempDir()
	root := filepath.Join(checkout, "tools", "gomad3")
	path := filepath.Join(root, "toolchain", "version", "version.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	descriptor, err := os.ReadFile(filepath.Join("..", "toolchain", "version", "version.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, descriptor, 0o600); err != nil {
		t.Fatal(err)
	}
	before := readAdapterCheckoutTree(t, checkout)
	var stdout bytes.Buffer
	status, err := RegenerateAdapter(root, "github.com/Masterminds/sprig/v3", "v3.2.3", "", &stdout)
	if status != 0 || err != nil {
		t.Fatalf("dry run status=%d: %v", status, err)
	}
	if !strings.Contains(stdout.String(), "Approval SHA-256:") {
		t.Fatalf("dry run report = %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Approval SHA-256: sha256:1a47941c061adf15f45843b757d5ba22c139271a661c1e2595d628abe517d5fd") {
		t.Fatalf("dry run changed the reviewed Sprig digest: %s", stdout.String())
	}
	updated, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(updated, descriptor) {
		t.Fatalf("dry run changed descriptor: %v", err)
	}
	if after := readAdapterCheckoutTree(t, checkout); !maps.Equal(after, before) {
		t.Fatalf("dry run changed checkout tree: before=%v after=%v", before, after)
	}
	stdout.Reset()
	status, err = RegenerateAdapter(root, "github.com/Masterminds/sprig/v3", "v3.2.3", "sha256:wrong", &stdout)
	if status == 0 || err == nil || !strings.Contains(err.Error(), "approval does not match") {
		t.Fatalf("wrong approval status=%d: %v", status, err)
	}
	if after := readAdapterCheckoutTree(t, checkout); !maps.Equal(after, before) {
		t.Fatalf("wrong approval changed checkout tree: before=%v after=%v", before, after)
	}
}

func readAdapterCheckoutTree(t *testing.T, root string) map[string]string {
	t.Helper()
	entries := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			entries[relative] = "directory"
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries[relative] = string(contents)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestAdapterApprovalBindsUpstreamSourceAndAnchors(t *testing.T) {
	result := deterministicio.AdapterRegeneration{Module: "example.com/module", Version: "v1.2.3", Sum: "h1:example", Sources: []deterministicio.RegeneratedSource{{Path: "source.go", Source: []byte("original"), Replacement: []byte("adapted"), SourceSHA256: "sha256:source", ReplacementSHA256: "sha256:replacement"}}}
	baseline, err := adapterRegenerationApproval(result)
	if err != nil {
		t.Fatal(err)
	}
	result.Sources[0].Source = []byte("changed upstream source")
	changedSource, err := adapterRegenerationApproval(result)
	if err != nil {
		t.Fatal(err)
	}
	result.Sources[0].Source = []byte("original")
	result.Sources[0].Replacement = []byte("changed replacement")
	changedAnchor, err := adapterRegenerationApproval(result)
	if err != nil {
		t.Fatal(err)
	}
	if baseline == changedSource || baseline == changedAnchor {
		t.Fatalf("approval did not bind source and rewrite: %s, %s, %s", baseline, changedSource, changedAnchor)
	}
}

func TestEditAdapterPinsPreservesCommentsAndRequiresOldPin(t *testing.T) {
	source := []byte("package test\n\n// Preserve this comment.\nconst old = \"sha256:old\"\nvar prepared = map[string]string{\"darwin/arm64\": \"sha256:hostold\", \"linux/amd64\": \"sha256:hostold\"}\n")
	regenerated := deterministicio.AdapterRegeneration{Sources: []deterministicio.RegeneratedSource{{OldSourceSHA256: "sha256:old", SourceSHA256: "sha256:new"}}, PreparedSourceSets: map[string]string{"darwin/arm64": "sha256:darwin", "linux/amd64": "sha256:linux"}}
	result, err := editAdapterPins(source, regenerated, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "// Preserve this comment.") || !strings.Contains(string(result), "sha256:new") || !strings.Contains(string(result), "sha256:darwin") || !strings.Contains(string(result), "sha256:linux") {
		t.Fatalf("edited source = %s", result)
	}
	regenerated.Sources[0].OldSourceSHA256 = "sha256:missing"
	if _, err := editAdapterPins(source, regenerated, "", ""); err == nil || !strings.Contains(err.Error(), "missing pin") {
		t.Fatalf("missing pin error = %v", err)
	}
	regenerated.Sources[0].OldSourceSHA256 = "sha256:old"
	duplicate := append(append([]byte{}, source...), []byte("const second = \"sha256:old\"\n")...)
	if _, err := editAdapterPins(duplicate, regenerated, "", ""); err == nil || !strings.Contains(err.Error(), "duplicate pin") {
		t.Fatalf("duplicate pin error = %v", err)
	}
}

func TestAdapterPublicationRejectsCheckoutDrift(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "source.go")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := readPublicationFile(root, "source.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("other"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = publishAdapterFiles(root, []adapterPublicationFile{{Path: "source.go", Old: before, New: publicationContent{Present: true, Bytes: []byte("new")}}})
	if err == nil || !strings.Contains(err.Error(), "changed since staging") {
		t.Fatalf("publishAdapterFiles() error = %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "other" {
		t.Fatalf("checkout = %q, %v", contents, err)
	}
}

func TestAdapterPublicationRejectsUnchangedInputDrift(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "go.mod")
	if err := os.WriteFile(input, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := readPublicationFile(root, "go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = publishAdapterFilesWithSnapshot(root, []adapterPublicationFile{{Path: "generated.go", New: publicationContent{Present: true, Bytes: []byte("generated")}}}, map[string]publicationContent{"go.mod": before}, nil)
	if err == nil || !strings.Contains(err.Error(), "checkout changed since staging: go.mod") {
		t.Fatalf("unchanged input drift error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "generated.go")); !os.IsNotExist(err) {
		t.Fatalf("generated output unexpectedly published: %v", err)
	}
}

func TestAdapterPublicationRejectsNewStagedInput(t *testing.T) {
	for _, test := range []struct{ name, path string }{
		{"adapter request", filepath.Join("tools", "gomad3", "internal", "compatibilitypack", "requests", "new.json")},
		{"qualification", filepath.Join("tools", "gomad3integration", "qualification", "new.json")},
		{"tests", filepath.Join("tests", "new_test.go")},
	} {
		t.Run(test.name, func(t *testing.T) {
			testAdapterPublicationRejectsNewStagedInput(t, test.path)
		})
	}
}

func testAdapterPublicationRejectsNewStagedInput(t *testing.T, newInput string) {
	t.Helper()
	root := t.TempDir()
	paths := map[string]string{
		filepath.Join("tools", "gomad3", "deterministicio", "sprig_adapter.go"):                    "old adapter",
		filepath.Join("tools", "gomad3", "toolchain", "version", "generated.go"):                   "old generated",
		filepath.Join("tools", "gomad3", "internal", "compatibilitypack", "requests", "base.json"): "old request",
		filepath.Join("tools", "gomad3integration", "qualification", "tests.json"):                 "old manifest",
		filepath.Join("tests", "test.go"):                                                          "old test",
		"go.mod":                                                                                   "old module",
		"go.sum":                                                                                   "old sums",
	}
	snapshot := map[string]publicationContent{}
	for path, contents := range paths {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		snapshot[path] = publicationContent{Present: true, Bytes: []byte(contents)}
	}
	if err := os.WriteFile(filepath.Join(root, newInput), []byte("new input"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapterPath := filepath.Join("tools", "gomad3", "deterministicio", "sprig_adapter.go")
	generatedPath := filepath.Join("tools", "gomad3", "toolchain", "version", "generated.go")
	files := []adapterPublicationFile{
		{Path: adapterPath, Old: snapshot[adapterPath], New: publicationContent{Present: true, Bytes: []byte("new adapter")}},
		{Path: generatedPath, Old: snapshot[generatedPath], New: publicationContent{Present: true, Bytes: []byte("new generated")}},
	}
	err := publishAdapterFilesWithSnapshot(root, files, snapshot, adapterRegenerationInputTrees)
	if err == nil || !strings.Contains(err.Error(), "checkout changed since staging: "+newInput) {
		t.Fatalf("new staged input error = %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(root, adapterPath))
	if err != nil || string(contents) != "old adapter" {
		t.Fatalf("published adapter despite new input: %q, %v", contents, err)
	}
	contents, err = os.ReadFile(filepath.Join(root, generatedPath))
	if err != nil || string(contents) != "old generated" {
		t.Fatalf("published generated output despite new input: %q, %v", contents, err)
	}
	if _, err := os.Stat(adapterPublicationMarker(root)); !os.IsNotExist(err) {
		t.Fatalf("publication marker unexpectedly exists: %v", err)
	}
}

func TestAdapterPublicationAcceptsIgnoredCacheDrift(t *testing.T) {
	root := t.TempDir()
	snapshot := map[string]publicationContent{}
	for path, contents := range map[string]string{
		filepath.Join("tools", "gomad3", "deterministicio", "sprig_adapter.go"):    "old adapter",
		filepath.Join("tools", "gomad3integration", "qualification", "tests.json"): "old manifest",
		filepath.Join("tests", "test.go"):                                          "old test",
		"go.mod":                                                                   "old module",
		"go.sum":                                                                   "old sums",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		snapshot[path] = publicationContent{Present: true, Bytes: []byte(contents)}
	}
	for _, path := range []string{
		filepath.Join("tools", "gomad3", ".toolchain", "generator-cache", "new"),
		filepath.Join("tools", "gomad3", ".bin", "gomad"),
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("cache"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	adapterPath := filepath.Join("tools", "gomad3", "deterministicio", "sprig_adapter.go")
	files := []adapterPublicationFile{{Path: adapterPath, Old: snapshot[adapterPath], New: publicationContent{Present: true, Bytes: []byte("new adapter")}}}
	if err := publishAdapterFilesWithSnapshot(root, files, snapshot, adapterRegenerationInputTrees); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(root, adapterPath))
	if err != nil || string(contents) != "new adapter" {
		t.Fatalf("published adapter = %q, %v", contents, err)
	}
	if _, err := os.Stat(adapterPublicationMarker(root)); !os.IsNotExist(err) {
		t.Fatalf("publication marker remains: %v", err)
	}
}

func TestAdapterPublicationRecoversInterruptedSet(t *testing.T) {
	root := t.TempDir()
	names := []string{filepath.Join("tools", "gomad3", "deterministicio", "sprig_adapter.go"), filepath.Join("tools", "gomad3integration", "qualification", "tests.json")}
	for _, name := range names {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files := []adapterPublicationFile{}
	for _, name := range names {
		files = append(files, adapterPublicationFile{Path: name, Old: publicationContent{Present: true, Bytes: []byte("old")}, New: publicationContent{Present: true, Bytes: []byte("new")}})
	}
	if err := recordAdapterPublication(root, files); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, names[0]), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recoverAdapterPublication(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		contents, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(contents) != "old" {
			t.Fatalf("recovered %s = %q, %v", name, contents, err)
		}
	}
	if _, err := os.Stat(adapterPublicationMarker(root)); !os.IsNotExist(err) {
		t.Fatalf("recovery marker remains: %v", err)
	}
}

func TestReportStaleLibcAdapterPacks(t *testing.T) {
	var output bytes.Buffer
	if err := reportStaleAdapterPacks(&output, "modernc.org/libc", "v1.72.4"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Stale adapter-bound pack: modernc-libc") {
		t.Fatalf("stale libc pack report = %s", output.String())
	}
}

func TestAdapterPublicationSerializesCompetingApplies(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "source.go")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, next := range []string{"first", "second"} {
		workers.Add(1)
		go func(next string) {
			defer workers.Done()
			<-start
			results <- publishAdapterFiles(root, []adapterPublicationFile{{Path: "source.go", Old: publicationContent{Present: true, Bytes: []byte("old")}, New: publicationContent{Present: true, Bytes: []byte(next)}}})
		}(next)
	}
	close(start)
	workers.Wait()
	close(results)
	successes, failures := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	contents, err := os.ReadFile(path)
	if err != nil || successes != 1 || failures != 1 || string(contents) != "first" && string(contents) != "second" {
		t.Fatalf("competing publications: successes=%d failures=%d contents=%q error=%v", successes, failures, contents, err)
	}
}

func TestAdapterGenerationFailureLeavesCheckoutUnchanged(t *testing.T) {
	checkout := t.TempDir()
	moduleRoot := filepath.Join(checkout, "tools", "gomad3")
	for _, directory := range []string{filepath.Join(moduleRoot, "deterministicio", "testdata"), filepath.Join(moduleRoot, "toolchain", "version"), filepath.Join(checkout, "tools", "gomad3integration", "qualification"), filepath.Join(checkout, "tests")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, contents string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(checkout, "go.mod"), "module example.com/root\n")
	write(filepath.Join(checkout, "go.sum"), "")
	write(filepath.Join(checkout, "tools", "gomad3integration", "qualification", "tests.json"), "old manifest")
	write(filepath.Join(moduleRoot, "Makefile"), "generate:\n\tfalse\n")
	adapter := filepath.Join(moduleRoot, "deterministicio", "sprig_adapter.go")
	oldAdapter := "package deterministicio\nconst version = \"v3.3.0\"\nconst sum = \"h1:old\"\nvar pins = map[string]string{\"darwin/arm64\": \"sha256:old\", \"linux/amd64\": \"sha256:old\"}\n"
	write(adapter, oldAdapter)
	versionPath := filepath.Join(moduleRoot, "toolchain", "version", "version.json")
	write(versionPath, "{\"adapters\":[{\"module\":\"github.com/Masterminds/sprig/v3\",\"version\":\"v3.3.0\",\"sum\":\"h1:old\"}]}")
	before := readAdapterCheckoutTree(t, checkout)
	result := deterministicio.AdapterRegeneration{Module: "github.com/Masterminds/sprig/v3", SourceFile: "sprig_adapter.go", Version: "v3.2.3", Sum: "h1:new", PreparedSourceSets: map[string]string{"darwin/arm64": "sha256:new", "linux/amd64": "sha256:new"}}
	_, _, err := stageAdapterRegeneration(moduleRoot, result, "v3.3.0", "h1:old", "h1:oldmod", "h1:newmod", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "generate staged adapter artifacts") {
		t.Fatalf("stageAdapterRegeneration() error = %v", err)
	}
	contents, err := os.ReadFile(adapter)
	if err != nil || string(contents) != oldAdapter {
		t.Fatalf("checkout adapter = %q, %v", contents, err)
	}
	contents, err = os.ReadFile(versionPath)
	if err != nil || !strings.Contains(string(contents), "v3.3.0") {
		t.Fatalf("checkout descriptor = %q, %v", contents, err)
	}
	if after := readAdapterCheckoutTree(t, checkout); !maps.Equal(after, before) {
		t.Fatalf("generation failure changed checkout tree: before=%v after=%v", before, after)
	}
}
