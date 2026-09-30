package deterministicio

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// rewrittenModuleAdapters lists the adapters built on prepareRewrittenModule
// with the package whose prepared source set each one pins.
var rewrittenModuleAdapters = []struct {
	name, module, version, sum, preparedPackage string
	importPath                                  string
	prepare                                     func(string, string, gomadversion.AdapterIdentity) (adapterPreparation, error)
	rewrites                                    []sourceRewrite
	originalInventorySHA256                     string
	replacementInventorySHA256                  string
	preparedSourceSetSHA256                     string
	removed, retained                           []string
	// outsideServerGraph marks an adapter for a module the server does not
	// depend on; its prepared source set is reviewed through a fixture module.
	outsideServerGraph bool
}{
	{
		name: "fx", module: fxModulePath, version: fxVersion, sum: fxSum, preparedPackage: fxModulePath, importPath: fxModulePath,
		prepare: prepareFx, rewrites: fxRewrites,
		originalInventorySHA256: fxOriginalSourceInventorySHA256, replacementInventorySHA256: fxReplacementSourceInventorySHA256, preparedSourceSetSHA256: fxPreparedSourceSetSHA256,
		removed:  []string{"\"os/signal\"", "signal.Notify", "signal.Stop", "\"golang.org/x/sys/unix\"", "unix.SIG"},
		retained: []string{"func (recv *signalReceivers) Start() {", "recv.notify(recv.signals, os.Interrupt, _sigINT, _sigTERM)", "_sigTERM hostSignal = \"terminated\""},
	},
	{
		name: "temporal-sdk", module: temporalSDKModulePath, version: temporalSDKVersion, sum: temporalSDKSum, preparedPackage: temporalSDKModulePath + "/internal", importPath: temporalSDKModulePath + "/client",
		prepare: prepareTemporalSDK, rewrites: temporalSDKRewrites,
		originalInventorySHA256: temporalSDKOriginalSourceInventorySHA256, replacementInventorySHA256: temporalSDKReplacementSourceInventorySHA256, preparedSourceSetSHA256: temporalSDKPreparedSourceSetSHA256,
		removed:  []string{"\"os/signal\"", "\"syscall\"", "signal.Notify", "syscall.SIGTERM"},
		retained: []string{"func InterruptCh() <-chan interface{} {", "os.Hostname()"},
	},
	{
		name: "otel-sdk", module: otelSDKModulePath, version: otelSDKVersion, sum: otelSDKSum, preparedPackage: otelSDKModulePath + "/resource", importPath: otelSDKModulePath + "/resource",
		prepare: prepareOtelSDK, rewrites: otelSDKRewrites,
		originalInventorySHA256: otelSDKOriginalSourceInventorySHA256, replacementInventorySHA256: otelSDKReplacementSourceInventorySHA256, preparedSourceSetSHA256: otelSDKPreparedSourceSetSHA256,
		removed:  []string{"\"os/user\"", "user.Current", "\"golang.org/x/sys/unix\"", "unix.Uname", "unix.Utsname", "\"os/exec\"", "exec.CommandContext"},
		retained: []string{"func (processOwnerDetector) Detect(context.Context) (*Resource, error) {", "semconv.ProcessOwner(owner.Username)", "func platformOSDescription() (string, error) {", "func execCommand(string, ...string) (string, error) {"},
	},
	{
		name: "go-sockaddr", module: sockaddrModulePath, version: sockaddrVersion, sum: sockaddrSum, preparedPackage: sockaddrModulePath, importPath: sockaddrModulePath,
		prepare: prepareSockaddr, rewrites: sockaddrRewrites,
		originalInventorySHA256: sockaddrOriginalSourceInventorySHA256, replacementInventorySHA256: sockaddrReplacementSourceInventorySHA256, preparedSourceSetSHA256: sockaddrPreparedSourceSetSHA256,
		removed:            []string{"\"os/exec\"", "exec.Command", "exec.LookPath"},
		retained:           []string{"func (ri routeInfo) GetDefaultInterfaceName() (string, error) {", "func routeCommandOutput([]string) ([]byte, error) {"},
		outsideServerGraph: true,
	},
}

func TestRewrittenModuleInventoriesMatchPinnedModules(t *testing.T) {
	moduleCache := pinnedModuleCache(t)
	for _, adapter := range rewrittenModuleAdapters {
		if adapter.outsideServerGraph {
			downloadPinnedModule(t, adapter.module, adapter.version)
		}
		prepared, err := adapter.prepare(moduleCache, t.TempDir(), gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.version, Sum: adapter.sum})
		if err != nil {
			t.Fatalf("%s: %v", adapter.name, err)
		}
		if prepared.replacement != prepared.evidence.ReplacementRoot || prepared.evidence.Module != adapter.module || prepared.evidence.Version != adapter.version || prepared.evidence.Sum != adapter.sum {
			t.Fatalf("%s prepared adapter = %#v", adapter.name, prepared)
		}
		if prepared.evidence.OriginalSourceInventorySHA256 != adapter.originalInventorySHA256 || prepared.evidence.ReplacementSourceInventorySHA256 != adapter.replacementInventorySHA256 {
			t.Fatalf("%s adapter inventory evidence = %#v", adapter.name, prepared.evidence)
		}
		if prepared.evidence.PreparedPackage != adapter.preparedPackage || prepared.evidence.PreparedSourceSetSHA256 != adapter.preparedSourceSetSHA256 || prepared.evidence.SourceSHA256 != adapter.rewrites[0].sourceSHA256 || prepared.evidence.ReplacementSHA256 != adapter.rewrites[0].replacementSHA256 {
			t.Fatalf("%s prepared package evidence = %#v", adapter.name, prepared.evidence)
		}
		var rewritten strings.Builder
		for _, rewrite := range adapter.rewrites {
			contents, err := os.ReadFile(filepath.Join(prepared.replacement, filepath.FromSlash(rewrite.path)))
			if err != nil {
				t.Fatal(err)
			}
			if got := digestBytes(contents); got != rewrite.replacementSHA256 {
				t.Fatalf("%s replacement %s digest = %s, want %s", adapter.name, rewrite.path, got, rewrite.replacementSHA256)
			}
			rewritten.Write(contents)
		}
		for _, removed := range adapter.removed {
			if strings.Contains(rewritten.String(), removed) {
				t.Fatalf("%s rewritten sources retained %q", adapter.name, removed)
			}
		}
		for _, retained := range adapter.retained {
			if !strings.Contains(rewritten.String(), retained) {
				t.Fatalf("%s rewritten sources omitted %q", adapter.name, retained)
			}
		}
	}
}

