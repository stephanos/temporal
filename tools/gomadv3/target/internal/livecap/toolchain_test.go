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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	process := exec.CommandContext(ctx, command, arguments...)
	process.Dir = directory
	process.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
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
