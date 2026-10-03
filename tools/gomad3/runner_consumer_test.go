package gomad3_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRunnerExternalConsumerCompiles builds testdata/runnerconsumer as a
// separate module that replaces this one with the working tree. A consumer
// outside the Runner subtree cannot import runner/internal/execution, so the
// build fails if constructing a request or substituting preparation or replay
// requires an internal type.
func TestRunnerExternalConsumerCompiles(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	module := t.TempDir()
	source, err := os.ReadFile(filepath.Join("testdata", "runnerconsumer", "consumer.go"))
	if err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile("go.sum")
	if err != nil {
		t.Fatal(err)
	}
	goMod := "module example.com/runnerconsumer\n\ngo 1.27.1\n\n" +
		"require " + modulePath + " v0.0.0\n\n" +
		"replace " + modulePath + " => " + root + "\n"
	for name, contents := range map[string][]byte{"consumer.go": source, "go.sum": sums, "go.mod": []byte(goMod)} {
		if err := os.WriteFile(filepath.Join(module, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// -mod=mod lets the build record this module's own requirements;
	// GOPROXY=off keeps it to the module cache this module already uses.
	command := exec.Command("go", "build", "./...")
	command.Dir = module
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("external Runner consumer does not compile: %v\n%s", err, output)
	}
}
