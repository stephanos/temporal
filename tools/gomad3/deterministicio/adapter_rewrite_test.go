package deterministicio

import (
	"bytes"
	"context"
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
	// The fields below drive the shared adapter test template; an adapter
	// without cacheDir is covered only by the tests above the template.
	//
	// cacheDir is the module's escaped module-cache directory without its
	// version, unrewritten a module file no rewrite touches, and
	// ambiguousAnchor a text that occurs more than once in every rewritten
	// file. otherModule and otherVersion are changed identities the adapter
	// must reject.
	cacheDir, unrewritten, ambiguousAnchor string
	otherModule, otherVersion              string
	// rewriteDropsComments marks an adapter whose rewrite removes code
	// together with its comments, so original comments are not kept.
	rewriteDropsComments          bool
	preparedSourceSetSHA256ByHost map[string]string
	// consumer is the stock consumer fixture under testdata/<name>.
	consumer *adapterConsumer
}{
	{
		name: "sprig", module: sprigModulePath, version: sprigVersion, sum: sprigSum, preparedPackage: sprigModulePath, importPath: sprigModulePath,
		prepare: prepareSprig, rewrites: sprigRewrites,
		originalInventorySHA256: sprigOriginalSourceInventorySHA256, replacementInventorySHA256: sprigReplacementSourceInventorySHA256, preparedSourceSetSHA256: sprigPreparedSourceSetSHA256,
		cacheDir: "github.com/!masterminds/sprig/v3", unrewritten: "functions.go", ambiguousAnchor: "name",
		otherModule: "github.com/Masterminds/sprig/v3/other", otherVersion: "v0.0.0",
		preparedSourceSetSHA256ByHost: sprigPreparedSourceSetSHA256ByHost,
		consumer:                      &adapterConsumer{testFlags: []string{"-p=2", "-buildvcs=false"}},
	},
	{
		name: "validator", module: validatorModulePath, version: validatorVersion, sum: validatorSum, preparedPackage: validatorModulePath, importPath: validatorModulePath,
		prepare: prepareValidator, rewrites: validatorRewrites,
		originalInventorySHA256: validatorOriginalSourceInventorySHA256, replacementInventorySHA256: validatorReplacementSourceInventorySHA256, preparedSourceSetSHA256: validatorPreparedSourceSetSHA256,
		outsideServerGraph: true,
		cacheDir:           "github.com/go-playground/validator/v10", unrewritten: "validator.go", ambiguousAnchor: "fl",
		otherModule: "github.com/go-playground/validator/v10/other", otherVersion: "v0.0.0",
		preparedSourceSetSHA256ByHost: validatorPreparedSourceSetSHA256ByHost,
		consumer:                      &adapterConsumer{testFlags: []string{"-p=2", "-buildvcs=false"}},
	},
	{
		name: "pebble", module: pebbleModulePath, version: pebbleVersion, sum: pebbleSum, preparedPackage: pebbleModulePath + "/vfs", importPath: pebbleModulePath + "/vfs",
		prepare: preparePebble, rewrites: pebbleRewrites,
		originalInventorySHA256: pebbleOriginalSourceInventorySHA256, replacementInventorySHA256: pebbleReplacementSourceInventorySHA256, preparedSourceSetSHA256: pebblePreparedSourceSetSHA256,
		outsideServerGraph: true,
		// The MemFS implementation is the unrewritten file, so preparation
		// must copy it unchanged.
		cacheDir: "github.com/cockroachdb/pebble", unrewritten: "vfs/mem_fs.go", ambiguousAnchor: "File",
		otherModule: "github.com/cockroachdb/other", otherVersion: "v0.0.0-other",
		preparedSourceSetSHA256ByHost: pebblePreparedSourceSetSHA256ByHost,
		consumer: &adapterConsumer{
			testFlags: []string{"-tags=gomad,hashicorpmetrics,integration,test_dep", "-p=2"},
			listFlags: []string{"-tags=gomad,hashicorpmetrics,integration,test_dep"},
			platform:  pebbleConsumerLinksNoNativeVFS,
		},
	},
	{
		name: "cactusstatsd", module: cactusStatsDModulePath, version: cactusStatsDVersion, sum: cactusStatsDSum, preparedPackage: cactusStatsDModulePath + "/statsd", importPath: cactusStatsDModulePath + "/statsd",
		prepare: prepareCactusStatsD, rewrites: cactusStatsDRewrites,
		originalInventorySHA256: cactusStatsDOriginalSourceInventorySHA256, replacementInventorySHA256: cactusStatsDReplacementSourceInventorySHA256, preparedSourceSetSHA256: cactusStatsDPreparedSourceSetSHA256,
		cacheDir: "github.com/cactus/go-statsd-client/v5", unrewritten: "statsd/client.go", ambiguousAnchor: "return",
		otherModule: "github.com/cactus/other", otherVersion: "v5.0.0",
		preparedSourceSetSHA256ByHost: cactusStatsDPreparedSourceSetSHA256ByHost,
		consumer:                      &adapterConsumer{testFlags: []string{"-p=2", "-tags=test_dep", "-timeout=3m", "-buildvcs=false"}},
	},
	{
		name: "memberlist", module: memberlistModulePath, version: memberlistVersion, sum: memberlistSum, preparedPackage: memberlistModulePath, importPath: memberlistModulePath,
		prepare: prepareMemberlist, rewrites: memberlistRewrites,
		originalInventorySHA256: memberlistOriginalSourceInventorySHA256, replacementInventorySHA256: memberlistReplacementSourceInventorySHA256, preparedSourceSetSHA256: memberlistPreparedSourceSetSHA256,
		outsideServerGraph: true,
		cacheDir:           "github.com/hashicorp/memberlist", unrewritten: "config.go", ambiguousAnchor: "return",
		otherModule: "github.com/hashicorp/other", otherVersion: "v0.5.3",
		preparedSourceSetSHA256ByHost: memberlistPreparedSourceSetSHA256ByHost,
		consumer:                      &adapterConsumer{testFlags: []string{"-p=2", "-tags=test_dep", "-timeout=3m", "-buildvcs=false"}},
	},
	{
		name: "sentry", module: sentryModulePath, version: sentryVersion, sum: sentrySum, preparedPackage: sentryModulePath, importPath: sentryModulePath,
		prepare: prepareSentry, rewrites: sentryRewrites,
		originalInventorySHA256: sentryOriginalSourceInventorySHA256, replacementInventorySHA256: sentryReplacementSourceInventorySHA256, preparedSourceSetSHA256: sentryPreparedSourceSetSHA256,
		removed:            []string{"\"golang.org/x/sys/execabs\"", "exec.LookPath", "exec.Command"},
		retained:           []string{"func defaultRelease() (release string) {", "func revisionFromBuildInfo(info *debug.BuildInfo) string {", "\"SENTRY_RELEASE\"", "debug.ReadBuildInfo()"},
		outsideServerGraph: true,
		cacheDir:           "github.com/getsentry/sentry-go", unrewritten: "client.go", ambiguousAnchor: "release",
		otherModule: "github.com/getsentry/other", otherVersion: "v0.45.0",
		preparedSourceSetSHA256ByHost: sentryPreparedSourceSetSHA256ByHost,
		consumer: &adapterConsumer{
			testFlags: []string{"-p=2", "-buildvcs=false"},
			prepared:  sentryReplacementPassesUpstreamBuildInfoTests,
			platform:  sentryConsumerHasNoSubprocessImports,
		},
	},
	{
		name: "hashicorp-metrics", module: hashicorpMetricsModulePath, version: hashicorpMetricsVersion, sum: hashicorpMetricsSum, preparedPackage: hashicorpMetricsModulePath, importPath: hashicorpMetricsModulePath,
		prepare: prepareHashicorpMetrics, rewrites: hashicorpMetricsRewrites,
		originalInventorySHA256: hashicorpMetricsOriginalSourceInventorySHA256, replacementInventorySHA256: hashicorpMetricsReplacementSourceInventorySHA256, preparedSourceSetSHA256: hashicorpMetricsPreparedSourceSetSHA256,
		removed:            []string{"\"os/signal\"", "signal.Notify", "signal.Stop", "go i.run()"},
		retained:           []string{"sig syscall.Signal", "func (i *InmemSignal) Stop() {", "func (i *InmemSignal) dumpStats() {", "func (i *InmemSignal) flattenLabels(name string, labels []Label) string {"},
		outsideServerGraph: true,
		// Its consumer, TestHashicorpMetricsAdapterConsumer, goes through the
		// default registry and capability review instead of the template.
		cacheDir: "github.com/hashicorp/go-metrics", unrewritten: "statsd.go", ambiguousAnchor: "\ti.stop",
		otherModule: "github.com/armon/go-metrics", otherVersion: "v0.5.3",
		rewriteDropsComments:          true,
		preparedSourceSetSHA256ByHost: hashicorpMetricsPreparedSourceSetSHA256ByHost,
	},
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
		if adapter.unrewritten != "" {
			original, err := readAdapterSource(adapter.module, filepath.Join(moduleCache, filepath.FromSlash(adapter.cacheDir+"@"+adapter.version)), adapter.unrewritten)
			if err != nil {
				t.Fatal(err)
			}
			copied, err := readAdapterSource(adapter.module, prepared.replacement, adapter.unrewritten)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(original, copied) {
				t.Fatalf("%s adapter changed the unrewritten %s", adapter.name, adapter.unrewritten)
			}
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

// TestRewrittenModulesRejectChangedIdentity requires every adapter to reject
// a changed sum and, where the template names them, a changed module path and
// version, before it reads the module cache.
func TestRewrittenModulesRejectChangedIdentity(t *testing.T) {
	for _, cache := range []string{"empty", "populated"} {
		t.Run(cache, func(t *testing.T) {
			moduleCache := t.TempDir()
			if cache == "populated" {
				moduleCache = pinnedModuleCache(t)
			}
			for _, adapter := range rewrittenModuleAdapters {
				if cache == "populated" && adapter.outsideServerGraph {
					downloadPinnedModule(t, adapter.module, adapter.version)
				}
				identities := []gomadversion.AdapterIdentity{{Module: adapter.module, Version: adapter.version, Sum: "h1:changed"}}
				if adapter.otherModule != "" {
					identities = append(identities,
						gomadversion.AdapterIdentity{Module: adapter.otherModule, Version: adapter.version, Sum: adapter.sum},
						gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.otherVersion, Sum: adapter.sum})
				}
				for _, identity := range identities {
					if _, err := adapter.prepare(moduleCache, t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
						t.Fatalf("%s changed identity %#v: %v", adapter.name, identity, err)
					}
				}
			}
		})
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

func TestProfileSelectsSentryAdapter(t *testing.T) {
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
	spec, adapters, err := Default().PrepareBuildAdapters(target.Spec{
		Kind: target.KindGoTest, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: t.TempDir(), BuildTags: []string{"test_dep", "integration", "hashicorpmetrics", "gomad"},
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != sentryModulePath || adapters[0].PreparedSourceSetSHA256 != sentryPreparedSourceSetSHA256 {
		t.Fatalf("Sentry profile adapter selection = %#v", adapters)
	}
	command := exec.CommandContext(t.Context(), "go", "env", "GOROOT")
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	toolchainRoot, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	spec.ToolchainRoot = strings.TrimSpace(string(toolchainRoot))
	if _, err := target.ReviewCapabilities(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
}

// TestProfileRejectsAdapterConfigurationDrift requires the default profile to
// refuse a target whose go.mod or go.sum the adapter for its module cannot
// replace exactly.
func TestProfileRejectsAdapterConfigurationDrift(t *testing.T) {
	for _, adapter := range []struct{ name, module, version, otherVersion string }{
		{name: "sentry", module: sentryModulePath, version: sentryVersion, otherVersion: "v0.45.0"},
		{name: "hashicorp-metrics", module: hashicorpMetricsModulePath, version: hashicorpMetricsVersion, otherVersion: "v0.5.3"},
		{name: "grpc", module: grpcModulePath, version: grpcVersion, otherVersion: "v1.80.1"},
	} {
		for _, test := range []struct {
			name         string
			version      string
			replace      string
			sum          string
			buildModFile bool
			want         string
		}{
			{name: "version", version: adapter.otherVersion, want: "unsupported " + adapter.module + " version"},
			{name: "replacement", replace: "replace " + adapter.module + " => ./adapted\n", want: "already replaces " + adapter.module},
			{name: "version-replacement", replace: "replace " + adapter.module + " " + adapter.version + " => ./adapted\n", want: "already replaces " + adapter.module},
			{name: "replacement-block", replace: "replace (\n\t" + adapter.module + " => ./adapted\n)\n", want: "already replaces " + adapter.module},
			{name: "sum", sum: "h1:changed", want: "module sum"},
			{name: "missing-sum", want: "module sum"},
			{name: "build-modfile", buildModFile: true, want: "existing build modfile"},
		} {
			t.Run(adapter.name+"/"+test.name, func(t *testing.T) {
				workingDirectory := t.TempDir()
				version := test.version
				if version == "" {
					version = adapter.version
				}
				modFile := "module example.test\n\ngo 1.27.1\n\nrequire " + adapter.module + " " + version + "\n" + test.replace
				if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte(modFile), 0o600); err != nil {
					t.Fatal(err)
				}
				if test.sum != "" {
					if err := os.WriteFile(filepath.Join(workingDirectory, "go.sum"), []byte(adapter.module+" "+version+" "+test.sum+"\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				spec := target.Spec{WorkingDir: workingDirectory, PreparationRoot: t.TempDir()}
				if test.buildModFile {
					spec.BuildModFile = filepath.Join(workingDirectory, "existing.mod")
				}
				if _, _, err := Default().PrepareBuildAdapters(spec, t.TempDir()); !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("prepare configuration: %v, want %s", err, test.want)
				}
			})
		}
	}
}

// TestRewrittenModuleRewritesPreserveCommentsAndRejectDrift checks each
// rewritten file of every template adapter: the rewrite keeps every original
// comment, and a changed upstream file, a missing or ambiguous anchor, or a
// replacement that does not match its digest is rejected.
func TestRewrittenModuleRewritesPreserveCommentsAndRejectDrift(t *testing.T) {
	const missingAnchor = "absent Gomad adapter anchor"
	for _, adapter := range rewrittenModuleAdapters {
		if adapter.cacheDir == "" {
			continue
		}
		t.Run(adapter.name, func(t *testing.T) {
			downloadPinnedModule(t, adapter.module, adapter.version)
			moduleRoot := filepath.Join(pinnedModuleCache(t), filepath.FromSlash(adapter.cacheDir+"@"+adapter.version))
			for _, rewrite := range adapter.rewrites {
				t.Run(rewrite.path, func(t *testing.T) {
					source, err := readAdapterSource(adapter.module, moduleRoot, rewrite.path)
					if err != nil {
						t.Fatal(err)
					}
					t.Run("comments", func(t *testing.T) {
						if adapter.rewriteDropsComments {
							t.Skipf("the %s rewrite removes code with its comments", adapter.name)
						}
						replacement, err := rewriteAdapterSource(adapter.module, rewrite, source)
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
							t.Fatalf("%s adapter changed original comments in %s", adapter.name, rewrite.path)
						}
					})
					if bytes.Contains(source, []byte(missingAnchor)) {
						t.Fatalf("%s contains the missing-anchor text", rewrite.path)
					}
					for _, test := range []struct {
						name, want string
						change     func(*sourceRewrite, *[]byte)
					}{
						{name: "upstream-file", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
						{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte(missingAnchor) }},
						{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte(adapter.ambiguousAnchor) }},
						{name: "replacement-digest", want: "replacement identity mismatch", change: func(changed *sourceRewrite, _ *[]byte) { changed.replacementSHA256 = rewrite.sourceSHA256 }},
					} {
						t.Run(test.name, func(t *testing.T) {
							changed := rewrite
							changed.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
							contents := append([]byte(nil), source...)
							test.change(&changed, &contents)
							if _, err := rewriteAdapterSource(adapter.module, changed, contents); err == nil || !strings.Contains(err.Error(), test.want) {
								t.Fatalf("%s rewrite of %s: %v, want %s", adapter.name, rewrite.path, err, test.want)
							}
						})
					}
				})
			}
		})
	}
}

// TestRewrittenModulesRejectUnrewrittenInventoryDrift changes a module file
// that no rewrite touches; the module's source inventory must reject it.
func TestRewrittenModulesRejectUnrewrittenInventoryDrift(t *testing.T) {
	for _, adapter := range rewrittenModuleAdapters {
		if adapter.cacheDir == "" {
			continue
		}
		t.Run(adapter.name, func(t *testing.T) {
			downloadPinnedModule(t, adapter.module, adapter.version)
			moduleDirectory := filepath.FromSlash(adapter.cacheDir + "@" + adapter.version)
			moduleCache := t.TempDir()
			moduleRoot := filepath.Join(moduleCache, moduleDirectory)
			if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := copyAdapterModule(filepath.Join(pinnedModuleCache(t), moduleDirectory), moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(moduleRoot, filepath.FromSlash(adapter.unrewritten))
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
			identity := gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.version, Sum: adapter.sum}
			if _, err := adapter.prepare(moduleCache, t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
				t.Fatalf("changed %s module inventory: %v", adapter.name, err)
			}
		})
	}
}

// adapterConsumer runs an adapter's stock consumer fixture, testdata/<name>
// with <name>_test.go, against the prepared build modfile.
type adapterConsumer struct {
	// testFlags go between "go test -v" and the modfile flags; listFlags
	// between "go list" and the modfile flags.
	testFlags, listFlags []string
	// prepared runs once after the fixture's tests pass.
	prepared func(t *testing.T, run consumerRun, adapter BuildAdapter)
	// platform runs for each platform after its prepared source set matches.
	platform func(t *testing.T, run consumerRun, environment []string, host string, pkg consumerPackage)
}

// consumerRun runs the go command in dir and fails the test on error.
type consumerRun func(dir string, environment []string, args ...string) []byte

// consumerPackage is the go list -json view of the prepared package.
type consumerPackage struct {
	Dir        string
	GoFiles    []string
	Imports    []string
	workingDir string
	modFile    string
}

// TestRewrittenModuleConsumers prepares each adapter for its stock consumer
// fixture, runs the fixture's tests through the prepared build modfile, and
// requires the prepared package's source set on darwin/arm64 and linux/amd64
// to equal that platform's pin.
func TestRewrittenModuleConsumers(t *testing.T) {
	for _, adapter := range rewrittenModuleAdapters {
		if adapter.consumer == nil {
			continue
		}
		t.Run(adapter.name, func(t *testing.T) {
			downloadPinnedModule(t, adapter.module, adapter.version)
			workingDirectory := t.TempDir()
			for _, name := range []string{"go.mod", "go.sum", adapter.name + "_test.go"} {
				contents, err := os.ReadFile(filepath.Join("testdata", adapter.name, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			identity := gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.version, Sum: adapter.sum}
			registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{identity}, []adapterImplementation{
				{module: adapter.module, prepare: adapter.prepare},
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
			if len(adapters) != 1 || adapters[0].Module != adapter.module || adapters[0].PreparedSourceSetSHA256 != adapter.preparedSourceSetSHA256 {
				t.Fatalf("%s consumer adapter selection = %#v", adapter.name, adapters)
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
					t.Fatalf("prepared %s consumer %v: %v\n%s", adapter.name, args, err, output)
				}
				return output
			}
			consumer := adapter.consumer
			modFlags := []string{"-mod=readonly", "-modfile=" + spec.BuildModFile}
			testArguments := append(append(append([]string{"test", "-v"}, consumer.testFlags...), modFlags...), ".")
			t.Logf("prepared stock %s consumer:\n%s", adapter.name, run(workingDirectory, nil, testArguments...))
			if consumer.prepared != nil {
				consumer.prepared(t, run, adapters[0])
			}
			for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
				host := platform.goos + "/" + platform.goarch
				environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
				listArguments := append(append(append([]string{"list"}, consumer.listFlags...), modFlags...), "-json", adapter.preparedPackage)
				var pkg consumerPackage
				if err := json.Unmarshal(run(workingDirectory, environment, listArguments...), &pkg); err != nil {
					t.Fatal(err)
				}
				pkg.workingDir, pkg.modFile = workingDirectory, spec.BuildModFile
				slices.Sort(pkg.GoFiles)
				sources := make([]compatibility.Source, len(pkg.GoFiles))
				for index, name := range pkg.GoFiles {
					contents, err := os.ReadFile(filepath.Join(pkg.Dir, name))
					if err != nil {
						t.Fatal(err)
					}
					sources[index] = compatibility.Source{Name: name, SHA256: digestBytes(contents)}
				}
				want := adapter.preparedSourceSetSHA256ByHost[host]
				if got := compatibility.DigestSources(sources); got != want {
					t.Fatalf("prepared %s source set on %s = %s, want %s", adapter.name, host, got, want)
				}
				if consumer.platform != nil {
					consumer.platform(t, run, environment, host, pkg)
				}
				t.Logf("prepared %s source set %s", host, want)
			}
		})
	}
}

// sentryReplacementPassesUpstreamBuildInfoTests runs the upstream build-info
// tests inside the prepared Sentry replacement.
func sentryReplacementPassesUpstreamBuildInfoTests(t *testing.T, run consumerRun, adapter BuildAdapter) {
	t.Helper()
	t.Logf("prepared upstream build-info tests:\n%s", run(adapter.ReplacementRoot, nil, "test", "-v", "-p=2", "-mod=readonly", "-run=^TestRevisionFromBuildInfo", "."))
}

// sentryConsumerHasNoSubprocessImports requires the prepared Sentry package to
// reach neither os/exec nor golang.org/x/sys/execabs on the platform.
func sentryConsumerHasNoSubprocessImports(t *testing.T, run consumerRun, environment []string, host string, pkg consumerPackage) {
	t.Helper()
	modFlags := []string{"-mod=readonly", "-modfile=" + pkg.modFile}
	output := run(pkg.workingDir, environment, append(append([]string{"list"}, modFlags...), "-deps", "-f={{.ImportPath}}", sentryModulePath)...)
	for _, path := range strings.Fields(string(output)) {
		if path == "golang.org/x/sys/execabs" {
			t.Fatalf("prepared Sentry %s retains subprocess dependency %s", host, path)
		}
	}
	output = run(pkg.workingDir, environment, append(append([]string{"list"}, modFlags...), "-deps", `-f={{if not .Standard}}{{.ImportPath}} {{join .Imports " "}}{{end}}`, sentryModulePath)...)
	for _, line := range strings.Split(string(output), "\n") {
		for _, path := range strings.Fields(line) {
			if path == "os/exec" || path == "golang.org/x/sys/execabs" {
				t.Fatalf("prepared Sentry %s nonstandard subprocess edge: %s", host, line)
			}
		}
	}
	for _, path := range pkg.Imports {
		if path == "os/exec" || path == "golang.org/x/sys/execabs" {
			t.Fatalf("prepared Sentry %s directly imports subprocess package %s", host, path)
		}
	}
}

// pebbleConsumerLinksNoNativeVFS links the Pebble consumer's test binary for
// the platform and requires that no OS-backed VFS wrapper method is linked.
func pebbleConsumerLinksNoNativeVFS(t *testing.T, run consumerRun, environment []string, host string, pkg consumerPackage) {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "pebble.test")
	run(pkg.workingDir, environment, "test", "-c", "-tags=gomad,hashicorpmetrics,integration,test_dep", "-p=2", "-mod=readonly", "-modfile="+pkg.modFile, "-o="+binary, ".")
	output := run(pkg.workingDir, environment, "tool", "nm", binary)
	for _, name := range []string{"unixFile", "linuxFile", "linuxDir"} {
		if strings.Contains(string(output), pebbleModulePath+"/vfs.(*"+name+")") {
			t.Fatalf("prepared %s retains OS-backed VFS wrapper %s", host, name)
		}
	}
}
