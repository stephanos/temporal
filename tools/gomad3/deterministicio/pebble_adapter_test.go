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

func pebbleAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: pebbleModulePath, Version: pebbleVersion, Sum: pebbleSum}
}

func TestPebbleAdapterConsumer(t *testing.T) {
	downloadPinnedModule(t, pebbleModulePath, pebbleVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "pebble_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "pebble", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{pebbleAdapterIdentity()}, []adapterImplementation{
		{module: pebbleModulePath, prepare: preparePebble},
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
	if len(adapters) != 1 || adapters[0].Module != pebbleModulePath || adapters[0].PreparedSourceSetSHA256 != pebblePreparedSourceSetSHA256 {
		t.Fatalf("Pebble consumer adapter selection = %#v", adapters)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	run := func(environment []string, args ...string) []byte {
		t.Helper()
		command := exec.CommandContext(t.Context(), goCommand, args...)
		command.Dir = workingDirectory
		command.Env = append(os.Environ(), append([]string{"GOWORK=off", "GOFLAGS="}, environment...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("prepared Pebble consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	t.Logf("prepared stock Pebble consumer:\n%s", run(nil, "test", "-v", "-tags=gomad,hashicorpmetrics,integration,test_dep", "-p=2", "-mod=readonly", "-modfile="+spec.BuildModFile, "."))
	for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
		host := platform.goos + "/" + platform.goarch
		environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
		output := run(environment, "list", "-tags=gomad,hashicorpmetrics,integration,test_dep", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", pebbleModulePath+"/vfs")
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
		if got := compatibility.DigestSources(sources); got != pebblePreparedSourceSetSHA256ByHost[host] {
			t.Fatalf("prepared Pebble source set on %s = %s, want %s", host, got, pebblePreparedSourceSetSHA256ByHost[host])
		}
		binary := filepath.Join(t.TempDir(), "pebble.test")
		run(environment, "test", "-c", "-tags=gomad,hashicorpmetrics,integration,test_dep", "-p=2", "-mod=readonly", "-modfile="+spec.BuildModFile, "-o="+binary, ".")
		output = run(environment, "tool", "nm", binary)
		for _, name := range []string{"unixFile", "linuxFile", "linuxDir"} {
			if strings.Contains(string(output), pebbleModulePath+"/vfs.(*"+name+")") {
				t.Fatalf("prepared %s retains OS-backed VFS wrapper %s", host, name)
			}
		}
		t.Logf("prepared %s source set %s; linked native wrapper methods absent", host, pebblePreparedSourceSetSHA256ByHost[host])
	}
}

func TestPebbleRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: "github.com/cockroachdb/other", Version: pebbleVersion, Sum: pebbleSum},
		{Module: pebbleModulePath, Version: "v0.0.0-other", Sum: pebbleSum},
		{Module: pebbleModulePath, Version: pebbleVersion, Sum: "h1:changed"},
	} {
		if _, err := preparePebble(t.TempDir(), t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("changed Pebble identity %#v: %v", identity, err)
		}
	}
}

func TestPebbleRewriteRejectsDrift(t *testing.T) {
	downloadPinnedModule(t, pebbleModulePath, pebbleVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "cockroachdb", "pebble@"+pebbleVersion)
	for _, original := range pebbleRewrites {
		source, err := readAdapterSource(pebbleModulePath, moduleRoot, original.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range []struct {
			name, want string
			change     func(*sourceRewrite, *[]byte)
		}{
			{name: "upstream-file", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
			{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) {
				rewrite.rewrites[0].anchor = []byte("absent file constructor")
			}},
			{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("File") }},
			{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = original.sourceSHA256 }},
		} {
			t.Run(original.path+"/"+test.name, func(t *testing.T) {
				rewrite := original
				rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
				contents := append([]byte(nil), source...)
				test.change(&rewrite, &contents)
				if _, err := rewriteAdapterSource(pebbleModulePath, rewrite, contents); err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("Pebble rewrite: %v, want %s", err, test.want)
				}
			})
		}
	}
}

func TestPebbleRewritePreservesCommentsAndMemFS(t *testing.T) {
	downloadPinnedModule(t, pebbleModulePath, pebbleVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "cockroachdb", "pebble@"+pebbleVersion)
	prepared, err := preparePebble(pinnedModuleCache(t), t.TempDir(), pebbleAdapterIdentity())
	if err != nil {
		t.Fatal(err)
	}
	comments := func(name string, contents []byte) []string {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), name, contents, parser.ParseComments)
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
	for _, rewrite := range pebbleRewrites {
		source, err := readAdapterSource(pebbleModulePath, moduleRoot, rewrite.path)
		if err != nil {
			t.Fatal(err)
		}
		replacement, err := rewriteAdapterSource(pebbleModulePath, rewrite, source)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(comments(rewrite.path, source), comments(rewrite.path, replacement)) {
			t.Fatalf("Pebble adapter changed original comments in %s", rewrite.path)
		}
	}
	source, err := readAdapterSource(pebbleModulePath, moduleRoot, "vfs/mem_fs.go")
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := readAdapterSource(pebbleModulePath, prepared.replacement, "vfs/mem_fs.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != string(replacement) {
		t.Fatal("Pebble adapter changed MemFS implementation")
	}
}

func TestPebbleRejectsUnrewrittenInventoryDrift(t *testing.T) {
	downloadPinnedModule(t, pebbleModulePath, pebbleVersion)
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "github.com", "cockroachdb", "pebble@"+pebbleVersion)
	if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(pinnedModuleCache(t), "github.com", "cockroachdb", "pebble@"+pebbleVersion)
	if err := copyAdapterModule(source, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleRoot, "vfs", "mem_fs.go")
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
	if _, err := preparePebble(moduleCache, t.TempDir(), pebbleAdapterIdentity()); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("unrewritten Pebble inventory drift: %v", err)
	}
}
