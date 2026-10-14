package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

type diagnosticHelperAdmission struct {
	Receipt  string
	Features []string
}

type diagnosticModuleAdmission struct {
	Schema, Source, SourceHead, CPUInfoSHA256       string
	Directory, Target, Kind                         string
	Tags                                            []string
	Module, ModuleSHA256, Compiler, CompilerVersion string
	Overlay, RuntimeKey, RuntimeSHA256, ModelSHA256 string
	GOWASM                                          string
	Command, Environment                            []string
	Inputs                                          map[string]string
}

type diagnosticAdmission struct {
	Repo, SourceHead, CPUInfo, BootID, Receipt string
	Inputs                                     map[string]string
	Helpers                                    map[string]diagnosticHelperAdmission
	Modules                                    map[string]diagnosticModuleAdmission
	ModuleReceipts                             map[string]string
}

func diagnosticGit(repo string, args ...string) (string, error) {
	result, err := hostexec.Run(context.Background(), hostexec.Request{Command: append([]string{"git", "-C", repo}, args...), Dir: repo, Timeout: 30 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20})
	if err != nil || result.ExitCode != 0 || result.Termination != hostexec.TerminationExit || !result.GroupGone || result.Stdout.Truncated || result.Stderr.Truncated {
		return "", fmt.Errorf("diagnostic source identity: %v, exit=%d, termination=%s", err, result.ExitCode, result.Termination)
	}
	return strings.TrimSpace(string(result.Stdout.RawBytes)), nil
}

func diagnosticCurrentHost(t *testing.T) (string, string) {
	t.Helper()
	cpu, admitted := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if !admitted || runtime.Version() != "go1.27.1" {
		t.Fatal("diagnostic measurements require actual Linux/arm64 CPU evidence and stock Go 1.27.1")
	}
	boot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	if err != nil || len(bytes.TrimSpace(boot)) == 0 {
		t.Fatal("diagnostic measurements require bounded current boot identity", err)
	}
	return cpu, strings.TrimSpace(string(boot))
}

func newDiagnosticAdmission(t *testing.T, repo, root string) *diagnosticAdmission {
	t.Helper()
	cpu, boot := diagnosticCurrentHost(t)
	head, err := diagnosticGit(repo, "rev-parse", "HEAD")
	if err != nil || head == "" {
		t.Fatal("diagnostic source HEAD unavailable", err)
	}
	a := &diagnosticAdmission{Repo: repo, SourceHead: head, CPUInfo: cpu, BootID: boot, Inputs: map[string]string{}, Helpers: map[string]diagnosticHelperAdmission{}, Modules: map[string]diagnosticModuleAdmission{}, ModuleReceipts: map[string]string{}}
	a.checkHost(t, true)
	a.Receipt = filepath.Join(root, "host-admission.json")
	temporalJSON(t, a.Receipt, map[string]any{"schema": "gomad-diagnostic-host/v1", "source": repo, "source_head": head, "go_version": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH, "cpuinfo": cpu, "boot_id": boot, "minimum_mem_available_bytes": uint64(8 << 30)})
	a.Inputs[a.Receipt] = fileHash(t, a.Receipt)
	return a
}

func (a *diagnosticAdmission) checkHost(t *testing.T, before bool) {
	t.Helper()
	cpu, boot := diagnosticCurrentHost(t)
	head, headErr := diagnosticGit(a.Repo, "rev-parse", "HEAD")
	status, statusErr := diagnosticGit(a.Repo, "status", "--porcelain", "--untracked-files=all")
	if cpu != a.CPUInfo || boot != a.BootID || headErr != nil || statusErr != nil || head != a.SourceHead || status != "" {
		t.Fatal("diagnostic CPU/boot/source identity changed or worktree is dirty", headErr, statusErr)
	}
	if before {
		if _, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", "1", func() (Result, error) { return Result{}, nil }); err != nil {
			t.Fatal(err)
		}
	}
}

