package livecap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPinnedToolchainProjectsOnlyReachableForbiddenImport(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	goVersion := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOVERSION"))

	expectation := Expectation{GoVersion: goVersion, ToolchainBuildKey: buildKey, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	for _, test := range []struct {
		name   string
		source string
		live   bool
	}{
		{name: "dead", source: "package main\nimport \"os/exec\"\nfunc dead() { _ = exec.Command(\"true\") }\nfunc main() {}\n"},
		{name: "data-only", source: "package main\nimport \"os/exec\"\nvar data = exec.ErrNotFound\nfunc main() { _ = data }\n"},
		{name: "direct", source: "package main\nimport \"os/exec\"\nfunc main() { _ = exec.Command(\"true\") }\n", live: true},
		{name: "init", source: "package main\nimport \"os/exec\"\nfunc init() { _ = exec.Command(\"true\") }\nfunc main() {}\n", live: true},
		{name: "function-value", source: "package main\nimport \"os/exec\"\nfunc main() { call := exec.Command; _ = call(\"true\") }\n", live: true},
		{name: "interface", source: "package main\nimport \"os/exec\"\ntype runner interface { run() }\ntype commandRunner struct{}\nfunc (commandRunner) run() { _ = exec.Command(\"true\") }\nfunc main() { var value runner = commandRunner{}; value.run() }\n", live: true},
		{name: "reflection", source: "package main\nimport (\"os/exec\"; \"reflect\")\ntype commandRunner struct{}\nfunc (commandRunner) Run() { _ = exec.Command(\"true\") }\nfunc main() { reflect.ValueOf(commandRunner{}).MethodByName(\"Run\").Call(nil) }\n", live: true},
		{name: "inlining", source: "package main\nimport \"os/exec\"\nfunc invoke() { _ = exec.Command(\"true\") }\nfunc main() { invoke() }\n", live: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := buildLiveCapabilityFixture(t, goCommand, buildKey, test.source)
			record, err := Read(target, expectation)
			if err != nil {
				t.Fatal(err)
			}
			if got := hasCapability(record.Manifest.Facts, "import:os/exec"); got != test.live {
				t.Fatalf("import:os/exec live = %t, want %t; owners = %v", got, test.live, capabilityOwners(record.Manifest.Facts, "import:os/exec"))
			}
		})
	}
}

func TestPinnedToolchainGuardsReachableForbiddenImport(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	goVersion := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOVERSION"))
	target := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "os/exec")
func main() { _ = exec.Command("true"); fmt.Println("after") }
`)
	record, err := Read(target, Expectation{GoVersion: goVersion, ToolchainBuildKey: buildKey, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFact(record.Manifest.Facts, FactKindGuard, "import:os/exec") || hasFact(record.Manifest.Facts, FactKindCapability, "import:os/exec") {
		t.Fatalf("guarded facts = %#v", record.Manifest.Facts)
	}
	if output, err := exec.Command(target).CombinedOutput(); err != nil || string(output) != "after\n" {
		t.Fatalf("native guarded target = %v: %s", err, output)
	}
	command := exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "GOMAD_CAPABILITY_DENIED") || strings.Contains(string(output), "after") {
		t.Fatalf("Gomad guarded target = %v: %s", err, output)
	}

	target = buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "os/exec")
func main() { _ = (&exec.Cmd{}).Run(); fmt.Println("after") }
`)
	command = exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err = command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "GOMAD_CAPABILITY_DENIED") || strings.Contains(string(output), "after") {
		t.Fatalf("Gomad guarded method target = %v: %s", err, output)
	}
}

func TestPinnedToolchainAllowsPureForbiddenPackageInitializationHelper(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	target := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; _ "syscall")
func main() { fmt.Println("after") }
`)
	command := exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err := command.CombinedOutput()
	if err != nil || string(output) != "after\n" {
		t.Fatalf("Gomad initialized syscall package = %v: %s", err, output)
	}

	target = buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "syscall")
func main() { var limit syscall.Rlimit; _ = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); fmt.Println("after") }
`)
	command = exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err = command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "GOMAD_CAPABILITY_DENIED") || strings.Contains(string(output), "after") {
		t.Fatalf("Gomad callable syscall capability = %v: %s", err, output)
	}

	target = buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import "syscall"
