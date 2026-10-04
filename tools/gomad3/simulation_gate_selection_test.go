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
	files, err := filepath.Glob("../../tools/gomad3sim/*toolchain_test.go")
	if err != nil {
		t.Fatal(err)
	}
	handles := make(map[string]int)
	for _, path := range files {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			prefix := ""
			for _, family := range []string{"TestProcessNetworkHandle", "TestProcessFilesystemHandle"} {
				if strings.HasPrefix(function.Name.Name, family) {
					prefix = family
				}
			}
			if prefix == "" {
				continue
			}
			handles[prefix]++
			if !selected(root+function.Name.Name, filters[0][1]) {
				t.Errorf("canonical main simulation gate excludes %s", function.Name.Name)
			}
			if selected(root+function.Name.Name, filters[1][1]) {
				t.Errorf("process handle case %s leaks into separate forward-delay gate", function.Name.Name)
			}
		}
	}
	for prefix, want := range map[string]int{"TestProcessNetworkHandle": 13, "TestProcessFilesystemHandle": 4} {
		if handles[prefix] != want {
			t.Fatalf("%s cases=%d, want %d", prefix, handles[prefix], want)
		}
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
