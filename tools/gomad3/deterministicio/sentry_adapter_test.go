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

func sentryAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: sentryModulePath, Version: sentryVersion, Sum: sentrySum}
}

func TestSentryAdapterConsumer(t *testing.T) {
	downloadPinnedModule(t, sentryModulePath, sentryVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "sentry_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "sentry", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{sentryAdapterIdentity()}, []adapterImplementation{
		{module: sentryModulePath, prepare: prepareSentry},
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
	if len(adapters) != 1 || adapters[0].Module != sentryModulePath {
		t.Fatalf("Sentry consumer adapter selection = %#v", adapters)
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
			t.Fatalf("prepared Sentry consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	output := run(workingDirectory, nil, "test", "-v", "-p=2", "-buildvcs=false", "-mod=readonly", "-modfile="+spec.BuildModFile, ".")
	t.Logf("prepared stock Sentry consumer:\n%s", output)
	output = run(adapters[0].ReplacementRoot, nil, "test", "-v", "-p=2", "-mod=readonly", "-run=^TestRevisionFromBuildInfo", ".")
	t.Logf("prepared upstream build-info tests:\n%s", output)
	for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
		environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
		output := run(workingDirectory, environment, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-deps", "-f={{.ImportPath}}", sentryModulePath)
		for _, path := range strings.Fields(string(output)) {
			if path == "golang.org/x/sys/execabs" {
				t.Fatalf("prepared Sentry %s/%s retains subprocess dependency %s", platform.goos, platform.goarch, path)
			}
		}
		output = run(workingDirectory, environment, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-deps", `-f={{if not .Standard}}{{.ImportPath}} {{join .Imports " "}}{{end}}`, sentryModulePath)
		for _, line := range strings.Split(string(output), "\n") {
			for _, path := range strings.Fields(line) {
				if path == "os/exec" || path == "golang.org/x/sys/execabs" {
					t.Fatalf("prepared Sentry %s/%s nonstandard subprocess edge: %s", platform.goos, platform.goarch, line)
				}
			}
		}
		output = run(workingDirectory, environment, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", sentryModulePath)
		var pkg struct {
			Dir     string
			GoFiles []string
			Imports []string
		}
		if err := json.Unmarshal(output, &pkg); err != nil {
			t.Fatal(err)
		}
		for _, path := range pkg.Imports {
			if path == "os/exec" || path == "golang.org/x/sys/execabs" {
				t.Fatalf("prepared Sentry %s/%s directly imports subprocess package %s", platform.goos, platform.goarch, path)
			}
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
		if got := compatibility.DigestSources(sources); got != sentryPreparedSourceSetSHA256 {
			t.Fatalf("prepared Sentry source set on %s/%s = %s, want %s", platform.goos, platform.goarch, got, sentryPreparedSourceSetSHA256)
		}
		t.Logf("prepared %s/%s source set %s; Sentry subprocess imports and execabs closure absent", platform.goos, platform.goarch, sentryPreparedSourceSetSHA256)
	}
}

func TestSentryRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: "github.com/getsentry/other", Version: sentryVersion, Sum: sentrySum},
		{Module: sentryModulePath, Version: "v0.45.0", Sum: sentrySum},
		{Module: sentryModulePath, Version: sentryVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareSentry(t.TempDir(), t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("changed Sentry identity %#v: %v", identity, err)
		}
	}
}

func TestSentryRewriteRejectsDrift(t *testing.T) {
	downloadPinnedModule(t, sentryModulePath, sentryVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "getsentry", "sentry-go@"+sentryVersion)
	source, err := readAdapterSource(sentryModulePath, moduleRoot, sentryUtilPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, want string
		change     func(*sourceRewrite, *[]byte)
	}{
		{name: "upstream-file", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
		{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("absent Git import") }},
		{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("release") }},
		{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = sentryUtilSourceSHA256 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			rewrite := sentryRewrites[0]
			rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			contents := append([]byte(nil), source...)
			test.change(&rewrite, &contents)
			if _, err := rewriteAdapterSource(sentryModulePath, rewrite, contents); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Sentry rewrite: %v, want %s", err, test.want)
			}
		})
	}
}

func TestSentryRewritePreservesComments(t *testing.T) {
	downloadPinnedModule(t, sentryModulePath, sentryVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "getsentry", "sentry-go@"+sentryVersion)
	source, err := readAdapterSource(sentryModulePath, moduleRoot, sentryUtilPath)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := rewriteAdapterSource(sentryModulePath, sentryRewrites[0], source)
	if err != nil {
		t.Fatal(err)
	}
	comments := func(contents []byte) []string {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), sentryUtilPath, contents, parser.ParseComments)
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
		t.Fatal("Sentry adapter changed original comments")
	}
}

func TestSentryRejectsUnrewrittenInventoryDrift(t *testing.T) {
	downloadPinnedModule(t, sentryModulePath, sentryVersion)
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "github.com", "getsentry", "sentry-go@"+sentryVersion)
	if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(pinnedModuleCache(t), "github.com", "getsentry", "sentry-go@"+sentryVersion)
	if err := copyAdapterModule(source, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleRoot, "client.go")
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
	if _, err := prepareSentry(moduleCache, t.TempDir(), sentryAdapterIdentity()); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("changed Sentry module inventory: %v", err)
	}
}
