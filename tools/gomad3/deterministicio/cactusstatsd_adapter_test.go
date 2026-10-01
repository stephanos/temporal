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

func cactusStatsDAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: cactusStatsDModulePath, Version: cactusStatsDVersion, Sum: cactusStatsDSum}
}

func TestCactusStatsDAdapterConsumer(t *testing.T) {
	downloadPinnedModule(t, cactusStatsDModulePath, cactusStatsDVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "cactusstatsd_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "cactusstatsd", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{cactusStatsDAdapterIdentity()}, []adapterImplementation{
		{module: cactusStatsDModulePath, prepare: prepareCactusStatsD},
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
	if len(adapters) != 1 || adapters[0].Module != cactusStatsDModulePath {
		t.Fatalf("StatsD consumer adapters = %#v", adapters)
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
			t.Fatalf("prepared StatsD consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	t.Logf("prepared stock StatsD consumer:\n%s", run(nil, "test", "-v", "-p=2", "-tags=test_dep", "-timeout=3m", "-buildvcs=false", "-mod=readonly", "-modfile="+spec.BuildModFile, "."))
	for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
		output := run([]string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", cactusStatsDModulePath+"/statsd")
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
		if got := compatibility.DigestSources(sources); got != cactusStatsDPreparedSourceSetSHA256 {
			t.Fatalf("prepared StatsD source set on %s/%s = %s, want %s", platform.goos, platform.goarch, got, cactusStatsDPreparedSourceSetSHA256)
		}
		t.Logf("prepared %s/%s source set %s", platform.goos, platform.goarch, cactusStatsDPreparedSourceSetSHA256)
	}
}

func TestCactusStatsDRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: "github.com/cactus/other", Version: cactusStatsDVersion, Sum: cactusStatsDSum},
		{Module: cactusStatsDModulePath, Version: "v5.0.0", Sum: cactusStatsDSum},
		{Module: cactusStatsDModulePath, Version: cactusStatsDVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareCactusStatsD(t.TempDir(), t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("changed StatsD identity %#v: %v", identity, err)
		}
	}
}

func TestCactusStatsDRewritePreservesCommentsAndRejectsDrift(t *testing.T) {
	downloadPinnedModule(t, cactusStatsDModulePath, cactusStatsDVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "cactus", "go-statsd-client", "v5@"+cactusStatsDVersion)
	for _, rewrite := range cactusStatsDRewrites {
		t.Run(rewrite.path, func(t *testing.T) {
			source, err := readAdapterSource(cactusStatsDModulePath, moduleRoot, rewrite.path)
			if err != nil {
				t.Fatal(err)
			}
			replacement, err := rewriteAdapterSource(cactusStatsDModulePath, rewrite, source)
			if err != nil {
				t.Fatal(err)
			}
			comments := func(contents []byte) []string {
				t.Helper()
				file, err := parser.ParseFile(token.NewFileSet(), rewrite.path, contents, parser.ParseComments)
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
				t.Fatal("StatsD adapter changed original comments")
			}
			for _, test := range []struct{ name, want string }{{"source", "source identity mismatch"}, {"anchor", "anchor mismatch"}, {"ambiguous", "anchor mismatch"}, {"replacement", "replacement identity mismatch"}} {
				t.Run(test.name, func(t *testing.T) {
					changed := rewrite
					changed.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
					contents := append([]byte(nil), source...)
					switch test.name {
					case "source":
						contents = append(contents, '\n')
					case "anchor":
						changed.rewrites[0].anchor = []byte("absent StatsD constructor")
					case "ambiguous":
						changed.rewrites[0].anchor = []byte("return")
					case "replacement":
						changed.replacementSHA256 = rewrite.sourceSHA256
					default:
						t.Fatalf("unknown drift case %q", test.name)
					}
					if _, err := rewriteAdapterSource(cactusStatsDModulePath, changed, contents); err == nil || !strings.Contains(err.Error(), test.want) {
						t.Fatalf("StatsD rewrite: %v, want %s", err, test.want)
					}
				})
			}
		})
	}
}

func TestCactusStatsDRejectsUnrewrittenInventoryDrift(t *testing.T) {
	downloadPinnedModule(t, cactusStatsDModulePath, cactusStatsDVersion)
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "github.com", "cactus", "go-statsd-client", "v5@"+cactusStatsDVersion)
	if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(pinnedModuleCache(t), "github.com", "cactus", "go-statsd-client", "v5@"+cactusStatsDVersion)
	if err := copyAdapterModule(source, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleRoot, "statsd", "client.go")
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
	if _, err := prepareCactusStatsD(moduleCache, t.TempDir(), cactusStatsDAdapterIdentity()); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("changed StatsD module inventory: %v", err)
	}
}
