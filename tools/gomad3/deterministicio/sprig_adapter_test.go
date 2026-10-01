package deterministicio

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func sprigAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: sprigModulePath, Version: sprigVersion, Sum: sprigSum}
}

func TestSprigAdapterConsumer(t *testing.T) {
	downloadPinnedModule(t, sprigModulePath, sprigVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "sprig_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "sprig", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{sprigAdapterIdentity()}, []adapterImplementation{
		{module: sprigModulePath, prepare: prepareSprig},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := registry.prepare(target.Spec{
		Kind: target.KindGoTest, Source: ".", WorkingDir: workingDirectory, PreparationRoot: t.TempDir(),
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != sprigModulePath {
		t.Fatalf("Sprig consumer adapter selection = %#v", adapters)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	run := func(dir string, environment []string, args ...string) []byte {
		t.Helper()
		command := exec.CommandContext(t.Context(), goCommand, args...)
		command.Dir = dir
		command.Env = append(os.Environ(), append([]string{"GOWORK=off", "GOFLAGS="}, environment...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("prepared Sprig consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	output := run(workingDirectory, nil, "test", "-v", "-p=2", "-buildvcs=false", "-mod=readonly", "-modfile="+spec.BuildModFile, ".")
	t.Logf("prepared stock Sprig consumer:\n%s", output)
	for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
		environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
		output := run(workingDirectory, environment, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", sprigModulePath)
		var pkg struct {
			Dir     string
			GoFiles []string
		}
		if err := json.Unmarshal(output, &pkg); err != nil {
			t.Fatal(err)
		}
		slices.Sort(pkg.GoFiles)
		sources := make([]compatibility.Source, len(pkg.GoFiles))
		for index, name := range pkg.GoFiles {
			contents, err := os.ReadFile(filepath.Join(pkg.Dir, name))
			if err != nil {
				t.Fatal(err)
			}
			sources[index] = compatibility.Source{Name: name, SHA256: digestBytes(contents)}
		}
		if got := compatibility.DigestSources(sources); got != sprigPreparedSourceSetSHA256 {
			t.Fatalf("prepared Sprig source set on %s/%s = %s, want %s", platform.goos, platform.goarch, got, sprigPreparedSourceSetSHA256)
		}
		t.Logf("prepared %s/%s source set %s", platform.goos, platform.goarch, sprigPreparedSourceSetSHA256)
	}
}

func TestSprigRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: "github.com/Masterminds/sprig/v3/other", Version: sprigVersion, Sum: sprigSum},
		{Module: sprigModulePath, Version: "v0.0.0", Sum: sprigSum},
		{Module: sprigModulePath, Version: sprigVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareSprig(t.TempDir(), t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("changed Sprig identity %#v: %v", identity, err)
		}
	}
}

func TestSprigRewriteRejectsDrift(t *testing.T) {
	downloadPinnedModule(t, sprigModulePath, sprigVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "!masterminds", "sprig", "v3@"+sprigVersion)
	source, err := readAdapterSource(sprigModulePath, moduleRoot, sprigNetworkPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, want string
		change     func(*sourceRewrite, *[]byte)
	}{
		{name: "upstream-file", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
		{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) {
			rewrite.rewrites[0].anchor = []byte("absent resolver callback")
		}},
		{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("name") }},
		{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = sprigNetworkSourceSHA256 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			rewrite := sprigRewrites[0]
			rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			contents := append([]byte(nil), source...)
			test.change(&rewrite, &contents)
			if _, err := rewriteAdapterSource(sprigModulePath, rewrite, contents); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Sprig rewrite: %v, want %s", err, test.want)
			}
		})
	}
}

func TestSprigRewritePreservesComments(t *testing.T) {
	downloadPinnedModule(t, sprigModulePath, sprigVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "!masterminds", "sprig", "v3@"+sprigVersion)
	source, err := readAdapterSource(sprigModulePath, moduleRoot, sprigNetworkPath)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := rewriteAdapterSource(sprigModulePath, sprigRewrites[0], source)
	if err != nil {
		t.Fatal(err)
	}
	comments := func(contents []byte) []string {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), sprigNetworkPath, contents, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		var result []string
		for _, group := range file.Comments {
			for _, comment := range group.List {
				result = append(result, comment.Text)
			}
		}
		return result
	}
	if !reflect.DeepEqual(comments(source), comments(replacement)) {
		t.Fatal("Sprig adapter changed original comments")
	}
}

func TestSprigRejectsUnrewrittenInventoryDrift(t *testing.T) {
	downloadPinnedModule(t, sprigModulePath, sprigVersion)
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "github.com", "!masterminds", "sprig", "v3@"+sprigVersion)
	if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(pinnedModuleCache(t), "github.com", "!masterminds", "sprig", "v3@"+sprigVersion)
	if err := copyAdapterModule(source, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleRoot, "functions.go")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(contents, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareSprig(moduleCache, t.TempDir(), sprigAdapterIdentity()); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("changed Sprig module inventory: %v", err)
	}
}
