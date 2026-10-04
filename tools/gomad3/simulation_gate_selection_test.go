package gomad3_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Exercise make's expanded gate commands, not merely the existence of test
// names: the direct root invocation skips tests that need Runner transport.
func TestSimulationGateSelectsProcessNetworkHandles(t *testing.T) {
	output, err := exec.Command("make", "-n", "-o", "toolchain", "test-simulation").CombinedOutput()
	if err != nil {
		t.Fatalf("expand simulation gate: %v: %s", err, output)
	}
	filters := regexp.MustCompile(`-run '([^']+)'`).FindAllStringSubmatch(string(output), -1)
	if len(filters) != 2 {
		t.Fatalf("simulation integration filters = %d, want main and separate forward-delay gates: %s", len(filters), output)
	}
	selected := func(name string, filter string) bool {
		t.Helper()
		names, patterns := strings.Split(name, "/"), strings.Split(filter, "/")
		for index, pattern := range patterns {
			matcher, err := regexp.Compile(pattern)
			if err != nil {
				t.Fatalf("invalid gate filter %q: %v", filter, err)
			}
			if index >= len(names) || !matcher.MatchString(names[index]) {
				return false
			}
		}
		return true
	}
	root := "TestRootProcessSimulationUsesRunnerTransport/"
	files, err := filepath.Glob("../../tools/gomad3sim/network*toolchain_test.go")
	if err != nil {
		t.Fatal(err)
	}
	handles := 0
	for _, path := range files {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(function.Name.Name, "TestProcessNetworkHandle") {
				continue
			}
			handles++
			if !selected(root+function.Name.Name, filters[0][1]) {
				t.Errorf("canonical main simulation gate excludes %s", function.Name.Name)
			}
			if selected(root+function.Name.Name, filters[1][1]) {
				t.Errorf("process handle case %s leaks into separate forward-delay gate", function.Name.Name)
			}
		}
	}
	if handles != 13 {
		t.Fatalf("process network handle cases = %d, want 13", handles)
	}
	delay := root + "TestProcessBackendSynchronizesNodeClockWithModelDelay/"
	for _, filter := range filters {
		if selected(delay+"strict", filter[1]) {
			t.Errorf("canonical gate selects recorded strict-delay watchdog case: %q", filter[1])
		}
	}
	if selected(delay+"forward", filters[0][1]) || !selected(delay+"forward", filters[1][1]) {
		t.Fatal("forward-delay case must retain its separate canonical invocation")
	}
}