func (a *diagnosticAdmission) addHelper(t *testing.T, helper, receipt string, features []string) {
	t.Helper()
	if !filepath.IsAbs(receipt) {
		t.Fatal("diagnostic helper requires an absolute authentic helper-admission receipt")
	}
	crate := filepath.Join(a.Repo, "tools/gomad/internal/backends/wasm/wasmhost")
	inputs := terminalExitHelperInputsFeatures(t, receipt, Request{HelperPath: helper, HelperSHA256: fileHash(t, helper)}, crate, features)
	var admission terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, receipt), &admission); err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(mustTerminalExitRead(t, admission.BuildReceipt), &build); err != nil || !diagnosticHelperCommand(build, features) {
		t.Fatal("diagnostic helper must retain the exact pinned, single-job feature build command", err)
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(filepath.Dir(admission.BuildReceipt), name)
		if build.Inputs[path] != fileHash(t, path) {
			t.Fatal("diagnostic helper is missing authenticated raw compiler evidence", name)
		}
	}
	var execution struct {
		Command  []string
		ExitCode int
	}
	if err := json.Unmarshal(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admission.BuildReceipt), "cargo-build.json")), &execution); err != nil || execution.ExitCode != 0 || !slices.Equal(execution.Command, build.Command) {
		t.Fatal("diagnostic helper command differs from actual retained execution", err)
	}
	if string(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admission.BuildReceipt), "rustc-version.stdout"))) != build.RustcVersion || string(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admission.BuildReceipt), "cargo-version.stdout"))) != build.CargoVersion {
		t.Fatal("diagnostic helper compiler versions differ from actual retained output")
	}
	a.Helpers[helper] = diagnosticHelperAdmission{Receipt: receipt, Features: slices.Clone(features)}
	maps.Copy(a.Inputs, inputs)
}

func (a *diagnosticAdmission) check(t *testing.T, root, name string, request Request, before bool) {
	t.Helper()
	a.checkHost(t, before)
	if err := diagnosticVerifyInputs(a.Inputs); err != nil {
		t.Fatal(err)
	}
	helper, ok := a.Helpers[request.HelperPath]
	if !ok {
		t.Fatal("diagnostic dispatch has no admitted helper")
	}
	terminalExitHelperInputsFeatures(t, helper.Receipt, request, filepath.Join(a.Repo, "tools/gomad/internal/backends/wasm/wasmhost"), helper.Features)
	module, ok := a.Modules[request.ModulePath]
	if !ok || !diagnosticModuleMatches(module, request) {
		t.Fatal("diagnostic dispatch has no matching actual module build")
	}
	temporalJSON(t, filepath.Join(root, name+"-admission.json"), map[string]any{
		"host_receipt": a.Receipt, "host_receipt_sha256": a.Inputs[a.Receipt],
		"helper_receipt": helper.Receipt, "helper_receipt_sha256": a.Inputs[helper.Receipt],
		"module_receipt": a.ModuleReceipts[request.ModulePath], "module_receipt_sha256": a.Inputs[a.ModuleReceipts[request.ModulePath]],
		"helper_sha256": request.HelperSHA256, "module_sha256": request.ModuleSHA256, "source_head": a.SourceHead, "before_dispatch": before,
	})
}

func diagnosticModuleMatches(module diagnosticModuleAdmission, request Request) bool {
	return request.Runtime != nil && request.Runtime.Choice != nil && module.Schema == "gomad-diagnostic-module-build/v1" && module.Module == request.ModulePath && module.ModuleSHA256 == request.ModuleSHA256 && module.RuntimeKey == request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey && module.RuntimeSHA256 == toolchain.ImplementationSHA256() && module.ModelSHA256 == ImplementationSHA256()
}

// This producer is deliberately test-only: production preparation fixes GOWASM="".
func (a *diagnosticAdmission) buildModule(t *testing.T, output, gowasm string) {
	t.Helper()
	a.buildModuleTarget(t, a.Repo, output, gowasm, "./tests", "test", []string{"disable_grpc_modules", "test_dep", "sqlite3_dotlk"})
}