func TestRewrittenModulesRejectChangedIdentity(t *testing.T) {
	moduleCache := pinnedModuleCache(t)
	for _, adapter := range rewrittenModuleAdapters {
		if adapter.outsideServerGraph {
			downloadPinnedModule(t, adapter.module, adapter.version)
		}
		identity := gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.version, Sum: "h1:changed"}
		if _, err := adapter.prepare(moduleCache, t.TempDir(), identity); err == nil {
			t.Fatalf("%s accepted a changed identity", adapter.name)
		}
	}
}

func TestRewriteAdapterSourceRejectsDrift(t *testing.T) {
	moduleCache := pinnedModuleCache(t)
	moduleRoot := filepath.Join(moduleCache, "go.uber.org", "fx@"+fxVersion)
	source, err := readAdapterSource(fxModulePath, moduleRoot, fxSignalPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rewriteAdapterSource(fxModulePath, fxRewrites[0], append(source, '\n')); err == nil {
		t.Fatal("rewriteAdapterSource() accepted changed source")
	}
	duplicated := sourceRewrite{path: fxSignalPath, sourceSHA256: fxSignalSourceSHA256, replacementSHA256: fxSignalReplacementSHA256, rewrites: []anchorRewrite{{anchor: []byte("\trecv.m.Lock()\n")}}}
	if _, err := rewriteAdapterSource(fxModulePath, duplicated, source); err == nil || !strings.Contains(err.Error(), "anchor mismatch") {
		t.Fatalf("rewriteAdapterSource() ambiguous anchor error = %v", err)
	}
	wrongOutput := sourceRewrite{path: fxSignalPath, sourceSHA256: fxSignalSourceSHA256, replacementSHA256: fxSignalSourceSHA256, rewrites: fxRewrites[0].rewrites}
	if _, err := rewriteAdapterSource(fxModulePath, wrongOutput, source); err == nil || !strings.Contains(err.Error(), "replacement identity mismatch") {
		t.Fatalf("rewriteAdapterSource() replacement digest error = %v", err)
	}
}

// TestRewrittenModulePreparedPackageSourceSetIdentity reviews each prepared
// package through the server module's dependency graph, so the recorded
// source-set pins are the ones a Temporal target observes.
func TestRewrittenModulePreparedPackageSourceSetIdentity(t *testing.T) {
	serverRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum"} {
		contents, err := os.ReadFile(filepath.Join(serverRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var imports strings.Builder
	for _, adapter := range rewrittenModuleAdapters {
		if adapter.outsideServerGraph {
			continue
		}
		imports.WriteString("import _ \"" + adapter.importPath + "\"\n")
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "main.go"), []byte("package main\n\n"+imports.String()+"\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := Default().PrepareBuildAdapters(target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	selected := map[string]bool{}
	for _, adapter := range adapters {
		selected[adapter.Module] = true
	}
	for _, adapter := range rewrittenModuleAdapters {
		if !selected[adapter.module] && !adapter.outsideServerGraph {
			t.Fatalf("%s adapter was not selected: %#v", adapter.name, adapters)
		}
	}
	if _, err := target.ReviewCapabilities(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
}

// TestRewrittenModuleOutsideServerGraphPreparedSourceSetIdentity reviews the
// adapters for modules the server does not depend on through a fixture module
// that does, so their prepared source-set pins match what such a target sees.
func TestRewrittenModuleOutsideServerGraphPreparedSourceSetIdentity(t *testing.T) {
	if sockaddrPreparedSourceSetSHA256 == "" {
		t.Skip("the address-library adapter is pinned for darwin/arm64 only")
	}
	fixture, err := filepath.Abs(filepath.Join("testdata", "sockaddr"))
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "main.go"} {
		contents, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	// PrepareTargetBuildAdapters downloads the pinned module, which the
	// server's own module cache does not hold.
	spec, adapters, err := Default().PrepareTargetBuildAdapters(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != sockaddrModulePath {
		t.Fatalf("fixture adapters = %#v", adapters)
	}
	if _, err := target.ReviewCapabilities(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
}

// downloadPinnedModule fetches an adapter module the server does not depend on
// into the pinned toolchain's module cache, from outside every module so no
// go.sum is rewritten.
func downloadPinnedModule(t *testing.T, module, version string) {
	t.Helper()
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "mod", "download", module+"@"+version)
	command.Dir = t.TempDir()
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("download %s@%s: %v\n%s", module, version, err, output)
	}
}