func main() { _, _ = syscall.Write(9, []byte("escape")) }
`)
	command = exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err = command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "GOMAD_CAPABILITY_DENIED") {
		t.Fatalf("Gomad raw descriptor write = %v: %s", err, output)
	}
}

func TestPinnedToolchainAllowsPureSyscallErrorHelpers(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	target := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "io/fs"; "syscall")
func main() {
    fmt.Printf("%q %t %t %t\n", syscall.ENOENT.Error(), syscall.ENOENT.Is(fs.ErrNotExist), syscall.EINTR.Temporary(), syscall.ETIMEDOUT.Timeout())
}
`)
	command := exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err := command.CombinedOutput()
	if err != nil || string(output) != `"no such file or directory" true true true
` {
		t.Fatalf("Gomad pure syscall error helpers = %v: %s", err, output)
	}
}

func TestPinnedToolchainUsesDeterministicEnvironment(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	target := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "os")
func main() {
    before, beforeOK := os.LookupEnv("GOMAD3_HOST_ONLY")
    seed, seedOK := os.LookupEnv("GOMADSEED")
    if err := os.Setenv("GOMAD3_HOST_ONLY", "modeled"); err != nil { panic(err) }
    modeled, modeledOK := os.LookupEnv("GOMAD3_HOST_ONLY")
    if err := os.Unsetenv("GOMAD3_HOST_ONLY"); err != nil { panic(err) }
    _, afterOK := os.LookupEnv("GOMAD3_HOST_ONLY")
    fmt.Printf("%q %t %q %t %q %t %t\n", before, beforeOK, seed, seedOK, modeled, modeledOK, afterOK)
}
`)
	command := exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1", "GOMAD3_HOST_ONLY=host")
	output, err := command.CombinedOutput()
	if err != nil || string(output) != `"" false "" false "modeled" true false
` {
		t.Fatalf("Gomad deterministic environment = %v: %s", err, output)
	}
}

func TestPinnedToolchainGuardsReachableDeniedBoundary(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	goVersion := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOVERSION"))
	target := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "net")
func main() { _, _ = net.InterfaceAddrs(); fmt.Println("after") }
`)
	record, err := Read(target, Expectation{GoVersion: goVersion, ToolchainBuildKey: buildKey, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFact(record.Manifest.Facts, FactKindGuard, "network.interface-addresses") || hasFact(record.Manifest.Facts, FactKindBoundary, "network.interface-addresses") {
		t.Fatalf("guarded boundary owners = %v; objdump:\n%s", capabilityOwners(record.Manifest.Facts, "network.interface-addresses"), runToolchainCommand(t, filepath.Dir(target), goCommand, "tool", "objdump", "-s", "net.InterfaceAddrs", target))
	}
	if output, err := exec.Command(target).CombinedOutput(); err != nil || string(output) != "after\n" {
		t.Fatalf("native guarded boundary target = %v: %s", err, output)
	}
	command := exec.Command(target)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "GOMAD_CAPABILITY_DENIED") || strings.Contains(string(output), "after") {
		t.Fatalf("Gomad guarded boundary target = %v: %s", err, output)
	}
}

func TestPinnedToolchainDoesNotGuardModeledUnsupportedBoundary(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	goVersion := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOVERSION"))
	target := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import "net"
func main() { _, _ = net.Interfaces() }
`)
	record, err := Read(target, Expectation{GoVersion: goVersion, ToolchainBuildKey: buildKey, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFact(record.Manifest.Facts, FactKindBoundary, "network.interfaces") || hasFact(record.Manifest.Facts, FactKindGuard, "network.interfaces") {
		t.Fatalf("modeled boundary owners = %v", capabilityOwners(record.Manifest.Facts, "network.interfaces"))
	}
}

func TestPinnedToolchainGuardsForbiddenPackageExceptModeledBoundary(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("live capability records are supported only on darwin/arm64")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(moduleRoot, ".toolchain", "bin", "go")
	goRoot := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOROOT"))
	buildKey := filepath.Base(goRoot)
	goVersion := strings.TrimSpace(runToolchainCommand(t, moduleRoot, goCommand, "env", "GOVERSION"))
	expectation := Expectation{GoVersion: goVersion, ToolchainBuildKey: buildKey, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}

	modeledTarget := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "os/user")
func main() { current, err := user.Current(); fmt.Printf("%t %t\n", current == nil, err != nil) }
`)
	modeledRecord, err := Read(modeledTarget, expectation)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFact(modeledRecord.Manifest.Facts, FactKindBoundary, "user.current") || hasFact(modeledRecord.Manifest.Facts, FactKindGuard, "import:os/user") {
		t.Fatalf("modeled os/user boundary facts = %#v", modeledRecord.Manifest.Facts)
	}

	guardedTarget := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "os/user")