func diagnosticModuleCommands(compiler, overlay, output, target, kind string, tags []string) ([]string, []string, error) {
	if kind != "test" && kind != "build" {
		return nil, nil, fmt.Errorf("diagnostic module kind %q", kind)
	}
	list := []string{compiler, "list", "-p=1", "-deps"}
	command := []string{compiler, kind}
	if kind == "test" {
		list = append(list, "-test")
		command = append(command, "-c")
	}
	list = append(list, "-json", "-overlay", overlay)
	command = append(command, "-p=1", "-trimpath")
	if kind == "build" {
		command = append(command, "-buildvcs=false")
	}
	command = append(command, "-overlay", overlay, "-o", output)
	if len(tags) != 0 {
		list = append(list, "-tags", strings.Join(tags, ","))
		command = append(command, "-tags", strings.Join(tags, ","))
	}
	return append(list, target), append(command, target), nil
}

func (a *diagnosticAdmission) buildModuleTarget(t *testing.T, directory, output, gowasm, target, kind string, tags []string) {
	t.Helper()
	if gowasm != "" && gowasm != "satconv,signext" {
		t.Fatal("unexpected diagnostic GOWASM variant")
	}
	a.checkHost(t, true)
	evidence, err := diagnosticRunRoot(filepath.Dir(output), "module-build-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatal("refusing to overwrite a diagnostic module", err)
	}
	compiler, overlay, key := prototypeToolchain(t)
	compilerSHA256 := fileHash(t, compiler)
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains([]string{"PATH", "HOME", "GOCACHE", "GOMODCACHE", "GOPATH", "TMPDIR", "GOTMPDIR", "GOPROXY", "GOSUMDB", "GONOPROXY", "GONOSUMDB", "GOPRIVATE"}, name) {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "GOROOT="+filepath.Dir(filepath.Dir(compiler)), "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOWASM="+gowasm, "GOMAXPROCS=1")
	version := temporalCommand(t, evidence, "go-version", directory, environment, []string{compiler, "version"})
	if version.Stdout.Truncated || version.Stderr.Truncated || !strings.HasPrefix(string(version.Stdout.RawBytes), "go version go1.27.1 ") {
		t.Fatal("diagnostic module requires pinned stock Go 1.27.1")
	}
	listCommand, command, err := diagnosticModuleCommands(compiler, overlay, output, target, kind, tags)
	if err != nil {
		t.Fatal(err)
	}
	listed := temporalCommand(t, evidence, "sources-before", directory, environment, listCommand)
	if listed.Stdout.Truncated || listed.Stderr.Truncated {
		t.Fatal("truncated diagnostic source closure")
	}
	inputs := diagnosticModuleSourceInputs(t, listed.Stdout.RawBytes, compiler, overlay, directory)
	if inputs[compiler] != compilerSHA256 {
		t.Fatal("diagnostic compiler changed before build")
	}
	built := temporalCommand(t, evidence, "module-build", directory, environment, command)
	if built.Stdout.Truncated || built.Stderr.Truncated {
		t.Fatal("truncated diagnostic compiler output")
	}
	listed = temporalCommand(t, evidence, "sources-after", directory, environment, listCommand)
	if listed.Stdout.Truncated || listed.Stderr.Truncated {
		t.Fatal("truncated diagnostic source closure")
	}
	if !maps.Equal(inputs, diagnosticModuleSourceInputs(t, listed.Stdout.RawBytes, compiler, overlay, directory)) {
		t.Fatal("diagnostic compiler, overlay or source closure changed during module build")
	}
	a.checkHost(t, false)
	for _, operation := range []string{"go-version", "sources-before", "module-build", "sources-after"} {
		for _, suffix := range []string{".json", ".stdout", ".stderr"} {
			path := filepath.Join(evidence, operation+suffix)
			inputs[path] = fileHash(t, path)
		}
	}
	module := diagnosticModuleAdmission{Schema: "gomad-diagnostic-module-build/v1", Source: a.Repo, SourceHead: a.SourceHead, CPUInfoSHA256: digest([]byte(a.CPUInfo)), Directory: directory, Target: target, Kind: kind, Tags: slices.Clone(tags), Module: output, ModuleSHA256: fileHash(t, output), Compiler: compiler, CompilerVersion: string(version.Stdout.RawBytes), Overlay: overlay, RuntimeKey: key, RuntimeSHA256: toolchain.ImplementationSHA256(), ModelSHA256: ImplementationSHA256(), GOWASM: gowasm, Command: command, Environment: environment, Inputs: inputs}
	receipt := filepath.Join(evidence, "module-admission.json")
	temporalJSON(t, receipt, module)
	a.Modules[output], a.ModuleReceipts[output] = module, receipt
	maps.Copy(a.Inputs, inputs)
	a.Inputs[output], a.Inputs[receipt] = module.ModuleSHA256, fileHash(t, receipt)
}

