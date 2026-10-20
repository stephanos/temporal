package archcheck_test

import (
	"os"
	"os/exec"
	"path/filepath"
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
		{"tools/gomad/internal/model/cluster/simulation.go", "wasm=true\n"},
		{"tools/gomad/internal/backends/wasm/backend/provider.go", "wasm=true\n"},
		{"tools/gomad/internal/evidence/record/record.go", "wasm=true\n"},
		{"tools/gomad/internal/model/schema/iowire.json", "wasm=true\n"},
		{"tools/gomad/internal/model/generation/protocol/generate.go", "wasm=true\n"},
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
