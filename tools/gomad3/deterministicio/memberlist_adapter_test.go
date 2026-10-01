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
	"golang.org/x/mod/modfile"
)

func memberlistAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: memberlistModulePath, Version: memberlistVersion, Sum: memberlistSum}
}

func TestMemberlistAdapterConsumer(t *testing.T) {
	downloadPinnedModule(t, memberlistModulePath, memberlistVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "memberlist_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "memberlist", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{memberlistAdapterIdentity()}, []adapterImplementation{
		{module: memberlistModulePath, prepare: prepareMemberlist},
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
	if len(adapters) != 1 || adapters[0].Module != memberlistModulePath {
		t.Fatalf("Memberlist consumer adapters = %#v", adapters)
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
			t.Fatalf("prepared Memberlist consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	t.Logf("prepared stock Memberlist consumer:\n%s", run(nil, "test", "-v", "-p=2", "-tags=test_dep", "-timeout=3m", "-buildvcs=false", "-mod=readonly", "-modfile="+spec.BuildModFile, "."))
	for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
		output := run([]string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", memberlistModulePath)
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
		if got := compatibility.DigestSources(sources); got != memberlistPreparedSourceSetSHA256 {
			t.Fatalf("prepared Memberlist source set on %s/%s = %s, want %s", platform.goos, platform.goarch, got, memberlistPreparedSourceSetSHA256)
		}
		t.Logf("prepared %s/%s source set %s", platform.goos, platform.goarch, memberlistPreparedSourceSetSHA256)
	}
}

func TestMemberlistRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: "github.com/hashicorp/other", Version: memberlistVersion, Sum: memberlistSum},
		{Module: memberlistModulePath, Version: "v0.5.3", Sum: memberlistSum},
		{Module: memberlistModulePath, Version: memberlistVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareMemberlist(t.TempDir(), t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("changed Memberlist identity %#v: %v", identity, err)
		}
	}
}

func TestMemberlistRewritePreservesCommentsAndRejectsDrift(t *testing.T) {
	downloadPinnedModule(t, memberlistModulePath, memberlistVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "hashicorp", "memberlist@"+memberlistVersion)
	for _, rewrite := range memberlistRewrites {
		t.Run(rewrite.path, func(t *testing.T) {
			source, err := readAdapterSource(memberlistModulePath, moduleRoot, rewrite.path)
			if err != nil {
				t.Fatal(err)
			}
			replacement, err := rewriteAdapterSource(memberlistModulePath, rewrite, source)
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
				t.Fatal("Memberlist adapter changed original comments")
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
						changed.rewrites[0].anchor = []byte("absent Memberlist constructor")
					case "ambiguous":
						changed.rewrites[0].anchor = []byte("return")
					case "replacement":
						changed.replacementSHA256 = rewrite.sourceSHA256
					default:
						t.Fatalf("unknown drift case %q", test.name)
					}
					if _, err := rewriteAdapterSource(memberlistModulePath, changed, contents); err == nil || !strings.Contains(err.Error(), test.want) {
						t.Fatalf("Memberlist rewrite: %v, want %s", err, test.want)
					}
				})
			}
		})
	}
}

func TestMemberlistRejectsUnrewrittenInventoryDrift(t *testing.T) {
	downloadPinnedModule(t, memberlistModulePath, memberlistVersion)
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "github.com", "hashicorp", "memberlist@"+memberlistVersion)
	if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(pinnedModuleCache(t), "github.com", "hashicorp", "memberlist@"+memberlistVersion)
	if err := copyAdapterModule(source, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleRoot, "config.go")
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
	if _, err := prepareMemberlist(moduleCache, t.TempDir(), memberlistAdapterIdentity()); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("changed Memberlist module inventory: %v", err)
	}
}

func TestMemberlistSuppliedTCPConsumer(t *testing.T) {
	workingDirectory := os.Getenv("GOMAD_MEMBERLIST_TCP_CONSUMER_DIR")
	if workingDirectory == "" {
		t.Skip("set GOMAD_MEMBERLIST_TCP_CONSUMER_DIR to a consumer checkout for its real TCP membership lifecycle")
	}
	modulePath := os.Getenv("GOMAD_MEMBERLIST_TCP_CONSUMER_MODULE")
	moduleDirectory := filepath.Join(workingDirectory, modulePath)
	downloadPinnedModule(t, memberlistModulePath, memberlistVersion)
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{memberlistAdapterIdentity()}, []adapterImplementation{
		{module: memberlistModulePath, prepare: prepareMemberlist},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := registry.prepare(target.Spec{
		Kind: target.KindGoTest, Source: "./testutil", WorkingDir: moduleDirectory, PreparationRoot: t.TempDir(),
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != memberlistModulePath {
		t.Fatalf("TCP consumer adapters = %#v", adapters)
	}
	contents, err := os.ReadFile(spec.BuildModFile)
	if err != nil {
		t.Fatal(err)
	}
	moduleFile, err := modfile.Parse(spec.BuildModFile, contents, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range moduleFile.Replace {
		if replacement.New.Version == "" && !filepath.IsAbs(replacement.New.Path) {
			if err := moduleFile.AddReplace(replacement.Old.Path, replacement.Old.Version, filepath.Join(spec.WorkingDir, replacement.New.Path), ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	contents, err = moduleFile.Format()
	if err != nil {
		t.Fatal(err)
	}
	wrapperModFile := filepath.Join(t.TempDir(), "wrapper.mod")
	if err := os.WriteFile(wrapperModFile, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile(strings.TrimSuffix(spec.BuildModFile, ".mod") + ".sum")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(strings.TrimSuffix(wrapperModFile, ".mod")+".sum", sums, 0o600); err != nil {
		t.Fatal(err)
	}
	spec.BuildModFile = wrapperModFile
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selection := exec.CommandContext(t.Context(), goCommand, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", memberlistModulePath)
	selection.Dir = spec.WorkingDir
	selection.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := selection.CombinedOutput()
	if err != nil {
		t.Fatalf("select prepared Memberlist in real TCP consumer: %v\n%s", err, output)
	}
	var pkg struct {
		Dir    string
		Module struct{ Path, Version string }
	}
	if err := json.Unmarshal(output, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Dir != adapters[0].ReplacementRoot || pkg.Module.Path != memberlistModulePath || pkg.Module.Version != memberlistVersion {
		t.Fatalf("real TCP consumer selected %#v; want %s@%s from %s", pkg, memberlistModulePath, memberlistVersion, adapters[0].ReplacementRoot)
	}
	t.Logf("real TCP consumer selected %s@%s, replacement inventory %s, source set %s", pkg.Module.Path, pkg.Module.Version, adapters[0].ReplacementSourceInventorySHA256, adapters[0].PreparedSourceSetSHA256)
	mise, err := exec.LookPath("mise")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), mise, "run", "test", "--run", "^TestTCPTransport_MembershipUpdateFailureAndRejoin$", "--tags=test_dep,hashicorpmetrics", "-t", "3m", "-o", "fn107-memberlist-tcp.log", "./"+filepath.ToSlash(filepath.Join(modulePath, "testutil")))
	command.Dir = workingDirectory
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-p=2 -modfile="+spec.BuildModFile)
	output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("prepared real TCP memberlist lifecycle: %v\n%s", err, output)
	}
	t.Logf("prepared real TCP memberlist lifecycle:\n%s", output)
}
