package archcheck_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Run the workflow's own gate script against real commits so the routing of
// each path is checked as CI evaluates it.
func TestCIBackendGatesRoutePaths(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "gomad3.yml"))
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
		{"tools/gomad3sim/simulation.go", "native=true\nwasm=true\n"},
		{"tools/gomad3/backends/wasm/backend/provider.go", "native=false\nwasm=true\n"},
		{"tools/gomad3/backends/native/build/build.go", "native=true\nwasm=false\n"},
		{"tools/gomad3/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go", "native=true\nwasm=true\n"},
		{"tools/gomad3/record/record.go", "native=true\nwasm=true\n"},
		{"tools/gomad3/backends/native/qualification/tests.json", "native=true\nwasm=false\n"},
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
