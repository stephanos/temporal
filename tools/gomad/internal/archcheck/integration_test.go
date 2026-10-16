package archcheck_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Run the workflow's own gate script against real commits so the routing of
// each path is checked as CI evaluates it.
func TestCIBackendGatesRoutePaths(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "gomad.yml"))
	if err != nil {
		t.Fatal(err)
	}
	_, step, found := strings.Cut(string(workflow), "      - name: Select backend gates\n")
	if !found {
		t.Fatal("workflow has no backend gate step")
	}
	_, step, found = strings.Cut(step, "        run: |\n")
	if !found {
		t.Fatal("backend gate step has no run script")
	}
	var script strings.Builder
	for _, line := range strings.SplitAfter(step, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, strings.Repeat(" ", 10)) {
			break
		}
		script.WriteString(strings.TrimPrefix(line, strings.Repeat(" ", 10)))
	}
	for _, test := range []struct {
		path, want string
	}{
		{"tools/gomad/internal/model/cluster/simulation.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/wasm/backend/provider.go", "native=false\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/build/build.go", "native=true\nwasm=false\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/fs.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/handles.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/local_handles.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/volume.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/hash.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/evidence/record/record.go", "native=true\nwasm=true\n"},
		{"tools/gomad/internal/backends/native/qualification/tests.json", "native=true\nwasm=false\n"},
	} {
		t.Run(test.path, func(t *testing.T) {
			repository := t.TempDir()
			environment := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
				"GIT_AUTHOR_NAME=gomad", "GIT_AUTHOR_EMAIL=gomad@example.com", "GIT_COMMITTER_NAME=gomad", "GIT_COMMITTER_EMAIL=gomad@example.com")
			git := func(arguments ...string) string {
				t.Helper()
				command := exec.Command("git", arguments...)
				command.Dir, command.Env = repository, environment
				output, err := command.CombinedOutput()
				if err != nil {
					t.Fatalf("git %s: %v: %s", strings.Join(arguments, " "), err, output)
				}
				return strings.TrimSpace(string(output))
			}
			git("init", "-q")
			git("commit", "-q", "--allow-empty", "-m", "base")
			base := git("rev-parse", "HEAD")
			if err := os.MkdirAll(filepath.Join(repository, filepath.Dir(test.path)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repository, test.path), []byte("changed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			git("add", ".")
			git("commit", "-q", "-m", "head")
			head := git("rev-parse", "HEAD")
			output := filepath.Join(t.TempDir(), "output")
			command := exec.Command("bash", "-c", script.String())
			command.Dir = repository
			command.Env = append(environment, "BASH_ENV=", "GITHUB_EVENT_NAME=push", "BASE_SHA="+base, "HEAD_SHA="+head, "GITHUB_OUTPUT="+output)
			if diagnostic, err := command.CombinedOutput(); err != nil {
				t.Fatalf("gate script: %v: %s", err, diagnostic)
			}
			got, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("gates = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNativeWrapperFixtureSelectorsResolve(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "internal/backends/native/qualification/integration_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	selectors := 0
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		selector, ok := strings.CutPrefix(value, "GOMAD_PACKAGES=")
		if !ok {
			return true
		}
		selectors++
		files, err := filepath.Glob(filepath.Join(selector, "*_test.go"))
		if err != nil || len(files) == 0 {
			t.Errorf("native wrapper fixture %q selects no test files: %v", selector, err)
		}
		return true
	})
	if selectors != 3 {
		t.Fatalf("native wrapper fixture selectors = %d, want 3", selectors)
	}
}

func TestHostGatePreservesRelocatedPackageSelectors(t *testing.T) {
	output, err := exec.Command("make", "-n", "-o", "toolchain", "test-host").CombinedOutput()
	if err != nil {
		t.Fatalf("expand host gate: %v: %s", err, output)
	}
	root := t.TempDir()
	tool := filepath.Join(root, ".toolchain", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(tool), 0o755); err != nil {
		t.Fatal(err)
	}
	shim := "#!/bin/sh\nprintf '%s\\n' \"GOMADSEED=${GOMADSEED-unset}\" \"GOMAD_CHILD_SEED=${GOMAD_CHILD_SEED-unset}\" \"GOMAD_STOCK_GO=$GOMAD_STOCK_GO\" \"GOEXPERIMENT=$GOEXPERIMENT\" \"GOWORK=$GOWORK\" \"$@\"\n"
	if err := os.WriteFile(tool, []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", "-c", string(output))
	command.Dir = root
	command.Env = append(os.Environ(), "GOMAD_STOCK_GO=/stock/go", "GOMADSEED=17", "GOMAD_CHILD_SEED=23")
	invocation, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run expanded host gate: %v: %s", err, invocation)
	}
	arguments := strings.Split(strings.TrimSpace(string(invocation)), "\n")
	wantPrefix := []string{"GOMADSEED=unset", "GOMAD_CHILD_SEED=unset", "GOMAD_STOCK_GO=/stock/go", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "test", "-tags", "test_dep"}
	if len(arguments) < len(wantPrefix) || !slices.Equal(arguments[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("host gate environment and flags = %q, want prefix %q", arguments, wantPrefix)
	}
	for _, selector := range []string{
		"./cmd/...", "./internal/runner/...", "./internal/backends/native/execution/...", "./internal/model/clock/...",
		"./internal/evidence/qualification/...", "./internal/backends/target/...", "./internal/evidence/record/...", "./internal/evidence/artifact/...", "./internal/model/choice/...",
		"./internal/evidence/artifact/capture/...", "./internal/model/volume/...", "./internal/backends/native/transcript/...", "./internal/model/world/...",
		"./internal/backends/native/toolchain/version", "./internal/backends/native/toolchain/installation",
		"./internal/backends/native/upgrade/...", "./internal/backends/native/preparation/...", "./internal/backends/native/qualification/...",
		"./internal/backends/native/generation/...", "./internal/backends/native/build/...", "./internal/backends/native/cmd/...",
		"./internal/backends/native/compatibilitypack/...", "./internal/backends/native/sourceinventory",
		"./internal/host/...", "./internal/archcheck", "./internal/model/iowire", "./internal/model/strictjson",
	} {
		if !slices.Contains(arguments[len(wantPrefix):], selector) {
			t.Errorf("canonical host gate excludes %s", selector)
		}
	}
}

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
	runner, err := parser.ParseFile(token.NewFileSet(), "internal/runner/internal/exploration/integration_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	registered := make(map[string]bool)
	for _, declaration := range runner.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "TestRootProcessSimulationUsesRunnerTransport" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			loop, ok := node.(*ast.RangeStmt)
			if !ok {
				return true
			}
			list, ok := loop.X.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, element := range list.Elts {
				literal, ok := element.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				name, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				registered[name] = true
			}
			return true
		})
	}
	files, err := filepath.Glob("internal/model/cluster/*toolchain_test.go")
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
			for _, family := range []string{"TestProcessNetworkHandle", "TestProcessFilesystemHandle", "TestProcessNetworkHeldFIFO", "TestProcessNetworkStallExpiry", "TestProcessNetworkStallOrdering", "TestProcessNetworkPartitionedDialStall", "TestProcessNetworkFramedPartition"} {
				if strings.HasPrefix(function.Name.Name, family) {
					prefix = family
				}
			}
			if prefix == "" {
				continue
			}
			handles[prefix]++
			if !registered[function.Name.Name] {
				t.Errorf("Runner simulation transport does not register %s", function.Name.Name)
			}
			if !selected(root+function.Name.Name, filters[0][1]) {
				t.Errorf("canonical main simulation gate excludes %s", function.Name.Name)
			}
			if selected(root+function.Name.Name, filters[1][1]) {
				t.Errorf("process handle case %s leaks into separate forward-delay gate", function.Name.Name)
			}
		}
	}
	for prefix, want := range map[string]int{"TestProcessNetworkHandle": 16, "TestProcessFilesystemHandle": 4, "TestProcessNetworkHeldFIFO": 1, "TestProcessNetworkStallExpiry": 1, "TestProcessNetworkStallOrdering": 1, "TestProcessNetworkPartitionedDialStall": 1, "TestProcessNetworkFramedPartition": 1} {
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
