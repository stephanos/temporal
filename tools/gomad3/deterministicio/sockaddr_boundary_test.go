package deterministicio

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func sockaddrBoundaryAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: sockaddrModulePath, Version: sockaddrVersion, Sum: sockaddrSum}
}

func TestSockaddrBoundaryConsumer(t *testing.T) {
	downloadPinnedModule(t, sockaddrModulePath, sockaddrVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "main.go", "boundary_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "sockaddr", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{sockaddrBoundaryAdapterIdentity()}, []adapterImplementation{
		{module: sockaddrModulePath, prepare: prepareSockaddr},
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
	if len(adapters) != 1 || adapters[0].Module != sockaddrModulePath {
		t.Fatalf("sockaddr consumer adapter selection = %#v", adapters)
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
			t.Fatalf("prepared sockaddr consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	output := run(workingDirectory, nil, "test", "-v", "-p=2", "-buildvcs=false", "-mod=readonly", "-modfile="+spec.BuildModFile, ".")
	t.Logf("prepared stock sockaddr consumer:\n%s", output)
	output = run(adapters[0].ReplacementRoot, nil, "test", "-v", "-p=2", "-mod=readonly", "-run=^Test(SockAddr_IPv[46]Addr|IPv[46])", ".")
	t.Logf("prepared upstream address computations:\n%s", output)
	for _, platform := range []struct{ goos, goarch, pin string }{
		{"darwin", "arm64", "sha256:05bad9b7f5550962a542a4b8c7101d135b1d0fdf199f2c01ce4cc9f33ab96f45"}, {"linux", "amd64", "sha256:08ac1f34ac338d5d5090f13f6a9dbed65cadb17c473cf6ed1f9b6b4813990b43"},
	} {
		environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
		output := run(workingDirectory, environment, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", sockaddrModulePath)
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
		if got := compatibility.DigestSources(sources); got != platform.pin {
			t.Fatalf("prepared sockaddr source set on %s/%s = %s, want %s", platform.goos, platform.goarch, got, platform.pin)
		}
		t.Logf("prepared %s/%s source set %s", platform.goos, platform.goarch, platform.pin)
	}
}

func TestSockaddrBoundaryRewritesRejectDrift(t *testing.T) {
	downloadPinnedModule(t, sockaddrModulePath, sockaddrVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "hashicorp", "go-sockaddr@"+sockaddrVersion)
	for _, original := range sockaddrRewrites {
		t.Run(original.path, func(t *testing.T) {
			source, err := readAdapterSource(sockaddrModulePath, moduleRoot, original.path)
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				name, want string
				change     func(*sourceRewrite, *[]byte)
			}{
				{name: "upstream-file", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
				{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("absent boundary anchor") }},
				{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("func") }},
				{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = original.sourceSHA256 }},
			} {
				t.Run(test.name, func(t *testing.T) {
					rewrite := original
					rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
					contents := append([]byte(nil), source...)
					test.change(&rewrite, &contents)
					if _, err := rewriteAdapterSource(sockaddrModulePath, rewrite, contents); err == nil || !strings.Contains(err.Error(), test.want) {
						t.Fatalf("sockaddr rewrite: %v, want %s", err, test.want)
					}
				})
			}
		})
	}
}

func TestSockaddrBoundaryPreservesOriginalComments(t *testing.T) {
	downloadPinnedModule(t, sockaddrModulePath, sockaddrVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "hashicorp", "go-sockaddr@"+sockaddrVersion)
	for _, rewrite := range sockaddrRewrites {
		t.Run(rewrite.path, func(t *testing.T) {
			source, err := readAdapterSource(sockaddrModulePath, moduleRoot, rewrite.path)
			if err != nil {
				t.Fatal(err)
			}
			replacement, err := rewriteAdapterSource(sockaddrModulePath, rewrite, source)
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
			remaining := comments(replacement)
			for _, want := range comments(source) {
				index := slices.Index(remaining, want)
				if index < 0 {
					t.Fatalf("sockaddr adapter removed or reordered original comment %q", want)
				}
				remaining = remaining[index+1:]
			}
		})
	}
}