func diagnosticModuleSourceInputs(t *testing.T, data []byte, compiler, overlay, repo string) map[string]string {
	t.Helper()
	inputs := temporalSourceInputs(t, data)
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg struct {
			Dir               string
			HFiles, SysoFiles []string
			Module            *struct {
				GoMod   string
				Replace *struct{ GoMod string }
			}
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		for _, name := range append(pkg.HFiles, pkg.SysoFiles...) {
			if !filepath.IsAbs(name) {
				name = filepath.Join(pkg.Dir, name)
			}
			inputs[name] = fileHash(t, name)
		}
		if pkg.Module != nil {
			gomod := pkg.Module.GoMod
			if pkg.Module.Replace != nil {
				gomod = pkg.Module.Replace.GoMod
			}
			if gomod != "" {
				inputs[gomod] = fileHash(t, gomod)
			}
		}
	}
	var replacement struct{ Replace map[string]string }
	if err := json.Unmarshal(mustTerminalExitRead(t, overlay), &replacement); err != nil || len(replacement.Replace) == 0 {
		t.Fatal("diagnostic module requires actual cooperative overlay bindings", err)
	}
	for from, to := range replacement.Replace {
		inputs[to] = fileHash(t, to)
		if _, err := os.Stat(from); err == nil {
			inputs[from] = fileHash(t, from)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	for _, path := range []string{compiler, overlay, filepath.Join(repo, "go.mod"), filepath.Join(repo, "go.sum"), filepath.Join(filepath.Dir(filepath.Dir(compiler)), "VERSION")} {
		inputs[path] = fileHash(t, path)
	}
	for _, tool := range []string{"compile", "link", "asm"} {
		path := filepath.Join(filepath.Dir(filepath.Dir(compiler)), "pkg/tool", runtime.GOOS+"_"+runtime.GOARCH, tool)
		inputs[path] = fileHash(t, path)
	}
	return inputs
}

func TestBuildDiagnosticProfileHelper(t *testing.T) {
	diagnosticBuildHelper(t, "GOMAD_WASM_DIAGNOSTIC_HELPER_BUILD_ROOT", []string{"diagnostic-profile"})
}

func TestBuildDiagnosticGuestProfileHelper(t *testing.T) {
	diagnosticBuildHelper(t, "GOMAD_WASM_GUEST_HELPER_BUILD_ROOT", []string{"diagnostic-profile", "guest-profile"})
}

func TestBuildDiagnosticFuelOffHelper(t *testing.T) {
	diagnosticBuildHelper(t, "GOMAD_WASM_FUEL_OFF_HELPER_BUILD_ROOT", []string{"diagnostic-fuel-off", "diagnostic-profile"})
}

func diagnosticBuildHelper(t *testing.T, variable string, features []string) {
	t.Helper()
	root := os.Getenv(variable)
	if root == "" {
		t.Skipf("set %s to retain a pinned, feature-exact diagnostic helper build and admission", variable)
	}
	terminalExitBuildHelper(t, root, features)
}

func diagnosticHelperCommand(build terminalExitHelperBuild, features []string) bool {
	if len(build.Command) == 0 || !filepath.IsAbs(build.Command[0]) || build.Inputs[build.Command[0]] == "" {
		return false
	}
	expected := []string{build.Command[0], "build", "--manifest-path", filepath.Join(build.Crate, "Cargo.toml"), "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json", "--features", strings.Join(features, ",")}
	return slices.Equal(build.Command, expected)
}

func diagnosticVerifyInputs(inputs map[string]string) error {
	for path, want := range inputs {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, readErr := io.Copy(hash, file)
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if fmt.Sprintf("%x", hash.Sum(nil)) != want {
			return fmt.Errorf("diagnostic input changed: %s", path)
		}
	}
	return nil
}

func TestDiagnosticAdmissionCommand(t *testing.T) {
	features := []string{"diagnostic-profile", "guest-profile"}
	command := []string{"/pinned/cargo", "build", "--manifest-path", "/crate/Cargo.toml", "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json", "--features", "diagnostic-profile,guest-profile"}
	build := terminalExitHelperBuild{Crate: "/crate", Command: command, Inputs: map[string]string{"/pinned/cargo": "digest"}}
	if !diagnosticHelperCommand(build, features) {
		t.Fatal("exact command rejected")
	}
	for index := range command {
		changed := build
		changed.Command = slices.Clone(command)
		changed.Command[index] = "wrong"
		if diagnosticHelperCommand(changed, features) {
			t.Fatalf("changed command element %d admitted", index)
		}
	}
}

func TestDiagnosticModuleBuildCommands(t *testing.T) {
	for _, test := range []struct {
		kind        string
		tags        []string
		list, build []string
	}{
		{"test", []string{"test_dep"}, []string{"/go", "list", "-p=1", "-deps", "-test", "-json", "-overlay", "/overlay", "-tags", "test_dep", "./fixture"}, []string{"/go", "test", "-c", "-p=1", "-trimpath", "-overlay", "/overlay", "-o", "/module", "-tags", "test_dep", "./fixture"}},
		{"build", nil, []string{"/go", "list", "-p=1", "-deps", "-json", "-overlay", "/overlay", "./fixture"}, []string{"/go", "build", "-p=1", "-trimpath", "-buildvcs=false", "-overlay", "/overlay", "-o", "/module", "./fixture"}},
	} {
		listed, built, err := diagnosticModuleCommands("/go", "/overlay", "/module", "./fixture", test.kind, test.tags)
		if err != nil || !slices.Equal(listed, test.list) || !slices.Equal(built, test.build) {
			t.Fatalf("%s compiler commands: %v %v %v", test.kind, listed, built, err)
		}
	}
	if _, _, err := diagnosticModuleCommands("/go", "/overlay", "/module", "./fixture", "run", nil); err == nil {
		t.Fatal("non-build compiler operation admitted")
	}
}

func TestDiagnosticModuleSourceBindings(t *testing.T) {
	root := t.TempDir()
	compiler := filepath.Join(root, "stock/bin/go")
	overlay := filepath.Join(root, "overlay.json")
	paths := []string{compiler, filepath.Join(root, "stock/VERSION"), filepath.Join(root, "go.mod"), filepath.Join(root, "go.sum"), filepath.Join(root, "fixture.go"), filepath.Join(root, "fixture.h"), filepath.Join(root, "fixture.syso"), filepath.Join(root, "replacement.go")}
	for _, tool := range []string{"compile", "link", "asm"} {
		paths = append(paths, filepath.Join(root, "stock/pkg/tool", runtime.GOOS+"_"+runtime.GOARCH, tool))
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		temporalWrite(t, path, []byte("original"))
	}
	temporalJSON(t, overlay, struct{ Replace map[string]string }{map[string]string{filepath.Join(root, "fixture.go"): filepath.Join(root, "replacement.go")}})
	listed, err := json.Marshal(struct {
		Dir                        string
		GoFiles, HFiles, SysoFiles []string
	}{root, []string{"fixture.go"}, []string{"fixture.h"}, []string{"fixture.syso"}})
	if err != nil {
		t.Fatal(err)
	}
	inputs := diagnosticModuleSourceInputs(t, listed, compiler, overlay, root)
	paths = append(paths, overlay)
	for _, path := range paths {
		if inputs[path] != fileHash(t, path) {
			t.Fatalf("producer omitted %s", path)
		}
		original := mustTerminalExitRead(t, path)
		temporalWrite(t, path, []byte("changed"))
		if err := diagnosticVerifyInputs(inputs); err == nil {
			t.Fatalf("changed compiler/source/tool admitted: %s", path)
		}
		temporalWrite(t, path, original)
	}
}

func TestDiagnosticRecoveryCompilerEnvironment(t *testing.T) {
	t.Setenv("GOMAXPROCS", "1")
	t.Setenv("GOTMPDIR", "/private/compiler-temp")
	environment := temporalBuildEnvironment("/pinned/bin/go")
	for _, expected := range []string{"GOMAXPROCS=1", "GOTMPDIR=/private/compiler-temp", "GOENV=off", "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS="} {
		if !slices.Contains(environment, expected) {
			t.Fatalf("compiler environment lost %s", expected)
		}
	}
}

func TestDiagnosticAdmissionInputs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{path: fileHash(t, path)}
	if err := diagnosticVerifyInputs(inputs); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := diagnosticVerifyInputs(inputs); err == nil {
		t.Fatal("changed input admitted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := diagnosticVerifyInputs(inputs); err == nil {
		t.Fatal("missing input admitted")
	}
}

func TestDiagnosticClusterPairAdmission(t *testing.T) {
	if root := os.Getenv("GOMAD_DIAGNOSTIC_PAIR_CHILD"); root != "" {
		phase, drift := os.Getenv("GOMAD_DIAGNOSTIC_PAIR_PHASE"), os.Getenv("GOMAD_DIAGNOSTIC_PAIR_DRIFT")
		input, memory := filepath.Join(root, "module"), filepath.Join(root, "meminfo")
		temporalWrite(t, input, []byte("admitted"))
		temporalWrite(t, memory, []byte("MemAvailable: 8388608 kB\n"))
		inputs := map[string]string{input: fileHash(t, input)}
		if drift == "before" {
			temporalWrite(t, input, []byte("changed"))
		}
		if drift == "resource" {
			temporalWrite(t, memory, []byte("MemAvailable: 8388607 kB\n"))
		}
		observer := temporalPairObserver{
			Admission: func(t *testing.T, evidence, label string, requests []Request, before bool) {
				if evidence != root || label != phase || len(requests) != 2 {
					t.Fatal("pair boundary lost")
				}
				if err := diagnosticVerifyInputs(inputs); err != nil {
					t.Fatal(err)
				}
				if before {
					if _, err := matchedPrepareAdmitted("linux", memory, "1", func() (Result, error) { return Result{}, nil }); err != nil {
						t.Fatal(err)
					}
				}
			},
			Observe: func(t *testing.T, evidence, label string, requests []Request, results []Result, failures []error) {
				if len(results) != 2 || len(failures) != 2 {
					t.Fatal("pair evidence lost")
				}
				temporalWrite(t, filepath.Join(root, "observed"), []byte(label))
			},
		}
		temporalObservedPair(t, root, phase, []Request{{}, {}}, []temporalPairObserver{observer}, func() ([]Result, []error) {
			temporalWrite(t, filepath.Join(root, "dispatched"), []byte(phase))
			if drift == "after" {
				temporalWrite(t, input, []byte("changed"))
			}
			return []Result{{}, {}}, []error{nil, nil}
		})
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"cluster-crash", "cluster-reopened", "cluster-prefix-replay", "cluster-full-replay"} {
		for _, drift := range []string{"none", "before", "after", "resource"} {
			t.Run(phase+"/"+drift, func(t *testing.T) {
				root := t.TempDir()
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				child := exec.CommandContext(ctx, executable, "-test.run=^TestDiagnosticClusterPairAdmission$")
				child.Env = append(os.Environ(), "GOMAD_DIAGNOSTIC_PAIR_CHILD="+root, "GOMAD_DIAGNOSTIC_PAIR_PHASE="+phase, "GOMAD_DIAGNOSTIC_PAIR_DRIFT="+drift)
				output, err := child.CombinedOutput()
				if (err == nil) != (drift == "none") {
					t.Fatalf("phase admission: %v\n%s", err, output)
				}
				if ctx.Err() != nil {
					t.Fatal("controlled pair admission timed out", ctx.Err())
				}
				if drift != "none" {
					reason := "diagnostic input changed"
					if drift == "resource" {
						reason = "at least 8GiB MemAvailable"
					}
					if !strings.Contains(string(output), reason) {
						t.Fatalf("pair failed without admission rejection: %s", output)
					}
				}
				_, dispatchErr := os.Stat(filepath.Join(root, "dispatched"))
				if (dispatchErr == nil) != (drift == "none" || drift == "after") {
					t.Fatal("unadmitted pair dispatched")
				}
				_, observedErr := os.Stat(filepath.Join(root, "observed"))
				if (observedErr == nil) != (drift == "none") {
					t.Fatal("unadmitted pair published measurements")
				}
			})
		}
	}
}

func TestDiagnosticAdmissionModuleBindings(t *testing.T) {
	request := Request{ModulePath: "/module", ModuleSHA256: "module-sha", Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{}}}
	request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey = "runtime-key"
	module := diagnosticModuleAdmission{Schema: "gomad-diagnostic-module-build/v1", Module: request.ModulePath, ModuleSHA256: request.ModuleSHA256, RuntimeKey: "runtime-key", RuntimeSHA256: toolchain.ImplementationSHA256(), ModelSHA256: ImplementationSHA256()}
	if !diagnosticModuleMatches(module, request) {
		t.Fatal("matching module rejected")
	}
	for _, name := range []string{"schema", "module", "digest", "runtime-key", "runtime-source", "model"} {
		t.Run(name, func(t *testing.T) {
			changed := module
			switch name {
			case "schema":
				changed.Schema = "gomad-wasm.provenance/v1"
			case "module":
				changed.Module = "/other"
			case "digest":
				changed.ModuleSHA256 = "stale"
			case "runtime-key":
				changed.RuntimeKey = "stale"
			case "runtime-source":
				changed.RuntimeSHA256 = "stale"
			case "model":
				changed.ModelSHA256 = "stale"
			}
			if diagnosticModuleMatches(changed, request) {
				t.Fatal("stale module admitted")
			}
		})
	}
	request.Runtime = nil
	if diagnosticModuleMatches(module, request) {
		t.Fatal("missing runtime admitted")
	}
}

func TestDiagnosticAdmissionHelperEvidence(t *testing.T) {
	if root := os.Getenv("GOMAD_DIAGNOSTIC_ADMISSION_CHILD"); root != "" {
		a := &diagnosticAdmission{Repo: root, Helpers: map[string]diagnosticHelperAdmission{}, Inputs: map[string]string{}}
		features := []string{"diagnostic-profile", "guest-profile"}
		if os.Getenv("GOMAD_DIAGNOSTIC_ADMISSION_CLUSTER") == "1" {
			features = []string{"diagnostic-profile"}
		}
		a.addHelper(t, filepath.Join(root, "admitted-helper"), filepath.Join(root, "helper-admission.json"), features)
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"valid", "cluster-valid", "cluster-extra-feature", "artifact", "helper", "source", "compiler", "receipt", "wrapper", "wrong-compiler", "missing-feature", "extra-feature", "raw-command"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			crate := filepath.Join(root, "tools/gomad/internal/backends/wasm/wasmhost")
			if err := os.MkdirAll(filepath.Join(crate, "src"), 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"src/main.rs", "Cargo.toml", "Cargo.lock", "rust-toolchain.toml", "../Makefile"} {
				temporalWrite(t, filepath.Join(crate, name), []byte(name))
			}
			rustc, cargo := filepath.Join(root, "rustc"), filepath.Join(root, "cargo")
			artifact, helper := filepath.Join(root, "artifact"), filepath.Join(root, "admitted-helper")
			for _, path := range []string{rustc, cargo, artifact, helper} {
				if err := os.WriteFile(path, []byte("fixture; never executed"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			features := []string{"diagnostic-profile", "guest-profile"}
			if mutation == "cluster-valid" {
				features = []string{"diagnostic-profile"}
			}
			command := []string{cargo, "build", "--manifest-path", filepath.Join(crate, "Cargo.toml"), "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json", "--features", strings.Join(features, ",")}
			encoded, err := json.Marshal(artifact)
			if err != nil {
				t.Fatal(err)
			}
			encodedFeatures, err := json.Marshal(features)
			if err != nil {
				t.Fatal(err)
			}
			temporalWrite(t, filepath.Join(root, "cargo-build.stdout"), []byte(`{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":`+string(encoded)+`,"features":`+string(encodedFeatures)+`} {"reason":"build-finished","success":true}`))
			rawCommand := slices.Clone(command)
			if mutation == "raw-command" {
				rawCommand[6] = "--jobs=8"
			}
			temporalJSON(t, filepath.Join(root, "cargo-build.json"), struct {
				Command  []string
				ExitCode int
			}{rawCommand, 0})
			temporalWrite(t, filepath.Join(root, "rustc-version.stdout"), []byte("rustc 1.94.1 fixture"))
			temporalWrite(t, filepath.Join(root, "cargo-version.stdout"), []byte("cargo 1.94.1 fixture"))
			inputs := terminalExitRustInputs(t, crate)
			for _, path := range []string{rustc, cargo, filepath.Join(root, "cargo-build.stdout"), filepath.Join(root, "cargo-build.json"), filepath.Join(root, "rustc-version.stdout"), filepath.Join(root, "cargo-version.stdout")} {
				inputs[path] = fileHash(t, path)
			}
			build := terminalExitHelperBuild{Crate: crate, Helper: artifact, HelperSHA256: fileHash(t, artifact), RustcPath: rustc, Inputs: inputs, Command: command, Environment: []string{"RUSTC=" + rustc, "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, Features: features, RustcVersion: "rustc 1.94.1 fixture", CargoVersion: "cargo 1.94.1 fixture"}
			switch mutation {
			case "wrapper":
				build.Environment[1] = "RUSTC_WRAPPER=/wrong"
			case "wrong-compiler":
				build.Environment[0] = "RUSTC=/wrong"
			case "missing-feature":
				build.Features = []string{"guest-profile"}
			case "extra-feature":
				build.Features = append(build.Features, "unexpected")
			}
			receipt := filepath.Join(root, "helper-build.json")
			temporalJSON(t, receipt, build)
			temporalJSON(t, filepath.Join(root, "helper-admission.json"), terminalExitHelperAdmission{BuildReceipt: receipt, BuildReceiptSHA256: fileHash(t, receipt), CargoArtifact: artifact, CargoArtifactSHA256: fileHash(t, artifact), Helper: helper, HelperSHA256: fileHash(t, helper)})
			changed := map[string]string{"artifact": artifact, "helper": helper, "source": filepath.Join(crate, "src/main.rs"), "compiler": rustc, "receipt": receipt}[mutation]
			if changed != "" {
				temporalWrite(t, changed, []byte("tampered"))
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, executable, "-test.run=^TestDiagnosticAdmissionHelperEvidence$")
			child.Env = append(os.Environ(), "GOMAD_DIAGNOSTIC_ADMISSION_CHILD="+root)
			if strings.HasPrefix(mutation, "cluster-") {
				child.Env = append(child.Env, "GOMAD_DIAGNOSTIC_ADMISSION_CLUSTER=1")
			}
			output, err := child.CombinedOutput()
			if (err == nil) != (mutation == "valid" || mutation == "cluster-valid") {
				t.Fatalf("admission for %s: %v\n%s", mutation, err, output)
			}
		})
	}
}

func TestDiagnosticAdmissionCargoFeatures(t *testing.T) {
	for _, features := range [][]string{{"diagnostic-profile"}, {"diagnostic-profile", "guest-profile"}, {"diagnostic-fuel-off", "diagnostic-profile"}} {
		for _, test := range []struct {
			name     string
			actual   []string
			accepted bool
		}{
			{"exact", features, true},
			{"missing", features[:len(features)-1], false},
			{"missing-first", features[1:], false},
			{"extra", append(append([]string{}, features...), "unexpected"), false},
		} {
			t.Run(features[0]+"/"+test.name, func(t *testing.T) {
				encoded, err := json.Marshal(test.actual)
				if err != nil {
					t.Fatal(err)
				}
				data := []byte(`{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":` + string(encoded) + `} {"reason":"build-finished","success":true}`)
				_, err = terminalExitCargoHelperFeatures(data, features)
				if (err == nil) != test.accepted {
					t.Fatalf("features %v admitted=%t", test.actual, err == nil)
				}
				if _, err := terminalExitCargoHelper(data); err == nil && len(test.actual) != 0 {
					t.Fatal("release wrapper admitted diagnostic features")
				}
			})
		}
	}
}