func main() { _, _ = user.Lookup("gomad"); fmt.Println("after") }
`)
	guardedRecord, err := Read(guardedTarget, expectation)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFact(guardedRecord.Manifest.Facts, FactKindGuard, "import:os/user") {
		t.Fatalf("unmodeled os/user boundary facts = %#v", guardedRecord.Manifest.Facts)
	}
	command := exec.Command(guardedTarget)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "GOMAD_CAPABILITY_DENIED") || strings.Contains(string(output), "after") {
		t.Fatalf("Gomad unmodeled os/user boundary = %v: %s", err, output)
	}

	signalTarget := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "os"; "os/signal")
func main() { signal.Stop(make(chan os.Signal, 1)); fmt.Println("after") }
`)
	signalRecord, err := Read(signalTarget, expectation)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFact(signalRecord.Manifest.Facts, FactKindBoundary, "signal.stop") || hasFact(signalRecord.Manifest.Facts, FactKindGuard, "import:os/signal") {
		t.Fatalf("modeled os/signal boundary facts = %#v", signalRecord.Manifest.Facts)
	}
	command = exec.Command(signalTarget)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err = command.CombinedOutput()
	if err != nil || string(output) != "after\n" {
		t.Fatalf("Gomad modeled os/signal boundary = %v: %s", err, output)
	}

	guardedSignalTarget := buildGuardedCapabilityFixture(t, goCommand, buildKey, `package main
import ("fmt"; "os"; "os/signal")
func main() { signal.Notify(make(chan os.Signal, 1)); fmt.Println("after") }
`)
	guardedSignalRecord, err := Read(guardedSignalTarget, expectation)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFact(guardedSignalRecord.Manifest.Facts, FactKindGuard, "import:os/signal") {
		t.Fatalf("unmodeled os/signal boundary facts = %#v", guardedSignalRecord.Manifest.Facts)
	}
	command = exec.Command(guardedSignalTarget)
	command.Env = append(os.Environ(), "GOMADSEED=1")
	output, err = command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "GOMAD_CAPABILITY_DENIED") || strings.Contains(string(output), "after") {
		t.Fatalf("Gomad unmodeled os/signal boundary = %v: %s", err, output)
	}
}

func buildLiveCapabilityFixture(t *testing.T, goCommand, buildKey, source string) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/livecapfixture\n\ngo 1.26\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(directory, "target")
	runToolchainCommand(t, directory, goCommand, "build", "-trimpath", "-gcflags=all=-gomadcap", "-ldflags=-linkmode=internal -gomadcap="+buildKey, "-o", target, ".")
	return target
}

func buildGuardedCapabilityFixture(t *testing.T, goCommand, buildKey, source string) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/guardedcapfixture\n\ngo 1.26\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(directory, "target")
	runToolchainCommand(t, directory, goCommand, "build", "-trimpath", "-gcflags=all=-gomadcap -gomadguard", "-ldflags=-linkmode=internal -gomadcap="+buildKey, "-o", target, ".")
	return target
}

func runToolchainCommand(t *testing.T, directory, command string, arguments ...string) string {
	t.Helper()
	toolchainRoot := filepath.Dir(filepath.Dir(command))
	buildKeyBytes, err := os.ReadFile(filepath.Join(toolchainRoot, "build-key"))
	if err != nil {
		t.Fatal(err)
	}
	buildCache := filepath.Join(toolchainRoot, "builds", strings.TrimSpace(string(buildKeyBytes)), "target-test-cache")
	if err := os.MkdirAll(buildCache, 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	process := exec.CommandContext(ctx, command, arguments...)
	process.Dir = directory
	process.Env = make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GOCACHE=") {
			process.Env = append(process.Env, entry)
		}
	}
	process.Env = append(process.Env, "GOWORK=off", "CGO_ENABLED=0", "GOCACHE="+buildCache)
	output, err := process.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v: %s", command, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func hasCapability(facts []Fact, capability string) bool {
	for _, fact := range facts {
		if fact.Capability == capability {
			return true
		}
	}
	return false
}

func hasFact(facts []Fact, kind FactKind, capability string) bool {
	for _, fact := range facts {
		if fact.Kind == kind && fact.Capability == capability {
			return true
		}
	}
	return false
}

func capabilityOwners(facts []Fact, capability string) []string {
	owners := []string{}
	for _, fact := range facts {
		if fact.Capability == capability {
			owners = append(owners, fact.OwnerPackage+":"+fact.OwnerSymbol+"->"+fact.ReferencedSymbol)
		}
	}
	return owners
}
